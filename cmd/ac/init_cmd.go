package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"github.com/zanetworker/agent-compose/pkg/compose"
)

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create ~/.agentctl/ with default config",
		RunE: func(cmd *cobra.Command, args []string) error {
			home, _ := os.UserHomeDir()
			dir := filepath.Join(home, ".agentctl")
			if err := os.MkdirAll(filepath.Join(dir, "skills"), 0755); err != nil {
				return err
			}

			cfgPath := filepath.Join(dir, "config.yaml")
			if _, err := os.Stat(cfgPath); err == nil {
				fmt.Fprintf(os.Stderr, "config already exists at %s\n", cfgPath)
				return nil
			}

			cfg := compose.DefaultConfig()
			data, err := yaml.Marshal(cfg)
			if err != nil {
				return err
			}
			if err := os.WriteFile(cfgPath, data, 0644); err != nil {
				return err
			}

			fmt.Printf("Created %s\n", dir)
			fmt.Printf("Edit %s to add your inference providers and MCP servers.\n", cfgPath)
			return nil
		},
	}
}
