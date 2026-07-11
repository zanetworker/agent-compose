package compose

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ProviderProfile represents an OpenShell provider profile YAML structure.
type ProviderProfile struct {
	ID               string              `yaml:"id"`
	DisplayName      string              `yaml:"display_name"`
	Description      string              `yaml:"description,omitempty"`
	Category         string              `yaml:"category"`
	InferenceCapable bool                `yaml:"inference_capable,omitempty"`
	Credentials      []ProfileCredential `yaml:"credentials,omitempty"`
	Endpoints        []ProfileEndpoint   `yaml:"endpoints,omitempty"`
}

type ProfileCredential struct {
	Name     string   `yaml:"name"`
	EnvVars  []string `yaml:"env_vars"`
	Required bool     `yaml:"required"`
}

type ProfileEndpoint struct {
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	Protocol    string `yaml:"protocol"`
	Access      string `yaml:"access"`
	Enforcement string `yaml:"enforcement"`
}

// GenerateProfiles builds OpenShell provider profiles from the config.
// It generates one profile per inference provider and one per MCP server.
// Runtime profiles are NOT pushed (OpenShell doesn't handle image/entrypoint).
func GenerateProfiles(cfg *Config) []ProviderProfile {
	var profiles []ProviderProfile

	// Generate profiles for inference providers
	for name, spec := range cfg.Inference {
		profile := ProviderProfile{
			ID:               name,
			DisplayName:      formatDisplayName(spec.Provider),
			Description:      fmt.Sprintf("%s inference gateway", spec.Provider),
			Category:         "inference",
			InferenceCapable: true,
		}

		// Add credential
		if spec.Provider != "" {
			profile.Credentials = []ProfileCredential{
				{
					Name:     spec.Provider,
					EnvVars:  []string{strings.ToUpper(spec.Provider) + "_API_KEY"},
					Required: true,
				},
			}
		}

		// Parse egress into endpoints
		for _, egress := range spec.Egress {
			endpoint := parseEgress(egress)
			profile.Endpoints = append(profile.Endpoints, endpoint)
		}

		profiles = append(profiles, profile)
	}

	// Generate profiles for MCP servers
	for name, spec := range cfg.MCP {
		profile := ProviderProfile{
			ID:          name,
			DisplayName: formatDisplayName(spec.Provider),
			Description: fmt.Sprintf("%s MCP server", spec.Provider),
			Category:    "other",
		}

		// Add credential
		if spec.Provider != "" {
			profile.Credentials = []ProfileCredential{
				{
					Name:     spec.Provider,
					EnvVars:  []string{strings.ToUpper(spec.Provider) + "_API_KEY"},
					Required: true,
				},
			}
		}

		// Parse egress into endpoints
		for _, egress := range spec.Egress {
			endpoint := parseEgress(egress)
			profile.Endpoints = append(profile.Endpoints, endpoint)
		}

		profiles = append(profiles, profile)
	}

	return profiles
}

// SyncProfiles generates provider profiles from config and imports them
// into the OpenShell gateway via `openshell provider profile import`.
// It writes each profile to a temp file and calls the CLI.
// Returns the list of profile IDs that were synced.
func SyncProfiles(ctx context.Context, cfg *Config, openshellBin string) ([]string, error) {
	profiles := GenerateProfiles(cfg)
	var synced []string

	for _, profile := range profiles {
		// Marshal to YAML
		data, err := yaml.Marshal(profile)
		if err != nil {
			return synced, fmt.Errorf("marshaling profile %s: %w", profile.ID, err)
		}

		// Write to temp file
		tmpfile, err := os.CreateTemp("", fmt.Sprintf("profile-%s-*.yaml", profile.ID))
		if err != nil {
			return synced, fmt.Errorf("creating temp file for %s: %w", profile.ID, err)
		}
		defer os.Remove(tmpfile.Name())

		if _, err := tmpfile.Write(data); err != nil {
			tmpfile.Close()
			return synced, fmt.Errorf("writing profile %s: %w", profile.ID, err)
		}
		tmpfile.Close()

		// Import via openshell CLI
		cmd := exec.CommandContext(ctx, openshellBin, "provider", "profile", "import", "-f", tmpfile.Name())
		output, err := cmd.CombinedOutput()
		if err != nil {
			// Check if profile already exists (idempotent)
			if strings.Contains(string(output), "already exists") {
				continue
			}
			return synced, fmt.Errorf("importing profile %s: %w\noutput: %s", profile.ID, err, output)
		}

		synced = append(synced, profile.ID)
	}

	return synced, nil
}

// parseEgress parses an egress string (format "host:port") into a ProfileEndpoint.
func parseEgress(egress string) ProfileEndpoint {
	parts := strings.Split(egress, ":")
	host := parts[0]
	port := 443 // default to HTTPS

	if len(parts) > 1 {
		if p, err := strconv.Atoi(parts[1]); err == nil {
			port = p
		}
	}

	return ProfileEndpoint{
		Host:        host,
		Port:        port,
		Protocol:    "rest",
		Access:      "read-write",
		Enforcement: "enforce",
	}
}

// formatDisplayName formats a provider name into a display name.
func formatDisplayName(provider string) string {
	switch provider {
	case "maas":
		return "MaaS Gateway"
	case "openai":
		return "OpenAI"
	case "anthropic":
		return "Anthropic"
	case "github":
		return "GitHub"
	default:
		// Capitalize first letter
		if len(provider) > 0 {
			return strings.ToUpper(provider[:1]) + provider[1:]
		}
		return provider
	}
}
