package compose

import (
	"fmt"
	"os"
	"path/filepath"
)

// CheckResult represents the result of a single health check
type CheckResult struct {
	Category string `json:"category"` // Runtimes, Inference, MCP, Skills, Agents, OpenShell
	Name     string `json:"name"`
	Check    string `json:"check"`
	Status   string `json:"status"` // "ok" or "fail"
	Message  string `json:"message,omitempty"`
}

// Doctor runs health checks against the config. It does NOT make network
// calls (that would require a live cluster). Instead it checks:
// - Config integrity: all references resolve (agent references a runtime that exists, etc.)
// - Skills exist on disk
// - No duplicate names
// - Required fields are populated
func Doctor(cfg *Config, skillsDir string) []CheckResult {
	var results []CheckResult

	// Check runtimes
	for name, runtime := range cfg.Runtimes {
		results = append(results, checkRuntime(name, runtime)...)
	}

	// Check inference
	for name, inf := range cfg.Inference {
		results = append(results, checkInference(name, inf)...)
	}

	// Check MCP
	for name, mcp := range cfg.MCP {
		results = append(results, checkMCP(name, mcp)...)
	}

	// Check skills exist on disk
	skillChecks := make(map[string]bool)
	for _, agent := range cfg.Agents {
		for _, skillName := range agent.Skills {
			if _, checked := skillChecks[skillName]; !checked {
				skillChecks[skillName] = true
				results = append(results, checkSkill(skillName, skillsDir))
			}
		}
	}

	// Check agents
	for name, agent := range cfg.Agents {
		results = append(results, checkAgent(name, agent, cfg)...)
	}

	// Check default inference if set
	if cfg.Defaults.Inference != "" {
		if _, exists := cfg.Inference[cfg.Defaults.Inference]; !exists {
			results = append(results, CheckResult{
				Category: "Defaults",
				Name:     "default inference",
				Check:    "exists in config",
				Status:   "fail",
				Message:  fmt.Sprintf("default inference %q not found in inference config", cfg.Defaults.Inference),
			})
		} else {
			results = append(results, CheckResult{
				Category: "Defaults",
				Name:     "default inference",
				Check:    "exists in config",
				Status:   "ok",
			})
		}
	}

	return results
}

func checkRuntime(name string, runtime RuntimeProfile) []CheckResult {
	var results []CheckResult

	// Check image is non-empty
	if runtime.Image == "" {
		results = append(results, CheckResult{
			Category: "Runtimes",
			Name:     name,
			Check:    "image specified",
			Status:   "fail",
			Message:  "image field is empty",
		})
	} else {
		results = append(results, CheckResult{
			Category: "Runtimes",
			Name:     name,
			Check:    "image specified",
			Status:   "ok",
		})
	}

	// Check entrypoint is non-empty
	if len(runtime.Entrypoint) == 0 {
		results = append(results, CheckResult{
			Category: "Runtimes",
			Name:     name,
			Check:    "entrypoint specified",
			Status:   "fail",
			Message:  "entrypoint field is empty",
		})
	} else {
		results = append(results, CheckResult{
			Category: "Runtimes",
			Name:     name,
			Check:    "entrypoint specified",
			Status:   "ok",
		})
	}

	// Check env-mapping is non-empty
	if len(runtime.EnvMapping) == 0 {
		results = append(results, CheckResult{
			Category: "Runtimes",
			Name:     name,
			Check:    "env-mapping specified",
			Status:   "fail",
			Message:  "env-mapping field is empty",
		})
	} else {
		results = append(results, CheckResult{
			Category: "Runtimes",
			Name:     name,
			Check:    "env-mapping specified",
			Status:   "ok",
		})
	}

	return results
}

func checkInference(name string, inf InferenceSpec) []CheckResult {
	var results []CheckResult

	// Check endpoint is non-empty
	if inf.Endpoint == "" {
		results = append(results, CheckResult{
			Category: "Inference",
			Name:     name,
			Check:    "endpoint specified",
			Status:   "fail",
			Message:  "endpoint field is empty",
		})
	} else {
		results = append(results, CheckResult{
			Category: "Inference",
			Name:     name,
			Check:    "endpoint specified",
			Status:   "ok",
		})
	}

	// Check provider is non-empty
	if inf.Provider == "" {
		results = append(results, CheckResult{
			Category: "Inference",
			Name:     name,
			Check:    "provider specified",
			Status:   "fail",
			Message:  "provider field is empty",
		})
	} else {
		results = append(results, CheckResult{
			Category: "Inference",
			Name:     name,
			Check:    "provider specified",
			Status:   "ok",
		})
	}

	// Check default-model is non-empty
	if inf.DefaultModel == "" {
		results = append(results, CheckResult{
			Category: "Inference",
			Name:     name,
			Check:    "default-model specified",
			Status:   "fail",
			Message:  "default-model field is empty",
		})
	} else {
		results = append(results, CheckResult{
			Category: "Inference",
			Name:     name,
			Check:    "default-model specified",
			Status:   "ok",
		})
	}

	return results
}

func checkMCP(name string, mcp MCPSpec) []CheckResult {
	var results []CheckResult

	// Check provider is non-empty
	if mcp.Provider == "" {
		results = append(results, CheckResult{
			Category: "MCP",
			Name:     name,
			Check:    "provider specified",
			Status:   "fail",
			Message:  "provider field is empty",
		})
	} else {
		results = append(results, CheckResult{
			Category: "MCP",
			Name:     name,
			Check:    "provider specified",
			Status:   "ok",
		})
	}

	return results
}

func checkSkill(name, skillsDir string) CheckResult {
	skillPath := filepath.Join(skillsDir, name, "SKILL.md")
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		return CheckResult{
			Category: "Skills",
			Name:     name,
			Check:    "exists on disk",
			Status:   "fail",
			Message:  fmt.Sprintf("skill directory not found at %s", skillPath),
		}
	}
	return CheckResult{
		Category: "Skills",
		Name:     name,
		Check:    "exists on disk",
		Status:   "ok",
	}
}

func checkAgent(name string, agent Agent, cfg *Config) []CheckResult {
	var results []CheckResult

	// Check runtime exists if specified (or has inline image)
	if agent.Runtime != "" {
		if _, exists := cfg.Runtimes[agent.Runtime]; !exists {
			results = append(results, CheckResult{
				Category: "Agents",
				Name:     name,
				Check:    "runtime exists",
				Status:   "fail",
				Message:  fmt.Sprintf("runtime %q not found in config", agent.Runtime),
			})
		} else {
			results = append(results, CheckResult{
				Category: "Agents",
				Name:     name,
				Check:    "runtime exists",
				Status:   "ok",
			})
		}
	} else if agent.Image != "" {
		// Agent has inline image, that's also valid
		results = append(results, CheckResult{
			Category: "Agents",
			Name:     name,
			Check:    "runtime exists",
			Status:   "ok",
			Message:  "using inline image",
		})
	} else {
		// Neither runtime nor inline image
		results = append(results, CheckResult{
			Category: "Agents",
			Name:     name,
			Check:    "runtime exists",
			Status:   "fail",
			Message:  "no runtime reference or inline image specified",
		})
	}

	// Check inference exists if specified
	if agent.Inference != "" {
		if _, exists := cfg.Inference[agent.Inference]; !exists {
			results = append(results, CheckResult{
				Category: "Agents",
				Name:     name,
				Check:    "inference exists",
				Status:   "fail",
				Message:  fmt.Sprintf("inference %q not found in config", agent.Inference),
			})
		} else {
			results = append(results, CheckResult{
				Category: "Agents",
				Name:     name,
				Check:    "inference exists",
				Status:   "ok",
			})
		}
	}

	// Check all MCP references exist
	for _, mcpName := range agent.MCP {
		if _, exists := cfg.MCP[mcpName]; !exists {
			results = append(results, CheckResult{
				Category: "Agents",
				Name:     name,
				Check:    "mcp exists",
				Status:   "fail",
				Message:  fmt.Sprintf("mcp %q not found in config", mcpName),
			})
		} else {
			results = append(results, CheckResult{
				Category: "Agents",
				Name:     name,
				Check:    "mcp exists",
				Status:   "ok",
				Message:  mcpName,
			})
		}
	}

	return results
}
