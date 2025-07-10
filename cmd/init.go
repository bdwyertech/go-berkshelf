package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new Berksfile",
	Long: `Initialize a new Berksfile in the current directory.
This creates a basic Berksfile with common configuration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if Berksfile already exists
		if _, err := os.Stat(berksfilePath); err == nil {
			return fmt.Errorf("Berksfile already exists at %s", berksfilePath) //lint:ignore ST1005 error strings should not be capitalized
		}

		// Create a basic Berksfile
		content := `source "https://supermarket.chef.io"

metadata
`

		// Ensure directory exists
		dir := filepath.Dir(berksfilePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		// Write the Berksfile
		if err := os.WriteFile(berksfilePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to create Berksfile: %w", err)
		}

		log.Infof("Successfully created %s\n", berksfilePath)
		return nil
	},
}
