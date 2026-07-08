package compose

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type SkillResolver interface {
	Resolve(ctx context.Context, name string) (*Skill, error)
	List(ctx context.Context) ([]Skill, error)
}

type LocalSkillResolver struct {
	dir string
}

func NewLocalSkillResolver(dir string) *LocalSkillResolver {
	return &LocalSkillResolver{dir: dir}
}

func (r *LocalSkillResolver) Resolve(_ context.Context, name string) (*Skill, error) {
	skillDir := filepath.Join(r.dir, name)
	skillFile := filepath.Join(skillDir, "SKILL.md")

	data, err := os.ReadFile(skillFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("skill %q: %w", name, ErrNotFound)
		}
		return nil, fmt.Errorf("reading skill %q: %w", name, err)
	}

	skill := &Skill{Name: name}
	prompt, fm := parseFrontmatter(data)
	skill.Prompt = prompt
	if fm != nil {
		skill.RequiresMCP = fm.Requires.MCP
		skill.RequiresTools = fm.Requires.Tools
	}

	refsDir := filepath.Join(skillDir, "references")
	if entries, err := os.ReadDir(refsDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				skill.References = append(skill.References, filepath.Join(refsDir, e.Name()))
			}
		}
	}

	return skill, nil
}

func (r *LocalSkillResolver) List(_ context.Context) ([]Skill, error) {
	entries, err := os.ReadDir(r.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var skills []Skill
	for _, e := range entries {
		if e.IsDir() {
			if _, err := os.Stat(filepath.Join(r.dir, e.Name(), "SKILL.md")); err == nil {
				skills = append(skills, Skill{Name: e.Name()})
			}
		}
	}
	return skills, nil
}

func parseFrontmatter(data []byte) (prompt string, fm *SkillFrontmatter) {
	content := string(data)
	if !strings.HasPrefix(content, "---\n") {
		return content, nil
	}

	end := strings.Index(content[4:], "\n---\n")
	if end == -1 {
		return content, nil
	}

	fmData := content[4 : 4+end]
	rest := content[4+end+5:] // skip past closing ---\n

	// Strip leading newline if present
	rest = strings.TrimPrefix(rest, "\n")

	var parsed SkillFrontmatter
	if err := yaml.NewDecoder(bytes.NewReader([]byte(fmData))).Decode(&parsed); err == nil {
		fm = &parsed
	}

	return rest, fm
}
