package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zanetworker/agent-compose/pkg/compose"
)

func doctorCmd() *cobra.Command {
	var openshellBin string

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Validate config and check that all references resolve",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := compose.LoadConfig(configPath)
			if err != nil {
				// If config doesn't exist, that's the first problem
				if errors.Is(err, os.ErrNotExist) {
					fmt.Println("No config found. Run 'ac init' first.")
					return nil
				}
				return err
			}
			results := compose.Doctor(cfg, skillsDir, openshellBin)

			if jsonOutput {
				data, _ := json.MarshalIndent(results, "", "  ")
				fmt.Fprintln(os.Stdout, string(data))
				return nil
			}

			// Pretty print grouped by category
			hasFailures := false
			currentCategory := ""
			for _, r := range results {
				if r.Category != currentCategory {
					if currentCategory != "" {
						fmt.Println()
					}
					fmt.Println(r.Category)
					currentCategory = r.Category
				}
				status := "ok"
				if r.Status == "fail" {
					status = "FAIL"
					hasFailures = true
				}
				line := fmt.Sprintf("  %-20s %-30s %s", r.Name, r.Check, status)
				if r.Message != "" {
					line += fmt.Sprintf(" (%s)", r.Message)
				}
				fmt.Println(line)
			}

			if hasFailures {
				fmt.Println("\nFix the issues above and re-run 'ac doctor'.")
				os.Exit(1)
			} else {
				fmt.Println("\nAll checks passed.")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&openshellBin, "openshell-bin", "openshell", "Path to openshell binary")
	return cmd
}
