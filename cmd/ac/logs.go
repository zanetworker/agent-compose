package main

import (
	"io"
	"os"

	"github.com/spf13/cobra"
)

func logsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logs <agent-name>",
		Short: "Stream sandbox output",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			engine, err := buildEngine()
			if err != nil {
				return err
			}
			reader, err := engine.Logs(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			defer reader.Close()
			io.Copy(os.Stdout, reader)
			return nil
		},
	}
}
