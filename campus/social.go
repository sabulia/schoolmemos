package campus

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/usememos/memos/store"
)

// ============================================================
// 好友系统（仿微信：搜索添加 → 申请 → 通过 → 好友列表/动态）
// ============================================================

type friendEntry struct {
	ID        int32     `json:"id"`
	User      *userLite `json:"user"`
	Status    string    `json:"status"` // PENDING（待通过）/ ACCEPTED（已互为好友）
	Direction string    `json:"direction"`
	CreatedAt time.Time `json:"createdAt"`
}

// acceptedFriendIDs 返回已互为好友的用户 ID 列表。
func (s *Service) acceptedFriendIDs(ctx context.Context, userID int32) ([]int32, error) {
	rows, err := s.query(ctx, `SELECT friend_id FROM campus_friend WHERE user_id = ? AND status = 'ACCEPTED'`, userID)
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

// handleListFriends 好友列表 + 收到的好友申请。
func (s *Service) handleListFriends(c *echo.Context) error {
	user, err := s.requireUser(c)
	if err != nil {
		return err
	}
	ctx := c.Request().Context()
	rows, err := s.query(ctx, `SELECT id, user_id, friend_id, status, created_ts FROM campus_friend
		WHERE user_id = ? OR (friend_id = ? AND status = 'PENDING') ORDER BY created_ts DESC`, user.ID, user.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	friends := []friendEntry{}
	requests := []friendEntry{}
	for rows.Next() {
		var id, uid, fid int32
		var status string
		var ts int64
		if err := rows.Scan(&id, &uid, &fid, &status, &ts); err != nil {
			return err
		}
		otherID := fid
		direction := "out"
		if fid == user.ID { // 别人发给我的申请
			otherID = uid
			direction = "in"
		}
		other, err := s.Store.GetUser(ctx, &store.FindUser{ID: &otherID})
		if err != nil || other == nil {
			continue
		}
		entry := friendEntry{
			ID:        id,
			User:      &userLite{Username: other.Username, Nickname: other.Nickname, AvatarURL: other.AvatarURL},
			Status:    status,
			Direction: direction,
			CreatedAt: time.Unix(ts, 0),
		}
		if status == "PENDING" && direction == "in" {
			requests = append(requests, entry)
		} else if status == "ACCEPTED" {
			friends = append(friends, entry)
		}
	}
	return c.JSON(http.StatusOK, map[string]any{"friends": friends, "requests": requests})
}

type friendRequestBody struct {
	Username string `json:"username"`
}

// handleFriendRequest 按用户名发送好友申请。
func (s *Service) handleFriendRequest(c *echo.Context) error {
	user, err := s.requireUser(c)
	if err != nil {
		return err
	}
	req := &friendRequestBody{}
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
	}
	ctx := c.Request().Context()
	target, err := s.Store.GetUser(ctx, &store.FindUser{Username: &req.Username})
	if err != nil {
		return err
	}
	if target == nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}
	if target.ID == user.ID {
		return echo.NewHTTPError(http.StatusBadRequest, "cannot add yourself")
	}
	// 幂等：已存在记录则直接返回。
	var cnt int64
	if err := s.queryRow(ctx, `SELECT COUNT(1) FROM campus_friend WHERE user_id = ? AND friend_id = ?`, user.ID, target.ID).Scan(&cnt); err != nil {
		return err
	}
	if cnt == 0 {
		if _, err := s.exec(ctx, `INSERT INTO campus_friend (user_id, friend_id, status, created_ts) VALUES (?, ?, 'PENDING', ?)`,
			user.ID, target.ID, time.Now().Unix()); err != nil {
			return err
		}
	}
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

type friendAcceptBody struct {
	Username string `json:"username"` // 申请人的用户名
}

// handleFriendAccept 通过好友申请：双向写入 ACCEPTED。
func (s *Service) handleFriendAccept(c *echo.Context) error {
	user, err := s.requireUser(c)
	if err != nil {
		return err
	}
	req := &friendAcceptBody{}
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
	}
	ctx := c.Request().Context()
	requester, err := s.Store.GetUser(ctx, &store.FindUser{Username: &req.Username})
	if err != nil {
		return err
	}
	if requester == nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}
	// 必须存在指向我的待处理申请。
	var cnt int64
	if err := s.queryRow(ctx, `SELECT COUNT(1) FROM campus_friend WHERE user_id = ? AND friend_id = ? AND status = 'PENDING'`,
		requester.ID, user.ID).Scan(&cnt); err != nil {
		return err
	}
	if cnt == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "no pending request from this user")
	}
	now := time.Now().Unix()
	if _, err := s.exec(ctx, `UPDATE campus_friend SET status = 'ACCEPTED' WHERE user_id = ? AND friend_id = ?`,
		requester.ID, user.ID); err != nil {
		return err
	}
	// 双向关系：写入反向记录（幂等）。
	var rev int64
	if err := s.queryRow(ctx, `SELECT COUNT(1) FROM campus_friend WHERE user_id = ? AND friend_id = ?`, user.ID, requester.ID).Scan(&rev); err != nil {
		return err
	}
	if rev == 0 {
		if _, err := s.exec(ctx, `INSERT INTO campus_friend (user_id, friend_id, status, created_ts) VALUES (?, ?, 'ACCEPTED', ?)`,
			user.ID, requester.ID, now); err != nil {
			return err
		}
	} else {
		if _, err := s.exec(ctx, `UPDATE campus_friend SET status = 'ACCEPTED' WHERE user_id = ? AND friend_id = ?`,
			user.ID, requester.ID); err != nil {
			return err
		}
	}
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

// handleFriendDelete 删除好友（双向）。
func (s *Service) handleFriendDelete(c *echo.Context) error {
	user, err := s.requireUser(c)
	if err != nil {
		return err
	}
	friendID, err := strconv.ParseInt(c.Param("friendId"), 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid friend id")
	}
	ctx := c.Request().Context()
	if _, err := s.exec(ctx, `DELETE FROM campus_friend WHERE (user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)`,
		user.ID, friendID, friendID, user.ID); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}
