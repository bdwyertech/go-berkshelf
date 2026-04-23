package harness

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// RunResult holds the captured output from a single tool invocation.
type RunResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Lockfile string // Contents of Berksfile.lock if produced
	TimedOut bool
}

// ToolRunner executes berks commands and captures output.
type ToolRunner struct {
	RubyBerksCmd []string          // e.g. ["bundle", "exec", "berks"]
	GoBerksPath  string            // path to go-berkshelf binary
	Timeout      time.Duration     // per-command timeout
	BundlerEnv   map[string]string // env vars for bundle exec
}

// DefaultGoBerksPath is the default path to the Go binary relative to the harness directory.
const DefaultGoBerksPath = "./go-berkshelf"

// NewToolRunner creates a ToolRunner with sensible defaults.
// If goBerksPath is empty, DefaultGoBerksPath is used.
func NewToolRunner(goBerksPath string, timeout time.Duration, bundlerEnv map[string]string) *ToolRunner {
	if goBerksPath == "" {
		goBerksPath = DefaultGoBerksPath
	}
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	return &ToolRunner{
		RubyBerksCmd: []string{"bundle", "exec", "berks"},
		GoBerksPath:  goBerksPath,
		Timeout:      timeout,
		BundlerEnv:   bundlerEnv,
	}
}

// BuildArgs constructs the argument list from a CommandSpec (subcommand + args).
// This is a shared function used by both RunRuby and RunGo to ensure argument identity.
func BuildArgs(cmd CommandSpec) []string {
	args := []string{cmd.Subcommand}
	args = append(args, cmd.Args...)
	return args
}

// RunRuby executes Ruby berks in the given working directory.
// It runs `bundle exec berks <subcommand> <args>` with the configured Bundler environment.
func (r *ToolRunner) RunRuby(ctx context.Context, workDir string, cmd CommandSpec) (*RunResult, error) {
	args := append(r.RubyBerksCmd[1:], BuildArgs(cmd)...)
	return r.run(ctx, workDir, r.RubyBerksCmd[0], args, r.BundlerEnv)
}

// RunGo executes Go berks in the given working directory.
// It runs the Go binary with the same subcommand and args as Ruby.
func (r *ToolRunner) RunGo(ctx context.Context, workDir string, cmd CommandSpec) (*RunResult, error) {
	return r.run(ctx, workDir, r.GoBerksPath, BuildArgs(cmd), nil)
}

// run is the internal execution method shared by RunRuby and RunGo.
func (r *ToolRunner) run(ctx context.Context, workDir string, binary string, args []string, extraEnv map[string]string) (*RunResult, error) {
	ctx, cancel := context.WithTimeout(ctx, r.Timeout)
	defer cancel()

	c := exec.CommandContext(ctx, binary, args...)
	c.Dir = workDir

	// Build environment with isolation vars
	env := os.Environ()
	// Set BERKSHELF_PATH to a per-invocation temp cache dir for isolation
	berkshelfPath, err := os.MkdirTemp("", "berkshelf-cache-*")
	if err != nil {
		return nil, fmt.Errorf("creating berkshelf cache dir: %w", err)
	}
	defer os.RemoveAll(berkshelfPath)

	// Set HOME to an isolated temp dir to prevent reading user-level Chef configs
	homeDir, err := os.MkdirTemp("", "berkshelf-home-*")
	if err != nil {
		return nil, fmt.Errorf("creating isolated home dir: %w", err)
	}
	defer os.RemoveAll(homeDir)

	env = append(env,
		"BERKSHELF_PATH="+berkshelfPath,
		"HOME="+homeDir,
	)

	// Add extra env vars (e.g., Bundler env for Ruby)
	for k, v := range extraEnv {
		env = append(env, k+"="+v)
	}

	c.Env = env

	// Capture stdout and stderr separately
	var stdoutBuf, stderrBuf bytes.Buffer
	c.Stdout = io.Writer(&stdoutBuf)
	c.Stderr = io.Writer(&stderrBuf)

	result := &RunResult{}

	runErr := c.Run()

	result.Stdout = stdoutBuf.String()
	result.Stderr = stderrBuf.String()

	if runErr != nil {
		// Check if the context deadline was exceeded (timeout)
		if ctx.Err() != nil && errors.Is(ctx.Err(), context.DeadlineExceeded) {
			result.TimedOut = true
			result.ExitCode = -1
		} else if exitErr, ok := runErr.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("executing %s: %w", binary, runErr)
		}
	}

	// Read Berksfile.lock if produced
	lockfilePath := filepath.Join(workDir, "Berksfile.lock")
	if data, err := os.ReadFile(lockfilePath); err == nil {
		result.Lockfile = string(data)
	}

	return result, nil
}

// CopyFixtureToTempDir copies all files from a fixture directory into a new
// temporary directory and returns the temp dir path. The caller is responsible
// for cleaning up the returned directory.
func CopyFixtureToTempDir(fixtureDir string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "harness-fixture-*")
	if err != nil {
		return "", fmt.Errorf("creating temp dir: %w", err)
	}

	err = filepath.Walk(fixtureDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(fixtureDir, path)
		if err != nil {
			return fmt.Errorf("computing relative path: %w", err)
		}

		destPath := filepath.Join(tmpDir, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		return copyFile(path, destPath, info.Mode())
	})

	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("copying fixture files: %w", err)
	}

	return tmpDir, nil
}

// copyFile copies a single file from src to dst with the given permissions.
func copyFile(src, dst string, mode os.FileMode) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source file %s: %w", src, err)
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("creating destination file %s: %w", dst, err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("copying %s to %s: %w", src, dst, err)
	}

	return nil
}
