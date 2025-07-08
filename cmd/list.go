package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list [COOKBOOK...]",
	Short: "List all cookbooks and their dependencies",
	Long: `List all cookbooks defined in the Berksfile and their resolved versions.

This command will show:
- All cookbooks from the Berksfile
- Their resolved versions from Berksfile.lock
- Dependency relationships
- Source locations

Examples:
  berks list                    # List all cookbooks
  berks list --format table    # Show as table (default)
  berks list --format json     # Show as JSON
  berks list nginx apt          # List specific cookbooks`,
	RunE: runList,
}

var listFormat string

func init() {
	rootCmd.AddCommand(listCmd)

	// Add flags
	listCmd.Flags().StringVarP(&listFormat, "format", "f", "table", "Output format (table, json)")
}

type CookbookListItem struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Source       string            `json:"source"`
	Dependencies map[string]string `json:"dependencies,omitempty"`
}

func runList(cmd *cobra.Command, args []string) error {
	// Parse Berksfile
	bf, err := LoadBerksfile()
	if err != nil {
		return err
	}

	// Try to read lock file
	lockFile, _, err := LoadLockFile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: No lock file found. Run 'berks install' to generate resolved versions.\n\n")
	}

	// Build cookbook list
	var cookbooks []CookbookListItem
	berksfileCookbooks := make(map[string]bool)

	// Add cookbooks from Berksfile
	for _, cookbook := range bf.Cookbooks {
		berksfileCookbooks[cookbook.Name] = true

		item := CookbookListItem{
			Name: cookbook.Name,
		}

		// Add version and source from lock file if available
		if lockFile != nil {
			for _, source := range lockFile.Sources {
				if lockedCookbook, exists := source.Cookbooks[cookbook.Name]; exists {
					item.Version = lockedCookbook.Version
					item.Source = source.URL
					item.Dependencies = lockedCookbook.Dependencies
					break
				}
			}
		}

		// If no version from lock file, show constraint from Berksfile
		if item.Version == "" && cookbook.Constraint != nil {
			item.Version = cookbook.Constraint.String() + " (not resolved)"
		}

		// Set source from Berksfile if not from lock file
		if item.Source == "" && cookbook.Source.Type != "" {
			item.Source = cookbook.Source.URL
		}

		cookbooks = append(cookbooks, item)
	}

	// Add any additional cookbooks from lock file (transitive dependencies)
	if lockFile != nil {
		for _, source := range lockFile.Sources {
			for cookbookName, lockedCookbook := range source.Cookbooks {
				if !berksfileCookbooks[cookbookName] {
					item := CookbookListItem{
						Name:         cookbookName,
						Version:      lockedCookbook.Version,
						Source:       source.URL,
						Dependencies: lockedCookbook.Dependencies,
					}
					cookbooks = append(cookbooks, item)
				}
			}
		}
	}

	// Filter cookbooks if specific ones were requested
	if len(args) > 0 {
		filteredCookbooks := []CookbookListItem{}
		requestedSet := make(map[string]bool)
		for _, name := range args {
			requestedSet[name] = true
		}

		for _, cookbook := range cookbooks {
			if requestedSet[cookbook.Name] {
				filteredCookbooks = append(filteredCookbooks, cookbook)
			}
		}
		cookbooks = filteredCookbooks
	}

	// Sort cookbooks by name
	sort.Slice(cookbooks, func(i, j int) bool {
		return cookbooks[i].Name < cookbooks[j].Name
	})

	// Output in requested format
	switch strings.ToLower(listFormat) {
	case "json":
		return outputJSON(cookbooks)
	case "table":
		return outputTable(cookbooks)
	default:
		return fmt.Errorf("unsupported format: %s (supported: table, json)", listFormat)
	}
}

func outputTable(cookbooks []CookbookListItem) error {
	if len(cookbooks) == 0 {
		fmt.Println("No cookbooks found.")
		return nil
	}

	// Print status information
	table := tablewriter.NewTable(os.Stdout)
	table.Configure(func(config *tablewriter.Config) {
		config.Row.Alignment.Global = tw.AlignLeft
	})
	table.Header("COOKBOOK", "VERSION", "SOURCE")

	data := [][]any{}
	for _, cookbook := range cookbooks {
		version := cookbook.Version
		if version == "" {
			version = "(unknown)"
		}
		source := cookbook.Source
		if source == "" {
			source = "(local)"
		}
		data = append(data, []any{cookbook.Name, version, source})
	}

	table.Bulk(data)
	return table.Render()
}
