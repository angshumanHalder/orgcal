package cmd

import (
	"fmt"

	"github.com/angshumanhalder/orgcal/internal/gcal"
	"github.com/angshumanhalder/orgcal/internal/org"
	"github.com/spf13/cobra"
)

var exportDir string

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export org TODOs to Google Calendar",
	RunE: func(cmd *cobra.Command, args []string) error {
		todos, err := org.ReadTodos(exportDir)
		if err != nil {
			return err
		}
		client, err := gcal.NewClient()
		if err != nil {
			return err
		}
		exported, _, err := client.ExportTodos(todos)
		if err != nil {
			return err
		}
		fmt.Printf("Exported %d todos to Google Calendar\n", exported)
		return nil
	},
}

func init() {
	exportCmd.Flags().StringVarP(&exportDir, "dir", "d", "~/org", "Org directory")
}
