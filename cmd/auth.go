package cmd

import (
	"fmt"

	"github.com/angshumanhalder/orgcal/internal/auth"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with Google Calendar",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := auth.Authenticate(); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
		fmt.Println("Authentication successful!")
		return nil
	},
}
