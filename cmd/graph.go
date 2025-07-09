// Package cmd implements the CLI commands for go-berkshelf.
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/bdwyertech/go-berkshelf/pkg/lockfile"
	"github.com/spf13/cobra"
)

var graphFormat string

func init() {
	rootCmd.AddCommand(graphCmd)

	// Add flags
	graphCmd.Flags().StringVarP(&graphFormat, "format", "f", "text", "Output format (dot, text)")
}

var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Display the dependency graph of resolved cookbooks",
	Long: `Display the dependency graph of resolved cookbooks, including their dependencies,
subdependencies, and versions. The graph can be output in DOT/Graphviz format or as a text tree.

Examples:
  berks graph                   # Output graph as a text tree (default)
  berks graph --format dot      # Output graph in DOT format`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load lock file
		workDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		manager := lockfile.NewManager(workDir)
		lockFile, err := manager.Load()
		if err != nil {
			return fmt.Errorf("failed to load lock file: %w", err)
		}

		// Generate graph
		switch strings.ToLower(graphFormat) {
		case "dot":
			fmt.Println("digraph dependencies {")
			for _, source := range lockFile.Sources {
				for cookbookName, cookbook := range source.Cookbooks {
					for depName := range cookbook.Dependencies {
						fmt.Printf("  \"%s (%s)\" -> \"%s\";\n", cookbookName, cookbook.Version, depName)
					}
				}
			}
			fmt.Println("}")
			return nil
		case "text":
			for _, source := range lockFile.Sources {
				for cookbookName, cookbook := range source.Cookbooks {
					fmt.Printf("%s (%s)\n", cookbookName, cookbook.Version)
					for depName := range cookbook.Dependencies {
						fmt.Printf("  └── %s\n", depName)
					}
				}
			}
			return nil
		default:
			return fmt.Errorf("unsupported format: %s (supported: dot, text)", graphFormat)
		}
	},
}
