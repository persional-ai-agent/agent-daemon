package tools

import (
	"sync"
	"time"
)

type ApprovalStore struct {
	mu         sync.Mutex
	items      map[string]time.Time
	defaultTTL time.Duration
}

func NewApprovalStore(defaultTTL time.Duration) *ApprovalStore {
	if defaultTTL <= 0 {
		defaultTTL = 5 * time.Minute
	}
	return &ApprovalStore{
		items:      map[string]time.Time{},
		defaultTTL: defaultTTL,
	}
}

func (s *ApprovalStore) Grant(sessionID string, ttl time.Duration) time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ttl <= 0 {
		ttl = s.defaultTTL
	}
	expiresAt := time.Now().Add(ttl)
	s.items[sessionID] = expiresAt
	return expiresAt
}

func (s *ApprovalStore) Revoke(sessionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.items[sessionID]
	delete(s.items, sessionID)
	return ok
}

func (s *ApprovalStore) IsApproved(sessionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	expiresAt, ok := s.items[sessionID]
	if !ok {
		return false
	}
	if time.Now().After(expiresAt) {
		delete(s.items, sessionID)
		return false
	}
	return true
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
