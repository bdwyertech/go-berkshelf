package cmd

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/bdwyertech/go-berkshelf/pkg/berksfile"
	"github.com/bdwyertech/go-berkshelf/pkg/berkshelf"
	"github.com/bdwyertech/go-berkshelf/pkg/lockfile"
	"github.com/bdwyertech/go-berkshelf/pkg/resolver"
	"github.com/bdwyertech/go-berkshelf/pkg/source"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(installCmd)

	// Add flags
	installCmd.Flags().StringSliceP("only", "o", nil, "Only install cookbooks in specified groups")
	installCmd.Flags().StringSliceP("except", "e", nil, "Install all cookbooks except those in specified groups")
	installCmd.Flags().BoolP("force", "f", false, "Force installation even if Berksfile.lock is up to date")
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install cookbooks from Berksfile",
	Long: `Install all cookbook dependencies specified in the Berksfile.

This command will:
- Parse the Berksfile to find cookbook requirements
- Resolve all dependencies using configured sources
- Download cookbooks to the cache
- Generate or update Berksfile.lock

Examples:
  berks install                 # Install all dependencies
  berks install --only group1   # Install only group1 dependencies
  berks install --except test   # Install all except test group`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Info("Installing cookbooks from Berksfile...")

		// 1. Parse Berksfile
		log.Info("Parsing Berksfile...")
		berks, err := LoadBerksfile()
		if err != nil {
			return err
		}

		workDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		// 2. Check lock file status
		lockManager := lockfile.NewManager(workDir)
		log.Info("Checking lock file status...")

		shouldProceed, err := CheckLockFileStatus(lockManager, viper.GetBool("force"))
		if err != nil {
			return err
		}
		if !shouldProceed {
			return nil
		}

		// Filter cookbooks by groups
		only, except := viper.GetStringSlice("only"), viper.GetStringSlice("except")

		cookbooks := berksfile.FilterCookbooksByGroup(berks.Cookbooks, only, except)
		if len(only) > 0 || len(except) > 0 {
			log.Infof("Filtered to %d cookbooks based on group selection", len(cookbooks))
		}

		// 3. Create requirements from cookbooks
		log.Info("Creating requirements...")
		requirements := CreateRequirementsFromCookbooks(cookbooks)
		if berks.HasMetadata {
			pathSrc, err := source.NewPathSource(".")
			if err != nil {
				return fmt.Errorf("failed to create path source for metadata: %w", err)
			}
			metadata, err := pathSrc.ReadMetadata(".")
			if err != nil {
				return fmt.Errorf("failed to read metadata: %w", err)
			}

			log.Debugf("Found cookbook %s (%s) via metadata", metadata.Name, metadata.Version)

			req := resolver.NewRequirementWithSource(metadata.Name, nil, &berkshelf.SourceLocation{
				Type: "path",
				Path: ".",
			})
			requirements = append(requirements, req)
		}

		// 4. Set up sources
		log.Info("Setting up sources...")
		sourceManager, err := SetupSourcesFromBerksfile(berks)
		if err != nil {
			return err
		}

		// 5. Resolve dependencies
		log.Info("Resolving dependencies...")
		resolution, err := ResolveDependencies(cmd.Context(), requirements, sourceManager.GetSources())
		if err != nil {
			return err
		}

		log.Infof("Resolved %d cookbooks", resolution.CookbookCount())

		// 6. Generate/update lock files
		log.Info("Updating Berksfile.lock...")

		// Extract direct dependencies from Berksfile for DEPENDENCIES section
		berksfilePath := "Berksfile"
		var groups []string
		if len(only) > 0 {
			groups = only
		}

		dependencies, err := lockfile.ExtractDirectDependencies(berksfilePath, groups)
		if err != nil {
			log.Warnf("Failed to extract direct dependencies for Ruby lock file: %v", err)
			// Continue with empty dependencies list
			dependencies = []string{}
		}

		// Update both JSON and Ruby lock files
		if err := lockManager.UpdateBoth(resolution, dependencies); err != nil {
			return fmt.Errorf("failed to update lock files: %w", err)
		}

		log.Info("")
		log.Info("Installation complete!")
		log.Infof("Resolved %d cookbooks", resolution.CookbookCount())
		log.Infof("Updated %s", lockManager.GetPath())
		log.Infof("Generated %s", lockManager.GetRubyPath())

		return nil
	},
}
