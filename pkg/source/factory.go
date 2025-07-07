package source

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/bdwyer/go-berkshelf/pkg/berksfile"
	"github.com/bdwyer/go-berkshelf/pkg/berkshelf"
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
	for _, sourceLocation := range bf.Sources {
		source, err := f.CreateFromLocation(sourceLocation)
		if err != nil {
			return nil, fmt.Errorf("creating source from %s: %w", sourceLocation.URL, err)
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
func (f *Factory) CreateFromLocation(location *berkshelf.SourceLocation) (CookbookSource, error) {
	if location == nil {
		return nil, fmt.Errorf("location cannot be nil")
	}

	switch location.Type {
	case "git":
		// Git source - pass location directly since NewGitSource extracts what it needs
		return NewGitSource(location.URL, location)

	case "github":
		// GitHub is a special case of git
		// Create a new location with github type
		githubLocation := &berkshelf.SourceLocation{
			Type:    "github",
			URL:     location.URL,
			Ref:     location.Ref,
			Path:    location.Path,
			Options: location.Options,
		}
		return NewGitSource(location.URL, githubLocation)

	case "path":
		path := location.Path
		if path == "" {
			path = location.URL
		}
		return NewPathSource(path)

	case "supermarket":
		url := location.URL
		if url == "" {
			url = "https://supermarket.chef.io"
		}
		return NewSupermarketSource(url), nil

	case "chef_server":
		// Extract authentication details from options
		clientName := getStringOption(location.Options, "client_name")
		clientKey := getStringOption(location.Options, "client_key")

		if clientName == "" || clientKey == "" {
			return nil, fmt.Errorf("chef_server source requires client_name and client_key options")
		}

		return NewChefServerSource(location.URL, clientName, clientKey)

	default:
		return nil, fmt.Errorf("unknown source type: %s", location.Type)
	}
}

// getStringOption safely extracts a string value from a map[string]any
func getStringOption(options map[string]any, key string) string {
	if options == nil {
		return ""
	}
	if v, ok := options[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// createFromURL creates a source from a URL string.
func (f *Factory) createFromURL(uri string) (CookbookSource, error) {
	// Handle Chef Server URLs with authentication
	if strings.HasPrefix(uri, "chef_server://") {
		// Parse chef_server://hostname?client_name=name&client_key=path
		chefUrl, err := url.Parse(strings.TrimPrefix(uri, "chef_server://"))
		if err != nil {
			return nil, fmt.Errorf("error parsing %w", err)
		}

		// Parse query parameters
		q := chefUrl.Query()
		clientName := q.Get("client_name")
		clientKey := q.Get("client_key")

		chefUrl.Path = ""

		if clientName == "" || clientKey == "" {
			return nil, fmt.Errorf("chef_server URL missing client_name or client_key: %s", chefUrl.String())
		}

		return NewChefServerSource(chefUrl.String(), clientName, clientKey)
	}

	// Determine the type of source from the URL
	if strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://") {
		// Assume it's a Supermarket API endpoint
		return NewSupermarketSource(uri), nil
	}

	if strings.HasPrefix(uri, "git://") || strings.HasPrefix(uri, "git@") {
		// Git source
		opts := &berkshelf.SourceLocation{
			Type:    "git",
			URL:     uri,
			Options: make(map[string]any),
		}
		return NewGitSource(uri, opts)
	}

	if strings.HasPrefix(uri, "file://") {
		// Local path
		path := strings.TrimPrefix(uri, "file://")
		return NewPathSource(path)
	}

	// Default to Supermarket
	return NewSupermarketSource(uri), nil
}

// CreateFromURL creates a source from a URL string (public method)
func (f *Factory) CreateFromURL(url string) (CookbookSource, error) {
	return f.createFromURL(url)
}

// CreateSourceForCookbook creates appropriate sources for a cookbook definition.
func (f *Factory) CreateSourceForCookbook(cookbook *berksfile.CookbookDef) ([]CookbookSource, error) {
	sources := make([]CookbookSource, 0)

	// If cookbook has a specific source, use that
	if cookbook.Source != nil && cookbook.Source.Type != "" {
		source, err := f.CreateFromLocation(cookbook.Source)
		if err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}

	return sources, nil
}
