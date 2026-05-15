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

type GatewayIdentityRef struct {
	Platform string
	UserID   string
}

type GatewaySetIdentityArgs struct {
	Platform string
	UserID   string
	GlobalID string
}

type GatewayModelSpec struct {
	Provider string
	Model    string
}

const GatewayModelUsage = "/model [provider:model|provider model]"
const GatewaySetHomeUsage = "/sethome <platform> <chat_id> | /sethome <platform:chat_id>"
const GatewayContinuityUsage = "/continuity [off|user_id|user_name]"
const GatewayWhoamiUsage = "/whoami <platform> <user_id>"
const GatewayResolveUsage = "/resolve <platform> <chat_type> <chat_id> <user_id> [user_name]"
const GatewaySetIDUsage = "/setid <platform> <user_id> <global_user_id>"
const GatewayUnsetIDUsage = "/unsetid <platform> <user_id>"
const GatewaySetIDGatewayUsage = "/setid <global_user_id>"
const GatewayUnsetIDGatewayUsage = "/unsetid"

func GatewayModelUsageEN() string {
	return "Usage: " + GatewayModelUsage
}

func GatewayModelUsageZH() string {
	return "用法: " + GatewayModelUsage
}

func GatewaySetHomeUsageEN() string      { return "Usage: " + GatewaySetHomeUsage }
func GatewaySetHomeUsageZH() string      { return "用法: " + GatewaySetHomeUsage }
func GatewayContinuityUsageEN() string   { return "Usage: " + GatewayContinuityUsage }
func GatewayContinuityUsageZH() string   { return "用法: " + GatewayContinuityUsage }
func GatewayWhoamiUsageEN() string       { return "Usage: " + GatewayWhoamiUsage }
func GatewayWhoamiUsageZH() string       { return "用法: " + GatewayWhoamiUsage }
func GatewayResolveUsageEN() string      { return "Usage: " + GatewayResolveUsage }
func GatewayResolveUsageZH() string      { return "用法: " + GatewayResolveUsage }
func GatewaySetIDUsageZH() string        { return "用法: " + GatewaySetIDUsage }
func GatewayUnsetIDUsageZH() string      { return "用法: " + GatewayUnsetIDUsage }
func GatewaySetIDGatewayUsageEN() string { return "Usage: " + GatewaySetIDGatewayUsage }
func GatewayUnsetIDGatewayUsageEN() string {
	return "Usage: " + GatewayUnsetIDGatewayUsage
}

func GatewayContinuityInvalidArgumentEN() string {
	return "mode must be off|user_id|user_name"
}

func GatewayIdentityRequiredEN() string {
	return "platform/user_id required"
}

func GatewaySetIdentityRequiredEN() string {
	return "platform/user_id/global_id required"
}

func GatewayResolveRequiredEN() string {
	return "platform/chat_type/chat_id/user_id required"
}

type GatewayWhoamiResult struct {
	Platform       string `json:"platform"`
	UserID         string `json:"user_id"`
	UserName       string `json:"user_name"`
	ActiveSession  string `json:"active_session"`
	GlobalID       string `json:"global_id"`
	ContinuityMode string `json:"continuity_mode"`
	AutoGlobalID   string `json:"auto_global_id,omitempty"`
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
	mode, err := ResolveGatewayContinuityMode(workdir)
	if err != nil {
		return GatewaySessionResolveResult{}, err
	}
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

func ParseGatewayIdentityRefArgs(args []string) (GatewayIdentityRef, error) {
	if len(args) != 2 {
		return GatewayIdentityRef{}, fmt.Errorf("invalid identity args")
	}
	out := GatewayIdentityRef{
		Platform: strings.ToLower(strings.TrimSpace(args[0])),
		UserID:   strings.TrimSpace(args[1]),
	}
	if out.Platform == "" || out.UserID == "" {
		return GatewayIdentityRef{}, fmt.Errorf("invalid identity args")
	}
	return out, nil
}

func ParseGatewaySetIdentityArgs(args []string) (GatewaySetIdentityArgs, error) {
	if len(args) != 3 {
		return GatewaySetIdentityArgs{}, fmt.Errorf("invalid setid args")
	}
	out := GatewaySetIdentityArgs{
		Platform: strings.ToLower(strings.TrimSpace(args[0])),
		UserID:   strings.TrimSpace(args[1]),
		GlobalID: strings.TrimSpace(args[2]),
	}
	if out.Platform == "" || out.UserID == "" || out.GlobalID == "" {
		return GatewaySetIdentityArgs{}, fmt.Errorf("invalid setid args")
	}
	return out, nil
}

func ParseGatewayGlobalIDArg(args []string) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("invalid global id arg")
	}
	globalID := strings.TrimSpace(args[0])
	if globalID == "" {
		return "", fmt.Errorf("invalid global id arg")
	}
	return globalID, nil
}

func ParseGatewayContinuityModeArg(args []string) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("invalid continuity args")
	}
	mode := NormalizeContinuityMode(args[0])
	switch mode {
	case "off", "user_id", "user_name":
		return mode, nil
	default:
		return "", fmt.Errorf("invalid continuity mode")
	}
}

func ParseGatewayModelSpecArgs(args []string) (GatewayModelSpec, error) {
	if len(args) == 1 {
		spec := strings.TrimSpace(args[0])
		parts := strings.SplitN(spec, ":", 2)
		if len(parts) != 2 {
			return GatewayModelSpec{}, fmt.Errorf("invalid model spec")
		}
		out := GatewayModelSpec{
			Provider: strings.ToLower(strings.TrimSpace(parts[0])),
			Model:    strings.TrimSpace(parts[1]),
		}
		if out.Provider == "" || out.Model == "" {
			return GatewayModelSpec{}, fmt.Errorf("invalid model spec")
		}
		return out, nil
	}
	if len(args) == 2 {
		out := GatewayModelSpec{
			Provider: strings.ToLower(strings.TrimSpace(args[0])),
			Model:    strings.TrimSpace(args[1]),
		}
		if out.Provider == "" || out.Model == "" {
			return GatewayModelSpec{}, fmt.Errorf("invalid model spec")
		}
		return out, nil
	}
	return GatewayModelSpec{}, fmt.Errorf("invalid model spec")
}

func BuildGatewaySessionResolvePayload(resolved GatewaySessionResolveResult) map[string]any {
	return map[string]any{
		"platform":         resolved.Platform,
		"chat_type":        resolved.ChatType,
		"chat_id":          resolved.ChatID,
		"user_id":          resolved.UserID,
		"user_name":        resolved.UserName,
		"route_session":    resolved.RouteSession,
		"mapped_session":   resolved.MappedSession,
		"resolved_session": resolved.ResolvedSession,
		"global_id":        resolved.GlobalID,
		"global_source":    resolved.GlobalSource,
		"continuity_mode":  resolved.ContinuityMode,
	}
}

func BuildGatewayWhoamiPayload(result GatewayWhoamiResult) map[string]any {
	payload := map[string]any{
		"platform":        result.Platform,
		"user_id":         result.UserID,
		"user_name":       result.UserName,
		"active_session":  result.ActiveSession,
		"global_id":       result.GlobalID,
		"continuity_mode": result.ContinuityMode,
	}
	if strings.TrimSpace(result.AutoGlobalID) != "" {
		payload["auto_global_id"] = strings.TrimSpace(result.AutoGlobalID)
	}
	return payload
}

func RenderGatewayWhoamiText(result GatewayWhoamiResult) string {
	reply := "platform=" + result.Platform + "\nuser_id=" + result.UserID + "\nuser_name=" + result.UserName + "\nactive_session=" + result.ActiveSession
	if strings.TrimSpace(result.GlobalID) != "" {
		reply += "\nglobal_id=" + result.GlobalID
	} else {
		reply += "\nglobal_id=(not set)"
	}
	reply += "\ncontinuity_mode=" + result.ContinuityMode
	if strings.TrimSpace(result.AutoGlobalID) != "" {
		reply += "\nauto_global_id=" + result.AutoGlobalID
	}
	return reply
}

func RenderGatewaySessionResolveText(resolved GatewaySessionResolveResult) string {
	return "platform=" + resolved.Platform +
		"\nchat_type=" + resolved.ChatType +
		"\nchat_id=" + resolved.ChatID +
		"\nuser_id=" + resolved.UserID +
		"\nuser_name=" + resolved.UserName +
		"\nroute_session=" + resolved.RouteSession +
		"\nmapped_session=" + resolved.MappedSession +
		"\nresolved_session=" + resolved.ResolvedSession +
		"\nglobal_id=" + resolved.GlobalID +
		"\nglobal_source=" + resolved.GlobalSource +
		"\ncontinuity_mode=" + resolved.ContinuityMode
}

func BuildGatewayIdentityPayload(platform, userID, globalID string, updated, deleted bool) map[string]any {
	payload := map[string]any{
		"platform": strings.TrimSpace(platform),
		"user_id":  strings.TrimSpace(userID),
	}
	if strings.TrimSpace(globalID) != "" {
		payload["global_id"] = strings.TrimSpace(globalID)
	}
	if updated {
		payload["updated"] = true
	}
	if deleted {
		payload["deleted"] = true
	}
	return payload
}
