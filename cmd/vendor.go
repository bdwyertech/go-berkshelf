package cmd

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/bdwyer/go-berkshelf/pkg/vendor"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(vendorCmd)

	// Add flags
	vendorCmd.Flags().Bool("delete", false, "Delete the target directory before vendoring")
	vendorCmd.Flags().Bool("dry-run", false, "Show what would be done without actually doing it")
}

var vendorCmd = &cobra.Command{
	Use:   "vendor [PATH]",
	Short: "Download cookbooks to a directory",
	Long: `Download all cookbooks and their dependencies to a specified directory.

This command downloads cookbooks from the lock file to a target directory,
maintaining the proper directory structure for Chef. This is useful for
packaging cookbooks for deployment or creating a self-contained cookbook bundle.

Examples:
  berks vendor ./vendor           # Download to ./vendor  
  berks vendor --delete           # Delete target directory first`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targetPath := args[0]

		// Parse Berksfile
		bf, err := LoadBerksfile()
		if err != nil {
			return err
		}

		// Load lock file
		lockFile, _, err := LoadLockFile()
		if err != nil {
			return fmt.Errorf("no lock file found. Run 'berks install' first: %w", err)
		}

		// Create source manager
		sourceManager, err := CreateSourceManager(bf)
		if err != nil {
			return err
		}

		// Create vendor options
		options := vendor.Options{
			TargetPath: targetPath,
			Delete:     viper.GetBool("delete"),
			DryRun:     viper.GetBool("dry-run"),
		}

		// Create vendorer
		vendorer := vendor.New(lockFile, sourceManager, options)

		if options.DryRun {
			log.Infof("Dry run: Would vendor cookbooks to: %s\n", targetPath)
			if options.Delete {
				log.Infof("Would delete existing directory first\n")
			}
		} else {
			log.Infof("Vendoring cookbooks to: %s\n", targetPath)
		}

		result, err := vendorer.Vendor(cmd.Context())
		if err != nil {
			return fmt.Errorf("vendor failed: %w", err)
		}

		// Report results
		if options.DryRun {
			log.Infof("\nDry run completed. %d cookbook(s) would be downloaded.\n", result.TotalCookbooks)
		} else {
			log.Infof("\nVendoring completed. %d cookbook(s) successfully downloaded to %s\n",
				result.SuccessfulDownloads, result.TargetPath)

			if len(result.FailedDownloads) > 0 {
				log.Warnf("\nWarning: Failed to download %d cookbook(s):\n", len(result.FailedDownloads))
				for _, name := range result.FailedDownloads {
					log.Warnf("  - %s\n", name)
				}
			}
		}

		return nil
	},
}
