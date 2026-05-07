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
