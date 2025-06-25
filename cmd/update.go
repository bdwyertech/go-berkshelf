package cmd

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/bdwyer/go-berkshelf/pkg/berksfile"
	"github.com/bdwyer/go-berkshelf/pkg/berkshelf"
	"github.com/bdwyer/go-berkshelf/pkg/lockfile"
	"github.com/bdwyer/go-berkshelf/pkg/resolver"

	"github.com/spf13/cobra"
)

var (
	updateExcept []string
	updateOnly   []string
)

func init() {
	rootCmd.AddCommand(updateCmd)

	// Add flags
	updateCmd.Flags().StringSliceVar(&updateExcept, "except", []string{}, "Exclude groups from update")
	updateCmd.Flags().StringSliceVar(&updateOnly, "only", []string{}, "Include only specified groups")
}

var updateCmd = &cobra.Command{
	Use:   "update [COOKBOOK...]",
	Short: "Update cookbook dependencies",
	Long: `Update cookbook dependencies to their latest versions.

If no cookbooks are specified, all cookbooks will be updated.
If specific cookbooks are provided, only those will be updated.

This command will:
1. Parse the Berksfile
2. Update specified cookbooks (or all if none specified)
3. Resolve dependencies with updated constraints
4. Update the lock file with new versions

Examples:
  berks update              # Update all cookbooks
  berks update nginx        # Update only nginx cookbook
  berks update nginx apache # Update nginx and apache cookbooks`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Infoln("Updating cookbook dependencies...")

		// Parse Berksfile
		bf, err := LoadBerksfile()
		if err != nil {
			return err
		}

		// Get cookbooks to update
		var cookbooksToUpdate []*berksfile.CookbookDef
		if len(args) > 0 {
			// Update specific cookbooks
			requestedSet := make(map[string]bool)
			for _, name := range args {
				requestedSet[name] = true
			}

			for _, cookbook := range bf.Cookbooks {
				if requestedSet[cookbook.Name] {
					cookbooksToUpdate = append(cookbooksToUpdate, cookbook)
				}
			}

			// Check if all requested cookbooks were found
			if len(cookbooksToUpdate) != len(args) {
				missing := []string{}
				for _, name := range args {
					found := false
					for _, cookbook := range cookbooksToUpdate {
						if cookbook.Name == name {
							found = true
							break
						}
					}
					if !found {
						missing = append(missing, name)
					}
				}
				if len(missing) > 0 {
					return fmt.Errorf("cookbooks not found in Berksfile: %v", missing)
				}
			}
		} else {
			// Update all cookbooks, filtered by groups if specified
			cookbooksToUpdate = berksfile.FilterCookbooksByGroup(bf.Cookbooks, updateOnly, updateExcept)
		}

		if len(cookbooksToUpdate) == 0 {
			fmt.Println("No cookbooks to update.")
			return nil
		}

		// Display what will be updated
		log.Infof("Updating %d cookbook(s):", len(cookbooksToUpdate))
		for _, cookbook := range cookbooksToUpdate {
			fmt.Printf("  - %s", cookbook.Name)
		}
		fmt.Println("")

		// Create source manager
		manager, err := CreateSourceManager(bf)
		if err != nil {
			return err
		}

		// Create resolver
		defaultResolver := resolver.NewResolver(manager.GetSources())

		// Convert to berkshelf requirements (for all cookbooks, not just those being updated)
		requirements := make([]*resolver.Requirement, 0, len(bf.Cookbooks))
		for _, cookbook := range bf.Cookbooks {
			// For cookbooks being updated, remove version constraints to get latest
			constraint := cookbook.Constraint
			isBeingUpdated := false
			for _, updateCookbook := range cookbooksToUpdate {
				if updateCookbook.Name == cookbook.Name {
					isBeingUpdated = true
					break
				}
			}

			// If being updated, use unconstrained requirement to get latest
			if isBeingUpdated {
				constraint = nil // This will default to ">= 0.0.0" for latest
			}

			// Convert source
			var sourceLocation *berkshelf.SourceLocation
			if cookbook.Source != nil && cookbook.Source.Type != "" {
				sourceLocation = cookbook.Source
			}

			req := &resolver.Requirement{
				Name:       cookbook.Name,
				Constraint: constraint,
				Source:     sourceLocation,
			}
			requirements = append(requirements, req)
		}

		// Resolve dependencies
		log.Info("Resolving dependencies...")

		resolution, err := defaultResolver.Resolve(cmd.Context(), requirements)
		if err != nil {
			return fmt.Errorf("dependency resolution failed: %w", err)
		}

		if len(resolution.Errors) > 0 {
			log.Info("Resolution errors:")
			for _, resolverErr := range resolution.Errors {
				log.Infof("  - %v", resolverErr)
			}
			return fmt.Errorf("dependency resolution completed with errors")
		}

		log.Infof("Resolved %d cookbook(s)", len(resolution.Cookbooks))

		// Update lock file
		lockManager := lockfile.NewManager(".")
		lockFile, err := lockManager.Generate(resolution)
		if err != nil {
			return fmt.Errorf("failed to generate lock file: %w", err)
		}

		if err := lockManager.Save(lockFile); err != nil {
			return fmt.Errorf("failed to save lock file: %w", err)
		}

		log.Infof("Lock file updated: %s", lockManager.GetPath())

		// Show what was updated
		log.Info("\nUpdated cookbooks:")
		for _, cookbook := range cookbooksToUpdate {
			if resolvedCookbook, exists := resolution.Cookbooks[cookbook.Name]; exists {
				fmt.Printf("  - %s (%s)", cookbook.Name, resolvedCookbook.Cookbook.Version)
			}
		}

		return nil
	},
}
