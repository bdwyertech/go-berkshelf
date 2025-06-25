package lockfile

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bdwyer/go-berkshelf/pkg/berkshelf"
	"github.com/bdwyer/go-berkshelf/pkg/resolver"
	"github.com/bdwyer/go-berkshelf/pkg/source"
)

const (
	// DefaultLockFileName is the default name for lock files
	DefaultLockFileName = "Berksfile.go.lock"
)

// Manager handles lock file operations
type Manager struct {
	lockFilePath string
}

// NewManager creates a new lock file manager
func NewManager(workDir string) *Manager {
	return &Manager{
		lockFilePath: filepath.Join(workDir, DefaultLockFileName),
	}
}

// NewManagerWithPath creates a new lock file manager with custom path
func NewManagerWithPath(lockFilePath string) *Manager {
	return &Manager{
		lockFilePath: lockFilePath,
	}
}

// Exists checks if the lock file exists
func (m *Manager) Exists() bool {
	_, err := os.Stat(m.lockFilePath)
	return err == nil
}

// Load reads and parses the lock file
func (m *Manager) Load() (*LockFile, error) {
	data, err := os.ReadFile(m.lockFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return NewLockFile(), nil
		}
		return nil, fmt.Errorf("failed to read lock file %s: %w", m.lockFilePath, err)
	}

	lockFile, err := FromJSON(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse lock file %s: %w", m.lockFilePath, err)
	}

	return lockFile, nil
}

// Save writes the lock file to disk
func (m *Manager) Save(lockFile *LockFile) error {
	// Update generation time
	lockFile.UpdateGeneratedAt()

	data, err := lockFile.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize lock file: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(m.lockFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create lock file directory: %w", err)
	}

	// Write with proper permissions
	if err := os.WriteFile(m.lockFilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write lock file %s: %w", m.lockFilePath, err)
	}

	return nil
}

// Generate creates a lock file from a resolution result
func (m *Manager) Generate(resolution *resolver.Resolution) (*LockFile, error) {
	lockFile := NewLockFile()

	// Process each resolved cookbook
	for _, resolvedCookbook := range resolution.Cookbooks {
		// Determine source information
		sourceInfo := &SourceInfo{
			Type: getSourceType(resolvedCookbook.Source),
			URL:  getSourceURL(resolvedCookbook.Source),
		}

		// Add to lock file
		lockFile.AddCookbook(sourceInfo.URL, resolvedCookbook.Cookbook, sourceInfo)
	}

	return lockFile, nil
}

// Update updates an existing lock file with new resolution data
func (m *Manager) Update(resolution *resolver.Resolution) error {
	// Load existing lock file or create new one
	existingLock, err := m.Load()
	if err != nil {
		return fmt.Errorf("failed to load existing lock file: %w", err)
	}

	// Generate new lock file from resolution
	newLock, err := m.Generate(resolution)
	if err != nil {
		return fmt.Errorf("failed to generate new lock file: %w", err)
	}

	// Merge lock files (for now, replace completely)
	// TODO: Implement intelligent merging for partial updates
	*existingLock = *newLock

	// Save updated lock file
	return m.Save(existingLock)
}

// IsOutdated checks if the lock file needs updating
func (m *Manager) IsOutdated() (bool, error) {
	if !m.Exists() {
		return true, nil
	}

	lockFile, err := m.Load()
	if err != nil {
		return true, err
	}

	// Check if Berksfile is newer than lock file
	berksfilePath := filepath.Join(filepath.Dir(m.lockFilePath), "Berksfile")
	berksfileInfo, err := os.Stat(berksfilePath)
	if err != nil {
		if os.IsNotExist(err) {
			// No Berksfile, so lock file is not outdated
			return false, nil
		}
		return true, fmt.Errorf("failed to check Berksfile: %w", err)
	}

	// Compare modification times
	return berksfileInfo.ModTime().After(lockFile.GeneratedAt), nil
}

// Validate checks if the lock file is valid and consistent
func (m *Manager) Validate() error {
	if !m.Exists() {
		return fmt.Errorf("lock file does not exist: %s", m.lockFilePath)
	}

	lockFile, err := m.Load()
	if err != nil {
		return fmt.Errorf("failed to load lock file: %w", err)
	}

	// Check revision compatibility
	if lockFile.Revision != 7 {
		return fmt.Errorf("unsupported lock file revision: %d (expected: 7)", lockFile.Revision)
	}

	// Validate cookbook entries
	for sourceURL, source := range lockFile.Sources {
		if sourceURL == "" {
			return fmt.Errorf("empty source URL found in lock file")
		}

		for cookbookName, cookbook := range source.Cookbooks {
			if cookbookName == "" {
				return fmt.Errorf("empty cookbook name found in source %s", sourceURL)
			}

			if cookbook.Version == "" {
				return fmt.Errorf("empty version for cookbook %s in source %s", cookbookName, sourceURL)
			}

			// Validate version format
			if _, err := berkshelf.NewVersion(cookbook.Version); err != nil {
				return fmt.Errorf("invalid version %s for cookbook %s: %w", cookbook.Version, cookbookName, err)
			}
		}
	}

	return nil
}

// GetPath returns the lock file path
func (m *Manager) GetPath() string {
	return m.lockFilePath
}

// Remove deletes the lock file
func (m *Manager) Remove() error {
	if !m.Exists() {
		return nil // Already doesn't exist
	}

	err := os.Remove(m.lockFilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove lock file %s: %w", m.lockFilePath, err)
	}

	return nil
}

// Backup creates a backup of the lock file
func (m *Manager) Backup() error {
	if !m.Exists() {
		return fmt.Errorf("no lock file to backup: %s", m.lockFilePath)
	}

	backupPath := m.lockFilePath + ".backup"

	srcFile, err := os.Open(m.lockFilePath)
	if err != nil {
		return fmt.Errorf("failed to open lock file for backup: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer dstFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get lock file info: %w", err)
	}

	// Copy file contents
	data := make([]byte, srcInfo.Size())
	if _, err := srcFile.Read(data); err != nil {
		return fmt.Errorf("failed to read lock file: %w", err)
	}

	if _, err := dstFile.Write(data); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	return nil
}

// Helper functions

func getSourceType(source any) string {
	if source == nil {
		return "unknown"
	}

	// Type assert to berkshelf.SourceLocation
	if sourceLocation, ok := source.(*berkshelf.SourceLocation); ok && sourceLocation != nil {
		if sourceLocation.Type != "" {
			return sourceLocation.Type
		}
	}

	// Default fallback
	return "supermarket"
}

func getSourceURL(src any) string {
	if src == nil {
		return source.PUBLIC_SUPERMARKET
	}

	// Type assert to berkshelf.SourceLocation
	if sourceLocation, ok := src.(*berkshelf.SourceLocation); ok && sourceLocation != nil {
		if sourceLocation.URL != "" {
			return sourceLocation.URL
		}
	}

	// Default fallback
	return source.PUBLIC_SUPERMARKET
}
