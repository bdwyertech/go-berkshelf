package harness

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"github.com/goccy/go-yaml"
)

// SupportedSubcommands lists the berks subcommands the harness can test.
var SupportedSubcommands = []string{
	"install",
	"update",
	"list",
	"graph",
	"info",
	"vendor",
	"outdated",
}

// FixtureConfig represents the declarative configuration for a test fixture.
type FixtureConfig struct {
	// Commands lists the subcommands to run with their arguments.
	Commands []CommandSpec `yaml:"commands"`
	// Skip disables this fixture without removing it.
	Skip bool `yaml:"skip,omitempty"`
	// Compare controls which outputs are compared.
	Compare CompareSpec `yaml:"compare,omitempty"`
}

// CommandSpec defines a single berks subcommand invocation.
type CommandSpec struct {
	// Subcommand is the berks subcommand (install, update, list, etc.)
	Subcommand string `yaml:"subcommand"`
	// Args are additional CLI arguments.
	Args []string `yaml:"args,omitempty"`
}

// CompareSpec controls which outputs are compared between tools.
type CompareSpec struct {
	Lockfile bool `yaml:"lockfile"`
	Stdout   bool `yaml:"stdout"`
	Stderr   bool `yaml:"stderr"`
	ExitCode bool `yaml:"exit_code"`
}

// FixtureInfo holds the name and path of a discovered fixture directory.
type FixtureInfo struct {
	Name string
	Path string
}

// DefaultFixtureConfig returns the default configuration used when no fixture.yaml exists.
// It runs the install command and compares lockfile + exit code.
func DefaultFixtureConfig() FixtureConfig {
	return FixtureConfig{
		Commands: []CommandSpec{{Subcommand: "install"}},
		Compare:  CompareSpec{Lockfile: true, ExitCode: true},
	}
}

// IsSupportedSubcommand checks whether a subcommand is in the supported list.
func IsSupportedSubcommand(sub string) bool {
	for _, s := range SupportedSubcommands {
		if s == sub {
			return true
		}
	}
	return false
}

// ValidateFixtureConfig checks that all commands use supported subcommands.
// It returns the list of unsupported subcommands found.
func ValidateFixtureConfig(cfg FixtureConfig) []string {
	var unsupported []string
	for _, cmd := range cfg.Commands {
		if !IsSupportedSubcommand(cmd.Subcommand) {
			unsupported = append(unsupported, cmd.Subcommand)
		}
	}
	return unsupported
}

// DiscoverFixtures scans fixturesDir for subdirectories containing a Berksfile.
// Directories without a Berksfile are skipped with a log warning.
func DiscoverFixtures(fixturesDir string) ([]FixtureInfo, error) {
	entries, err := os.ReadDir(fixturesDir)
	if err != nil {
		return nil, fmt.Errorf("reading fixtures directory: %w", err)
	}

	var fixtures []FixtureInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirPath := filepath.Join(fixturesDir, entry.Name())
		berksfilePath := filepath.Join(dirPath, "Berksfile")

		if _, err := os.Stat(berksfilePath); os.IsNotExist(err) {
			log.Warnf("skipping fixture %q: no Berksfile found", entry.Name())
			continue
		}

		fixtures = append(fixtures, FixtureInfo{
			Name: entry.Name(),
			Path: dirPath,
		})
	}

	return fixtures, nil
}

// LoadFixtureConfig reads fixture.yaml from fixtureDir, or returns defaults if absent.
// If the config contains unsupported subcommands, a warning is logged and an error returned.
func LoadFixtureConfig(fixtureDir string) (FixtureConfig, error) {
	configPath := filepath.Join(fixtureDir, "fixture.yaml")

	data, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		return DefaultFixtureConfig(), nil
	}
	if err != nil {
		return FixtureConfig{}, fmt.Errorf("reading fixture.yaml: %w", err)
	}

	var cfg FixtureConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return FixtureConfig{}, fmt.Errorf("parsing fixture.yaml: %w", err)
	}

	if unsupported := ValidateFixtureConfig(cfg); len(unsupported) > 0 {
		log.Warnf("fixture %q has unsupported subcommands: %v", fixtureDir, unsupported)
		return cfg, fmt.Errorf("unsupported subcommands: %v", unsupported)
	}

	return cfg, nil
}

// SerializeFixtureConfig marshals a FixtureConfig to YAML bytes.
func SerializeFixtureConfig(cfg FixtureConfig) ([]byte, error) {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("serializing fixture config: %w", err)
	}
	return data, nil
}
