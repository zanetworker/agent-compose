package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func applyCmd() *cobra.Command {
	var syncProfiles bool

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply configuration changes",
		RunE: func(cmd *cobra.Command, args []string) error {
			engine, err := buildEngine()
			if err != nil {
				return err
			}
			if syncProfiles {
				ids, err := engine.SyncProfiles(cmd.Context())
				if err != nil {
					return fmt.Errorf("syncing profiles: %w", err)
				}
				if len(ids) == 0 {
					fmt.Println("No profiles to sync")
				} else {
					fmt.Printf("Synced %d profile(s): %s\n", len(ids), strings.Join(ids, ", "))
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&syncProfiles, "sync-profiles", false, "push provider profiles to OpenShell gateway")
	return cmd
}
