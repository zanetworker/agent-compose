package compose

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeSkill(t *testing.T, dir, name, content string, refs ...string) {
	t.Helper()
	skillDir := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Join(skillDir, "references"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	for _, ref := range refs {
		if err := os.WriteFile(filepath.Join(skillDir, "references", ref), []byte("ref content"), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLocalSkillResolver_SimpleSkill(t *testing.T) {
	dir := t.TempDir()
	writeSkill(t, dir, "code-style", "# Code Style\n\nUse gofmt.\n")

	r := NewLocalSkillResolver(dir)
	skill, err := r.Resolve(context.Background(), "code-style")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if skill.Prompt != "# Code Style\n\nUse gofmt.\n" {
		t.Errorf("prompt = %q, want '# Code Style\\n\\nUse gofmt.\\n'", skill.Prompt)
	}
	if len(skill.RequiresMCP) != 0 {
		t.Errorf("requires.mcp len = %d, want 0", len(skill.RequiresMCP))
	}
}

func TestLocalSkillResolver_SkillWithFrontmatter(t *testing.T) {
	dir := t.TempDir()
	content := "---\nrequires:\n  mcp: [github]\n  tools: [shell, file-read]\n---\n\n# Security Review\n\nCheck for XSS.\n"
	writeSkill(t, dir, "security-review", content, "owasp.md")

	r := NewLocalSkillResolver(dir)
	skill, err := r.Resolve(context.Background(), "security-review")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if skill.Prompt != "# Security Review\n\nCheck for XSS.\n" {
		t.Errorf("prompt = %q", skill.Prompt)
	}
	if len(skill.RequiresMCP) != 1 || skill.RequiresMCP[0] != "github" {
		t.Errorf("requires.mcp = %v, want [github]", skill.RequiresMCP)
	}
	if len(skill.RequiresTools) != 2 {
		t.Errorf("requires.tools len = %d, want 2", len(skill.RequiresTools))
	}
	if len(skill.References) != 1 {
		t.Errorf("references len = %d, want 1", len(skill.References))
	}
}

func TestLocalSkillResolver_NotFound(t *testing.T) {
	dir := t.TempDir()
	r := NewLocalSkillResolver(dir)

	_, err := r.Resolve(context.Background(), "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestLocalSkillResolver_List(t *testing.T) {
	dir := t.TempDir()
	writeSkill(t, dir, "skill-a", "# Skill A")
	writeSkill(t, dir, "skill-b", "# Skill B")

	r := NewLocalSkillResolver(dir)
	skills, err := r.List(context.Background())
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(skills) != 2 {
		t.Errorf("len(skills) = %d, want 2", len(skills))
	}
}
