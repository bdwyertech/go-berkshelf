package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/bdwyer/go-berkshelf/pkg/berksfile"
	"github.com/bdwyer/go-berkshelf/pkg/info"
	"github.com/bdwyer/go-berkshelf/pkg/source"

	"github.com/spf13/cobra"
)

var infoFormat string

func init() {
	rootCmd.AddCommand(infoCmd)

	// Add flags
	infoCmd.Flags().StringVarP(&infoFormat, "format", "f", "text", "Output format (text, json)")
}

var infoCmd = &cobra.Command{
	Use:   "info COOKBOOK [VERSION]",
	Short: "Show detailed information about a cookbook",
	Long: `Show detailed information about a cookbook including metadata,
dependencies, and available versions.

Examples:
  berks info nginx           # Show info for nginx cookbook
  berks info nginx 2.7.6     # Show info for specific version
  berks info nginx --format json  # Output as JSON`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cookbookName := args[0]
		var requestedVersion string
		if len(args) > 1 {
			requestedVersion = args[1]
		}

		// Try to parse Berksfile to get sources
		var sourceManager *source.Manager

		if _, err := os.Stat("Berksfile"); err == nil {
			berksfileContent, err := os.ReadFile("Berksfile")
			if err == nil {
				bf, err := berksfile.ParseString(string(berksfileContent))
				if err == nil {
					factory := source.NewFactory()
					sourceManager, err = factory.CreateFromBerksfile(bf)
					if err != nil {
						log.Error(err)
					}
				}
			}
		}

		// If no Berksfile or failed to parse, create default source manager
		if sourceManager == nil {
			factory := source.NewFactory()
			sourceManager = source.NewManager()
			supermarketSource, err := factory.CreateFromURL(source.PUBLIC_SUPERMARKET)
			if err != nil {
				return fmt.Errorf("failed to create supermarket source: %w", err)
			}
			sourceManager.AddSource(supermarketSource)
		}

		// Create info provider
		provider := info.New(sourceManager)

		// Get cookbook information
		cookbookInfo, err := provider.GetInfo(cmd.Context(), cookbookName, requestedVersion)
		if err != nil {
			return fmt.Errorf("failed to get cookbook info: %w", err)
		}

		// Output in requested format
		switch strings.ToLower(infoFormat) {
		case "json":
			return outputInfoJSON(cookbookInfo)
		case "text":
			return outputInfoText(cookbookInfo)
		default:
			return fmt.Errorf("unsupported format: %s (supported: text, json)", infoFormat)
		}
	},
}

func outputInfoJSON(info *info.CookbookInfo) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(info)
}

func outputInfoText(info *info.CookbookInfo) error {
	fmt.Printf("Cookbook: %s\n", info.Name)

	if info.Version != "" {
		fmt.Printf("Version: %s\n", info.Version)
	}

	if info.Description != "" {
		fmt.Printf("Description: %s\n", info.Description)
	}

	if info.Maintainer != "" {
		fmt.Printf("Maintainer: %s\n", info.Maintainer)
	}

	if info.License != "" {
		fmt.Printf("License: %s\n", info.License)
	}

	fmt.Printf("Source: %s\n", info.Source)

	if len(info.Dependencies) > 0 {
		fmt.Printf("\nDependencies:\n")
		for depName, constraint := range info.Dependencies {
			fmt.Printf("  %s (%s)\n", depName, constraint)
		}
	}

	if len(info.Versions) > 0 {
		fmt.Printf("\nAvailable Versions:\n")
		for i, version := range info.Versions {
			if i < 10 { // Show first 10 versions
				marker := ""
				if version == info.Version {
					marker = " (current)"
				}
				fmt.Printf("  %s%s\n", version, marker)
			} else if i == 10 {
				fmt.Printf("  ... and %d more versions\n", len(info.Versions)-10)
				break
			}
		}
	}

	return nil
}
