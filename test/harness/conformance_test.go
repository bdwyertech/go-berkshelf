//go:build conformance

package harness_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bdwyertech/go-berkshelf/test/harness"
)

var (
	runner      *harness.ToolRunner
	fixturesDir string
)

var _ = BeforeSuite(func() {
	// Verify Ruby berks is functional
	bundleGemfile := filepath.Join(getHarnessDir(), "Gemfile")
	cmd := exec.Command("bundle", "exec", "berks", "--version")
	cmd.Env = append(os.Environ(), "BUNDLE_GEMFILE="+bundleGemfile)
	out, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred(),
		"Ruby berks is not functional. Run 'task setup' in test/harness/ first.\nOutput: %s", string(out))
	GinkgoWriter.Printf("Ruby berks version: %s", string(out))

	// Locate Go binary
	goBerksPath := os.Getenv("GO_BERKS_PATH")
	if goBerksPath == "" {
		goBerksPath = filepath.Join(getHarnessDir(), "go-berkshelf")
	}
	if _, err := os.Stat(goBerksPath); os.IsNotExist(err) {
		Fail(fmt.Sprintf("Go binary not found at %s. Run 'task build' in test/harness/ first.", goBerksPath))
	}
	GinkgoWriter.Printf("Go berks path: %s\n", goBerksPath)

	// Build the ToolRunner
	bundlerEnv := map[string]string{
		"BUNDLE_GEMFILE": bundleGemfile,
	}
	timeout := 5 * time.Minute
	if t := os.Getenv("CONFORMANCE_TIMEOUT"); t != "" {
		if d, err := time.ParseDuration(t); err == nil {
			timeout = d
		}
	}
	runner = harness.NewToolRunner(goBerksPath, timeout, bundlerEnv)

	// Discover fixtures directory
	fixturesDir = filepath.Join(getHarnessDir(), "fixtures")
})

var _ = Describe("Conformance", func() {
	Describe("Fixtures", func() {
		var fixtures []harness.FixtureInfo

		BeforeEach(func() {
			var err error
			fixtures, err = harness.DiscoverFixtures(fixturesDir)
			Expect(err).NotTo(HaveOccurred(), "Failed to discover fixtures")
		})

		It("should discover at least one fixture", func() {
			if len(fixtures) == 0 {
				Skip("No fixtures found in " + fixturesDir)
			}
		})

		It("should produce conformant output for all fixtures", func() {
			if len(fixtures) == 0 {
				Skip("No fixtures found")
			}

			verbose := os.Getenv("CONFORMANCE_VERBOSE") != ""
			normOpts := harness.DefaultNormalizeOptions()

			for _, fixture := range fixtures {
				By(fmt.Sprintf("Testing fixture: %s", fixture.Name))

				// Load fixture config
				cfg, err := harness.LoadFixtureConfig(fixture.Path)
				if err != nil {
					GinkgoWriter.Printf("WARNING: skipping fixture %q: %v\n", fixture.Name, err)
					continue
				}

				// Skip if configured
				if cfg.Skip {
					GinkgoWriter.Printf("SKIP: fixture %q has skip=true\n", fixture.Name)
					continue
				}

				// Run each command in the fixture
				for _, cmdSpec := range cfg.Commands {
					By(fmt.Sprintf("Running command: berks %s %v", cmdSpec.Subcommand, cmdSpec.Args))

					// Copy fixture to temp dirs for each tool
					rubyDir, err := harness.CopyFixtureToTempDir(fixture.Path)
					Expect(err).NotTo(HaveOccurred(), "Failed to copy fixture for Ruby")
					defer os.RemoveAll(rubyDir)

					goDir, err := harness.CopyFixtureToTempDir(fixture.Path)
					Expect(err).NotTo(HaveOccurred(), "Failed to copy fixture for Go")
					defer os.RemoveAll(goDir)

					ctx := context.Background()

					// Run Ruby berks
					rubyResult, err := runner.RunRuby(ctx, rubyDir, cmdSpec)
					Expect(err).NotTo(HaveOccurred(), "Ruby berks execution failed for fixture %q", fixture.Name)
					if rubyResult.TimedOut {
						Fail(fmt.Sprintf("Ruby berks timed out for fixture %q command %q", fixture.Name, cmdSpec.Subcommand))
					}

					// Run Go berks
					goResult, err := runner.RunGo(ctx, goDir, cmdSpec)
					Expect(err).NotTo(HaveOccurred(), "Go berks execution failed for fixture %q", fixture.Name)
					if goResult.TimedOut {
						Fail(fmt.Sprintf("Go berks timed out for fixture %q command %q", fixture.Name, cmdSpec.Subcommand))
					}

					// Compare outputs based on fixture config
					var failures []string

					// Compare exit codes
					if cfg.Compare.ExitCode {
						if rubyResult.ExitCode != goResult.ExitCode {
							failures = append(failures, fmt.Sprintf(
								"Exit code mismatch: Ruby=%d, Go=%d",
								rubyResult.ExitCode, goResult.ExitCode))
						}
					}

					// Compare lockfiles
					if cfg.Compare.Lockfile {
						if rubyResult.Lockfile != "" || goResult.Lockfile != "" {
							rubyLock, err := harness.ParseRubyLockfile(rubyResult.Lockfile)
							Expect(err).NotTo(HaveOccurred(), "Failed to parse Ruby lockfile for fixture %q", fixture.Name)

							goLock, err := harness.ParseRubyLockfile(goResult.Lockfile)
							Expect(err).NotTo(HaveOccurred(), "Failed to parse Go lockfile for fixture %q", fixture.Name)

							diff := harness.CompareLockfiles(rubyLock, goLock)
							if diff != "" {
								failures = append(failures, fmt.Sprintf("Lockfile diff:\n%s", diff))
							}
						}
					}

					// Compare stdout
					if cfg.Compare.Stdout {
						rubyStdout := harness.NormalizeCLIOutput(rubyResult.Stdout, normOpts)
						goStdout := harness.NormalizeCLIOutput(goResult.Stdout, normOpts)
						diff := harness.CompareText("stdout", rubyStdout, goStdout)
						if diff != "" {
							failures = append(failures, fmt.Sprintf("Stdout diff:\n%s", diff))
						}
					}

					// Compare stderr
					if cfg.Compare.Stderr {
						rubyStderr := harness.NormalizeCLIOutput(rubyResult.Stderr, normOpts)
						goStderr := harness.NormalizeCLIOutput(goResult.Stderr, normOpts)
						diff := harness.CompareText("stderr", rubyStderr, goStderr)
						if diff != "" {
							failures = append(failures, fmt.Sprintf("Stderr diff:\n%s", diff))
						}
					}

					// Verbose output
					if verbose {
						GinkgoWriter.Printf("\n=== Fixture: %s | Command: %s ===\n", fixture.Name, cmdSpec.Subcommand)
						GinkgoWriter.Printf("Ruby exit code: %d\n", rubyResult.ExitCode)
						GinkgoWriter.Printf("Go exit code:   %d\n", goResult.ExitCode)
						GinkgoWriter.Printf("Ruby stdout:\n%s\n", rubyResult.Stdout)
						GinkgoWriter.Printf("Go stdout:\n%s\n", goResult.Stdout)
						if rubyResult.Stderr != "" || goResult.Stderr != "" {
							GinkgoWriter.Printf("Ruby stderr:\n%s\n", rubyResult.Stderr)
							GinkgoWriter.Printf("Go stderr:\n%s\n", goResult.Stderr)
						}
					}

					// Assert no failures
					if len(failures) > 0 {
						failMsg := fmt.Sprintf("Conformance failure for fixture %q, command %q:\n",
							fixture.Name, cmdSpec.Subcommand)
						for _, f := range failures {
							failMsg += "\n" + f
						}
						Fail(failMsg)
					}
				}
			}
		})
	})
})

// getHarnessDir returns the absolute path to the test/harness directory.
func getHarnessDir() string {
	wd, err := os.Getwd()
	if err != nil {
		Fail("Failed to get working directory: " + err.Error())
	}
	return wd
}
