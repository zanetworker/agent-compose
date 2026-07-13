package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zanetworker/agent-compose/pkg/compose"
)

func runCmd() *cobra.Command {
	var (
		prompt          string
		workspace       string
		runtime         string
		inference       string
		model           string
		skipPermissions bool
		interactive     bool
		mcp             []string
		skills          []string
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
				Prompt:          prompt,
				Workspace:       workspace,
				Inference:       inference,
				Model:           model,
				SkipPermissions: skipPermissions,
				Interactive:     interactive,
			}

			if len(args) == 0 {
				if runtime == "" {
					return fmt.Errorf("either provide an agent name or --runtime")
				}
				opts.Agent = &compose.Agent{
					Runtime:   runtime,
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
	cmd.Flags().StringVar(&runtime, "runtime", "", "runtime profile (for inline agents)")
	cmd.Flags().StringVar(&inference, "inference", "", "override inference provider")
	cmd.Flags().StringVar(&model, "model", "", "override model")
	cmd.Flags().BoolVar(&skipPermissions, "skip-permissions", false, "skip agent permission prompts (use with caution)")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "attach an interactive terminal to the sandbox")
	cmd.Flags().StringSliceVar(&mcp, "mcp", nil, "MCP servers")
	cmd.Flags().StringSliceVar(&skills, "skills", nil, "skills")

	return cmd
}
