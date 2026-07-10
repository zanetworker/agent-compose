package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List running agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			engine, err := buildEngine()
			if err != nil {
				return err
			}
			agents, err := engine.List(cmd.Context())
			if err != nil {
				return err
			}
			if jsonOutput {
				data, _ := json.MarshalIndent(agents, "", "  ")
				fmt.Fprintln(os.Stdout, string(data))
				return nil
			}
			if len(agents) == 0 {
				fmt.Println("No running agents")
				return nil
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tSANDBOX\tSTATUS\tAGE")
			for _, a := range agents {
				age := time.Since(time.Unix(a.Since, 0)).Round(time.Second)
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", a.Name, a.Sandbox, a.Status, age)
			}
			w.Flush()
			return nil
		},
	}
}
