package compose

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateProfiles_InferenceProvider(t *testing.T) {
	cfg := &Config{
		Inference: map[string]InferenceSpec{
			"maas": {
				Name:         "maas",
				Endpoint:     "https://maas.apps.cluster.com/v1",
				Provider:     "maas",
				DefaultModel: "claude-3-5-sonnet-20241022",
				Egress:       []string{"maas.apps.cluster.com:443"},
			},
		},
	}

	profiles := GenerateProfiles(cfg)

	assert.Len(t, profiles, 1)
	profile := profiles[0]

	assert.Equal(t, "maas", profile.ID)
	assert.Equal(t, "MaaS Gateway", profile.DisplayName)
	assert.Equal(t, "inference", profile.Category)
	assert.True(t, profile.InferenceCapable)
	assert.Len(t, profile.Endpoints, 1)
	assert.Equal(t, "maas.apps.cluster.com", profile.Endpoints[0].Host)
	assert.Equal(t, 443, profile.Endpoints[0].Port)
	assert.Equal(t, "rest", profile.Endpoints[0].Protocol)
	assert.Equal(t, "read-write", profile.Endpoints[0].Access)
	assert.Equal(t, "enforce", profile.Endpoints[0].Enforcement)
}

func TestGenerateProfiles_MCPServer(t *testing.T) {
	cfg := &Config{
		MCP: map[string]MCPSpec{
			"github": {
				Name:     "github",
				Provider: "github",
				Egress:   []string{"api.github.com:443", "github.com:443"},
			},
		},
	}

	profiles := GenerateProfiles(cfg)

	assert.Len(t, profiles, 1)
	profile := profiles[0]

	assert.Equal(t, "github", profile.ID)
	assert.Equal(t, "GitHub", profile.DisplayName)
	assert.Equal(t, "other", profile.Category)
	assert.False(t, profile.InferenceCapable)
	assert.Len(t, profile.Endpoints, 2)

	// Check first endpoint
	assert.Equal(t, "api.github.com", profile.Endpoints[0].Host)
	assert.Equal(t, 443, profile.Endpoints[0].Port)

	// Check second endpoint
	assert.Equal(t, "github.com", profile.Endpoints[1].Host)
	assert.Equal(t, 443, profile.Endpoints[1].Port)
}

func TestGenerateProfiles_Empty(t *testing.T) {
	cfg := &Config{}

	profiles := GenerateProfiles(cfg)

	assert.Len(t, profiles, 0)
}

func TestGenerateProfiles_Combined(t *testing.T) {
	cfg := &Config{
		Inference: map[string]InferenceSpec{
			"maas": {
				Name:     "maas",
				Provider: "maas",
				Endpoint: "https://maas.apps.cluster.com/v1",
				Egress:   []string{"maas.apps.cluster.com:443"},
			},
			"openai": {
				Name:     "openai",
				Provider: "openai",
				Endpoint: "https://api.openai.com/v1",
				Egress:   []string{"api.openai.com:443"},
			},
		},
		MCP: map[string]MCPSpec{
			"github": {
				Name:     "github",
				Provider: "github",
				Egress:   []string{"api.github.com:443"},
			},
		},
	}

	profiles := GenerateProfiles(cfg)

	assert.Len(t, profiles, 3)

	// Count inference vs other
	inferenceCount := 0
	otherCount := 0
	for _, p := range profiles {
		if p.Category == "inference" {
			inferenceCount++
			assert.True(t, p.InferenceCapable)
		} else if p.Category == "other" {
			otherCount++
			assert.False(t, p.InferenceCapable)
		}
	}

	assert.Equal(t, 2, inferenceCount)
	assert.Equal(t, 1, otherCount)
}
