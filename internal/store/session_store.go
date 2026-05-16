package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

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

func (s *SessionStore) DB() *sql.DB { return s.db }

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

-- Full-text search index for message content.
-- If FTS5 is unavailable in the SQLite build, these statements will fail and we fall back to LIKE queries.
CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
  content,
  session_id UNINDEXED,
  role UNINDEXED,
  created_at UNINDEXED,
  content='messages',
  content_rowid='id'
);

CREATE TRIGGER IF NOT EXISTS messages_ai AFTER INSERT ON messages BEGIN
  INSERT INTO messages_fts(rowid, content, session_id, role, created_at)
  VALUES (new.id, COALESCE(new.content, ''), new.session_id, new.role, new.created_at);
END;

CREATE TRIGGER IF NOT EXISTS messages_ad AFTER DELETE ON messages BEGIN
  INSERT INTO messages_fts(messages_fts, rowid, content, session_id, role, created_at)
  VALUES('delete', old.id, COALESCE(old.content, ''), old.session_id, old.role, old.created_at);
END;

CREATE TRIGGER IF NOT EXISTS messages_au AFTER UPDATE ON messages BEGIN
  INSERT INTO messages_fts(messages_fts, rowid, content, session_id, role, created_at)
  VALUES('delete', old.id, COALESCE(old.content, ''), old.session_id, old.role, old.created_at);
  INSERT INTO messages_fts(rowid, content, session_id, role, created_at)
  VALUES (new.id, COALESCE(new.content, ''), new.session_id, new.role, new.created_at);
END;

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

CREATE TABLE IF NOT EXISTS session_summaries (
session_id TEXT PRIMARY KEY,
summary TEXT NOT NULL DEFAULT '',
keywords_json TEXT NOT NULL DEFAULT '[]',
facts_json TEXT NOT NULL DEFAULT '[]',
message_count INTEGER NOT NULL DEFAULT 0,
last_message_id INTEGER NOT NULL DEFAULT 0,
updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_session_summaries_updated_at ON session_summaries(updated_at);
`
	if _, err := s.db.Exec(schema); err != nil {
		// FTS5 might be unavailable; fall back to the core tables only.
		coreSchema := `
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

CREATE TABLE IF NOT EXISTS session_summaries (
session_id TEXT PRIMARY KEY,
summary TEXT NOT NULL DEFAULT '',
keywords_json TEXT NOT NULL DEFAULT '[]',
facts_json TEXT NOT NULL DEFAULT '[]',
message_count INTEGER NOT NULL DEFAULT 0,
last_message_id INTEGER NOT NULL DEFAULT 0,
updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_session_summaries_updated_at ON session_summaries(updated_at);
`
		_, coreErr := s.db.Exec(coreSchema)
		if coreErr != nil {
			return err
		}
	}
	return nil
}

func (s *SessionStore) Close() error { return s.db.Close() }

func (s *SessionStore) ListRecentSessions(limit int) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.Query(`
SELECT session_id, MAX(id) AS last_id, MAX(created_at) AS last_seen
FROM messages
GROUP BY session_id
ORDER BY last_id DESC
LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]map[string]any, 0)
	for rows.Next() {
		var sessionID string
		var lastID int64
		var lastSeen string
		if err := rows.Scan(&sessionID, &lastID, &lastSeen); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"session_id": sessionID,
			"last_id":    lastID,
			"last_seen":  lastSeen,
		})
	}
	return out, rows.Err()
}

func (s *SessionStore) LoadMessagesPage(sessionID string, offset, limit int) ([]core.Message, error) {
	if limit <= 0 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.db.Query(`SELECT role, content, name, tool_call_id, tool_calls_json FROM messages WHERE session_id = ? ORDER BY id ASC LIMIT ? OFFSET ?`, sessionID, limit, offset)
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

// CompactSession keeps only the latest keepLastN messages for a session and deletes older rows.
func (s *SessionStore) CompactSession(sessionID string, keepLastN int) (before int, after int, err error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return 0, 0, fmt.Errorf("session_id required")
	}
	if keepLastN <= 0 {
		keepLastN = 20
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM messages WHERE session_id = ?`, sessionID).Scan(&before); err != nil {
		return 0, 0, err
	}
	if before <= keepLastN {
		return before, before, nil
	}
	if _, err := s.db.Exec(`
DELETE FROM messages
WHERE session_id = ?
  AND id NOT IN (
    SELECT id FROM messages
    WHERE session_id = ?
    ORDER BY id DESC
    LIMIT ?
  )`, sessionID, sessionID, keepLastN); err != nil {
		return 0, 0, err
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM messages WHERE session_id = ?`, sessionID).Scan(&after); err != nil {
		return 0, 0, err
	}
	_, _ = s.db.Exec(`DELETE FROM session_summaries WHERE session_id = ?`, sessionID)
	return before, after, nil
}

func (s *SessionStore) SessionStats(sessionID string) (map[string]any, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, fmt.Errorf("session_id required")
	}
	var messageCount int64
	var toolCallCount int64
	var firstSeen string
	var lastSeen string
	err := s.db.QueryRow(`
SELECT
  COUNT(*) AS message_count,
  COALESCE(SUM(CASE WHEN tool_calls_json IS NOT NULL AND tool_calls_json <> '' THEN 1 ELSE 0 END), 0) AS tool_call_messages,
  COALESCE(MIN(created_at), '') AS first_seen,
  COALESCE(MAX(created_at), '') AS last_seen
FROM messages
WHERE session_id = ?`, sessionID).Scan(&messageCount, &toolCallCount, &firstSeen, &lastSeen)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"session_id":         sessionID,
		"message_count":      messageCount,
		"tool_call_messages": toolCallCount,
		"first_seen":         firstSeen,
		"last_seen":          lastSeen,
	}, nil
}

// DeleteSession removes all persisted records for a session.
func (s *SessionStore) DeleteSession(sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("session_id required")
	}
	if _, err := s.db.Exec(`DELETE FROM messages WHERE session_id = ?`, sessionID); err != nil {
		return err
	}
	if _, err := s.db.Exec(`DELETE FROM approvals WHERE session_id = ?`, sessionID); err != nil {
		return err
	}
	if _, err := s.db.Exec(`DELETE FROM session_summaries WHERE session_id = ?`, sessionID); err != nil {
		return err
	}
	return nil
}

func (s *SessionStore) Search(query string, limit int, sessionID string) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 5
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return s.recentSessionSummaries(limit, sessionID)
	}
	if s.hasFTS() {
		return s.searchFTS(query, limit, sessionID)
	}
	return s.searchLike(query, limit, sessionID)
}

func (s *SessionStore) hasFTS() bool {
	var count int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='messages_fts'`).Scan(&count)
	return count > 0
}

func (s *SessionStore) searchFTS(query string, limit int, sessionID string) ([]map[string]any, error) {
	matchQuery := buildFTSQuery(query)
	if matchQuery == "" {
		return s.searchLike(query, limit, sessionID)
	}
	baseSQL := `
SELECT m.session_id, COUNT(*) AS match_count, MAX(m.id) AS last_id, MAX(m.created_at) AS last_seen
FROM messages m
JOIN messages_fts ON messages_fts.rowid = m.id
WHERE messages_fts MATCH ?`
	args := []any{matchQuery}
	if strings.TrimSpace(sessionID) != "" {
		baseSQL += ` AND m.session_id <> ?`
		args = append(args, sessionID)
	}
	baseSQL += ` GROUP BY m.session_id ORDER BY match_count DESC, last_id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(baseSQL, args...)
	if err != nil {
		// If the build doesn't support MATCH, fall back.
		return s.searchLike(query, limit, sessionID)
	}
	return s.scanSessionSearchRows(rows, query)
}

func (s *SessionStore) searchLike(query string, limit int, sessionID string) ([]map[string]any, error) {
	like := "%" + query + "%"
	baseSQL := `
SELECT session_id, COUNT(*) AS match_count, MAX(id) AS last_id, MAX(created_at) AS last_seen
FROM messages
WHERE content LIKE ?`
	args := []any{like}
	if strings.TrimSpace(sessionID) != "" {
		baseSQL += ` AND session_id <> ?`
		args = append(args, sessionID)
	}
	baseSQL += ` GROUP BY session_id ORDER BY match_count DESC, last_id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(baseSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("search query failed: %w", err)
	}
	return s.scanSessionSearchRows(rows, query)
}

func (s *SessionStore) recentSessionSummaries(limit int, excludeSessionID string) ([]map[string]any, error) {
	baseSQL := `
SELECT session_id, COUNT(*) AS match_count, MAX(id) AS last_id, MAX(created_at) AS last_seen
FROM messages`
	args := []any{}
	if strings.TrimSpace(excludeSessionID) != "" {
		baseSQL += ` WHERE session_id <> ?`
		args = append(args, excludeSessionID)
	}
	baseSQL += ` GROUP BY session_id ORDER BY last_id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.Query(baseSQL, args...)
	if err != nil {
		return nil, err
	}
	return s.scanSessionSearchRows(rows, "")
}

func (s *SessionStore) scanSessionSearchRows(rows *sql.Rows, query string) ([]map[string]any, error) {
	type searchGroup struct {
		sessionID  string
		lastSeen   string
		matchCount int64
		lastID     int64
	}
	groups := make([]searchGroup, 0)
	for rows.Next() {
		var g searchGroup
		if err := rows.Scan(&g.sessionID, &g.matchCount, &g.lastID, &g.lastSeen); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	results := make([]map[string]any, 0)
	for _, g := range groups {
		summary, err := s.ensureSessionSummary(g.sessionID, g.lastID)
		if err != nil {
			return nil, err
		}
		highlights, _ := s.sessionHighlights(g.sessionID, query, 3)
		result := map[string]any{
			"session_id":    g.sessionID,
			"summary":       summary.Summary,
			"keywords":      summary.Keywords,
			"facts":         summary.Facts,
			"message_count": summary.MessageCount,
			"match_count":   g.matchCount,
			"last_seen":     g.lastSeen,
			"highlights":    highlights,
		}
		// Backward-compatible row-level fields for older callers.
		if len(highlights) > 0 {
			result["role"] = highlights[0]["role"]
			result["content"] = highlights[0]["content"]
			result["created_at"] = highlights[0]["created_at"]
		}
		results = append(results, result)
	}
	return results, nil
}

type SessionSummary struct {
	SessionID     string
	Summary       string
	Keywords      []string
	Facts         []string
	MessageCount  int64
	LastMessageID int64
	UpdatedAt     string
}

type sessionMessageRecord struct {
	id        int64
	role      string
	content   string
	createdAt string
}

func (s *SessionStore) ensureSessionSummary(sessionID string, lastMessageID int64) (SessionSummary, error) {
	var cached SessionSummary
	var keywordsJSON, factsJSON string
	err := s.db.QueryRow(`SELECT session_id, summary, keywords_json, facts_json, message_count, last_message_id, updated_at FROM session_summaries WHERE session_id = ?`, sessionID).
		Scan(&cached.SessionID, &cached.Summary, &keywordsJSON, &factsJSON, &cached.MessageCount, &cached.LastMessageID, &cached.UpdatedAt)
	if err == nil && cached.LastMessageID >= lastMessageID {
		_ = json.Unmarshal([]byte(keywordsJSON), &cached.Keywords)
		_ = json.Unmarshal([]byte(factsJSON), &cached.Facts)
		return cached, nil
	}
	if err != nil && err != sql.ErrNoRows {
		return SessionSummary{}, err
	}
	summary, err := s.BuildSessionSummary(sessionID)
	if err != nil {
		return SessionSummary{}, err
	}
	return summary, s.SaveSessionSummary(summary)
}

func (s *SessionStore) BuildSessionSummary(sessionID string) (SessionSummary, error) {
	rows, err := s.db.Query(`SELECT id, role, content, created_at FROM messages WHERE session_id = ? ORDER BY id ASC`, sessionID)
	if err != nil {
		return SessionSummary{}, err
	}
	defer rows.Close()

	records := make([]sessionMessageRecord, 0)
	allText := strings.Builder{}
	for rows.Next() {
		var r sessionMessageRecord
		if err := rows.Scan(&r.id, &r.role, &r.content, &r.createdAt); err != nil {
			return SessionSummary{}, err
		}
		records = append(records, r)
		if strings.TrimSpace(r.content) != "" {
			allText.WriteString(" ")
			allText.WriteString(r.content)
		}
	}
	if err := rows.Err(); err != nil {
		return SessionSummary{}, err
	}
	if len(records) == 0 {
		return SessionSummary{SessionID: sessionID, Summary: "(empty session)", UpdatedAt: time.Now().Format(time.RFC3339)}, nil
	}
	firstUser := firstContentByRole(records, "user")
	lastUser := lastContentByRole(records, "user")
	lastAssistant := lastContentByRole(records, "assistant")
	parts := []string{fmt.Sprintf("Conversation with %d messages.", len(records))}
	if firstUser != "" {
		parts = append(parts, "Initial user request: "+trimRunes(oneLine(firstUser), 220))
	}
	if lastUser != "" && lastUser != firstUser {
		parts = append(parts, "Latest user request: "+trimRunes(oneLine(lastUser), 220))
	}
	if lastAssistant != "" {
		parts = append(parts, "Latest assistant response: "+trimRunes(oneLine(lastAssistant), 260))
	}
	text := allText.String()
	last := records[len(records)-1]
	return SessionSummary{
		SessionID:     sessionID,
		Summary:       strings.Join(parts, " "),
		Keywords:      extractKeywords(text, 12),
		Facts:         extractFactLines(text, 8),
		MessageCount:  int64(len(records)),
		LastMessageID: last.id,
		UpdatedAt:     time.Now().Format(time.RFC3339),
	}, nil
}

func (s *SessionStore) SaveSessionSummary(summary SessionSummary) error {
	keywordsJSON, _ := json.Marshal(summary.Keywords)
	factsJSON, _ := json.Marshal(summary.Facts)
	if strings.TrimSpace(summary.UpdatedAt) == "" {
		summary.UpdatedAt = time.Now().Format(time.RFC3339)
	}
	_, err := s.db.Exec(`INSERT INTO session_summaries(session_id, summary, keywords_json, facts_json, message_count, last_message_id, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(session_id) DO UPDATE SET summary=excluded.summary, keywords_json=excluded.keywords_json, facts_json=excluded.facts_json, message_count=excluded.message_count, last_message_id=excluded.last_message_id, updated_at=excluded.updated_at`,
		summary.SessionID, summary.Summary, string(keywordsJSON), string(factsJSON), summary.MessageCount, summary.LastMessageID, summary.UpdatedAt)
	return err
}

func (s *SessionStore) sessionHighlights(sessionID, query string, limit int) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 3
	}
	query = strings.TrimSpace(query)
	args := []any{sessionID}
	sqlText := `SELECT role, content, created_at FROM messages WHERE session_id = ?`
	if query != "" {
		sqlText += ` AND content LIKE ?`
		args = append(args, "%"+query+"%")
	}
	sqlText += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.Query(sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]map[string]any, 0)
	for rows.Next() {
		var role, content, createdAt string
		if err := rows.Scan(&role, &content, &createdAt); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"role":       role,
			"content":    content,
			"snippet":    trimRunes(oneLine(content), 240),
			"created_at": createdAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) > 0 || query == "" {
		return out, nil
	}
	return s.sessionHighlights(sessionID, "", limit)
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

func buildFTSQuery(query string) string {
	tokens := tokenizeSearch(query)
	if len(tokens) == 0 {
		return ""
	}
	if len(tokens) == 1 {
		return quoteFTSToken(tokens[0])
	}
	quoted := make([]string, 0, len(tokens))
	for _, token := range tokens {
		quoted = append(quoted, quoteFTSToken(token))
	}
	return strings.Join(quoted, " OR ")
}

func quoteFTSToken(token string) string {
	token = strings.ReplaceAll(token, `"`, `""`)
	return `"` + token + `"`
}

func tokenizeSearch(s string) []string {
	fields := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_' || r == '-' || r == '.')
	})
	out := make([]string, 0, len(fields))
	seen := map[string]struct{}{}
	for _, f := range fields {
		f = strings.Trim(f, "._-")
		if len([]rune(f)) < 2 {
			continue
		}
		if _, skip := searchStopwords[f]; skip {
			continue
		}
		if _, ok := seen[f]; ok {
			continue
		}
		seen[f] = struct{}{}
		out = append(out, f)
	}
	return out
}

var searchStopwords = map[string]struct{}{
	"the": {}, "and": {}, "for": {}, "with": {}, "that": {}, "this": {}, "from": {}, "have": {}, "what": {}, "when": {}, "where": {}, "about": {}, "into": {}, "your": {}, "you": {}, "are": {}, "was": {}, "were": {}, "has": {}, "had": {},
	"一个": {}, "这个": {}, "那个": {}, "以及": {}, "然后": {}, "如果": {}, "我们": {}, "你们": {},
}

func firstContentByRole(records []sessionMessageRecord, role string) string {
	for _, r := range records {
		if r.role == role && strings.TrimSpace(r.content) != "" {
			return r.content
		}
	}
	return ""
}

func lastContentByRole(records []sessionMessageRecord, role string) string {
	for i := len(records) - 1; i >= 0; i-- {
		if records[i].role == role && strings.TrimSpace(records[i].content) != "" {
			return records[i].content
		}
	}
	return ""
}

func extractKeywords(text string, limit int) []string {
	tokens := tokenizeSearch(text)
	counts := map[string]int{}
	for _, token := range tokens {
		counts[token]++
	}
	type kv struct {
		key string
		n   int
	}
	items := make([]kv, 0, len(counts))
	for key, n := range counts {
		items = append(items, kv{key: key, n: n})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].n == items[j].n {
			return items[i].key < items[j].key
		}
		return items[i].n > items[j].n
	})
	if limit <= 0 || limit > len(items) {
		limit = len(items)
	}
	out := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, items[i].key)
	}
	return out
}

func extractFactLines(text string, limit int) []string {
	sentences := splitSentences(text)
	out := make([]string, 0, limit)
	seen := map[string]struct{}{}
	for _, sentence := range sentences {
		lower := strings.ToLower(sentence)
		if !looksLikeMemoryFact(lower) {
			continue
		}
		fact := trimRunes(oneLine(sentence), 220)
		if fact == "" {
			continue
		}
		key := strings.ToLower(fact)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, fact)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func splitSentences(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool {
		return r == '\n' || r == '.' || r == '!' || r == '?' || r == '。' || r == '！' || r == '？' || r == ';' || r == '；'
	})
}

func looksLikeMemoryFact(lower string) bool {
	patterns := []string{
		"i prefer", "i like", "i want", "my name", "my project", "we use", "we need", "project uses", "user prefers", "user likes",
		"我喜欢", "我希望", "我需要", "我的", "我们使用", "项目使用", "用户偏好", "用户喜欢",
	}
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func trimRunes(s string, limit int) string {
	s = strings.TrimSpace(s)
	if limit <= 0 {
		return s
	}
	r := []rune(s)
	if len(r) <= limit {
		return s
	}
	return string(r[:limit]) + "..."
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
