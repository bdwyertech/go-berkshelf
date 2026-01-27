package berkshelf

// SourceLocation represents where a cookbook comes from
type SourceLocation struct {
	Type    string         `json:"type"` // "supermarket", "git", "path", "chef_server"
	URL     string         `json:"url,omitempty"`
	Ref     string         `json:"ref,omitempty"`  // git branch/tag/commit
	Path    string         `json:"path,omitempty"` // local path or subdir in git
	Options map[string]any `json:"options,omitempty"`
}

// String returns a string representation of the source location
func (s *SourceLocation) String() string {
	if s == nil {
		return ""
	}

	// For simple cases, just return the URL
	if s.Type == "supermarket" && s.URL != "" {
		return s.URL
	}

	// For other types, include more detail if needed
	if s.URL != "" {
		return s.URL
	}

	if s.Path != "" {
		return s.Path
	}

	return s.Type
}
