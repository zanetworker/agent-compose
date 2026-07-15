package main

import (
	"github.com/spf13/cobra"
	"github.com/zanetworker/agent-compose/pkg/compose"
)

func attachCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "attach <sandbox-name>",
		Short:         "Attach to a running sandbox",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			engine, err := buildEngine()
			if err != nil {
				return err
			}
			return engine.Attach(cmd.Context(), args[0], compose.AttachOpts{})
		},
	}
	return cmd
}
