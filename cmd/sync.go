package cmd

import (
	"fmt"

	"github.com/angshumanhalder/orgcal/internal/sync"
	"github.com/spf13/cobra"
)

var syncDir string

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Bidirectional sync between org files and Google Calendar",
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := sync.Run(syncDir)
		if err != nil {
			return err
		}
		fmt.Printf("Imported: %d  Exported: %d\n", result.Imported, result.Exported)
		return nil
	},
}

func init() {
	syncCmd.Flags().StringVarP(&syncDir, "dir", "d", "~/org", "Org directory")
}
