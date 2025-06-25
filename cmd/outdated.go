package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/bdwyer/go-berkshelf/pkg/berksfile"
	"github.com/bdwyer/go-berkshelf/pkg/lockfile"
	"github.com/bdwyer/go-berkshelf/pkg/outdated"
	"github.com/bdwyer/go-berkshelf/pkg/source"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(outdatedCmd)

	// Add flags
	outdatedCmd.Flags().StringP("format", "f", "table", "Output format (table, json)")
}

var outdatedCmd = &cobra.Command{
	Use:   "outdated [COOKBOOK...]",
	Short: "Show outdated cookbook dependencies",
	Long: `Show cookbooks that have newer versions available.

This command compares the versions in your lock file with the latest
available versions from configured sources and shows which cookbooks
can be updated.

Examples:
  berks outdated           # Show all outdated cookbooks
  berks outdated nginx     # Check if nginx is outdated
  berks outdated --format json  # Output as JSON`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if Berksfile exists
		if _, err := os.Stat("Berksfile"); os.IsNotExist(err) {
			return fmt.Errorf("no Berksfile found in current directory")
		}

		// Parse Berksfile
		berksfileContent, err := os.ReadFile("Berksfile")
		if err != nil {
			return fmt.Errorf("failed to read Berksfile: %w", err)
		}

		bf, err := berksfile.ParseString(string(berksfileContent))
		if err != nil {
			return fmt.Errorf("failed to parse Berksfile: %w", err)
		}

		// Load lock file
		manager := lockfile.NewManager(".")
		lockFile, err := manager.Load()
		if err != nil {
			return fmt.Errorf("no lock file found. Run 'berks install' first: %w", err)
		}

		// Create source manager
		factory := source.NewFactory()
		sourceManager, err := factory.CreateFromBerksfile(bf)
		if err != nil {
			return fmt.Errorf("failed to create source manager: %w", err)
		}

		log.Infoln("Checking for outdated cookbooks...")

		// Create outdated checker
		checker := outdated.New(lockFile, sourceManager)

		// Check for outdated cookbooks
		outdatedCookbooks, err := checker.Check(cmd.Context(), args)
		if err != nil {
			return fmt.Errorf("failed to check for outdated cookbooks: %w", err)
		}

		// Output results
		if len(outdatedCookbooks) == 0 {
			fmt.Println("All cookbooks are up to date!")
			return nil
		}

		switch outdatedFormat := strings.ToLower(viper.GetString("format")); outdatedFormat {
		case "json":
			return outputOutdatedJSON(outdatedCookbooks)
		case "table":
			return outputOutdatedTable(outdatedCookbooks)
		default:
			return fmt.Errorf("unsupported format: %s (supported: table, json)", outdatedFormat)
		}
	},
}

func outputOutdatedJSON(cookbooks []outdated.Cookbook) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(cookbooks)
}

func outputOutdatedTable(cookbooks []outdated.Cookbook) error {
	log.Printf("Found %d outdated cookbook(s):\n\n", len(cookbooks))

	table := tablewriter.NewTable(os.Stdout)
	table.Configure(func(config *tablewriter.Config) {
		config.Row.Alignment.Global = tw.AlignLeft
	})
	table.Header("COOKBOOK", "CURRENT", "LATEST", "SOURCE")

	data := [][]any{}
	for _, cookbook := range cookbooks {
		data = append(data, []any{
			cookbook.Name,
			cookbook.CurrentVersion,
			cookbook.LatestVersion,
			cookbook.Source,
		})
	}

	table.Bulk(data)
	return table.Render()
}
