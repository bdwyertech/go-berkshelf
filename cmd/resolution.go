package cmd

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/bdwyer/go-berkshelf/pkg/berksfile"
	"github.com/bdwyer/go-berkshelf/pkg/resolver"
	"github.com/bdwyer/go-berkshelf/pkg/source"
)

// SetupSourcesFromBerksfile sets up the source manager with sources from the Berksfile
func SetupSourcesFromBerksfile(berks *berksfile.Berksfile) (*source.Manager, error) {
	sourceManager := source.NewManager()
	factory := source.NewFactory()

	// Add sources from Berksfile
	for _, sourceURL := range berks.Sources {
		src, err := factory.CreateFromURL(sourceURL)
		if err != nil {
			log.Warnf("failed to create source from %s: %v", sourceURL, err)
			continue
		}
		sourceManager.AddSource(src)
	}

	// Add default Supermarket if no sources specified
	if len(berks.Sources) == 0 {
		defaultSource := source.NewSupermarketSource(source.PUBLIC_SUPERMARKET)
		sourceManager.AddSource(defaultSource)
	}

	return sourceManager, nil
}

// ResolveDependencies resolves cookbook dependencies and handles errors
func ResolveDependencies(ctx context.Context, requirements []*resolver.Requirement, sources []source.CookbookSource) (*resolver.Resolution, error) {
	resolverImpl := resolver.NewResolver(sources)

	resolution, err := resolverImpl.Resolve(ctx, requirements)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve dependencies: %w", err)
	}

	if resolution.HasErrors() {
		log.Error("Resolution errors:")
		for _, resErr := range resolution.Errors {
			log.Error(resErr)
		}
		return nil, fmt.Errorf("dependency resolution failed with %d errors", len(resolution.Errors))
	}

	return resolution, nil
}

// CreateRequirementsFromCookbooks creates resolver requirements from cookbook definitions
func CreateRequirementsFromCookbooks(cookbooks []*berksfile.CookbookDef) []*resolver.Requirement {
	requirements := make([]*resolver.Requirement, 0, len(cookbooks))
	for _, cookbook := range cookbooks {
		req := resolver.NewRequirementWithSource(cookbook.Name, cookbook.Constraint, cookbook.Source)
		requirements = append(requirements, req)
	}
	return requirements
}
