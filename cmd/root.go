package cmd

import (
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Global flags
	berksfilePath string
	configFile    string
	debug         bool
	noColor       bool
)

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&berksfilePath, "berksfile", "b", "", "Path to Berksfile (default: ./Berksfile)")
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Config file (default: $HOME/.berkshelf/config.json)")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable debug output")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "berks",
	Short: "A dependency manager for Chef cookbooks",
	Long: `go-berkshelf is a Go implementation of Berkshelf, providing
dependency resolution and management for Chef cookbooks.

It resolves cookbook dependencies from various sources including:
- Chef Supermarket
- Git repositories  
- Local paths
- Chef Server`,
	SilenceUsage: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlags(cmd.Flags())
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if viper.GetBool("debug") || viper.GetBool("trace") {
		log.SetLevel(log.DebugLevel)
		if viper.GetBool("trace") {
			log.SetReportCaller(true)
		}
	}

	if configFile != "" {
		// TODO: Load configuration from file
		// For now, we'll just acknowledge it
		log.Debugf("Using config file: %s\n", configFile)
	}

	// Set default Berksfile path if not provided
	if berksfilePath == "" {
		berksfilePath = "Berksfile"
	}

	// TODO: Initialize logging based on debug flag
	// TODO: Initialize color output based on noColor flag
}
