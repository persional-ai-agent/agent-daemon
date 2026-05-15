package gateway

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type identityStore struct {
	path string
	mu   sync.Mutex
}

type identityRecord struct {
	Platform string `json:"platform"`
	UserID   string `json:"user_id"`
	GlobalID string `json:"global_id"`
}

func newIdentityStore(workdir string) *identityStore {
	return &identityStore{path: filepath.Join(strings.TrimSpace(workdir), ".agent-daemon", "gateway_identity_map.json")}
}

func (s *identityStore) bind(platform, userID, globalID string) error {
	platform = strings.ToLower(strings.TrimSpace(platform))
	userID = strings.TrimSpace(userID)
	globalID = strings.TrimSpace(globalID)
	if platform == "" || userID == "" || globalID == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.loadLocked()
	if err != nil {
		return err
	}
	found := false
	for i := range rows {
		if rows[i].Platform == platform && rows[i].UserID == userID {
			rows[i].GlobalID = globalID
			found = true
			break
		}
	}
	if !found {
		rows = append(rows, identityRecord{Platform: platform, UserID: userID, GlobalID: globalID})
	}
	return s.saveLocked(rows)
}

func (s *identityStore) resolve(platform, userID string) (string, error) {
	platform = strings.ToLower(strings.TrimSpace(platform))
	userID = strings.TrimSpace(userID)
	if platform == "" || userID == "" {
		return "", nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.loadLocked()
	if err != nil {
		return "", err
	}
	for _, r := range rows {
		if r.Platform == platform && r.UserID == userID {
			return strings.TrimSpace(r.GlobalID), nil
		}
	}
	return "", nil
}

func (s *identityStore) unbind(platform, userID string) error {
	platform = strings.ToLower(strings.TrimSpace(platform))
	userID = strings.TrimSpace(userID)
	if platform == "" || userID == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.loadLocked()
	if err != nil {
		return err
	}
	next := make([]identityRecord, 0, len(rows))
	for _, r := range rows {
		if r.Platform == platform && r.UserID == userID {
			continue
		}
		next = append(next, r)
	}
	return s.saveLocked(next)
}

func (s *identityStore) loadLocked() ([]identityRecord, error) {
	bs, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []identityRecord{}, nil
		}
		return nil, err
	}
	rows := []identityRecord{}
	if err := json.Unmarshal(bs, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *identityStore) saveLocked(rows []identityRecord) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	bs, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, bs, 0o644)
}
