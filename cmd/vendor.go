package cmd

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/bdwyertech/go-berkshelf/pkg/berksfile"
	"github.com/bdwyertech/go-berkshelf/pkg/vendor"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(vendorCmd)

	// Add flags
	vendorCmd.Flags().Bool("delete", false, "Delete the target directory before vendoring")
	vendorCmd.Flags().Bool("dry-run", false, "Show what would be done without actually doing it")
	vendorCmd.Flags().Bool("install", true, "Automatically create/update lockfile")
	vendorCmd.Flags().StringSliceP("only", "o", nil, "Only vendor cookbooks in specified groups")
	vendorCmd.Flags().StringSliceP("except", "e", nil, "Vendor all cookbooks except those in specified groups")
}

var vendorCmd = &cobra.Command{
	Use:   "vendor [PATH]",
	Short: "Download cookbooks to a directory",
	Long: `Download all cookbooks and their dependencies to a specified directory.

This command downloads cookbooks from the lock file to a target directory,
maintaining the proper directory structure for Chef. This is useful for
packaging cookbooks for deployment or creating a self-contained cookbook bundle.

If no PATH is provided, cookbooks will be vendored to ./berks-cookbooks.

Examples:
     berks vendor
     berks vendor ./vendor
 	 berks vendor --delete                    # Delete target directory first
 	 berks vendor ./vendor --only production  # Vendor only production group cookbooks
 	 berks vendor ./vendor --except test      # Vendor all except test group cookbooks`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targetPath := "berks-cookbooks"
		if len(args) == 1 {
			targetPath = args[0]
		}

		if viper.GetBool("install") {
			if err := installCmd.RunE(cmd, args); err != nil {
				return fmt.Errorf("failed to run install command: %w", err)
			}
		}

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

		// Filter cookbooks by groups if needed
		var allowedCookbooks []string
		only, except := viper.GetStringSlice("only"), viper.GetStringSlice("except")
		if len(only) > 0 || len(except) > 0 {
			// Filter cookbooks from Berksfile
			filtered := berksfile.FilterCookbooksByGroup(bf.Cookbooks, only, except)

			// Extract cookbook names
			filteredNames := make([]string, 0, len(filtered))
			for _, cb := range filtered {
				filteredNames = append(filteredNames, cb.Name)
			}

			// If using --only, include transitive dependencies
			if len(only) > 0 {
				allowedCookbooks = vendor.FindTransitiveDependencies(lockFile, filteredNames)
				log.Infof("Including %d cookbook(s) with dependencies", len(allowedCookbooks))
			} else {
				// For --except, don't include dependencies of excluded cookbooks
				allowedCookbooks = filteredNames
			}

			if len(allowedCookbooks) == 0 {
				return fmt.Errorf("no cookbooks match the specified group filters")
			}
		}

		// Create vendor options
		options := vendor.Options{
			TargetPath:    targetPath,
			Delete:        viper.GetBool("delete"),
			DryRun:        viper.GetBool("dry-run"),
			OnlyCookbooks: allowedCookbooks,
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
