package types

// Berksfile represents a parsed Berksfile
type Berksfile struct {
	// Sources are the cookbook sources (e.g., supermarket URLs)
	Sources []string

	// Cookbooks are the cookbook dependencies
	Cookbooks []*CookbookRef

	// Groups organize cookbooks
	Groups map[string][]*CookbookRef

	// HasMetadata indicates if the 'metadata' directive was used
	HasMetadata bool

	// MetadataPath is the path to the metadata file (if HasMetadata is true)
	MetadataPath string
}

// AllCookbooks returns all cookbooks including those in groups
func (b *Berksfile) AllCookbooks() []*CookbookRef {
	seen := make(map[string]bool)
	result := make([]*CookbookRef, 0)

	// Add non-grouped cookbooks
	for _, cb := range b.Cookbooks {
		if !seen[cb.Name] {
			result = append(result, cb)
			seen[cb.Name] = true
		}
	}

	// Add grouped cookbooks
	for _, group := range b.Groups {
		for _, cb := range group {
			if !seen[cb.Name] {
				result = append(result, cb)
				seen[cb.Name] = true
			}
		}
	}

	return result
}

// GetCookbook returns a cookbook by name
func (b *Berksfile) GetCookbook(name string) *CookbookRef {
	// Check non-grouped cookbooks
	for _, cb := range b.Cookbooks {
		if cb.Name == name {
			return cb
		}
	}

	// Check grouped cookbooks
	for _, group := range b.Groups {
		for _, cb := range group {
			if cb.Name == name {
				return cb
			}
		}
	}

	return nil
}

// AddCookbook adds a cookbook to the Berksfile
func (b *Berksfile) AddCookbook(cookbook *CookbookRef) {
	b.Cookbooks = append(b.Cookbooks, cookbook)
}

// AddGroupedCookbook adds a cookbook to a specific group
func (b *Berksfile) AddGroupedCookbook(groupName string, cookbook *CookbookRef) {
	if b.Groups == nil {
		b.Groups = make(map[string][]*CookbookRef)
	}

	cookbook.Groups = append(cookbook.Groups, groupName)
	b.Groups[groupName] = append(b.Groups[groupName], cookbook)
}
