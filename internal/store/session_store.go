package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dingjingmaster/agent-daemon/internal/core"
	_ "modernc.org/sqlite"
)

type SessionStore struct {
	db *sql.DB
}

func NewSessionStore(dbPath string) (*SessionStore, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	s := &SessionStore{db: db}
	if err := s.init(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *SessionStore) init() error {
	schema := `
CREATE TABLE IF NOT EXISTS messages (
id INTEGER PRIMARY KEY AUTOINCREMENT,
session_id TEXT NOT NULL,
role TEXT NOT NULL,
content TEXT,
name TEXT,
tool_call_id TEXT,
tool_calls_json TEXT,
created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_messages_session_id ON messages(session_id);
CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at);

CREATE TABLE IF NOT EXISTS approvals (
id INTEGER PRIMARY KEY AUTOINCREMENT,
session_id TEXT NOT NULL,
scope TEXT NOT NULL DEFAULT 'session',
pattern TEXT NOT NULL DEFAULT '',
granted_at TEXT NOT NULL,
expires_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_approvals_session_id ON approvals(session_id);
CREATE INDEX IF NOT EXISTS idx_approvals_expires_at ON approvals(expires_at);

CREATE TABLE IF NOT EXISTS oauth_tokens (
id INTEGER PRIMARY KEY AUTOINCREMENT,
provider TEXT NOT NULL,
access_token TEXT NOT NULL,
refresh_token TEXT NOT NULL DEFAULT '',
expires_at TEXT NOT NULL,
created_at TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_oauth_tokens_provider ON oauth_tokens(provider);
`
	_, err := s.db.Exec(schema)
	return err
}

func (s *SessionStore) Close() error { return s.db.Close() }

func (s *SessionStore) AppendMessage(sessionID string, msg core.Message) error {
	callsJSON := ""
	if len(msg.ToolCalls) > 0 {
		b, _ := json.Marshal(msg.ToolCalls)
		callsJSON = string(b)
	}
	_, err := s.db.Exec(`INSERT INTO messages(session_id, role, content, name, tool_call_id, tool_calls_json, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, sessionID, msg.Role, msg.Content, msg.Name, msg.ToolCallID, callsJSON, time.Now().Format(time.RFC3339))
	return err
}

func (s *SessionStore) LoadMessages(sessionID string, limit int) ([]core.Message, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.db.Query(`SELECT role, content, name, tool_call_id, tool_calls_json FROM messages WHERE session_id = ? ORDER BY id ASC LIMIT ?`, sessionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]core.Message, 0)
	for rows.Next() {
		var m core.Message
		var tcJSON string
		if err := rows.Scan(&m.Role, &m.Content, &m.Name, &m.ToolCallID, &tcJSON); err != nil {
			return nil, err
		}
		if strings.TrimSpace(tcJSON) != "" {
			_ = json.Unmarshal([]byte(tcJSON), &m.ToolCalls)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *SessionStore) Search(query string, limit int, sessionID string) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 5
	}
	like := "%" + query + "%"
	baseSQL := `SELECT session_id, role, content, created_at FROM messages WHERE content LIKE ?`
	args := []any{like}
	if strings.TrimSpace(sessionID) != "" {
		baseSQL += ` AND session_id <> ?`
		args = append(args, sessionID)
	}
	baseSQL += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(baseSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("search query failed: %w", err)
	}
	defer rows.Close()
	results := make([]map[string]any, 0)
	for rows.Next() {
		var sid, role, content, createdAt string
		if err := rows.Scan(&sid, &role, &content, &createdAt); err != nil {
			return nil, err
		}
		results = append(results, map[string]any{"session_id": sid, "role": role, "content": content, "created_at": createdAt})
	}
	return results, rows.Err()
}

type ApprovalRecord struct {
	ID        int64
	SessionID string
	Scope     string
	Pattern   string
	GrantedAt time.Time
	ExpiresAt time.Time
}

func (s *SessionStore) GrantApproval(sessionID, scope, pattern string, expiresAt time.Time) error {
	_, err := s.db.Exec(`INSERT INTO approvals(session_id, scope, pattern, granted_at, expires_at) VALUES (?, ?, ?, ?, ?)`,
		sessionID, scope, pattern, time.Now().Format(time.RFC3339), expiresAt.Format(time.RFC3339))
	return err
}

func (s *SessionStore) RevokeApproval(sessionID, scope, pattern string) (bool, error) {
	query := `DELETE FROM approvals WHERE session_id = ?`
	args := []any{sessionID}
	if scope != "" {
		query += ` AND scope = ?`
		args = append(args, scope)
	}
	if pattern != "" {
		query += ` AND pattern = ?`
		args = append(args, pattern)
	}
	result, err := s.db.Exec(query, args...)
	if err != nil {
		return false, err
	}
	n, _ := result.RowsAffected()
	return n > 0, nil
}

func (s *SessionStore) ListApprovals(sessionID string) ([]ApprovalRecord, error) {
	now := time.Now().Format(time.RFC3339)
	rows, err := s.db.Query(`SELECT id, session_id, scope, pattern, granted_at, expires_at FROM approvals WHERE session_id = ? AND expires_at > ? ORDER BY id ASC`, sessionID, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []ApprovalRecord
	for rows.Next() {
		var r ApprovalRecord
		var grantedAt, expiresAt string
		if err := rows.Scan(&r.ID, &r.SessionID, &r.Scope, &r.Pattern, &grantedAt, &expiresAt); err != nil {
			return nil, err
		}
		r.GrantedAt, _ = time.Parse(time.RFC3339, grantedAt)
		r.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
		records = append(records, r)
	}
	return records, rows.Err()
}

func (s *SessionStore) IsApproved(sessionID, scope, pattern string) (bool, error) {
	now := time.Now().Format(time.RFC3339)
	if scope == "session" {
		var count int
		err := s.db.QueryRow(`SELECT COUNT(*) FROM approvals WHERE session_id = ? AND scope = 'session' AND expires_at > ?`, sessionID, now).Scan(&count)
		return count > 0, err
	}
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM approvals WHERE session_id = ? AND scope = ? AND pattern = ? AND expires_at > ?`, sessionID, scope, pattern, now).Scan(&count)
	return count > 0, err
}

func (s *SessionStore) CleanupExpiredApprovals() (int64, error) {
	now := time.Now().Format(time.RFC3339)
	result, err := s.db.Exec(`DELETE FROM approvals WHERE expires_at <= ?`, now)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *SessionStore) SaveOAuthToken(provider, accessToken, refreshToken string, expiresAt time.Time) error {
	now := time.Now().Format(time.RFC3339)
	exp := expiresAt.Format(time.RFC3339)
	_, err := s.db.Exec(`INSERT INTO oauth_tokens(provider, access_token, refresh_token, expires_at, created_at) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(provider) DO UPDATE SET access_token=excluded.access_token, refresh_token=excluded.refresh_token, expires_at=excluded.expires_at`,
		provider, accessToken, refreshToken, exp, now)
	return err
}

func (s *SessionStore) LoadOAuthToken(provider string) (accessToken, refreshToken string, expiresAt time.Time, err error) {
	var expStr string
	err = s.db.QueryRow(`SELECT access_token, refresh_token, expires_at FROM oauth_tokens WHERE provider = ?`, provider).Scan(&accessToken, &refreshToken, &expStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", time.Time{}, nil
		}
		return "", "", time.Time{}, err
	}
	parsed, parseErr := time.Parse(time.RFC3339, expStr)
	if parseErr != nil {
		return accessToken, refreshToken, time.Time{}, nil
	}
	return accessToken, refreshToken, parsed, nil
}

func (s *SessionStore) DeleteOAuthToken(provider string) error {
	_, err := s.db.Exec(`DELETE FROM oauth_tokens WHERE provider = ?`, provider)
	return err
}
