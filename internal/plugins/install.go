package plugins

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func InstallLocal(src, destDir string) (Manifest, string, error) {
	src = strings.TrimSpace(src)
	destDir = strings.TrimSpace(destDir)
	if src == "" {
		return Manifest{}, "", fmt.Errorf("source path is required")
	}
	if destDir == "" {
		return Manifest{}, "", fmt.Errorf("destination directory is required")
	}
	info, err := os.Stat(src)
	if err != nil {
		return Manifest{}, "", err
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return Manifest{}, "", err
	}
	if info.IsDir() {
		manifestPath, err := findManifestInDir(src)
		if err != nil {
			return Manifest{}, "", err
		}
		manifest, err := LoadManifestFile(manifestPath)
		if err != nil {
			return Manifest{}, "", err
		}
		if err := ValidateManifest(manifest); err != nil {
			return Manifest{}, "", err
		}
		name := safePluginDirName(manifest.Name)
		if name == "" {
			return Manifest{}, "", fmt.Errorf("plugin name is required")
		}
		dest := filepath.Join(destDir, name)
		if err := copyDir(src, dest); err != nil {
			return Manifest{}, "", err
		}
		installed, err := LoadManifestFile(filepath.Join(dest, filepath.Base(manifestPath)))
		if err == nil {
			manifest = installed
		}
		return manifest, dest, nil
	}
	if !isManifestFileName(filepath.Base(src)) {
		return Manifest{}, "", fmt.Errorf("plugin install source must be a manifest file or plugin directory")
	}
	manifest, err := LoadManifestFile(src)
	if err != nil {
		return Manifest{}, "", err
	}
	if err := ValidateManifest(manifest); err != nil {
		return Manifest{}, "", err
	}
	name := safePluginDirName(manifest.Name)
	if name == "" {
		return Manifest{}, "", fmt.Errorf("plugin name is required")
	}
	dest := filepath.Join(destDir, name+filepath.Ext(src))
	if err := copyFile(src, dest, info.Mode()); err != nil {
		return Manifest{}, "", err
	}
	installed, err := LoadManifestFile(dest)
	if err == nil {
		manifest = installed
	}
	return manifest, dest, nil
}

func UninstallLocal(manifests []Manifest, name string, allowedDirs []string) (string, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return "", fmt.Errorf("plugin name is required")
	}
	allowed := make([]string, 0, len(allowedDirs))
	for _, dir := range allowedDirs {
		if abs, err := filepath.Abs(strings.TrimSpace(dir)); err == nil {
			allowed = append(allowed, abs)
		}
	}
	for _, m := range manifests {
		if strings.ToLower(strings.TrimSpace(m.Name)) != name {
			continue
		}
		target, err := uninstallTarget(m)
		if err != nil {
			return "", err
		}
		if !pathUnderAny(target, allowed) {
			return "", fmt.Errorf("refusing to uninstall plugin outside configured plugin dirs: %s", target)
		}
		if err := os.RemoveAll(target); err != nil {
			return "", err
		}
		return target, nil
	}
	return "", fmt.Errorf("plugin %q not found", name)
}

func uninstallTarget(m Manifest) (string, error) {
	file := strings.TrimSpace(m.File)
	if file == "" {
		return "", fmt.Errorf("plugin %q has no manifest file", m.Name)
	}
	base := strings.ToLower(filepath.Base(file))
	parent := filepath.Dir(file)
	if base == "plugin.json" || base == "manifest.json" || base == "plugin.yaml" || base == "plugin.yml" || base == "manifest.yaml" || base == "manifest.yml" {
		return parent, nil
	}
	return file, nil
}

func findManifestInDir(dir string) (string, error) {
	for _, name := range []string{"plugin.json", "manifest.json", "plugin.yaml", "plugin.yml", "manifest.yaml", "manifest.yml"} {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no plugin manifest found in %s", dir)
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		return copyFile(path, target, info.Mode())
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func safePluginDirName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		} else if r == ' ' {
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), ".-_")
}

func pathUnderAny(path string, roots []string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	for _, root := range roots {
		rel, err := filepath.Rel(root, abs)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}
