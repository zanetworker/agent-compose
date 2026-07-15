package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

func logsCmd() *cobra.Command {
	var system bool

	cmd := &cobra.Command{
		Use:   "logs <sandbox-name>",
		Short: "Show agent output",
		Args:  cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			engine, err := buildEngine()
			if err != nil {
				return err
			}

			if system {
				reader, err := engine.Logs(cmd.Context(), args[0])
				if err != nil {
					return err
				}
				defer reader.Close()
				io.Copy(os.Stdout, reader)
				return nil
			}

			output, err := engine.AgentOutput(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("reading agent output: %w", err)
			}
			fmt.Print(output)
			return nil
		},
	}

	cmd.Flags().BoolVar(&system, "system", false, "show gateway/supervisor logs instead of agent output")
	return cmd
}
