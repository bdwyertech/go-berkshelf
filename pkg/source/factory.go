package source

import (
	"fmt"
	"strings"

	"github.com/bdwyer/go-berkshelf/pkg/berksfile"
)

// Factory creates CookbookSource instances from Berksfile entries.
type Factory struct {
	defaultSources []CookbookSource
}

// NewFactory creates a new source factory.
func NewFactory() *Factory {
	return &Factory{
		defaultSources: make([]CookbookSource, 0),
	}
}

// AddDefaultSource adds a default source to use when no specific source is specified.
func (f *Factory) AddDefaultSource(source CookbookSource) {
	f.defaultSources = append(f.defaultSources, source)
}

// CreateFromBerksfile creates a Manager with sources from a Berksfile.
func (f *Factory) CreateFromBerksfile(bf *berksfile.Berksfile) (*Manager, error) {
	manager := NewManager()

	// Add sources from Berksfile
	for _, sourceURL := range bf.Sources {
		source, err := f.createFromURL(sourceURL)
		if err != nil {
			return nil, fmt.Errorf("creating source from %s: %w", sourceURL, err)
		}
		manager.AddSource(source)
	}

	// Add default sources if no sources were specified
	if len(bf.Sources) == 0 {
		for _, source := range f.defaultSources {
			manager.AddSource(source)
		}

		// If no defaults either, add the public Supermarket
		if len(f.defaultSources) == 0 {
			manager.AddSource(NewSupermarketSource("https://supermarket.chef.io"))
		}
	}

	return manager, nil
}

// CreateFromLocation creates a source from a SourceLocation.
func (f *Factory) CreateFromLocation(location berksfile.SourceLocation) (CookbookSource, error) {
	switch location.Type {
	case berksfile.SourceGit:
		opts := SourceLocation{
			Type:     string(location.Type),
			URI:      location.URI,
			Branch:   location.Options["branch"],
			Tag:      location.Options["tag"],
			Ref:      location.Options["ref"],
			Revision: location.Options["revision"],
			Options:  location.Options,
		}
		return NewGitSource(location.URI, opts)

	case "github": // Handle github as a string since it may not be a const
		// GitHub is a special case of git
		opts := SourceLocation{
			Type:     "github",
			URI:      location.URI,
			Branch:   location.Options["branch"],
			Tag:      location.Options["tag"],
			Ref:      location.Options["ref"],
			Revision: location.Options["revision"],
			Options:  location.Options,
		}
		return NewGitSource(location.URI, opts)

	case berksfile.SourcePath:
		path := location.Options["path"]
		if path == "" {
			path = location.URI
		}
		return NewPathSource(path)

	case berksfile.SourceSupermarket:
		url := location.URI
		if url == "" {
			url = "https://supermarket.chef.io"
		}
		return NewSupermarketSource(url), nil

	default:
		return nil, fmt.Errorf("unknown source type: %s", location.Type)
	}
}

// createFromURL creates a source from a URL string.
func (f *Factory) createFromURL(url string) (CookbookSource, error) {
	// Determine the type of source from the URL
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		// Assume it's a Supermarket API endpoint
		return NewSupermarketSource(url), nil
	}

	if strings.HasPrefix(url, "git://") || strings.HasPrefix(url, "git@") {
		// Git source
		opts := SourceLocation{Type: "git"}
		return NewGitSource(url, opts)
	}

	if strings.HasPrefix(url, "file://") {
		// Local path
		path := strings.TrimPrefix(url, "file://")
		return NewPathSource(path)
	}

	// Default to Supermarket
	return NewSupermarketSource(url), nil
}

// CreateSourceForCookbook creates appropriate sources for a cookbook definition.
func (f *Factory) CreateSourceForCookbook(cookbook *berksfile.CookbookDef) ([]CookbookSource, error) {
	sources := make([]CookbookSource, 0)

	// If cookbook has a specific source, use that
	if cookbook.Source.Type != "" {
		source, err := f.CreateFromLocation(cookbook.Source)
		if err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}

	return sources, nil
}
