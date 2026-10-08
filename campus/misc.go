package campus

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/usememos/memos/store"
)

// ============================================================
// 用户设置（推荐授权 / 隐私开关 / 界面个性化）、数据导出、
// 日程 ICS 订阅、统计汇总（用于向学校汇报与成果展示）。
// ============================================================

// 隐私开关 key：个人主页四类内容是否公开（默认全部不公开）。
var privacyKeys = []string{
	"privacy_show_calendar",  // 日期表
	"privacy_show_memos",     // 用户自己发的 memo
	"privacy_show_history",   // 用户浏览记录
	"privacy_show_favorites", // 用户收藏
}

func (s *Service) getSetting(ctx context.Context, userID int32, key string) (string, error) {
	var v string
	err := s.queryRow(ctx, `SELECT value FROM campus_setting WHERE user_id = ? AND key = ?`, userID, key).Scan(&v)
	if err != nil {
		return "", nil // 未设置视为空
	}
	return v, nil
}

func (s *Service) putSetting(ctx context.Context, userID int32, key, value string) error {
	var query string
	switch s.dialect {
	case dialectMySQL:
		query = `INSERT INTO campus_setting (user_id, key, value) VALUES (?, ?, ?)
			ON DUPLICATE KEY UPDATE value = VALUES(value)`
	default: // sqlite / postgres
		query = `INSERT INTO campus_setting (user_id, key, value) VALUES (?, ?, ?)
			ON CONFLICT (user_id, key) DO UPDATE SET value = excluded.value`
	}
	_, err := s.exec(ctx, query, userID, key, value)
	return err
}

// handleGetSettings 读取当前用户全部校园设置。
func (s *Service) handleGetSettings(c *echo.Context) error {
	user, err := s.requireUser(c)
	if err != nil {
		return err
	}
	rows, err := s.query(c.Request().Context(), `SELECT key, value FROM campus_setting WHERE user_id = ?`, user.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	settings := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return err
		}
		settings[k] = v
	}
	return c.JSON(http.StatusOK, map[string]any{"settings": settings})
}

// handlePutSettings 更新校园设置（仅接受白名单 key，防止注入任意配置）。
func (s *Service) handlePutSettings(c *echo.Context) error {
	user, err := s.requireUser(c)
	if err != nil {
		return err
	}
	body := map[string]string{}
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
	}
	// 注意：phone 仅注册流程可写，这里不开放。
	allowed := map[string]bool{
		settingRecommendOptIn:   true,
		"download_prompt_shown": true, // 注册后「是否下载」只问一次
	}
	for _, k := range privacyKeys {
		allowed[k] = true
	}
	// 界面个性化：section_style_<分区> = default | image:<url> | animated:<名称>
	for _, section := range []string{"friends", "groups", "circle", "topics", "recommend", "themes", "mine"} {
		allowed["section_style_"+section] = true
	}
	ctx := c.Request().Context()
	for k, v := range body {
		if !allowed[k] {
			continue
		}
		if len(v) > 500 {
			return echo.NewHTTPError(http.StatusBadRequest, "setting value too long")
		}
		if err := s.putSetting(ctx, user.ID, k, v); err != nil {
			return err
		}
	}
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

// ---- 数据导出 ----

type exportRow struct {
	UID        string `json:"uid"`
	Creator    string `json:"creator"`
	Anonymous  bool   `json:"anonymous"`
	Visibility string `json:"visibility"`
	Theme      string `json:"theme"`
	Keywords   string `json:"keywords"`
	Location   string `json:"location"`
	CreateTime string `json:"createTime"`
	Content    string `json:"content"`
}

// collectExportRows 收集导出数据：管理员导出全站，普通用户导出自己的。
func (s *Service) collectExportRows(c *echo.Context, user *store.User) ([]exportRow, error) {
	ctx := c.Request().Context()
	limit := 100000
	find := &store.FindMemo{
		RowStatus:       ptrOf(store.Normal),
		ExcludeComments: true,
		Limit:           &limit,
	}
	if user.Role != store.RoleAdmin {
		find.CreatorID = &user.ID
	}
	memos, err := s.Store.ListMemos(ctx, find)
	if err != nil {
		return nil, err
	}
	// 非管理员导出也做一遍权限过滤，双保险。
	if user.Role != store.RoleAdmin {
		filtered := memos[:0]
		for _, m := range memos {
			if m.CreatorID == user.ID {
				filtered = append(filtered, m)
			}
		}
		memos = filtered
	}
	ids := make([]int32, 0, len(memos))
	for _, m := range memos {
		ids = append(ids, m.ID)
	}
	metaMap, err := s.loadMetaMap(ctx, ids)
	if err != nil {
		return nil, err
	}
	rows := make([]exportRow, 0, len(memos))
	for _, m := range memos {
		meta := metaMap[m.ID]
		creatorName := ""
		creator, _ := s.Store.GetUser(ctx, &store.FindUser{ID: &m.CreatorID})
		if creator != nil {
			creatorName = creator.Username
		}
		row := exportRow{
			UID:        m.UID,
			Creator:    creatorName,
			Visibility: string(m.Visibility),
			CreateTime: time.Unix(m.CreatedTs, 0).Format(time.RFC3339),
			Content:    m.Content,
		}
		if meta != nil {
			row.Anonymous = meta.Anonymous
			row.Theme = meta.Theme
			row.Location = meta.Location
			kws := []string{}
			for _, k := range meta.Keywords {
				kws = append(kws, k.Text)
			}
			row.Keywords = strings.Join(kws, ",")
			if meta.FriendsOnly {
				row.Visibility = VisibilityFriends
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func (s *Service) handleExportJSON(c *echo.Context) error {
	user, err := s.requireUser(c)
	if err != nil {
		return err
	}
	rows, err := s.collectExportRows(c, user)
	if err != nil {
		return err
	}
	c.Response().Header().Set("Content-Disposition", `attachment; filename="campus-memos-export.json"`)
	return c.JSON(http.StatusOK, map[string]any{
		"exportedAt": time.Now().Format(time.RFC3339),
		"count":      len(rows),
		"memos":      rows,
	})
}

func (s *Service) handleExportCSV(c *echo.Context) error {
	user, err := s.requireUser(c)
	if err != nil {
		return err
	}
	rows, err := s.collectExportRows(c, user)
	if err != nil {
		return err
	}
	c.Response().Header().Set("Content-Type", "text/csv; charset=utf-8")
	c.Response().Header().Set("Content-Disposition", `attachment; filename="campus-memos-export.csv"`)
	w := csv.NewWriter(c.Response())
	// 写 UTF-8 BOM，方便 Excel 直接打开中文不乱码。
	_, _ = c.Response().Write([]byte{0xEF, 0xBB, 0xBF})
	_ = w.Write([]string{"UID", "作者", "匿名", "可见性", "主题分类", "关键词", "地点", "发布时间", "内容"})
	for _, r := range rows {
		_ = w.Write([]string{r.UID, r.Creator, strconv.FormatBool(r.Anonymous), r.Visibility, r.Theme, r.Keywords, r.Location, r.CreateTime, r.Content})
	}
	w.Flush()
	return nil
}

// handleStats 统计汇总：按主题 / 按日期的 memo 数，用于向学校汇报。
func (s *Service) handleStats(c *echo.Context) error {
	user, err := s.requireUser(c)
	if err != nil {
		return err
	}
	rows, err := s.collectExportRows(c, user)
	if err != nil {
		return err
	}
	byTheme := map[string]int{}
	byDay := map[string]int{}
	for _, r := range rows {
		theme := r.Theme
		if theme == "" {
			theme = DefaultTheme
		}
		byTheme[theme]++
		if len(r.CreateTime) >= 10 {
			byDay[r.CreateTime[:10]]++
		}
	}
	type kv struct {
		Key   string `json:"key"`
		Count int    `json:"count"`
	}
	toSorted := func(m map[string]int) []kv {
		list := []kv{}
		for k, v := range m {
			list = append(list, kv{k, v})
		}
		sort.Slice(list, func(i, j int) bool { return list[i].Count > list[j].Count })
		return list
	}
	return c.JSON(http.StatusOK, map[string]any{
		"total":   len(rows),
		"byTheme": toSorted(byTheme),
		"byDay":   toSorted(byDay),
	})
}

// ---- 日程 ICS 订阅（含 VALARM 到时提醒） ----

func (s *Service) handleCalendarICS(c *echo.Context) error {
	user, err := s.requireUser(c)
	if err != nil {
		return err
	}
	ctx := c.Request().Context()
	rows, err := s.query(ctx, `SELECT m.memo_id, m.schedule_ts, m.remind_minutes FROM campus_memo_meta m
		INNER JOIN memo ON memo.id = m.memo_id
		WHERE memo.creator_id = ? AND m.schedule_ts > 0 AND memo.row_status = 'NORMAL'`, user.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	type ev struct {
		memoID    int32
		ts        int64
		remindMin int64
	}
	events := []ev{}
	for rows.Next() {
		var e ev
		if err := rows.Scan(&e.memoID, &e.ts, &e.remindMin); err != nil {
			return err
		}
		events = append(events, e)
	}
	var b strings.Builder
	b.WriteString("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//campus-memos//CN\r\nCALSCALE:GREGORIAN\r\n")
	for _, e := range events {
		memo, err := s.Store.GetMemo(ctx, &store.FindMemo{ID: &e.memoID})
		if err != nil || memo == nil {
			continue
		}
		start := time.Unix(e.ts, 0).UTC()
		end := start.Add(30 * time.Minute)
		summary := strings.NewReplacer("\r", " ", "\n", " ", ",", "\\,", ";", "\\;").Replace(memo.Content)
		if len([]rune(summary)) > 60 {
			summary = string([]rune(summary)[:60])
		}
		fmt.Fprintf(&b, "BEGIN:VEVENT\r\nUID:%s@campus-memos\r\nDTSTART:%s\r\nDTEND:%s\r\nSUMMARY:%s\r\n",
			memo.UID, start.Format("20060102T150405Z"), end.Format("20060102T150405Z"), summary)
		if e.remindMin > 0 {
			fmt.Fprintf(&b, "BEGIN:VALARM\r\nACTION:DISPLAY\r\nTRIGGER:-PT%dM\r\nDESCRIPTION:日程提醒\r\nEND:VALARM\r\n", e.remindMin)
		}
		b.WriteString("END:VEVENT\r\n")
	}
	b.WriteString("END:VCALENDAR\r\n")
	c.Response().Header().Set("Content-Type", "text/calendar; charset=utf-8")
	c.Response().Header().Set("Content-Disposition", `attachment; filename="campus-schedule.ics"`)
	return c.String(http.StatusOK, b.String())
}
