package cmd

import (
	"fmt"

	"github.com/bdwyertech/go-berkshelf/internal/version"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information",
	Long:  `Display the version of go-berkshelf along with build information.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.GetBuildInfo().String())
	},
}
