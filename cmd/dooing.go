package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/angshumanhalder/orgcal/internal/dooing"
	"github.com/spf13/cobra"
)

var dooingFile string

var dooingCmd = &cobra.Command{
	Use:   "dooing",
	Short: "Sync dooing todos with Google Calendar",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := dooingFile
		if path == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			path = filepath.Join(home, ".local", "share", "nvim", "dooing_todos.json")
		}
		result, err := dooing.Sync(path)
		if err != nil {
			return err
		}
		fmt.Printf("Exported: %d  Deleted: %d\n", result.Exported, result.Deleted)
		return nil
	},
}

func init() {
	dooingCmd.Flags().StringVarP(&dooingFile, "file", "f", "", "Path to dooing JSON (default: ~/.local/share/nvim/dooing_todos.json)")
}
