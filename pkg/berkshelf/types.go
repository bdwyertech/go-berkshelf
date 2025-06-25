package berkshelf

// SourceLocation represents where a cookbook comes from
type SourceLocation struct {
	Type    string         `json:"type"` // "supermarket", "git", "path", "chef_server"
	URL     string         `json:"url,omitempty"`
	Ref     string         `json:"ref,omitempty"`  // git branch/tag/commit
	Path    string         `json:"path,omitempty"` // local path or subdir in git
	Options map[string]any `json:"options,omitempty"`
}
