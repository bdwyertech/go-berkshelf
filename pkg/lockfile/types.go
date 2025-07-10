package lockfile

import (
	"bytes"
	"encoding/json"
	"maps"
	"time"

	"github.com/bdwyertech/go-berkshelf/pkg/berkshelf"
)

// LockFile represents a Berksfile.lock file structure
type LockFile struct {
	Revision    int                    `json:"revision"`
	GeneratedAt time.Time              `json:"generated_at"`
	Sources     map[string]*SourceLock `json:"sources"`
}

// SourceLock represents a cookbook source in the lock file
type SourceLock struct {
	Type      string                   `json:"type,omitempty"`
	URL       string                   `json:"url,omitempty"`
	Cookbooks map[string]*CookbookLock `json:"cookbooks"`
}

// CookbookLock represents a locked cookbook with its resolved dependencies
type CookbookLock struct {
	Version      string            `json:"version"`
	Dependencies map[string]string `json:"dependencies,omitempty"`
	Source       *SourceInfo       `json:"source,omitempty"`
}

// SourceInfo contains additional source information for the cookbook
type SourceInfo struct {
	Type   string `json:"type"`
	URL    string `json:"url,omitempty"`
	Path   string `json:"path,omitempty"`
	Branch string `json:"branch,omitempty"`
	Tag    string `json:"tag,omitempty"`
	Ref    string `json:"ref,omitempty"`
}

// NewLockFile creates a new lock file with current revision
func NewLockFile() *LockFile {
	return &LockFile{
		Revision:    7, // Berkshelf lock file format version
		GeneratedAt: time.Now(),
		Sources:     make(map[string]*SourceLock),
	}
}

// AddCookbook adds a cookbook to the lock file under the specified source
func (lf *LockFile) AddCookbook(sourceURL string, cookbook *berkshelf.Cookbook, sourceInfo *SourceInfo) {
	// Handle nil sourceInfo by creating a default one
	if sourceInfo == nil {
		sourceInfo = &SourceInfo{
			Type: "supermarket",
			URL:  sourceURL,
		}
	}

	// Ensure source exists
	if lf.Sources[sourceURL] == nil {
		lf.Sources[sourceURL] = &SourceLock{
			Type:      sourceInfo.Type,
			URL:       sourceURL,
			Cookbooks: make(map[string]*CookbookLock),
		}
	}

	// Convert dependencies to string map
	deps := make(map[string]string)
	for name, constraint := range cookbook.Dependencies {
		deps[name] = constraint.String()
	}

	// Add cookbook lock
	lf.Sources[sourceURL].Cookbooks[cookbook.Name] = &CookbookLock{
		Version:      cookbook.Version.String(),
		Dependencies: deps,
		Source:       sourceInfo,
	}
}

// GetCookbook retrieves a cookbook from the lock file
func (lf *LockFile) GetCookbook(name string) (*CookbookLock, string, bool) {
	for sourceURL, source := range lf.Sources {
		if cookbook, exists := source.Cookbooks[name]; exists {
			return cookbook, sourceURL, true
		}
	}
	return nil, "", false
}

// HasCookbook checks if a cookbook exists in the lock file
func (lf *LockFile) HasCookbook(name string) bool {
	_, _, exists := lf.GetCookbook(name)
	return exists
}

// ListCookbooks returns all cookbooks in the lock file
func (lf *LockFile) ListCookbooks() map[string]*CookbookLock {
	cookbooks := make(map[string]*CookbookLock)
	for _, source := range lf.Sources {
		maps.Copy(cookbooks, source.Cookbooks)
	}
	return cookbooks
}

// ToJSON serializes the lock file to JSON
func (lf *LockFile) ToJSON() ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(lf); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// ToRubyFormat serializes the lock file to Ruby Berkshelf format
func (lf *LockFile) ToRubyFormat(dependencies []string) ([]byte, error) {
	var buffer bytes.Buffer

	// Write DEPENDENCIES section
	buffer.WriteString("DEPENDENCIES\n")
	for _, dep := range dependencies {
		buffer.WriteString("  " + dep + "\n")
	}
	buffer.WriteString("\n")

	// Write GRAPH section
	buffer.WriteString("GRAPH\n")

	// Collect all cookbooks and sort them alphabetically
	allCookbooks := make(map[string]*CookbookLock)
	for _, source := range lf.Sources {
		for name, cookbook := range source.Cookbooks {
			allCookbooks[name] = cookbook
		}
	}

	// Sort cookbook names for consistent output
	var sortedNames []string
	for name := range allCookbooks {
		sortedNames = append(sortedNames, name)
	}

	// Simple sort implementation
	for i := 0; i < len(sortedNames); i++ {
		for j := i + 1; j < len(sortedNames); j++ {
			if sortedNames[i] > sortedNames[j] {
				sortedNames[i], sortedNames[j] = sortedNames[j], sortedNames[i]
			}
		}
	}

	// Write each cookbook with its dependencies
	for _, name := range sortedNames {
		cookbook := allCookbooks[name]
		buffer.WriteString("  " + name + " (" + cookbook.Version + ")\n")

		// Sort dependencies for consistent output
		if len(cookbook.Dependencies) > 0 {
			var depNames []string
			for depName := range cookbook.Dependencies {
				depNames = append(depNames, depName)
			}

			// Simple sort implementation for dependencies
			for i := 0; i < len(depNames); i++ {
				for j := i + 1; j < len(depNames); j++ {
					if depNames[i] > depNames[j] {
						depNames[i], depNames[j] = depNames[j], depNames[i]
					}
				}
			}

			// Write dependencies with indentation
			for _, depName := range depNames {
				constraint := cookbook.Dependencies[depName]
				buffer.WriteString("    " + depName + " (" + constraint + ")\n")
			}
		}
	}

	return buffer.Bytes(), nil
}

// FromJSON deserializes a lock file from JSON
func FromJSON(data []byte) (*LockFile, error) {
	var lf LockFile
	err := json.Unmarshal(data, &lf)
	if err != nil {
		return nil, err
	}
	return &lf, nil
}

// IsOutdated checks if the lock file is older than the specified duration
func (lf *LockFile) IsOutdated(maxAge time.Duration) bool {
	return time.Since(lf.GeneratedAt) > maxAge
}

// GetRevision returns the lock file format revision
func (lf *LockFile) GetRevision() int {
	return lf.Revision
}

// UpdateGeneratedAt updates the generation timestamp
func (lf *LockFile) UpdateGeneratedAt() {
	lf.GeneratedAt = time.Now()
}
