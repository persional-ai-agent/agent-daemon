package tools

import (
	"sync"
	"time"

	"github.com/dingjingmaster/agent-daemon/internal/store"
)

type ApprovalStore struct {
	mu         sync.Mutex
	items      map[string]time.Time
	patterns   map[string]map[string]time.Time
	defaultTTL time.Duration
	persistent *store.SessionStore
}

func NewApprovalStore(defaultTTL time.Duration) *ApprovalStore {
	if defaultTTL <= 0 {
		defaultTTL = 5 * time.Minute
	}
	return &ApprovalStore{
		items:      map[string]time.Time{},
		patterns:   map[string]map[string]time.Time{},
		defaultTTL: defaultTTL,
	}
}

func NewPersistentApprovalStore(defaultTTL time.Duration, s *store.SessionStore) *ApprovalStore {
	a := NewApprovalStore(defaultTTL)
	a.persistent = s
	return a
}

func (s *ApprovalStore) Grant(sessionID string, ttl time.Duration) time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ttl <= 0 {
		ttl = s.defaultTTL
	}
	expiresAt := time.Now().Add(ttl)
	s.items[sessionID] = expiresAt
	if s.persistent != nil {
		_ = s.persistent.GrantApproval(sessionID, "session", "", expiresAt)
	}
	return expiresAt
}

func (s *ApprovalStore) GrantPattern(sessionID, pattern string, ttl time.Duration) time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ttl <= 0 {
		ttl = s.defaultTTL
	}
	expiresAt := time.Now().Add(ttl)
	if s.patterns[sessionID] == nil {
		s.patterns[sessionID] = map[string]time.Time{}
	}
	s.patterns[sessionID][pattern] = expiresAt
	if s.persistent != nil {
		_ = s.persistent.GrantApproval(sessionID, "pattern", pattern, expiresAt)
	}
	return expiresAt
}

func (s *ApprovalStore) Revoke(sessionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.items[sessionID]
	delete(s.items, sessionID)
	delete(s.patterns, sessionID)
	if s.persistent != nil {
		_, _ = s.persistent.RevokeApproval(sessionID, "", "")
	}
	return ok
}

func (s *ApprovalStore) RevokePattern(sessionID, pattern string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	patterns, ok := s.patterns[sessionID]
	if ok {
		_, ok = patterns[pattern]
		delete(patterns, pattern)
	}
	if s.persistent != nil {
		_, _ = s.persistent.RevokeApproval(sessionID, "pattern", pattern)
	}
	return ok
}

func (s *ApprovalStore) IsApproved(sessionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	expiresAt, ok := s.items[sessionID]
	if ok {
		if time.Now().After(expiresAt) {
			delete(s.items, sessionID)
			return false
		}
		return true
	}
	if s.persistent != nil {
		approved, err := s.persistent.IsApproved(sessionID, "session", "")
		if err == nil && approved {
			s.items[sessionID] = time.Time{}
			return true
		}
	}
	return false
}

func (s *ApprovalStore) IsApprovedPattern(sessionID, pattern string) bool {
	if s.IsApproved(sessionID) {
		return true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	patterns, ok := s.patterns[sessionID]
	if ok {
		expiresAt, ok2 := patterns[pattern]
		if ok2 {
			if time.Now().After(expiresAt) {
				delete(patterns, pattern)
				return false
			}
			return true
		}
	}
	if s.persistent != nil {
		approved, err := s.persistent.IsApproved(sessionID, "pattern", pattern)
		if err == nil && approved {
			if s.patterns[sessionID] == nil {
				s.patterns[sessionID] = map[string]time.Time{}
			}
			s.patterns[sessionID][pattern] = time.Time{}
			return true
		}
	}
	return false
}

func (s *ApprovalStore) Status(sessionID string) (approved bool, expiresAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	expiresAt, ok := s.items[sessionID]
	if !ok {
		return false, time.Time{}
	}
	if time.Now().After(expiresAt) {
		delete(s.items, sessionID)
		return false, time.Time{}
	}
	return true, expiresAt
}

func (s *ApprovalStore) ListApprovals(sessionID string) []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []map[string]any
	if expiresAt, ok := s.items[sessionID]; ok {
		if !time.Now().After(expiresAt) {
			result = append(result, map[string]any{
				"scope":      "session",
				"pattern":    "",
				"expires_at": expiresAt.Format(time.RFC3339),
			})
		}
	}
	for pattern, expiresAt := range s.patterns[sessionID] {
		if !time.Now().After(expiresAt) {
			result = append(result, map[string]any{
				"scope":      "pattern",
				"pattern":    pattern,
				"expires_at": expiresAt.Format(time.RFC3339),
			})
		}
	}
	if s.persistent != nil && len(result) == 0 {
		records, err := s.persistent.ListApprovals(sessionID)
		if err == nil {
			for _, r := range records {
				result = append(result, map[string]any{
					"scope":      r.Scope,
					"pattern":    r.Pattern,
					"expires_at": r.ExpiresAt.Format(time.RFC3339),
				})
			}
		}
	}
	return result
}

func (s *ApprovalStore) LoadFromStore(sessionID string) {
	if s.persistent == nil {
		return
	}
	records, err := s.persistent.ListApprovals(sessionID)
	if err != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range records {
		if r.Scope == "session" {
			s.items[sessionID] = r.ExpiresAt
		} else if r.Scope == "pattern" {
			if s.patterns[sessionID] == nil {
				s.patterns[sessionID] = map[string]time.Time{}
			}
			s.patterns[sessionID][r.Pattern] = r.ExpiresAt
		}
	}
}
