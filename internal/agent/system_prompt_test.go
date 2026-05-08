package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildSkillsIndexBlockEmptyDir(t *testing.T) {
	dir := t.TempDir()
	if got := buildSkillsIndexBlock(dir, nil); got != "" {
		t.Errorf("empty dir should return empty, got: %s", got)
	}
}

func TestBuildSkillsIndexBlockWithSkills(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, "skills")
	os.MkdirAll(filepath.Join(skillsDir, "test-skill"), 0o755)
	os.WriteFile(filepath.Join(skillsDir, "test-skill", "SKILL.md"), []byte("# A test skill for unit testing"), 0o644)
	os.MkdirAll(filepath.Join(skillsDir, "another-skill"), 0o755)
	os.WriteFile(filepath.Join(skillsDir, "another-skill", "SKILL.md"), []byte("# Another useful skill"), 0o644)

	result := buildSkillsIndexBlock(dir, nil)
	if !strings.Contains(result, "test-skill") {
		t.Errorf("expected test-skill in result: %s", result)
	}
	if !strings.Contains(result, "another-skill") {
		t.Errorf("expected another-skill in result: %s", result)
	}
	if !strings.Contains(result, "Available Skills") {
		t.Errorf("expected header: %s", result)
	}
}

func TestBuildSkillsIndexBlockIgnoresFiles(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, "skills")
	os.MkdirAll(skillsDir, 0o755)
	os.WriteFile(filepath.Join(skillsDir, "README.md"), []byte("# readme"), 0o644)

	result := buildSkillsIndexBlock(dir, nil)
	if result != "" {
		t.Errorf("files should be ignored: %s", result)
	}
}

func TestBuildSkillsIndexBlockMissingSkillsDir(t *testing.T) {
	dir := t.TempDir()
	result := buildSkillsIndexBlock(dir, nil)
	if result != "" {
		t.Errorf("missing dir should return empty: %s", result)
	}
}

func TestSkillFilterRequiresTools(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, "skills")
	os.MkdirAll(filepath.Join(skillsDir, "needs-terminal"), 0o755)
	os.WriteFile(filepath.Join(skillsDir, "needs-terminal", "SKILL.md"), []byte("---\nrequires_tools: [terminal]\n---\n# Needs terminal"), 0o644)
	os.MkdirAll(filepath.Join(skillsDir, "no-requirements"), 0o755)
	os.WriteFile(filepath.Join(skillsDir, "no-requirements", "SKILL.md"), []byte("# No requirements"), 0o644)

	result := buildSkillsIndexBlock(dir, nil)
	if strings.Contains(result, "needs-terminal") {
		t.Errorf("skill requiring terminal should be hidden when no tools: %s", result)
	}
	if !strings.Contains(result, "no-requirements") {
		t.Errorf("skill without requirements should show: %s", result)
	}
}

func TestSkillFilterFallbackTools(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, "skills")
	os.MkdirAll(filepath.Join(skillsDir, "fallback-web"), 0o755)
	os.WriteFile(filepath.Join(skillsDir, "fallback-web", "SKILL.md"), []byte("---\nfallback_for_tools: [web_search]\n---\n# Fallback for web"), 0o644)

	result := buildSkillsIndexBlock(dir, nil)
	if !strings.Contains(result, "fallback-web") {
		t.Errorf("fallback skill should show when web_search unavailable: %s", result)
	}
}
