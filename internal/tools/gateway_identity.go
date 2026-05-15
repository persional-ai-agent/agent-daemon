package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type GatewayIdentityRecord struct {
	Platform string `json:"platform"`
	UserID   string `json:"user_id"`
	GlobalID string `json:"global_id"`
}

type GatewaySessionResolveResult struct {
	Platform        string `json:"platform"`
	ChatType        string `json:"chat_type"`
	ChatID          string `json:"chat_id"`
	UserID          string `json:"user_id"`
	UserName        string `json:"user_name"`
	RouteSession    string `json:"route_session"`
	MappedSession   string `json:"mapped_session"`
	ResolvedSession string `json:"resolved_session"`
	GlobalID        string `json:"global_id"`
	GlobalSource    string `json:"global_source"`
	ContinuityMode  string `json:"continuity_mode"`
}

type GatewayResolveArgs struct {
	Platform string
	ChatType string
	ChatID   string
	UserID   string
	UserName string
}

var gatewayIdentityMu sync.Mutex

func NormalizeContinuityMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "user_id", "userid", "id":
		return "user_id"
	case "user_name", "username", "name":
		return "user_name"
	case "off":
		return "off"
	default:
		return "off"
	}
}

func AutoGlobalIdentity(mode, userID, userName string) string {
	switch NormalizeContinuityMode(mode) {
	case "user_id":
		if strings.TrimSpace(userID) == "" {
			return ""
		}
		return "uid:" + strings.TrimSpace(userID)
	case "user_name":
		n := strings.ToLower(strings.TrimSpace(userName))
		n = strings.ReplaceAll(n, " ", "_")
		if n == "" {
			return ""
		}
		return "uname:" + n
	default:
		return ""
	}
}

func GatewaySessionKey(platform, chatType, chatID string) string {
	return fmt.Sprintf("agent:main:%s:%s:%s", strings.TrimSpace(platform), strings.TrimSpace(chatType), strings.TrimSpace(chatID))
}

func GatewayGlobalSessionKey(globalID string) string {
	return GatewaySessionKey("global", "user", globalID)
}

func gatewayIdentityMapPath(workdir string) string {
	return filepath.Join(strings.TrimSpace(workdir), ".agent-daemon", "gateway_identity_map.json")
}

func LoadGatewayIdentityMap(workdir string) ([]GatewayIdentityRecord, error) {
	gatewayIdentityMu.Lock()
	defer gatewayIdentityMu.Unlock()
	return loadGatewayIdentityMapUnlocked(workdir)
}

func loadGatewayIdentityMapUnlocked(workdir string) ([]GatewayIdentityRecord, error) {
	path := gatewayIdentityMapPath(workdir)
	bs, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []GatewayIdentityRecord{}, nil
		}
		return nil, err
	}
	rows := []GatewayIdentityRecord{}
	if err := json.Unmarshal(bs, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

func saveGatewayIdentityMap(workdir string, rows []GatewayIdentityRecord) error {
	path := gatewayIdentityMapPath(workdir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	bs, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, bs, 0o644)
}

func ResolveGatewayIdentity(workdir, platformName, userID string) (string, error) {
	rows, err := LoadGatewayIdentityMap(workdir)
	if err != nil {
		return "", err
	}
	platformName = strings.ToLower(strings.TrimSpace(platformName))
	userID = strings.TrimSpace(userID)
	for _, row := range rows {
		if row.Platform == platformName && row.UserID == userID {
			return strings.TrimSpace(row.GlobalID), nil
		}
	}
	return "", nil
}

func UpsertGatewayIdentity(workdir, platformName, userID, globalID string) error {
	gatewayIdentityMu.Lock()
	defer gatewayIdentityMu.Unlock()
	rows, err := loadGatewayIdentityMapUnlocked(workdir)
	if err != nil {
		return err
	}
	platformName = strings.ToLower(strings.TrimSpace(platformName))
	userID = strings.TrimSpace(userID)
	globalID = strings.TrimSpace(globalID)
	found := false
	for i := range rows {
		if rows[i].Platform == platformName && rows[i].UserID == userID {
			rows[i].GlobalID = globalID
			found = true
			break
		}
	}
	if !found {
		rows = append(rows, GatewayIdentityRecord{Platform: platformName, UserID: userID, GlobalID: globalID})
	}
	return saveGatewayIdentityMap(workdir, rows)
}

func DeleteGatewayIdentity(workdir, platformName, userID string) error {
	gatewayIdentityMu.Lock()
	defer gatewayIdentityMu.Unlock()
	rows, err := loadGatewayIdentityMapUnlocked(workdir)
	if err != nil {
		return err
	}
	platformName = strings.ToLower(strings.TrimSpace(platformName))
	userID = strings.TrimSpace(userID)
	next := make([]GatewayIdentityRecord, 0, len(rows))
	for _, row := range rows {
		if row.Platform == platformName && row.UserID == userID {
			continue
		}
		next = append(next, row)
	}
	return saveGatewayIdentityMap(workdir, next)
}

func ResolveGatewaySessionMapping(workdir, platformName, chatType, chatID, userID, userName string) (GatewaySessionResolveResult, error) {
	platformName = strings.ToLower(strings.TrimSpace(platformName))
	chatType = strings.TrimSpace(chatType)
	chatID = strings.TrimSpace(chatID)
	userID = strings.TrimSpace(userID)
	userName = strings.TrimSpace(userName)
	mode, err := GetGatewaySetting(workdir, "continuity_mode")
	if err != nil {
		return GatewaySessionResolveResult{}, err
	}
	mode = NormalizeContinuityMode(mode)
	globalID, err := ResolveGatewayIdentity(workdir, platformName, userID)
	if err != nil {
		return GatewaySessionResolveResult{}, err
	}
	globalSource := "mapped"
	if strings.TrimSpace(globalID) == "" {
		globalSource = "none"
		if autoID := AutoGlobalIdentity(mode, userID, userName); strings.TrimSpace(autoID) != "" {
			globalID = autoID
			globalSource = "auto"
		}
	}
	routeSession := GatewaySessionKey(platformName, chatType, chatID)
	mappedSession := ""
	resolvedSession := routeSession
	if strings.TrimSpace(globalID) != "" {
		mappedSession = GatewayGlobalSessionKey(globalID)
		resolvedSession = mappedSession
	}
	return GatewaySessionResolveResult{
		Platform:        platformName,
		ChatType:        chatType,
		ChatID:          chatID,
		UserID:          userID,
		UserName:        userName,
		RouteSession:    routeSession,
		MappedSession:   mappedSession,
		ResolvedSession: resolvedSession,
		GlobalID:        globalID,
		GlobalSource:    globalSource,
		ContinuityMode:  mode,
	}, nil
}

func ParseGatewayResolveArgs(args []string) (GatewayResolveArgs, error) {
	if len(args) != 4 && len(args) != 5 {
		return GatewayResolveArgs{}, fmt.Errorf("invalid resolve args")
	}
	out := GatewayResolveArgs{
		Platform: strings.ToLower(strings.TrimSpace(args[0])),
		ChatType: strings.TrimSpace(args[1]),
		ChatID:   strings.TrimSpace(args[2]),
		UserID:   strings.TrimSpace(args[3]),
	}
	if len(args) == 5 {
		out.UserName = strings.TrimSpace(args[4])
	}
	if out.Platform == "" || out.ChatType == "" || out.ChatID == "" || out.UserID == "" {
		return GatewayResolveArgs{}, fmt.Errorf("invalid resolve args")
	}
	return out, nil
}

func ParseGatewayResolveArgsWithDefaults(args []string, defaults GatewayResolveArgs) (GatewayResolveArgs, error) {
	if len(args) == 0 {
		if strings.TrimSpace(defaults.Platform) == "" || strings.TrimSpace(defaults.ChatType) == "" || strings.TrimSpace(defaults.ChatID) == "" || strings.TrimSpace(defaults.UserID) == "" {
			return GatewayResolveArgs{}, fmt.Errorf("invalid resolve args")
		}
		defaults.Platform = strings.ToLower(strings.TrimSpace(defaults.Platform))
		defaults.ChatType = strings.TrimSpace(defaults.ChatType)
		defaults.ChatID = strings.TrimSpace(defaults.ChatID)
		defaults.UserID = strings.TrimSpace(defaults.UserID)
		defaults.UserName = strings.TrimSpace(defaults.UserName)
		return defaults, nil
	}
	return ParseGatewayResolveArgs(args)
}
