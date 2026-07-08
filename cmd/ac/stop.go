package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func stopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <agent-name>",
		Short: "Stop agent and delete sandbox",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			engine, err := buildEngine()
			if err != nil {
				return err
			}
			if err := engine.Stop(cmd.Context(), args[0]); err != nil {
				return err
			}
			fmt.Printf("Agent %s stopped\n", args[0])
			return nil
		},
	}
}
