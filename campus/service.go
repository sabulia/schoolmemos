package campus

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/usememos/memos/internal/profile"
	"github.com/usememos/memos/server/auth"
	"github.com/usememos/memos/store"
)

// Service 是校园版功能模块的 HTTP 服务。
// 全部接口挂载在 /api/campus 前缀下，与原项目的 /api/v1 完全隔离。
type Service struct {
	Store         *store.Store
	Profile       *profile.Profile
	Authenticator *auth.Authenticator
	dialect       dialect
}

// NewService 创建校园模块服务并初始化数据表。
func NewService(ctx context.Context, profile *profile.Profile, st *store.Store, secret string) (*Service, error) {
	d := dialectSQLite
	switch profile.Driver {
	case "postgres":
		d = dialectPostgres
	case "mysql":
		d = dialectMySQL
	}
	s := &Service{
		Store:         st,
		Profile:       profile,
		Authenticator: auth.NewAuthenticator(st, secret),
		dialect:       d,
	}
	if err := s.ensureSchema(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

// RegisterRoutes 注册校园模块全部路由。
func (s *Service) RegisterRoutes(e *echo.Echo) {
	g := e.Group("/api/campus")

	g.GET("/ping", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok", "module": "campus"})
	})

	// 校园注册 / 找回密码（短信验证，无需登录）。
	g.POST("/auth/sms", s.handleSendSMS)
	g.POST("/auth/register", s.handleRegister)
	g.POST("/auth/reset-password", s.handleResetPassword)

	// memo 校园扩展。
	g.POST("/memos", s.handleCreateMemo)
	g.GET("/feed", s.handleFeed)
	g.GET("/themes", s.handleThemes)
	g.POST("/reindex", s.handleReindex)
	g.POST("/memos/:uid/browse", s.handleBrowse)
	g.POST("/memos/:uid/favorite", s.handleToggleFavorite)
	g.GET("/favorites", s.handleFavorites)
	g.GET("/history", s.handleHistory)
	g.GET("/mine", s.handleMine)

	// 好友。
	g.GET("/friends", s.handleListFriends)
	g.POST("/friends/request", s.handleFriendRequest)
	g.POST("/friends/accept", s.handleFriendAccept)
	g.DELETE("/friends/:friendId", s.handleFriendDelete)

	// 设置（推荐授权 / 隐私开关 / 界面个性化）。
	g.GET("/settings", s.handleGetSettings)
	g.PUT("/settings", s.handlePutSettings)

	// 数据导出与日程。
	g.GET("/export.json", s.handleExportJSON)
	g.GET("/export.csv", s.handleExportCSV)
	g.GET("/stats", s.handleStats)
	g.GET("/calendar.ics", s.handleCalendarICS)
}

// currentUser 复用原项目的 refresh-token 会话机制解析当前登录用户。
// 未登录返回 nil, nil。
func (s *Service) currentUser(c *echo.Context) (*store.User, error) {
	authHeader := c.Request().Header.Get("Authorization")
	cookieHeader := c.Request().Header.Get("Cookie")
	return s.Authenticator.AuthenticateToUser(c.Request().Context(), authHeader, cookieHeader)
}

// requireUser 要求登录，否则返回 401。
func (s *Service) requireUser(c *echo.Context) (*store.User, error) {
	user, err := s.currentUser(c)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "login required")
	}
	return user, nil
}
