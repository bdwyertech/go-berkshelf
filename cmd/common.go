package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bdwyertech/go-berkshelf/pkg/berksfile"
	"github.com/bdwyertech/go-berkshelf/pkg/lockfile"
	"github.com/bdwyertech/go-berkshelf/pkg/source"
)

// CommonFlags holds flags that are used across multiple commands
type CommonFlags struct {
	Only   []string
	Except []string
}

// LoadBerksfile loads and parses the Berksfile from the current directory
func LoadBerksfile() (*berksfile.Berksfile, error) {
	berksfilePath := filepath.Join(".", "Berksfile")

	// Check if Berksfile exists
	if _, err := os.Stat(berksfilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("no Berksfile found in current directory. Run 'berks init' to create one")
	}

	// Parse Berksfile
	bf, err := berksfile.ParseFile(berksfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Berksfile: %w", err)
	}

	return bf, nil
}

// CreateSourceManager creates a source manager from a parsed Berksfile
func CreateSourceManager(bf *berksfile.Berksfile) (*source.Manager, error) {
	factory := source.NewFactory()
	manager, err := factory.CreateFromBerksfile(bf)
	if err != nil {
		return nil, fmt.Errorf("failed to create source manager: %w", err)
	}
	return manager, nil
}

// LoadLockFile loads the lock file from the current directory
func LoadLockFile() (*lockfile.LockFile, *lockfile.Manager, error) {
	manager := lockfile.NewManager(".")
	lockFile, err := manager.Load()
	if err != nil {
		return nil, manager, err
	}
	return lockFile, manager, nil
}

// CheckLockFileStatus checks if the lock file exists and whether it's outdated
func CheckLockFileStatus(manager *lockfile.Manager, force bool) (shouldProceed bool, err error) {
	if force {
		return true, nil
	}

	outdated, err := manager.IsOutdated()
	if err != nil {
		// If we can't check status, proceed with warning
		fmt.Printf("Warning: failed to check lock file status: %v\n", err)
		return true, nil
	}

	if !outdated && manager.Exists() {
		fmt.Println("Berksfile.lock is up to date. Use --force to reinstall.")
		return false, nil
	}

	return true, nil
}

func outputJSON(cookbooks []CookbookListItem) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(cookbooks)
}
