package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const GatewayContinuityEnvVar = "AGENT_GATEWAY_CONTINUITY"
const GatewayModelProviderEnvVar = "AGENT_MODEL_PROVIDER"
const GatewayModelNameEnvVar = "AGENT_MODEL"
const GatewayModelBaseURLEnvVar = "AGENT_BASE_URL"

type GatewayModelPreference struct {
	Provider string
	Model    string
	BaseURL  string
}

var gatewaySettingsMu sync.Mutex

func gatewaySettingsPath(workdir string) string {
	return filepath.Join(strings.TrimSpace(workdir), ".agent-daemon", "gateway_settings.json")
}

func SetGatewaySetting(workdir, key, value string) error {
	workdir = strings.TrimSpace(workdir)
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	if workdir == "" || key == "" {
		return nil
	}
	gatewaySettingsMu.Lock()
	defer gatewaySettingsMu.Unlock()
	m := map[string]string{}
	path := gatewaySettingsPath(workdir)
	if bs, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(bs, &m)
	}
	m[key] = value
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	bs, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, bs, 0o644)
}

func GetGatewaySetting(workdir, key string) (string, error) {
	workdir = strings.TrimSpace(workdir)
	key = strings.TrimSpace(key)
	if workdir == "" || key == "" {
		return "", nil
	}
	gatewaySettingsMu.Lock()
	defer gatewaySettingsMu.Unlock()
	path := gatewaySettingsPath(workdir)
	bs, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	m := map[string]string{}
	if err := json.Unmarshal(bs, &m); err != nil {
		return "", err
	}
	return strings.TrimSpace(m[key]), nil
}

func ResolveGatewayContinuityMode(workdir string) (string, error) {
	if v := strings.TrimSpace(os.Getenv(GatewayContinuityEnvVar)); v != "" {
		return NormalizeContinuityMode(v), nil
	}
	v, err := GetGatewaySetting(workdir, "continuity_mode")
	if err != nil {
		return "", err
	}
	return NormalizeContinuityMode(v), nil
}

func UpdateGatewayContinuityMode(workdir, rawMode string) (string, error) {
	mode, err := ParseGatewayContinuityModeArg([]string{rawMode})
	if err != nil {
		return "", err
	}
	_ = os.Setenv(GatewayContinuityEnvVar, mode)
	if err := SetGatewaySetting(workdir, "continuity_mode", mode); err != nil {
		return "", err
	}
	return mode, nil
}

func ResolveGatewayModelPreference(workdir string) (GatewayModelPreference, error) {
	provider, err := GetGatewaySetting(workdir, "model_provider")
	if err != nil {
		return GatewayModelPreference{}, err
	}
	modelName, err := GetGatewaySetting(workdir, "model_name")
	if err != nil {
		return GatewayModelPreference{}, err
	}
	baseURL, err := GetGatewaySetting(workdir, "model_base_url")
	if err != nil {
		return GatewayModelPreference{}, err
	}
	if strings.TrimSpace(provider) == "" {
		provider = strings.TrimSpace(os.Getenv(GatewayModelProviderEnvVar))
	}
	if strings.TrimSpace(modelName) == "" {
		modelName = strings.TrimSpace(os.Getenv(GatewayModelNameEnvVar))
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = strings.TrimSpace(os.Getenv(GatewayModelBaseURLEnvVar))
	}
	return GatewayModelPreference{
		Provider: strings.TrimSpace(provider),
		Model:    strings.TrimSpace(modelName),
		BaseURL:  strings.TrimSpace(baseURL),
	}, nil
}

func UpdateGatewayModelPreference(workdir string, spec GatewayModelSpec) error {
	if err := SetGatewaySetting(workdir, "model_provider", spec.Provider); err != nil {
		return err
	}
	if err := SetGatewaySetting(workdir, "model_name", spec.Model); err != nil {
		return err
	}
	_ = os.Setenv(GatewayModelProviderEnvVar, spec.Provider)
	_ = os.Setenv(GatewayModelNameEnvVar, spec.Model)
	return nil
}

func UpdateGatewayModelBaseURL(workdir, baseURL string) error {
	baseURL = strings.TrimSpace(baseURL)
	if err := SetGatewaySetting(workdir, "model_base_url", baseURL); err != nil {
		return err
	}
	_ = os.Setenv(GatewayModelBaseURLEnvVar, baseURL)
	return nil
}

func DisplayGatewayModelPreference(pref GatewayModelPreference) GatewayModelPreference {
	out := GatewayModelPreference{
		Provider: strings.TrimSpace(pref.Provider),
		Model:    strings.TrimSpace(pref.Model),
		BaseURL:  strings.TrimSpace(pref.BaseURL),
	}
	if out.Provider == "" {
		out.Provider = "openai"
	}
	if out.Model == "" {
		out.Model = "(default)"
	}
	if out.BaseURL == "" {
		out.BaseURL = "(default)"
	}
	return out
}
