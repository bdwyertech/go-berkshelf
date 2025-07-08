package policyfile

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bdwyer/go-berkshelf/pkg/berkshelf"
)

// LoadPolicyfile loads and parses a Policyfile.rb from the given path
func LoadPolicyfile(path string) (*Policyfile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read Policyfile.rb: %w", err)
	}

	return ParsePolicyfile(string(content))
}

// FindPolicyfile searches for a Policyfile.rb in the given directory and parent directories
func FindPolicyfile(startDir string) (string, error) {
	dir := startDir
	for {
		policyfilePath := filepath.Join(dir, "Policyfile.rb")
		if _, err := os.Stat(policyfilePath); err == nil {
			return policyfilePath, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root directory
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("Policyfile.rb not found in %s or any parent directory", startDir)
}

// ToBerksfileEquivalent converts a Policyfile to a structure that can be used
// with the existing Berkshelf resolver and source systems
func (p *Policyfile) ToBerksfileEquivalent() (*BerksfileEquivalent, error) {
	return &BerksfileEquivalent{
		Sources:   p.DefaultSources,
		Cookbooks: p.Cookbooks,
	}, nil
}

// BerksfileEquivalent represents the Berkshelf-compatible parts of a Policyfile
type BerksfileEquivalent struct {
	Sources   []*berkshelf.SourceLocation
	Cookbooks []*CookbookDef
}
