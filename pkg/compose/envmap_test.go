package compose

import "testing"

func TestExpandEnvMapping_BasicVars(t *testing.T) {
	mapping := map[string]string{
		"ANTHROPIC_BASE_URL":             "${endpoint}",
		"ANTHROPIC_API_KEY":              "${key}",
		"ANTHROPIC_DEFAULT_SONNET_MODEL": "${model}",
	}
	vars := map[string]string{
		"endpoint": "https://maas.apps.cluster.com/v1",
		"key":      "",
		"model":    "granite-3.3-8b-instruct",
	}
	result := ExpandEnvMapping(mapping, vars)

	if result["ANTHROPIC_BASE_URL"] != "https://maas.apps.cluster.com/v1" {
		t.Errorf("endpoint: got %q", result["ANTHROPIC_BASE_URL"])
	}
	if result["ANTHROPIC_DEFAULT_SONNET_MODEL"] != "granite-3.3-8b-instruct" {
		t.Errorf("model: got %q", result["ANTHROPIC_DEFAULT_SONNET_MODEL"])
	}
}

func TestExpandEnvMapping_LiteralValues(t *testing.T) {
	mapping := map[string]string{
		"CLAUDE_CODE_USE_VERTEX": "1",
		"CLOUD_ML_REGION":       "${region}",
	}
	vars := map[string]string{
		"region": "us-central1",
	}
	result := ExpandEnvMapping(mapping, vars)

	if result["CLAUDE_CODE_USE_VERTEX"] != "1" {
		t.Errorf("literal: got %q", result["CLAUDE_CODE_USE_VERTEX"])
	}
	if result["CLOUD_ML_REGION"] != "us-central1" {
		t.Errorf("region: got %q", result["CLOUD_ML_REGION"])
	}
}

func TestExpandEnvMapping_TierVars(t *testing.T) {
	mapping := map[string]string{
		"ANTHROPIC_DEFAULT_SONNET_MODEL": "${model}",
		"ANTHROPIC_DEFAULT_OPUS_MODEL":   "${model.opus}",
		"ANTHROPIC_DEFAULT_HAIKU_MODEL":  "${model.haiku}",
	}
	vars := map[string]string{
		"model":       "granite-3.3-8b-instruct",
		"model.opus":  "granite-3.3-8b-instruct",
		"model.haiku": "granite-3.3-2b-instruct",
	}
	result := ExpandEnvMapping(mapping, vars)

	if result["ANTHROPIC_DEFAULT_OPUS_MODEL"] != "granite-3.3-8b-instruct" {
		t.Errorf("opus: got %q", result["ANTHROPIC_DEFAULT_OPUS_MODEL"])
	}
	if result["ANTHROPIC_DEFAULT_HAIKU_MODEL"] != "granite-3.3-2b-instruct" {
		t.Errorf("haiku: got %q", result["ANTHROPIC_DEFAULT_HAIKU_MODEL"])
	}
}

func TestExpandEnvMapping_MissingVarOmitted(t *testing.T) {
	mapping := map[string]string{
		"ANTHROPIC_DEFAULT_OPUS_MODEL": "${model.opus}",
	}
	vars := map[string]string{}
	result := ExpandEnvMapping(mapping, vars)

	if _, ok := result["ANTHROPIC_DEFAULT_OPUS_MODEL"]; ok {
		t.Error("expected missing var to be omitted from result")
	}
}

func TestExpandEnvMapping_EmptyKeySkipped(t *testing.T) {
	mapping := map[string]string{
		"ANTHROPIC_API_KEY": "${key}",
	}
	vars := map[string]string{"key": ""}
	result := ExpandEnvMapping(mapping, vars)

	if _, ok := result["ANTHROPIC_API_KEY"]; ok {
		t.Error("expected empty-value var to be omitted")
	}
}
