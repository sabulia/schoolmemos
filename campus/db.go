// Package campus 是校园版改造新增的独立模块。
// 该模块全部为本项目新增代码（非原 memos 项目代码），通过
// store.Driver.GetDB() 使用独立前缀（campus_）的数据表，
// 不修改原项目任何表结构与接口行为。
package campus

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// dialect 适配 sqlite / postgres / mysql 三种数据库。
type dialect string

const (
	dialectSQLite   dialect = "sqlite"
	dialectPostgres dialect = "postgres"
	dialectMySQL    dialect = "mysql"
)

// rebind 将 ? 占位符转换为对应数据库的占位符风格。
func (d dialect) rebind(query string) string {
	if d != dialectPostgres {
		return query
	}
	var b strings.Builder
	n := 0
	for _, r := range query {
		if r == '?' {
			n++
			fmt.Fprintf(&b, "$%d", n)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (d dialect) autoIncrementPK() string {
	switch d {
	case dialectPostgres:
		return "BIGSERIAL PRIMARY KEY"
	case dialectMySQL:
		return "BIGINT PRIMARY KEY AUTO_INCREMENT"
	default:
		return "INTEGER PRIMARY KEY AUTOINCREMENT"
	}
}

func (d dialect) boolType() string {
	if d == dialectPostgres {
		return "BOOLEAN"
	}
	return "INTEGER" // sqlite / mysql 用 0/1 表示
}

// schemaDDL 返回校园模块全部数据表的建表语句。
// 设计原则：全部使用 campus_ 前缀，与原项目表完全隔离。
func (s *Service) schemaDDL() []string {
	pk := s.dialect.autoIncrementPK()
	boolean := s.dialect.boolType()
	return []string{
		// memo 的校园扩展元数据：匿名标志、仅好友可见、地点、主题分类、关键词、日程。
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS campus_memo_meta (
			memo_id BIGINT PRIMARY KEY,
			anonymous %s NOT NULL DEFAULT 0,
			friends_only %s NOT NULL DEFAULT 0,
			location TEXT NOT NULL DEFAULT '',
			theme TEXT NOT NULL DEFAULT '',
			keywords TEXT NOT NULL DEFAULT '[]',
			schedule_ts BIGINT NOT NULL DEFAULT 0,
			remind_minutes BIGINT NOT NULL DEFAULT 0,
			created_ts BIGINT NOT NULL DEFAULT 0
		)`, boolean, boolean),
		// 好友关系（PENDING / ACCEPTED）。
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS campus_friend (
			id %s,
			user_id BIGINT NOT NULL,
			friend_id BIGINT NOT NULL,
			status TEXT NOT NULL DEFAULT 'PENDING',
			created_ts BIGINT NOT NULL DEFAULT 0
		)`, pk),
		// 浏览记录（仅本人可见，除非用户在隐私设置中公开）。
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS campus_browse (
			id %s,
			user_id BIGINT NOT NULL,
			memo_id BIGINT NOT NULL,
			ts BIGINT NOT NULL DEFAULT 0
		)`, pk),
		// 收藏。
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS campus_favorite (
			user_id BIGINT NOT NULL,
			memo_id BIGINT NOT NULL,
			ts BIGINT NOT NULL DEFAULT 0,
			PRIMARY KEY (user_id, memo_id)
		)`),
		// 用户级校园设置（推荐授权、隐私开关、界面个性化、手机号等）。
		`CREATE TABLE IF NOT EXISTS campus_setting (
			user_id BIGINT NOT NULL,
			key TEXT NOT NULL,
			value TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (user_id, key)
		)`,
		// 短信验证码（注册 / 找回密码）。
		`CREATE TABLE IF NOT EXISTS campus_sms_code (
			phone TEXT NOT NULL,
			code TEXT NOT NULL,
			purpose TEXT NOT NULL,
			expire_ts BIGINT NOT NULL,
			used INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (phone, purpose)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_campus_friend_user ON campus_friend (user_id, status)`,
		`CREATE INDEX IF NOT EXISTS idx_campus_browse_user ON campus_browse (user_id, ts)`,
		`CREATE INDEX IF NOT EXISTS idx_campus_memo_meta_theme ON campus_memo_meta (theme)`,
	}
}

// ensureSchema 在服务启动时创建校园模块数据表（幂等）。
func (s *Service) ensureSchema(ctx context.Context) error {
	db := s.Store.GetDriver().GetDB()
	for _, ddl := range s.schemaDDL() {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			return fmt.Errorf("campus schema init failed: %w", err)
		}
	}
	return nil
}

// exec / query 是带方言占位符转换的小工具。
func (s *Service) exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return s.Store.GetDriver().GetDB().ExecContext(ctx, s.dialect.rebind(query), args...)
}

func (s *Service) query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return s.Store.GetDriver().GetDB().QueryContext(ctx, s.dialect.rebind(query), args...)
}

func (s *Service) queryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return s.Store.GetDriver().GetDB().QueryRowContext(ctx, s.dialect.rebind(query), args...)
}
