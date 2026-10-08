package campus

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/usememos/memos/store"
)

// ============================================================
// 推荐信息流：经用户明示同意后，基于点赞/收藏/浏览记录做
// 「关键词相关性」推荐，推送至「推荐」一栏。
// ============================================================

const settingRecommendOptIn = "recommend_optin"

// userKeywordProfile 汇总用户行为涉及 memo 的关键词，构成兴趣画像。
func (s *Service) userKeywordProfile(ctx context.Context, userID int32) (map[string]float64, error) {
	profile := map[string]float64{}
	collect := func(memoIDs []int32, weight float64) error {
		metaMap, err := s.loadMetaMap(ctx, memoIDs)
		if err != nil {
			return err
		}
		for _, meta := range metaMap {
			for _, kw := range meta.Keywords {
				profile[kw.Text] += kw.Weight * weight
			}
		}
		return nil
	}

	// 1) 点赞（reaction）过的 memo，权重最高。
	reactions, err := s.Store.ListReactions(ctx, &store.FindReaction{CreatorID: &userID})
	if err != nil {
		return nil, err
	}
	reactedIDs := make([]int32, 0, len(reactions))
	for _, r := range reactions {
		reactedIDs = append(reactedIDs, r.MemoID)
	}
	if err := collect(reactedIDs, 3.0); err != nil {
		return nil, err
	}

	// 2) 收藏的 memo。
	favIDs, err := s.favoriteMemoIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := collect(favIDs, 2.5); err != nil {
		return nil, err
	}

	// 3) 浏览记录（近 200 条）。
	browseIDs, err := s.recentBrowseMemoIDs(ctx, userID, 200)
	if err != nil {
		return nil, err
	}
	if err := collect(browseIDs, 1.0); err != nil {
		return nil, err
	}
	return profile, nil
}

// feedRecommend 推荐信息流。
func (s *Service) feedRecommend(c *echo.Context, user *store.User) error {
	ctx := c.Request().Context()

	// 未经明示同意，不做个性化推荐：前端据此弹窗询问。
	optIn, err := s.getSetting(ctx, user.ID, settingRecommendOptIn)
	if err != nil {
		return err
	}
	if optIn != "1" {
		return c.JSON(http.StatusOK, map[string]any{"needOptIn": true, "memos": []any{}})
	}

	profile, err := s.userKeywordProfile(ctx, user.ID)
	if err != nil {
		return err
	}

	limit := 300
	memos, err := s.Store.ListMemos(ctx, &store.FindMemo{
		RowStatus:       ptrOf(store.Normal),
		VisibilityList:  []store.Visibility{store.Public, store.Protected},
		ExcludeComments: true,
		Limit:           &limit,
	})
	if err != nil {
		return err
	}
	ids := make([]int32, 0, len(memos))
	for _, m := range memos {
		ids = append(ids, m.ID)
	}
	metaMap, err := s.loadMetaMap(ctx, ids)
	if err != nil {
		return err
	}

	type scored struct {
		memo  *store.Memo
		score float64
	}
	scoredList := make([]scored, 0, len(memos))
	for _, m := range memos {
		if m.CreatorID == user.ID {
			continue // 不推荐自己的内容
		}
		meta := metaMap[m.ID]
		if !s.canView(ctx, m, meta, user) {
			continue
		}
		score := 0.0
		if meta != nil {
			for _, kw := range meta.Keywords {
				if w, ok := profile[kw.Text]; ok {
					score += w * kw.Weight
				}
			}
		}
		// 时间衰减：越新略有加成。
		ageHours := time.Since(time.Unix(m.CreatedTs, 0)).Hours()
		score += 5.0 / (1.0 + ageHours/24.0)
		scoredList = append(scoredList, scored{memo: m, score: score})
	}
	sort.Slice(scoredList, func(i, j int) bool { return scoredList[i].score > scoredList[j].score })

	top := scoredList
	if len(top) > 50 {
		top = top[:50]
	}
	memoList := make([]*store.Memo, 0, len(top))
	for _, sc := range top {
		memoList = append(memoList, sc.memo)
	}
	cards, err := s.buildCards(ctx, memoList, user)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"needOptIn": false, "memos": cards})
}

// ---- 浏览记录 ----

func (s *Service) handleBrowse(c *echo.Context) error {
	user, err := s.requireUser(c)
	if err != nil {
		return err
	}
	memo, err := s.findMemoByUID(c, c.Param("uid"))
	if err != nil {
		return err
	}
	if _, err := s.exec(c.Request().Context(),
		`INSERT INTO campus_browse (user_id, memo_id, ts) VALUES (?, ?, ?)`,
		user.ID, memo.ID, time.Now().Unix()); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

func (s *Service) recentBrowseMemoIDs(ctx context.Context, userID int32, limit int) ([]int32, error) {
	rows, err := s.query(ctx, `SELECT memo_id FROM campus_browse WHERE user_id = ? ORDER BY ts DESC LIMIT ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []int32{}
	for rows.Next() {
		var id int32
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// handleHistory 我的浏览记录（仅本人可见；隐私设置可决定是否公开到个人主页）。
func (s *Service) handleHistory(c *echo.Context) error {
	user, err := s.requireUser(c)
	if err != nil {
		return err
	}
	ctx := c.Request().Context()
	ids, err := s.recentBrowseMemoIDs(ctx, user.ID, 100)
	if err != nil {
		return err
	}
	memos := make([]*store.Memo, 0, len(ids))
	for _, id := range ids {
		m, err := s.Store.GetMemo(ctx, &store.FindMemo{ID: &id})
		if err == nil && m != nil {
			memos = append(memos, m)
		}
	}
	cards, err := s.buildCards(ctx, memos, user)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"memos": cards})
}

// ---- 收藏 ----

func (s *Service) handleToggleFavorite(c *echo.Context) error {
	user, err := s.requireUser(c)
	if err != nil {
		return err
	}
	ctx := c.Request().Context()
	memo, err := s.findMemoByUID(c, c.Param("uid"))
	if err != nil {
		return err
	}
	var cnt int64
	if err := s.queryRow(ctx, `SELECT COUNT(1) FROM campus_favorite WHERE user_id = ? AND memo_id = ?`, user.ID, memo.ID).Scan(&cnt); err != nil {
		return err
	}
	favorited := false
	if cnt > 0 {
		if _, err := s.exec(ctx, `DELETE FROM campus_favorite WHERE user_id = ? AND memo_id = ?`, user.ID, memo.ID); err != nil {
			return err
		}
	} else {
		if _, err := s.exec(ctx, `INSERT INTO campus_favorite (user_id, memo_id, ts) VALUES (?, ?, ?)`, user.ID, memo.ID, time.Now().Unix()); err != nil {
			return err
		}
		favorited = true
	}
	return c.JSON(http.StatusOK, map[string]bool{"favorited": favorited})
}

func (s *Service) favoriteMemoIDs(ctx context.Context, userID int32) ([]int32, error) {
	rows, err := s.query(ctx, `SELECT memo_id FROM campus_favorite WHERE user_id = ? ORDER BY ts DESC LIMIT 200`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []int32{}
	for rows.Next() {
		var id int32
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *Service) handleFavorites(c *echo.Context) error {
	user, err := s.requireUser(c)
	if err != nil {
		return err
	}
	ctx := c.Request().Context()
	ids, err := s.favoriteMemoIDs(ctx, user.ID)
	if err != nil {
		return err
	}
	memos := make([]*store.Memo, 0, len(ids))
	for _, id := range ids {
		m, err := s.Store.GetMemo(ctx, &store.FindMemo{ID: &id})
		if err == nil && m != nil && s.canView(ctx, m, nil, user) {
			memos = append(memos, m)
		}
	}
	cards, err := s.buildCards(ctx, memos, user)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"memos": cards})
}

// handleMine 我的 memo（含匿名与仅好友可见，作者本人全量可见）。
func (s *Service) handleMine(c *echo.Context) error {
	user, err := s.requireUser(c)
	if err != nil {
		return err
	}
	ctx := c.Request().Context()
	limit := 200
	memos, err := s.Store.ListMemos(ctx, &store.FindMemo{
		RowStatus:       ptrOf(store.Normal),
		CreatorID:       &user.ID,
		ExcludeComments: true,
		Limit:           &limit,
	})
	if err != nil {
		return err
	}
	cards, err := s.buildCards(ctx, memos, user)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"memos": cards})
}

// ---- 主题分类索引与话题索引 ----

func (s *Service) handleThemes(c *echo.Context) error {
	ctx := c.Request().Context()
	user, _ := s.currentUser(c)
	limit := 500
	vis := []store.Visibility{store.Public}
	if user != nil {
		vis = append(vis, store.Protected)
	}
	memos, err := s.Store.ListMemos(ctx, &store.FindMemo{
		RowStatus:       ptrOf(store.Normal),
		VisibilityList:  vis,
		ExcludeComments: true,
		ExcludeContent:  true,
		Limit:           &limit,
	})
	if err != nil {
		return err
	}
	ids := make([]int32, 0, len(memos))
	for _, m := range memos {
		ids = append(ids, m.ID)
	}
	metaMap, err := s.loadMetaMap(ctx, ids)
	if err != nil {
		return err
	}
	counts := map[string]int{}
	kwCounts := map[string]map[string]int{}
	for _, meta := range metaMap {
		if meta.Theme == "" {
			continue
		}
		counts[meta.Theme]++
		if kwCounts[meta.Theme] == nil {
			kwCounts[meta.Theme] = map[string]int{}
		}
		for _, kw := range meta.Keywords {
			kwCounts[meta.Theme][kw.Text]++
		}
	}
	type themeInfo struct {
		Name     string   `json:"name"`
		Count    int      `json:"count"`
		Keywords []string `json:"keywords"`
	}
	themes := []themeInfo{}
	for _, t := range Themes {
		// 每个主题取热度最高的 5 个关键词作为「分类词条」。
		type kv struct {
			k string
			v int
		}
		pairs := []kv{}
		for k, v := range kwCounts[t.Name] {
			pairs = append(pairs, kv{k, v})
		}
		sort.Slice(pairs, func(i, j int) bool { return pairs[i].v > pairs[j].v })
		topKws := []string{}
		for i, p := range pairs {
			if i >= 5 {
				break
			}
			topKws = append(topKws, p.k)
		}
		themes = append(themes, themeInfo{Name: t.Name, Count: counts[t.Name], Keywords: topKws})
	}
	return c.JSON(http.StatusOK, map[string]any{"themes": themes})
}

// handleTopicsIndex 话题区首页：按 #话题 出现频次排出热榜。
func (s *Service) handleTopicsIndex(c *echo.Context, user *store.User) error {
	ctx := c.Request().Context()
	limit := 300
	vis := []store.Visibility{store.Public}
	if user != nil {
		vis = append(vis, store.Protected)
	}
	memos, err := s.Store.ListMemos(ctx, &store.FindMemo{
		RowStatus:       ptrOf(store.Normal),
		VisibilityList:  vis,
		ExcludeComments: true,
		Limit:           &limit,
	})
	if err != nil {
		return err
	}
	tagCounts := map[string]int{}
	for _, m := range memos {
		for _, match := range tagRegexp.FindAllStringSubmatch(m.Content, -1) {
			tagCounts[match[1]]++
		}
	}
	type tagInfo struct {
		Tag   string `json:"tag"`
		Count int    `json:"count"`
	}
	tags := []tagInfo{}
	for t, n := range tagCounts {
		tags = append(tags, tagInfo{Tag: t, Count: n})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Count > tags[j].Count })
	if len(tags) > 50 {
		tags = tags[:50]
	}
	return c.JSON(http.StatusOK, map[string]any{"tags": tags})
}

// handleReindex 重建全站关键词索引（管理员）。
func (s *Service) handleReindex(c *echo.Context) error {
	user, err := s.requireUser(c)
	if err != nil {
		return err
	}
	if user.Role != store.RoleAdmin {
		return echo.NewHTTPError(http.StatusForbidden, "admin only")
	}
	ctx := c.Request().Context()
	limit := 10000
	memos, err := s.Store.ListMemos(ctx, &store.FindMemo{
		RowStatus: ptrOf(store.Normal),
		Limit:     &limit,
	})
	if err != nil {
		return err
	}
	count := 0
	for _, m := range memos {
		keywords := ExtractKeywords(m.Content, 8)
		theme, _ := Classify(m.Content, keywords)
		if _, err := s.exec(ctx, `UPDATE campus_memo_meta SET theme = ?, keywords = ? WHERE memo_id = ?`,
			theme, mustJSON(keywords), m.ID); err != nil {
			return err
		}
		count++
	}
	return c.JSON(http.StatusOK, map[string]any{"reindexed": count})
}

// findMemoByUID 解析 uid 并校验查看权限。
func (s *Service) findMemoByUID(c *echo.Context, uid string) (*store.Memo, error) {
	memo, err := s.Store.GetMemo(c.Request().Context(), &store.FindMemo{UID: &uid})
	if err != nil {
		return nil, err
	}
	if memo == nil {
		return nil, echo.NewHTTPError(http.StatusNotFound, "memo not found")
	}
	return memo, nil
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
