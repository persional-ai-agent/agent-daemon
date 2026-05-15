package gateway

import (
	"strings"

	"github.com/dingjingmaster/agent-daemon/internal/tools"
)

type identityStore struct {
	workdir string
}

func newIdentityStore(workdir string) *identityStore {
	return &identityStore{workdir: strings.TrimSpace(workdir)}
}

func (s *identityStore) bind(platform, userID, globalID string) error {
	platform = strings.ToLower(strings.TrimSpace(platform))
	userID = strings.TrimSpace(userID)
	globalID = strings.TrimSpace(globalID)
	if platform == "" || userID == "" || globalID == "" {
		return nil
	}
	return tools.UpsertGatewayIdentity(s.workdir, platform, userID, globalID)
}

func (s *identityStore) resolve(platform, userID string) (string, error) {
	platform = strings.ToLower(strings.TrimSpace(platform))
	userID = strings.TrimSpace(userID)
	if platform == "" || userID == "" {
		return "", nil
	}
	return tools.ResolveGatewayIdentity(s.workdir, platform, userID)
}

func (s *identityStore) unbind(platform, userID string) error {
	platform = strings.ToLower(strings.TrimSpace(platform))
	userID = strings.TrimSpace(userID)
	if platform == "" || userID == "" {
		return nil
	}
	return tools.DeleteGatewayIdentity(s.workdir, platform, userID)
}
