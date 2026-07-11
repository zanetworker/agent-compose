package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/zanetworker/agent-compose/pkg/compose"
)

var (
	configPath string
	skillsDir  string
	dryRun     bool
	jsonOutput bool
)

func main() {
	root := &cobra.Command{
		Use:   "ac",
		Short: "Agent composition engine for OpenShell",
		Long:  "Compose agents with the right model, MCP servers, skills, and credentials, and run them securely in OpenShell sandboxes.",
	}

	home, _ := os.UserHomeDir()
	defaultConfig := filepath.Join(home, ".ac", "config.yaml")
	defaultSkills := filepath.Join(home, ".ac", "skills")

	root.PersistentFlags().StringVar(&configPath, "config", defaultConfig, "path to config.yaml")
	root.PersistentFlags().StringVar(&skillsDir, "skills-dir", defaultSkills, "path to skills directory")
	root.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "print openshell commands without executing")
	root.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output as JSON")

	root.AddCommand(initCmd())
	root.AddCommand(runCmd())
	root.AddCommand(stopCmd())
	root.AddCommand(listCmd())
	root.AddCommand(logsCmd())
	root.AddCommand(getCmd())
	root.AddCommand(applyCmd())
	root.AddCommand(doctorCmd())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func buildEngine() (*compose.Engine, error) {
	cfg, err := compose.LoadConfig(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cfg = compose.DefaultConfig()
		} else {
			return nil, fmt.Errorf("loading config: %w", err)
		}
	}

	var executor compose.Executor
	if dryRun {
		executor = compose.NewDryRunExecutor(os.Stdout)
	} else {
		executor = compose.NewCLIExecutor("openshell")
	}

	return compose.New(
		compose.WithConfig(cfg),
		compose.WithExecutor(executor),
		compose.WithSkillsDir(skillsDir),
	), nil
}
