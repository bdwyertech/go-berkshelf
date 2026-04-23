package harness

import (
	"fmt"
	"sort"
	"strings"
)

// RubyLockfile represents a parsed Ruby-format Berksfile.lock.
type RubyLockfile struct {
	Dependencies []string
	Graph        map[string]GraphEntry // cookbook name -> entry
}

// GraphEntry represents a single cookbook in the GRAPH section.
type GraphEntry struct {
	Version      string
	Dependencies map[string]string // dep name -> constraint
}

// ParseRubyLockfile parses a Ruby-format Berksfile.lock string.
// It extracts the DEPENDENCIES and GRAPH sections.
func ParseRubyLockfile(content string) (*RubyLockfile, error) {
	lf := &RubyLockfile{
		Graph: make(map[string]GraphEntry),
	}

	lines := strings.Split(content, "\n")
	section := ""
	var currentCookbook string

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Detect section headers
		if trimmed == "DEPENDENCIES" {
			section = "dependencies"
			continue
		}
		if trimmed == "GRAPH" {
			section = "graph"
			currentCookbook = ""
			continue
		}

		// Skip blank lines (they separate sections)
		if trimmed == "" {
			continue
		}

		switch section {
		case "dependencies":
			// DEPENDENCIES lines are indented 2 spaces: "  name" or "  name (~> 1.0)"
			if !strings.HasPrefix(line, "  ") {
				return nil, fmt.Errorf("line %d: expected 2-space indent in DEPENDENCIES section: %q", i+1, line)
			}
			entry := strings.TrimPrefix(line, "  ")
			lf.Dependencies = append(lf.Dependencies, entry)

		case "graph":
			// GRAPH cookbook lines are indented 2 spaces: "  name (version)"
			// GRAPH dependency lines are indented 4 spaces: "    name (constraint)"
			if strings.HasPrefix(line, "    ") {
				// This is a dependency of the current cookbook
				if currentCookbook == "" {
					return nil, fmt.Errorf("line %d: dependency line without a parent cookbook: %q", i+1, line)
				}
				depLine := strings.TrimPrefix(line, "    ")
				name, constraint, err := parseNameConstraint(depLine)
				if err != nil {
					return nil, fmt.Errorf("line %d: parsing graph dependency: %w", i+1, err)
				}
				entry := lf.Graph[currentCookbook]
				if entry.Dependencies == nil {
					entry.Dependencies = make(map[string]string)
				}
				entry.Dependencies[name] = constraint
				lf.Graph[currentCookbook] = entry
			} else if strings.HasPrefix(line, "  ") {
				// This is a cookbook entry
				cbLine := strings.TrimPrefix(line, "  ")
				name, version, err := parseNameConstraint(cbLine)
				if err != nil {
					return nil, fmt.Errorf("line %d: parsing graph cookbook: %w", i+1, err)
				}
				currentCookbook = name
				lf.Graph[name] = GraphEntry{
					Version:      version,
					Dependencies: nil,
				}
			} else {
				return nil, fmt.Errorf("line %d: unexpected line in GRAPH section: %q", i+1, line)
			}
		}
	}

	return lf, nil
}

// parseNameConstraint parses a string like "name (value)" and returns the name and value.
// The value in parentheses can be a version or a constraint like ">= 0.0.0" or "~> 8.0".
func parseNameConstraint(s string) (string, string, error) {
	parenIdx := strings.Index(s, "(")
	if parenIdx < 0 {
		return "", "", fmt.Errorf("expected parenthesized value in %q", s)
	}
	closeIdx := strings.LastIndex(s, ")")
	if closeIdx < 0 {
		return "", "", fmt.Errorf("expected closing parenthesis in %q", s)
	}

	name := strings.TrimSpace(s[:parenIdx])
	value := s[parenIdx+1 : closeIdx]

	if name == "" {
		return "", "", fmt.Errorf("empty name in %q", s)
	}

	return name, value, nil
}

// FormatRubyLockfile formats a RubyLockfile back into the Ruby Berksfile.lock string format.
func FormatRubyLockfile(lf *RubyLockfile) string {
	var b strings.Builder

	// Write DEPENDENCIES section
	b.WriteString("DEPENDENCIES\n")
	for _, dep := range lf.Dependencies {
		b.WriteString("  ")
		b.WriteString(dep)
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Write GRAPH section
	b.WriteString("GRAPH\n")

	// Sort cookbook names for deterministic output
	cookbooks := make([]string, 0, len(lf.Graph))
	for name := range lf.Graph {
		cookbooks = append(cookbooks, name)
	}
	sort.Strings(cookbooks)

	for _, name := range cookbooks {
		entry := lf.Graph[name]
		b.WriteString("  ")
		b.WriteString(name)
		b.WriteString(" (")
		b.WriteString(entry.Version)
		b.WriteString(")\n")

		// Sort dependencies for deterministic output
		if len(entry.Dependencies) > 0 {
			depNames := make([]string, 0, len(entry.Dependencies))
			for depName := range entry.Dependencies {
				depNames = append(depNames, depName)
			}
			sort.Strings(depNames)

			for _, depName := range depNames {
				b.WriteString("    ")
				b.WriteString(depName)
				b.WriteString(" (")
				b.WriteString(entry.Dependencies[depName])
				b.WriteString(")\n")
			}
		}
	}
	b.WriteString("\n")

	return b.String()
}
