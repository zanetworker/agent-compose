package compose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctor_ValidConfig(t *testing.T) {
	skillsDir := t.TempDir()

	// Create a valid skill directory structure
	analysisSkillDir := filepath.Join(skillsDir, "analysis")
	if err := os.MkdirAll(analysisSkillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(analysisSkillDir, "SKILL.md"), []byte("# Analysis Skill"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		Runtimes: map[string]RuntimeProfile{
			"claude-code": {
				Image:      "ghcr.io/anthropics/claude-code:latest",
				Entrypoint: []string{"claude"},
				EnvMapping: map[string]string{"ANTHROPIC_API_KEY": "${key}"},
			},
		},
		Inference: map[string]InferenceSpec{
			"vertex": {
				Endpoint:     "https://us-central1-aiplatform.googleapis.com",
				Provider:     "vertex",
				DefaultModel: "claude-sonnet-4-5",
			},
		},
		MCP: map[string]MCPSpec{
			"github": {
				Provider: "github",
			},
		},
		Defaults: Defaults{
			Inference: "vertex",
		},
		Agents: map[string]Agent{
			"researcher": {
				Name:      "researcher",
				Runtime:   "claude-code",
				Inference: "vertex",
				MCP:       []string{"github"},
				Skills:    []string{"analysis"},
			},
		},
	}

	// Pass non-existent binary so live checks fail gracefully
	results := Doctor(cfg, skillsDir, "/nonexistent/openshell")

	// Config checks should pass, live checks will fail (expected)
	for _, r := range results {
		// Skip live checks (they will fail since binary doesn't exist or endpoints aren't reachable)
		if r.Category == "OpenShell" {
			continue
		}
		if r.Category == "Inference" && (r.Check == "endpoint reachable" || strings.Contains(r.Check, "model")) {
			continue
		}
		if r.Status == "fail" {
			t.Errorf("Expected config checks to pass, but got failure: %s - %s - %s: %s", r.Category, r.Name, r.Check, r.Message)
		}
	}
}

func TestDoctor_MissingRuntime(t *testing.T) {
	skillsDir := t.TempDir()

	cfg := &Config{
		Runtimes: map[string]RuntimeProfile{
			"claude-code": {
				Image:      "ghcr.io/anthropics/claude-code:latest",
				Entrypoint: []string{"claude"},
				EnvMapping: map[string]string{"ANTHROPIC_API_KEY": "${key}"},
			},
		},
		Inference: map[string]InferenceSpec{
			"vertex": {
				Endpoint:     "https://us-central1-aiplatform.googleapis.com",
				Provider:     "vertex",
				DefaultModel: "claude-sonnet-4-5",
			},
		},
		Agents: map[string]Agent{
			"researcher": {
				Name:      "researcher",
				Runtime:   "missing-runtime",
				Inference: "vertex",
			},
		},
	}

	results := Doctor(cfg, skillsDir, "/nonexistent/openshell")

	// Should have at least one failure for missing runtime
	var foundFailure bool
	for _, r := range results {
		if r.Category == "Agents" && r.Status == "fail" && r.Check == "runtime exists" {
			foundFailure = true
			break
		}
	}

	if !foundFailure {
		t.Error("Expected failure for missing runtime reference")
	}
}

func TestDoctor_MissingSkill(t *testing.T) {
	skillsDir := t.TempDir()
	// Don't create the skill directory — it's missing

	cfg := &Config{
		Runtimes: map[string]RuntimeProfile{
			"claude-code": {
				Image:      "ghcr.io/anthropics/claude-code:latest",
				Entrypoint: []string{"claude"},
				EnvMapping: map[string]string{"ANTHROPIC_API_KEY": "${key}"},
			},
		},
		Inference: map[string]InferenceSpec{
			"vertex": {
				Endpoint:     "https://us-central1-aiplatform.googleapis.com",
				Provider:     "vertex",
				DefaultModel: "claude-sonnet-4-5",
			},
		},
		Agents: map[string]Agent{
			"researcher": {
				Name:      "researcher",
				Runtime:   "claude-code",
				Inference: "vertex",
				Skills:    []string{"missing-skill"},
			},
		},
	}

	results := Doctor(cfg, skillsDir, "/nonexistent/openshell")

	// Should have at least one failure for missing skill
	var foundFailure bool
	for _, r := range results {
		if r.Category == "Skills" && r.Status == "fail" {
			foundFailure = true
			break
		}
	}

	if !foundFailure {
		t.Error("Expected failure for missing skill")
	}
}

func TestDoctor_MissingInference(t *testing.T) {
	skillsDir := t.TempDir()

	cfg := &Config{
		Runtimes: map[string]RuntimeProfile{
			"claude-code": {
				Image:      "ghcr.io/anthropics/claude-code:latest",
				Entrypoint: []string{"claude"},
				EnvMapping: map[string]string{"ANTHROPIC_API_KEY": "${key}"},
			},
		},
		Inference: map[string]InferenceSpec{
			"vertex": {
				Endpoint:     "https://us-central1-aiplatform.googleapis.com",
				Provider:     "vertex",
				DefaultModel: "claude-sonnet-4-5",
			},
		},
		Agents: map[string]Agent{
			"researcher": {
				Name:      "researcher",
				Runtime:   "claude-code",
				Inference: "missing-inference",
			},
		},
	}

	results := Doctor(cfg, skillsDir, "/nonexistent/openshell")

	// Should have at least one failure for missing inference
	var foundFailure bool
	for _, r := range results {
		if r.Category == "Agents" && r.Status == "fail" && r.Check == "inference exists" {
			foundFailure = true
			break
		}
	}

	if !foundFailure {
		t.Error("Expected failure for missing inference reference")
	}
}

func TestDoctor_EmptyImage(t *testing.T) {
	skillsDir := t.TempDir()

	cfg := &Config{
		Runtimes: map[string]RuntimeProfile{
			"broken-runtime": {
				Image:      "", // Empty image
				Entrypoint: []string{"claude"},
				EnvMapping: map[string]string{"ANTHROPIC_API_KEY": "${key}"},
			},
		},
		Agents: make(map[string]Agent),
	}

	results := Doctor(cfg, skillsDir, "/nonexistent/openshell")

	// Should have at least one failure for empty image
	var foundFailure bool
	for _, r := range results {
		if r.Category == "Runtimes" && r.Status == "fail" && r.Check == "image specified" {
			foundFailure = true
			break
		}
	}

	if !foundFailure {
		t.Error("Expected failure for empty image")
	}
}

func TestDoctor_GatewayNotReachable(t *testing.T) {
	skillsDir := t.TempDir()

	cfg := &Config{
		Runtimes: map[string]RuntimeProfile{
			"claude-code": {
				Image:      "ghcr.io/anthropics/claude-code:latest",
				Entrypoint: []string{"claude"},
				EnvMapping: map[string]string{"ANTHROPIC_API_KEY": "${key}"},
			},
		},
		Inference: map[string]InferenceSpec{
			"vertex": {
				Endpoint:     "https://us-central1-aiplatform.googleapis.com",
				Provider:     "vertex",
				DefaultModel: "claude-sonnet-4-5",
			},
		},
		MCP: map[string]MCPSpec{
			"github": {
				Provider: "github",
			},
		},
		Agents: map[string]Agent{
			"researcher": {
				Name:      "researcher",
				Runtime:   "claude-code",
				Inference: "vertex",
				MCP:       []string{"github"},
			},
		},
	}

	// Pass non-existent binary path
	results := Doctor(cfg, skillsDir, "/nonexistent/openshell")

	// Gateway check should fail
	var foundGatewayFail bool
	var foundProfilesSkipped bool
	var foundProvidersSkipped bool

	for _, r := range results {
		if r.Category == "OpenShell" && r.Name == "gateway" && r.Check == "reachable" && r.Status == "fail" {
			foundGatewayFail = true
		}
		if r.Category == "OpenShell" && r.Name == "profiles" && r.Check == "synced" && r.Status == "fail" {
			if r.Message == "skipped (gateway not reachable)" {
				foundProfilesSkipped = true
			}
		}
		if r.Category == "OpenShell" && r.Name == "providers" && r.Check == "exist" && r.Status == "fail" {
			if r.Message == "skipped (gateway not reachable)" {
				foundProvidersSkipped = true
			}
		}
	}

	if !foundGatewayFail {
		t.Error("Expected gateway check to fail when binary doesn't exist")
	}
	if !foundProfilesSkipped {
		t.Error("Expected profiles check to be skipped when gateway is not reachable")
	}
	if !foundProvidersSkipped {
		t.Error("Expected providers check to be skipped when gateway is not reachable")
	}
}
