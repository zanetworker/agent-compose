package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zanetworker/agent-compose/pkg/compose"
)

func runCmd() *cobra.Command {
	var (
		prompt    string
		workspace string
		harness   string
		inference string
		mcp       []string
		skills    []string
	)

	cmd := &cobra.Command{
		Use:   "run [agent-name]",
		Short: "Resolve agent config, create sandbox, and start agent",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			engine, err := buildEngine()
			if err != nil {
				return err
			}

			opts := compose.RunOpts{
				Prompt:    prompt,
				Workspace: workspace,
			}

			if len(args) == 0 {
				if harness == "" {
					return fmt.Errorf("either provide an agent name or --harness")
				}
				opts.Agent = &compose.Agent{
					Harness:   harness,
					Inference: inference,
					MCP:       mcp,
					Skills:    skills,
					Prompt:    prompt,
					Workspace: workspace,
				}
				prompt = ""
			}

			name := ""
			if len(args) > 0 {
				name = args[0]
			}

			run, err := engine.Run(cmd.Context(), name, opts)
			if err != nil {
				return err
			}

			if !dryRun {
				fmt.Printf("Agent %s running in sandbox %s\n", run.Agent, run.Sandbox)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&prompt, "prompt", "", "task prompt")
	cmd.Flags().StringVar(&workspace, "workspace", "", "workspace path")
	cmd.Flags().StringVar(&harness, "harness", "", "harness profile (for inline agents)")
	cmd.Flags().StringVar(&inference, "inference", "", "inference provider (for inline agents)")
	cmd.Flags().StringSliceVar(&mcp, "mcp", nil, "MCP servers (for inline agents)")
	cmd.Flags().StringSliceVar(&skills, "skills", nil, "skills (for inline agents)")

	return cmd
}
