package lockfile

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bdwyertech/go-berkshelf/pkg/berksfile"
	"github.com/bdwyertech/go-berkshelf/pkg/berkshelf"
	"github.com/bdwyertech/go-berkshelf/pkg/resolver"
	"github.com/bdwyertech/go-berkshelf/pkg/source"
)

const (
	// DefaultLockFileName is the default name for lock files
	DefaultLockFileName = "Berksfile.go.lock"
	// RubyLockFileName is the Ruby Berkshelf compatible lock file name
	RubyLockFileName = "Berksfile.lock"
)

// Manager handles lock file operations for both JSON and Ruby formats
type Manager struct {
	lockFilePath     string
	rubyLockFilePath string
}

// NewManager creates a new lock file manager
func NewManager(workDir string) *Manager {
	return &Manager{
		lockFilePath:     filepath.Join(workDir, DefaultLockFileName),
		rubyLockFilePath: filepath.Join(workDir, RubyLockFileName),
	}
}

// NewManagerWithPath creates a new lock file manager with custom path
func NewManagerWithPath(lockFilePath string) *Manager {
	return &Manager{
		lockFilePath:     lockFilePath,
		rubyLockFilePath: filepath.Join(filepath.Dir(lockFilePath), RubyLockFileName),
	}
}

// Exists checks if the lock file exists
func (m *Manager) Exists() bool {
	_, err := os.Stat(m.lockFilePath)
	return err == nil
}

// RubyExists checks if the Ruby lock file exists
func (m *Manager) RubyExists() bool {
	_, err := os.Stat(m.rubyLockFilePath)
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

// Save writes the lock file to disk in JSON format
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

// SaveRuby writes the lock file in Ruby format to disk
func (m *Manager) SaveRuby(lockFile *LockFile, dependencies []string) error {
	// Update generation time
	lockFile.UpdateGeneratedAt()

	data, err := lockFile.ToRubyFormat(dependencies)
	if err != nil {
		return fmt.Errorf("failed to serialize lock file to Ruby format: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(m.rubyLockFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create lock file directory: %w", err)
	}

	// Write with proper permissions
	if err := os.WriteFile(m.rubyLockFilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write Ruby lock file %s: %w", m.rubyLockFilePath, err)
	}

	return nil
}

// SaveBoth writes both JSON and Ruby format lock files
func (m *Manager) SaveBoth(lockFile *LockFile, dependencies []string) error {
	// Save JSON format
	if err := m.Save(lockFile); err != nil {
		return err
	}

	// Save Ruby format
	if err := m.SaveRuby(lockFile, dependencies); err != nil {
		return err
	}

	return nil
}

// Generate creates a lock file from a resolution result
func (m *Manager) Generate(resolution *resolver.Resolution) (*LockFile, error) {
	lockFile := NewLockFile()

	// Process each resolved cookbook
	for _, resolvedCookbook := range resolution.Cookbooks {
		// Handle nil source (use default)
		var sourceInfo *SourceInfo
		var sourceKey string

		if resolvedCookbook.Source != nil {
			sourceInfo = createSourceInfoFromLocation(resolvedCookbook.Source)
			sourceKey = getSourceKey(resolvedCookbook.Source)
		} else {
			// Use default source if source is nil
			sourceKey = source.PUBLIC_SUPERMARKET
		}

		// Add to lock file
		lockFile.AddCookbook(sourceKey, resolvedCookbook.Cookbook, sourceInfo)
	}

	return lockFile, nil
}

// GenerateBoth creates and saves both JSON and Ruby format lock files
func (m *Manager) GenerateBoth(resolution *resolver.Resolution, dependencies []string) error {
	lockFile, err := m.Generate(resolution)
	if err != nil {
		return err
	}

	return m.SaveBoth(lockFile, dependencies)
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

// UpdateBoth updates both JSON and Ruby format lock files
func (m *Manager) UpdateBoth(resolution *resolver.Resolution, dependencies []string) error {
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
	*existingLock = *newLock

	// Save both formats
	return m.SaveBoth(existingLock, dependencies)
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

// GetPath returns the JSON lock file path
func (m *Manager) GetPath() string {
	return m.lockFilePath
}

// GetRubyPath returns the Ruby lock file path
func (m *Manager) GetRubyPath() string {
	return m.rubyLockFilePath
}

// Remove deletes the JSON lock file
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

// RemoveRuby deletes the Ruby lock file
func (m *Manager) RemoveRuby() error {
	if !m.RubyExists() {
		return nil // Already doesn't exist
	}

	err := os.Remove(m.rubyLockFilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove Ruby lock file %s: %w", m.rubyLockFilePath, err)
	}

	return nil
}

// RemoveBoth deletes both lock files
func (m *Manager) RemoveBoth() error {
	if err := m.Remove(); err != nil {
		return err
	}
	return m.RemoveRuby()
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

// createSourceInfoFromLocation creates SourceInfo from typed SourceLocation
func createSourceInfoFromLocation(loc *berkshelf.SourceLocation) *SourceInfo {
	if loc == nil {
		return &SourceInfo{
			Type: "supermarket",
			URL:  source.PUBLIC_SUPERMARKET,
		}
	}

	sourceInfo := &SourceInfo{
		Type: loc.Type,
		URL:  loc.URL,
		Path: loc.Path,
		Ref:  loc.Ref,
	}

	if loc.Options != nil {
		if branch, ok := loc.Options["branch"].(string); ok {
			sourceInfo.Branch = branch
		}
		if tag, ok := loc.Options["tag"].(string); ok {
			sourceInfo.Tag = tag
		}
		if revision, ok := loc.Options["revision"].(string); ok && sourceInfo.Ref == "" {
			sourceInfo.Ref = revision
		}
	}

	// Only set default URL for supermarket sources without a URL
	if sourceInfo.Type == "supermarket" && sourceInfo.URL == "" {
		sourceInfo.URL = source.PUBLIC_SUPERMARKET
	}

	return sourceInfo
}

// getSourceKey returns the appropriate key for grouping cookbooks by source
func getSourceKey(loc *berkshelf.SourceLocation) string {
	if loc == nil {
		return source.PUBLIC_SUPERMARKET
	}

	// For path sources, use "path" as the key to group all path cookbooks together
	if loc.Type == "path" {
		return "path"
	}

	// For git sources, use the URL as the key
	if loc.Type == "git" && loc.URL != "" {
		return loc.URL
	}

	// For supermarket and chef_server, use the URL
	if loc.URL != "" {
		return loc.URL
	}

	// Fallback to default supermarket
	return source.PUBLIC_SUPERMARKET
}

// ExtractDirectDependencies extracts the direct dependencies from a Berksfile
func ExtractDirectDependencies(berksfilePath string, groups []string) ([]string, error) {
	// Parse the Berksfile
	parsedBerksfile, err := berksfile.Load(berksfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Berksfile: %w", err)
	}

	// Get cookbooks, optionally filtered by groups
	var cookbooks []*berksfile.CookbookDef
	if len(groups) > 0 {
		cookbooks = parsedBerksfile.GetCookbooks(groups...)
	} else {
		cookbooks = parsedBerksfile.GetCookbooks()
	}

	// Extract cookbook names
	var dependencies []string
	for _, cookbook := range cookbooks {
		dependencies = append(dependencies, cookbook.Name)
	}

	// Sort dependencies for consistent output
	for i := 0; i < len(dependencies); i++ {
		for j := i + 1; j < len(dependencies); j++ {
			if dependencies[i] > dependencies[j] {
				dependencies[i], dependencies[j] = dependencies[j], dependencies[i]
			}
		}
	}

	return dependencies, nil
}
