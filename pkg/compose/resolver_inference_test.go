package compose

import (
	"context"
	"errors"
	"testing"
)

func TestConfigInferenceResolver_Resolve(t *testing.T) {
	cfg := &Config{
		Inference: map[string]InferenceSpec{
			"maas": {
				Endpoint:     "https://maas.example.com/v1",
				Provider:     "maas-anthropic",
				DefaultModel: "granite-3.3-8b",
				Egress:       []string{"maas.example.com:443"},
			},
		},
	}
	r := NewConfigInferenceResolver(cfg)

	spec, err := r.Resolve(context.Background(), "maas")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if spec.Endpoint != "https://maas.example.com/v1" {
		t.Errorf("endpoint = %q, want https://maas.example.com/v1", spec.Endpoint)
	}
	if spec.Provider != "maas-anthropic" {
		t.Errorf("provider = %q, want maas-anthropic", spec.Provider)
	}
	if spec.DefaultModel != "granite-3.3-8b" {
		t.Errorf("default-model = %q, want granite-3.3-8b", spec.DefaultModel)
	}
	if len(spec.Egress) != 1 {
		t.Errorf("egress len = %d, want 1", len(spec.Egress))
	}
	if spec.Name != "maas" {
		t.Errorf("name = %q, want maas", spec.Name)
	}
}

func TestConfigInferenceResolver_NotFound(t *testing.T) {
	cfg := &Config{Inference: map[string]InferenceSpec{}}
	r := NewConfigInferenceResolver(cfg)

	_, err := r.Resolve(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestConfigInferenceResolver_List(t *testing.T) {
	cfg := &Config{
		Inference: map[string]InferenceSpec{
			"maas": {
				Endpoint: "https://maas.example.com/v1",
				Provider: "maas-anthropic",
			},
			"openai": {
				Endpoint: "https://api.openai.com/v1",
				Provider: "openai",
			},
		},
	}
	r := NewConfigInferenceResolver(cfg)

	specs, err := r.List(context.Background())
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(specs) != 2 {
		t.Errorf("len = %d, want 2", len(specs))
	}
}
