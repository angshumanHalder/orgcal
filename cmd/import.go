package cmd

import (
	"fmt"

	"github.com/angshumanhalder/orgcal/internal/gcal"
	"github.com/angshumanhalder/orgcal/internal/org"
	"github.com/spf13/cobra"
)

var importDir string

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import Google Calendar events into org files",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := gcal.NewClient()
		if err != nil {
			return err
		}
		events, err := client.ListEvents()
		if err != nil {
			return err
		}
		if err := org.WriteEvents(importDir, events); err != nil {
			return err
		}
		fmt.Printf("Imported %d events to %s\n", len(events), importDir)
		return nil
	},
}

func init() {
	importCmd.Flags().StringVarP(&importDir, "dir", "d", "~/org", "Org directory")
}
