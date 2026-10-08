package campus

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/lithammer/shortuuid/v4"

	"github.com/usememos/memos/store"
)

// ============================================================
// 校园 memo 扩展：匿名发布、仅好友可见、地点、日程、主题分类。
// 数据存储：核心字段仍走原项目 memo 表；校园扩展字段存 campus_memo_meta。
// ============================================================

// 校园版可见性：在原项目三档（PRIVATE/PROTECTED/PUBLIC）之上增加 FRIENDS（仅好友可见）。
const (
	VisibilityPrivate   = "PRIVATE"   // 仅自己可见
	VisibilityFriends   = "FRIENDS"   // 仅好友可见（校园版新增）
	VisibilityProtected = "PROTECTED" // 登录可见
	VisibilityPublic    = "PUBLIC"    // 完全公开
)

// memoMeta 是 campus_memo_meta 表的内存表示。
type memoMeta struct {
	MemoID        int32
	Anonymous     bool
	FriendsOnly   bool
	Location      string
	Theme         string
	Keywords      []Keyword
	ScheduleTs    int64
	RemindMinutes int64
}

// memoCard 是校园各信息流统一的卡片结构。
type memoCard struct {
	UID           string    `json:"uid"`
	Content       string    `json:"content"`
	CreateTime    time.Time `json:"createTime"`
	UpdateTime    time.Time `json:"updateTime"`
	Visibility    string    `json:"visibility"`
	Pinned        bool      `json:"pinned"`
	Anonymous     bool      `json:"anonymous"`
	Location      string    `json:"location,omitempty"`
	Theme         string    `json:"theme"`
	Keywords      []string  `json:"keywords"`
	ScheduleTs    int64     `json:"scheduleTs,omitempty"`
	RemindMinutes int64     `json:"remindMinutes,omitempty"`
	Owned         bool      `json:"owned"`
	Creator       *userLite `json:"creator,omitempty"`
}

type userLite struct {
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatarUrl,omitempty"`
}

// ---- 工具：meta 读写 ----

func (s *Service) saveMeta(ctx context.Context, m *memoMeta) error {
	kws, err := json.Marshal(m.Keywords)
	if err != nil {
		return err
	}
	// 幂等写入：方言间冲突语法不同（MySQL 用 INSERT IGNORE，sqlite/postgres 用 ON CONFLICT DO NOTHING）。
	query := `INSERT INTO campus_memo_meta
		(memo_id, anonymous, friends_only, location, theme, keywords, schedule_ts, remind_minutes, created_ts)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (memo_id) DO NOTHING`
	if s.dialect == dialectMySQL {
		query = `INSERT IGNORE INTO campus_memo_meta
			(memo_id, anonymous, friends_only, location, theme, keywords, schedule_ts, remind_minutes, created_ts)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	}
	_, err = s.exec(ctx, query,
		m.MemoID, m.Anonymous, m.FriendsOnly, m.Location, m.Theme, string(kws), m.ScheduleTs, m.RemindMinutes, time.Now().Unix())
	return err
}

func (s *Service) loadMetaMap(ctx context.Context, memoIDs []int32) (map[int32]*memoMeta, error) {
	out := map[int32]*memoMeta{}
	if len(memoIDs) == 0 {
		return out, nil
	}
	placeholders := make([]string, len(memoIDs))
	args := make([]any, len(memoIDs))
	for i, id := range memoIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	rows, err := s.query(ctx, `SELECT memo_id, anonymous, friends_only, location, theme, keywords, schedule_ts, remind_minutes
		FROM campus_memo_meta WHERE memo_id IN (`+strings.Join(placeholders, ",")+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		m := &memoMeta{}
		var kws string
		if err := rows.Scan(&m.MemoID, &m.Anonymous, &m.FriendsOnly, &m.Location, &m.Theme, &kws, &m.ScheduleTs, &m.RemindMinutes); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(kws), &m.Keywords)
		out[m.MemoID] = m
	}
	return out, rows.Err()
}

// ---- 工具：好友关系判断 ----

func (s *Service) isFriend(ctx context.Context, userID, otherID int32) bool {
	var cnt int64
	if err := s.queryRow(ctx, `SELECT COUNT(1) FROM campus_friend
		WHERE user_id = ? AND friend_id = ? AND status = 'ACCEPTED'`, userID, otherID).Scan(&cnt); err != nil {
		return false
	}
	return cnt > 0
}

// canView 服务端统一鉴权：匿名与「仅好友可见」都在这里强制校验。
func (s *Service) canView(ctx context.Context, memo *store.Memo, meta *memoMeta, user *store.User) bool {
	if user != nil && memo.CreatorID == user.ID {
		return true
	}
	switch memo.Visibility {
	case store.Public:
		return true
	case store.Protected:
		return user != nil
	case store.Private:
		// 校园版「仅好友可见」底层存 PRIVATE + friends_only 标志。
		if meta != nil && meta.FriendsOnly && user != nil {
			return s.isFriend(ctx, user.ID, memo.CreatorID)
		}
		return false
	}
	return false
}

// toCard 将 memo + meta 转为卡片；匿名内容在此隐藏作者信息。
func (s *Service) toCard(memo *store.Memo, meta *memoMeta, viewer *store.User, creator *store.User) *memoCard {
	vis := string(memo.Visibility)
	card := &memoCard{
		UID:        memo.UID,
		Content:    memo.Content,
		CreateTime: time.Unix(memo.CreatedTs, 0),
		UpdateTime: time.Unix(memo.UpdatedTs, 0),
		Visibility: vis,
		Pinned:     memo.Pinned,
	}
	if meta != nil {
		card.Anonymous = meta.Anonymous
		card.Location = meta.Location
		card.Theme = meta.Theme
		card.ScheduleTs = meta.ScheduleTs
		card.RemindMinutes = meta.RemindMinutes
		if meta.FriendsOnly {
			card.Visibility = VisibilityFriends
		}
		for _, k := range meta.Keywords {
			card.Keywords = append(card.Keywords, k.Text)
		}
	}
	owned := viewer != nil && memo.CreatorID == viewer.ID
	card.Owned = owned
	switch {
	case card.Anonymous && !owned:
		// 匿名：浏览者不可见任何作者信息。
		card.Creator = &userLite{Username: "", Nickname: "匿名同学"}
	case creator != nil:
		card.Creator = &userLite{Username: creator.Username, Nickname: creator.Nickname, AvatarURL: creator.AvatarURL}
	}
	return card
}

// ---- 创建 memo（校园版发布器） ----

type createMemoRequest struct {
	Content       string `json:"content"`
	Visibility    string `json:"visibility"` // PRIVATE / FRIENDS / PROTECTED / PUBLIC
	Anonymous     bool   `json:"anonymous"`
	Location      string `json:"location"`
	ScheduleTs    int64  `json:"scheduleTs"`
	RemindMinutes int64  `json:"remindMinutes"`
}

func (s *Service) handleCreateMemo(c *echo.Context) error {
	user, err := s.requireUser(c)
	if err != nil {
		return err
	}
	req := &createMemoRequest{}
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
	}
	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "content is required")
	}

	// 可见性映射：FRIENDS 底层落 PRIVATE，由校园模块做好友鉴权。
	coreVis := store.Visibility(req.Visibility)
	friendsOnly := false
	switch req.Visibility {
	case VisibilityFriends:
		coreVis = store.Private
		friendsOnly = true
	case VisibilityPrivate:
		coreVis = store.Private
	case VisibilityProtected:
		coreVis = store.Protected
	case VisibilityPublic:
		coreVis = store.Public
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "invalid visibility: must be PRIVATE/FRIENDS/PROTECTED/PUBLIC")
	}

	memo, err := s.Store.CreateMemo(c.Request().Context(), &store.Memo{
		UID:        shortuuid.New(),
		CreatorID:  user.ID,
		Content:    req.Content,
		Visibility: coreVis,
	})
	if err != nil {
		return err
	}

	// 关键词提取 + 主题分类（创建时自动完成）。
	keywords := ExtractKeywords(req.Content, 8)
	theme, _ := Classify(req.Content, keywords)

	meta := &memoMeta{
		MemoID:        memo.ID,
		Anonymous:     req.Anonymous,
		FriendsOnly:   friendsOnly,
		Location:      strings.TrimSpace(req.Location),
		Theme:         theme,
		Keywords:      keywords,
		ScheduleTs:    req.ScheduleTs,
		RemindMinutes: req.RemindMinutes,
	}
	if err := s.saveMeta(c.Request().Context(), meta); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, s.toCard(memo, meta, user, user))
}

// ---- 信息流 ----

func (s *Service) handleFeed(c *echo.Context) error {
	user, err := s.currentUser(c)
	if err != nil {
		return err
	}
	box := c.QueryParam("box")
	switch box {
	case "circle":
		return s.feedCircle(c, user)
	case "friends":
		if user == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "login required")
		}
		return s.feedFriends(c, user)
	case "topics":
		return s.feedTopics(c, user)
	case "theme":
		return s.feedTheme(c, user)
	case "recommend":
		if user == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "login required")
		}
		return s.feedRecommend(c, user)
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "unknown feed box")
	}
}

// buildCards 统一做「权限过滤 + meta 挂载 + 匿名脱敏」。
func (s *Service) buildCards(ctx context.Context, memos []*store.Memo, viewer *store.User) ([]*memoCard, error) {
	ids := make([]int32, 0, len(memos))
	for _, m := range memos {
		ids = append(ids, m.ID)
	}
	metaMap, err := s.loadMetaMap(ctx, ids)
	if err != nil {
		return nil, err
	}
	cards := make([]*memoCard, 0, len(memos))
	for _, m := range memos {
		meta := metaMap[m.ID]
		if !s.canView(ctx, m, meta, viewer) {
			continue
		}
		creator, _ := s.Store.GetUser(ctx, &store.FindUser{ID: &m.CreatorID})
		cards = append(cards, s.toCard(m, meta, viewer, creator))
	}
	return cards, nil
}

// feedCircle 圈：全校 公开+登录可见 动态（仿朋友圈/QQ 动态）。
func (s *Service) feedCircle(c *echo.Context, user *store.User) error {
	ctx := c.Request().Context()
	limit := 100
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
	cards, err := s.buildCards(ctx, memos, user)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"memos": cards})
}

// feedFriends 好友动态：好友的公开/登录可见 memo + 好友发给我的「仅好友可见」memo。
func (s *Service) feedFriends(c *echo.Context, user *store.User) error {
	ctx := c.Request().Context()
	friendIDs, err := s.acceptedFriendIDs(ctx, user.ID)
	if err != nil {
		return err
	}
	cards := []*memoCard{}
	for _, fid := range friendIDs {
		limit := 50
		memos, err := s.Store.ListMemos(ctx, &store.FindMemo{
			RowStatus:       ptrOf(store.Normal),
			CreatorID:       &fid,
			VisibilityList:  []store.Visibility{store.Public, store.Protected, store.Private},
			ExcludeComments: true,
			Limit:           &limit,
		})
		if err != nil {
			return err
		}
		built, err := s.buildCards(ctx, memos, user) // canView 内部完成「仅好友可见」校验
		if err != nil {
			return err
		}
		cards = append(cards, built...)
	}
	sort.Slice(cards, func(i, j int) bool { return cards[i].CreateTime.After(cards[j].CreateTime) })
	if len(cards) > 100 {
		cards = cards[:100]
	}
	return c.JSON(http.StatusOK, map[string]any{"memos": cards})
}

// feedTopics 话题区：按 #话题 聚合（仿贴吧/小红书）。
func (s *Service) feedTopics(c *echo.Context, user *store.User) error {
	ctx := c.Request().Context()
	tag := strings.TrimPrefix(strings.TrimSpace(c.QueryParam("tag")), "#")
	if tag == "" {
		// 话题列表：统计各主题/关键词热度。
		return s.handleTopicsIndex(c, user)
	}
	limit := 200
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
	filtered := make([]*store.Memo, 0)
	needle := "#" + tag
	for _, m := range memos {
		if strings.Contains(m.Content, needle) {
			filtered = append(filtered, m)
		}
	}
	cards, err := s.buildCards(ctx, filtered, user)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"memos": cards, "tag": tag})
}

// feedTheme 主题分类：按关键词相关性归入的主题浏览（仿番茄小说分类书架）。
func (s *Service) feedTheme(c *echo.Context, user *store.User) error {
	ctx := c.Request().Context()
	theme := c.QueryParam("theme")
	if theme == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "theme is required")
	}
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
	ids := make([]int32, 0, len(memos))
	byID := map[int32]*store.Memo{}
	for _, m := range memos {
		ids = append(ids, m.ID)
		byID[m.ID] = m
	}
	metaMap, err := s.loadMetaMap(ctx, ids)
	if err != nil {
		return err
	}
	filtered := make([]*store.Memo, 0)
	for id, meta := range metaMap {
		if meta.Theme == theme {
			filtered = append(filtered, byID[id])
		}
	}
	cards, err := s.buildCards(ctx, filtered, user)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"memos": cards, "theme": theme})
}

func ptrOf[T any](v T) *T { return &v }
