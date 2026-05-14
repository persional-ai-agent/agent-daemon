package plugins

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Manifest struct {
	Name            string              `json:"name" yaml:"name"`
	Type            string              `json:"type" yaml:"type"`
	Kind            string              `json:"kind,omitempty" yaml:"kind,omitempty"`
	Version         string              `json:"version,omitempty" yaml:"version,omitempty"`
	Description     string              `json:"description,omitempty" yaml:"description,omitempty"`
	Author          string              `json:"author,omitempty" yaml:"author,omitempty"`
	Entry           string              `json:"entry,omitempty" yaml:"entry,omitempty"`
	Enabled         *bool               `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Tool            *ToolManifest       `json:"tool,omitempty" yaml:"tool,omitempty"`
	Tools           []ToolManifest      `json:"tools,omitempty" yaml:"tools,omitempty"`
	Provider        *ProviderManifest   `json:"provider,omitempty" yaml:"provider,omitempty"`
	Providers       []ProviderManifest  `json:"providers,omitempty" yaml:"providers,omitempty"`
	Commands        []CommandManifest   `json:"commands,omitempty" yaml:"commands,omitempty"`
	Dashboard       *DashboardManifest  `json:"dashboard,omitempty" yaml:"dashboard,omitempty"`
	Dashboards      []DashboardManifest `json:"dashboards,omitempty" yaml:"dashboards,omitempty"`
	ProvidesTools   []string            `json:"provides_tools,omitempty" yaml:"provides_tools,omitempty"`
	Hooks           []string            `json:"hooks,omitempty" yaml:"hooks,omitempty"`
	Platforms       []string            `json:"platforms,omitempty" yaml:"platforms,omitempty"`
	PipDependencies []string            `json:"pip_dependencies,omitempty" yaml:"pip_dependencies,omitempty"`
	Security        *SecurityManifest   `json:"security,omitempty" yaml:"security,omitempty"`
	Sandbox         *SandboxManifest    `json:"sandbox,omitempty" yaml:"sandbox,omitempty"`
	Env             map[string]string   `json:"env,omitempty" yaml:"env,omitempty"`
	File            string              `json:"file,omitempty" yaml:"file,omitempty"`
}

type ToolManifest struct {
	Name           string         `json:"name,omitempty" yaml:"name,omitempty"`
	Description    string         `json:"description,omitempty" yaml:"description,omitempty"`
	Command        string         `json:"command" yaml:"command"`
	Args           []string       `json:"args,omitempty" yaml:"args,omitempty"`
	Schema         map[string]any `json:"schema" yaml:"schema"`
	TimeoutSeconds int            `json:"timeout_seconds,omitempty" yaml:"timeout_seconds,omitempty"`
	PassContext    bool           `json:"pass_context,omitempty" yaml:"pass_context,omitempty"`
}

type ProviderManifest struct {
	Name           string   `json:"name,omitempty" yaml:"name,omitempty"`
	Description    string   `json:"description,omitempty" yaml:"description,omitempty"`
	Command        string   `json:"command" yaml:"command"`
	Args           []string `json:"args,omitempty" yaml:"args,omitempty"`
	TimeoutSeconds int      `json:"timeout_seconds,omitempty" yaml:"timeout_seconds,omitempty"`
	Model          string   `json:"model,omitempty" yaml:"model,omitempty"`
}

type CommandManifest struct {
	Name           string   `json:"name" yaml:"name"`
	Description    string   `json:"description,omitempty" yaml:"description,omitempty"`
	Command        string   `json:"command" yaml:"command"`
	Args           []string `json:"args,omitempty" yaml:"args,omitempty"`
	TimeoutSeconds int      `json:"timeout_seconds,omitempty" yaml:"timeout_seconds,omitempty"`
	PassContext    bool     `json:"pass_context,omitempty" yaml:"pass_context,omitempty"`
}

type DashboardManifest struct {
	Name        string         `json:"name,omitempty" yaml:"name,omitempty"`
	Label       string         `json:"label,omitempty" yaml:"label,omitempty"`
	Description string         `json:"description,omitempty" yaml:"description,omitempty"`
	Icon        string         `json:"icon,omitempty" yaml:"icon,omitempty"`
	Version     string         `json:"version,omitempty" yaml:"version,omitempty"`
	Entry       string         `json:"entry,omitempty" yaml:"entry,omitempty"`
	CSS         string         `json:"css,omitempty" yaml:"css,omitempty"`
	API         string         `json:"api,omitempty" yaml:"api,omitempty"`
	Tab         map[string]any `json:"tab,omitempty" yaml:"tab,omitempty"`
}

type SecurityManifest struct {
	PublicKey string         `json:"public_key,omitempty" yaml:"public_key,omitempty"`
	Signature string         `json:"signature,omitempty" yaml:"signature,omitempty"`
	Files     []FileChecksum `json:"files,omitempty" yaml:"files,omitempty"`
}

type FileChecksum struct {
	Path   string `json:"path" yaml:"path"`
	SHA256 string `json:"sha256" yaml:"sha256"`
}

type SandboxManifest struct {
	Enabled             *bool    `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Workdir             string   `json:"workdir,omitempty" yaml:"workdir,omitempty"`
	EnvPassthrough      []string `json:"env_passthrough,omitempty" yaml:"env_passthrough,omitempty"`
	AllowHostEnv        bool     `json:"allow_host_env,omitempty" yaml:"allow_host_env,omitempty"`
	AllowOutsideCommand bool     `json:"allow_outside_command,omitempty" yaml:"allow_outside_command,omitempty"`
}

func (m Manifest) IsEnabled() bool {
	if m.Enabled == nil {
		return true
	}
	return *m.Enabled
}

func ValidateManifest(m Manifest) error {
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("name is required")
	}
	switch manifestType(m) {
	case "tool":
		if m.Tool == nil && len(m.Tools) == 0 {
			return fmt.Errorf("tool spec is required for type=tool")
		}
		if m.Tool != nil {
			if err := validateToolSpec(*m.Tool, m.Name); err != nil {
				return err
			}
		}
		for i, spec := range m.Tools {
			if err := validateToolSpec(spec, m.Name); err != nil {
				return fmt.Errorf("tools[%d]: %w", i, err)
			}
		}
	case "provider":
		if m.Provider == nil && len(m.Providers) == 0 {
			return fmt.Errorf("provider spec is required for type=provider")
		}
		if m.Provider != nil {
			if err := validateProviderSpec(*m.Provider, m.Name); err != nil {
				return err
			}
		}
		for i, spec := range m.Providers {
			if err := validateProviderSpec(spec, m.Name); err != nil {
				return fmt.Errorf("providers[%d]: %w", i, err)
			}
		}
	case "plugin":
		if m.Tool != nil {
			if err := validateToolSpec(*m.Tool, m.Name); err != nil {
				return err
			}
		}
		for i, spec := range m.Tools {
			if err := validateToolSpec(spec, m.Name); err != nil {
				return fmt.Errorf("tools[%d]: %w", i, err)
			}
		}
		if m.Provider != nil {
			if err := validateProviderSpec(*m.Provider, m.Name); err != nil {
				return err
			}
		}
		for i, spec := range m.Providers {
			if err := validateProviderSpec(spec, m.Name); err != nil {
				return fmt.Errorf("providers[%d]: %w", i, err)
			}
		}
		for i, spec := range m.Commands {
			if err := validateCommandSpec(spec); err != nil {
				return fmt.Errorf("commands[%d]: %w", i, err)
			}
		}
	default:
		return fmt.Errorf("unsupported plugin type: %q", m.Type)
	}
	return nil
}

func VerifyManifest(m Manifest) error {
	if err := ValidateManifest(m); err != nil {
		return err
	}
	if m.Security == nil {
		return nil
	}
	if len(m.Security.Files) > 0 {
		if strings.TrimSpace(m.File) == "" {
			return fmt.Errorf("security.files requires manifest file path")
		}
		base := filepath.Dir(strings.TrimSpace(m.File))
		for _, file := range m.Security.Files {
			if err := verifyFileChecksum(base, file); err != nil {
				return err
			}
		}
	}
	pub := strings.TrimSpace(m.Security.PublicKey)
	sig := strings.TrimSpace(m.Security.Signature)
	if pub == "" && sig == "" {
		return nil
	}
	if pub == "" || sig == "" {
		return fmt.Errorf("security.public_key and security.signature must be provided together")
	}
	publicKey, err := decodeBinary(pub)
	if err != nil {
		return fmt.Errorf("invalid security.public_key: %w", err)
	}
	signature, err := decodeBinary(sig)
	if err != nil {
		return fmt.Errorf("invalid security.signature: %w", err)
	}
	if len(publicKey) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid security.public_key length: %d", len(publicKey))
	}
	if len(signature) != ed25519.SignatureSize {
		return fmt.Errorf("invalid security.signature length: %d", len(signature))
	}
	payload, err := ManifestSignaturePayload(m)
	if err != nil {
		return err
	}
	if !ed25519.Verify(ed25519.PublicKey(publicKey), payload, signature) {
		return fmt.Errorf("manifest signature verification failed")
	}
	return nil
}

func ManifestSignaturePayload(m Manifest) ([]byte, error) {
	cp := m
	cp.File = ""
	if cp.Security != nil {
		sec := *cp.Security
		sec.Signature = ""
		cp.Security = &sec
	}
	return json.Marshal(cp)
}

func verifyFileChecksum(base string, file FileChecksum) error {
	rel := strings.TrimSpace(file.Path)
	want := strings.ToLower(strings.TrimSpace(file.SHA256))
	if rel == "" || want == "" {
		return fmt.Errorf("security.files entries require path and sha256")
	}
	full := filepath.Join(base, rel)
	if !pathInside(base, full) {
		return fmt.Errorf("security file escapes plugin directory: %s", rel)
	}
	got, err := HashFile(full)
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("sha256 mismatch for %s: got %s want %s", rel, got, want)
	}
	return nil
}

func manifestType(m Manifest) string {
	t := strings.ToLower(strings.TrimSpace(m.Type))
	if t != "" {
		return t
	}
	if strings.TrimSpace(m.Kind) != "" || len(m.Commands) > 0 || m.Dashboard != nil || len(m.Dashboards) > 0 || len(m.ProvidesTools) > 0 || len(m.Hooks) > 0 {
		return "plugin"
	}
	if m.Tool != nil && len(m.Tools) == 0 && m.Provider == nil && len(m.Providers) == 0 {
		return "tool"
	}
	if m.Provider != nil && len(m.Providers) == 0 && m.Tool == nil && len(m.Tools) == 0 {
		return "provider"
	}
	return "plugin"
}

func validateToolSpec(spec ToolManifest, fallbackName string) error {
	if strings.TrimSpace(spec.Name) == "" && strings.TrimSpace(fallbackName) == "" {
		return fmt.Errorf("tool.name is required")
	}
	if strings.TrimSpace(spec.Command) == "" {
		return fmt.Errorf("tool.command is required")
	}
	if len(spec.Schema) == 0 {
		return fmt.Errorf("tool.schema is required")
	}
	return nil
}

func validateProviderSpec(spec ProviderManifest, fallbackName string) error {
	if strings.TrimSpace(spec.Name) == "" && strings.TrimSpace(fallbackName) == "" {
		return fmt.Errorf("provider.name is required")
	}
	if strings.TrimSpace(spec.Command) == "" {
		return fmt.Errorf("provider.command is required")
	}
	return nil
}

func validateCommandSpec(spec CommandManifest) error {
	if strings.TrimSpace(spec.Name) == "" {
		return fmt.Errorf("command.name is required")
	}
	if strings.TrimSpace(spec.Command) == "" {
		return fmt.Errorf("command.command is required")
	}
	return nil
}

func DefaultDirs(workdir string) []string {
	dirs := []string{}
	if strings.TrimSpace(workdir) != "" {
		dirs = append(dirs, filepath.Join(workdir, ".agent-daemon", "plugins"))
		dirs = append(dirs, filepath.Join(workdir, "plugins"))
	}
	return dirs
}

func LoadFromDirs(dirs []string) ([]Manifest, error) {
	out := make([]Manifest, 0)
	seen := map[string]struct{}{}
	for _, dir := range dirs {
		dir = strings.TrimSpace(dir)
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		candidates := make([]string, 0)
		for _, e := range entries {
			full := filepath.Join(dir, e.Name())
			if e.IsDir() {
				for _, name := range []string{"plugin.json", "manifest.json", "plugin.yaml", "plugin.yml", "manifest.yaml", "manifest.yml"} {
					candidates = append(candidates, filepath.Join(full, name))
				}
				continue
			}
			if isManifestFileName(e.Name()) {
				candidates = append(candidates, full)
			}
		}
		for _, full := range candidates {
			if _, ok := seen[full]; ok {
				continue
			}
			seen[full] = struct{}{}
			m, err := LoadManifestFile(full)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, err
			}
			if err := VerifyManifest(m); err != nil {
				continue
			}
			m.File = full
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Type == out[j].Type {
			return out[i].Name < out[j].Name
		}
		return out[i].Type < out[j].Type
	})
	return out, nil
}

func LoadManifestFile(path string) (Manifest, error) {
	bs, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var m Manifest
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(bs, &m); err != nil {
			return Manifest{}, err
		}
	default:
		if err := json.Unmarshal(bs, &m); err != nil {
			return Manifest{}, err
		}
	}
	m.File = path
	if strings.TrimSpace(m.Type) == "" {
		m.Type = manifestType(m)
	}
	return m, nil
}

type VerificationReport struct {
	File  string `json:"file"`
	Name  string `json:"name,omitempty"`
	Valid bool   `json:"valid"`
	Error string `json:"error,omitempty"`
}

func VerifyDirs(dirs []string) ([]VerificationReport, error) {
	files, err := discoverManifestFiles(dirs)
	if err != nil {
		return nil, err
	}
	out := make([]VerificationReport, 0, len(files))
	for _, file := range files {
		report := VerificationReport{File: file}
		m, err := LoadManifestFile(file)
		if err == nil {
			report.Name = m.Name
			err = VerifyManifest(m)
		}
		if err != nil {
			report.Error = err.Error()
		} else {
			report.Valid = true
		}
		out = append(out, report)
	}
	return out, nil
}

func discoverManifestFiles(dirs []string) ([]string, error) {
	out := make([]string, 0)
	seen := map[string]struct{}{}
	for _, dir := range dirs {
		dir = strings.TrimSpace(dir)
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, e := range entries {
			full := filepath.Join(dir, e.Name())
			candidates := []string{full}
			if e.IsDir() {
				candidates = candidates[:0]
				for _, name := range []string{"plugin.json", "manifest.json", "plugin.yaml", "plugin.yml", "manifest.yaml", "manifest.yml"} {
					candidates = append(candidates, filepath.Join(full, name))
				}
			} else if !isManifestFileName(e.Name()) {
				continue
			}
			for _, candidate := range candidates {
				if _, ok := seen[candidate]; ok {
					continue
				}
				if _, err := os.Stat(candidate); err != nil {
					continue
				}
				seen[candidate] = struct{}{}
				out = append(out, candidate)
			}
		}
	}
	sort.Strings(out)
	return out, nil
}

func isManifestFileName(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	if strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
		return true
	}
	return false
}

func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func decodeBinary(s string) ([]byte, error) {
	if b, err := base64.StdEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	if b, err := base64.RawStdEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	return hex.DecodeString(s)
}

func pathInside(root, path string) bool {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(rootAbs, pathAbs)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func ExpandRuntimeManifests(manifests []Manifest) []Manifest {
	out := make([]Manifest, 0, len(manifests))
	for _, m := range manifests {
		t := manifestType(m)
		if t == "tool" || t == "provider" {
			if strings.TrimSpace(m.Type) == "" {
				m.Type = t
			}
			out = append(out, m)
			continue
		}
		if m.Tool != nil {
			out = append(out, singleToolManifest(m, *m.Tool))
		}
		for _, spec := range m.Tools {
			out = append(out, singleToolManifest(m, spec))
		}
		if m.Provider != nil {
			out = append(out, singleProviderManifest(m, *m.Provider))
		}
		for _, spec := range m.Providers {
			out = append(out, singleProviderManifest(m, spec))
		}
	}
	return out
}

func singleToolManifest(parent Manifest, spec ToolManifest) Manifest {
	name := strings.TrimSpace(spec.Name)
	if name == "" {
		name = strings.TrimSpace(parent.Name)
	}
	desc := strings.TrimSpace(spec.Description)
	if desc == "" {
		desc = strings.TrimSpace(parent.Description)
	}
	return Manifest{
		Name:        name,
		Type:        "tool",
		Version:     parent.Version,
		Description: desc,
		Enabled:     parent.Enabled,
		Tool:        &spec,
		Env:         parent.Env,
		File:        parent.File,
	}
}

func singleProviderManifest(parent Manifest, spec ProviderManifest) Manifest {
	name := strings.TrimSpace(spec.Name)
	if name == "" {
		name = strings.TrimSpace(parent.Name)
	}
	desc := strings.TrimSpace(spec.Description)
	if desc == "" {
		desc = strings.TrimSpace(parent.Description)
	}
	return Manifest{
		Name:        name,
		Type:        "provider",
		Version:     parent.Version,
		Description: desc,
		Enabled:     parent.Enabled,
		Provider:    &spec,
		Env:         parent.Env,
		File:        parent.File,
	}
}
