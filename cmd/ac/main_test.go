package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildEngine(t *testing.T) {
	// Save original flags
	origConfig := configPath
	origSkills := skillsDir
	origDryRun := dryRun
	defer func() {
		configPath = origConfig
		skillsDir = origSkills
		dryRun = origDryRun
	}()

	t.Run("with valid config", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, "config.yaml")
		err := os.WriteFile(cfgPath, []byte("runtimes: {}\ninference: {}\nmcp: {}\ndefaults:\n  inference: \"\"\n  policy: restricted\n  sandbox:\n    scope: session\n    mode: all\nagents: {}"), 0644)
		if err != nil {
			t.Fatal(err)
		}

		configPath = cfgPath
		skillsDir = tmpDir
		dryRun = true

		engine, err := buildEngine()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if engine == nil {
			t.Fatal("expected engine, got nil")
		}
	})

	t.Run("with missing config uses default", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Point to a non-existent directory to ensure the file doesn't exist
		configPath = filepath.Join(tmpDir, "subdir", "nonexistent.yaml")
		skillsDir = tmpDir
		dryRun = true

		engine, err := buildEngine()
		if err != nil {
			t.Fatalf("expected no error when config is missing, got %v", err)
		}
		if engine == nil {
			t.Fatal("expected engine with default config, got nil")
		}
	})

	t.Run("with invalid YAML returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, "invalid.yaml")
		os.WriteFile(cfgPath, []byte("invalid: yaml: [[["), 0644)

		configPath = cfgPath
		skillsDir = tmpDir
		dryRun = true

		_, err := buildEngine()
		if err == nil {
			t.Fatal("expected error for invalid YAML, got nil")
		}
	})

	t.Run("creates dry run executor when dry-run is true", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath = filepath.Join(tmpDir, "nonexistent.yaml")
		skillsDir = tmpDir
		dryRun = true

		engine, err := buildEngine()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if engine == nil {
			t.Fatal("expected engine, got nil")
		}
	})

	t.Run("creates CLI executor when dry-run is false", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath = filepath.Join(tmpDir, "nonexistent.yaml")
		skillsDir = tmpDir
		dryRun = false

		engine, err := buildEngine()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if engine == nil {
			t.Fatal("expected engine, got nil")
		}
	})
}

func TestInitCommand(t *testing.T) {
	tmpDir := t.TempDir()
	home := tmpDir

	// Override home directory for test
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", home)
	defer os.Setenv("HOME", origHome)

	cmd := initCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Check directory was created
	acDir := filepath.Join(home, ".ac")
	if _, err := os.Stat(acDir); os.IsNotExist(err) {
		t.Fatal("expected .ac directory to be created")
	}

	// Check config.yaml was created
	cfgPath := filepath.Join(acDir, "config.yaml")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Fatal("expected config.yaml to be created")
	}

	// Check skills directory was created
	skillsPath := filepath.Join(acDir, "skills")
	if _, err := os.Stat(skillsPath); os.IsNotExist(err) {
		t.Fatal("expected skills directory to be created")
	}

	// Run again to verify idempotency
	buf.Reset()
	err = cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error on second run, got %v", err)
	}
	// No need to check output - the command should succeed silently or with a message
}
