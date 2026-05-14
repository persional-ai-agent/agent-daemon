package plugins

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type MarketplaceIndex struct {
	Plugins []MarketplacePlugin `json:"plugins" yaml:"plugins"`
}

type MarketplacePlugin struct {
	Name        string `json:"name" yaml:"name"`
	Version     string `json:"version,omitempty" yaml:"version,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Source      string `json:"source" yaml:"source"`
	SHA256      string `json:"sha256,omitempty" yaml:"sha256,omitempty"`
}

func LoadMarketplace(path string) (MarketplaceIndex, error) {
	bs, err := os.ReadFile(strings.TrimSpace(path))
	if err != nil {
		return MarketplaceIndex{}, err
	}
	var idx MarketplaceIndex
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		err = yaml.Unmarshal(bs, &idx)
	default:
		err = json.Unmarshal(bs, &idx)
	}
	if err != nil {
		return MarketplaceIndex{}, err
	}
	sort.Slice(idx.Plugins, func(i, j int) bool {
		return strings.ToLower(idx.Plugins[i].Name) < strings.ToLower(idx.Plugins[j].Name)
	})
	return idx, nil
}

func InstallFromMarketplace(indexPath, name, destDir string) (Manifest, string, error) {
	idx, err := LoadMarketplace(indexPath)
	if err != nil {
		return Manifest{}, "", err
	}
	key := strings.ToLower(strings.TrimSpace(name))
	for _, item := range idx.Plugins {
		if strings.ToLower(strings.TrimSpace(item.Name)) != key {
			continue
		}
		src := resolveMarketplaceSource(indexPath, item.Source)
		if strings.TrimSpace(item.SHA256) != "" {
			got, err := HashPath(src)
			if err != nil {
				return Manifest{}, "", err
			}
			if got != strings.ToLower(strings.TrimSpace(item.SHA256)) {
				return Manifest{}, "", fmt.Errorf("marketplace sha256 mismatch for %s: got %s want %s", item.Name, got, item.SHA256)
			}
		}
		return InstallLocal(src, destDir)
	}
	return Manifest{}, "", fmt.Errorf("marketplace plugin %q not found", name)
}

func resolveMarketplaceSource(indexPath, source string) string {
	source = strings.TrimSpace(source)
	if filepath.IsAbs(source) {
		return source
	}
	return filepath.Join(filepath.Dir(strings.TrimSpace(indexPath)), source)
}

func HashPath(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return HashFile(path)
	}
	files := make([]string, 0)
	if err := filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		files = append(files, p)
		return nil
	}); err != nil {
		return "", err
	}
	sort.Strings(files)
	h := sha256.New()
	for _, file := range files {
		rel, _ := filepath.Rel(path, file)
		sum, err := HashFile(file)
		if err != nil {
			return "", err
		}
		_, _ = h.Write([]byte(filepath.ToSlash(rel)))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(sum))
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
