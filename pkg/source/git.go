package source

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bdwyer/go-berkshelf/pkg/berkshelf"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

// GitSource implements CookbookSource for Git repositories.
type GitSource struct {
	uri      string
	branch   string
	tag      string
	ref      string
	revision string
	auth     transport.AuthMethod
	cacheDir string
	priority int
}

// NewGitSource creates a new Git source.
func NewGitSource(uri string, opts SourceLocation) (*GitSource, error) {
	if uri == "" {
		return nil, fmt.Errorf("git source requires URI")
	}

	// Expand GitHub shorthand if needed
	if strings.Contains(opts.Type, "github") && !strings.HasPrefix(uri, "http") && !strings.HasPrefix(uri, "git@") {
		// Convert "user/repo" to full GitHub URL
		uri = fmt.Sprintf("https://github.com/%s.git", uri)
	}

	source := &GitSource{
		uri:      uri,
		branch:   opts.Branch,
		tag:      opts.Tag,
		ref:      opts.Ref,
		revision: opts.Revision,
		cacheDir: filepath.Join(os.TempDir(), "berkshelf-git-cache"),
		priority: 50, // Lower priority than Supermarket
	}

	// Set up authentication if needed
	if err := source.setupAuth(opts); err != nil {
		return nil, err
	}

	return source, nil
}

// setupAuth configures authentication based on the URI and options.
func (g *GitSource) setupAuth(opts SourceLocation) error {
	// Check for SSH URL
	if strings.HasPrefix(g.uri, "git@") || strings.Contains(g.uri, "ssh://") {
		// Try to use SSH agent first
		if auth, err := ssh.NewSSHAgentAuth("git"); err == nil {
			g.auth = auth
		} else {
			// Try to use default SSH key
			homeDir, _ := os.UserHomeDir()
			keyFiles := []string{"id_ed25519", "id_ecdsa", "id_rsa"}
			for _, keyName := range keyFiles {
				keyPath := filepath.Join(homeDir, ".ssh", keyName)
				if _, err := os.Stat(keyPath); err == nil {
					publicKeys, err := ssh.NewPublicKeysFromFile("git", keyPath, "")
					if err != nil {
						return fmt.Errorf("setting up SSH auth: %w", err)
					}
					g.auth = publicKeys
					break
				}
			}
		}
	} else if strings.HasPrefix(g.uri, "https://") {
		// Check for username/password in options
		if user, ok := opts.Options["username"]; ok {
			if pass, ok := opts.Options["password"]; ok {
				g.auth = &http.BasicAuth{
					Username: user,
					Password: pass,
				}
			}
		}
		// Could also check for token-based auth
		if token, ok := opts.Options["token"]; ok {
			g.auth = &http.BasicAuth{
				Username: "token",
				Password: token,
			}
		}
	}

	return nil
}

// Name returns the name of this source.
func (g *GitSource) Name() string {
	return fmt.Sprintf("git (%s)", g.uri)
}

// Priority returns the priority of this source.
func (g *GitSource) Priority() int {
	return g.priority
}

// getCacheDir returns the cache directory for a specific cookbook.
func (g *GitSource) getCacheDir(name string) string {
	// Create a safe directory name from the URI
	safeName := strings.ReplaceAll(g.uri, "/", "_")
	safeName = strings.ReplaceAll(safeName, ":", "_")
	safeName = strings.ReplaceAll(safeName, ".", "_")

	return filepath.Join(g.cacheDir, safeName, name)
}

// clone clones or updates the repository.
func (g *GitSource) clone(ctx context.Context, name string) (*git.Repository, error) {
	cacheDir := g.getCacheDir(name)

	// Check if already cloned
	repo, err := git.PlainOpen(cacheDir)
	if err == nil {
		// Repository exists, try to fetch updates
		w, err := repo.Worktree()
		if err != nil {
			return nil, fmt.Errorf("getting worktree: %w", err)
		}

		err = w.Pull(&git.PullOptions{
			RemoteName: "origin",
			Auth:       g.auth,
		})
		if err != nil && err != git.NoErrAlreadyUpToDate {
			// If pull fails, continue with existing clone
		}

		return repo, nil
	}

	// Clone the repository
	cloneOpts := &git.CloneOptions{
		URL:      g.uri,
		Auth:     g.auth,
		Progress: nil, // Could add progress reporting
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("creating cache directory: %w", err)
	}

	repo, err = git.PlainCloneContext(ctx, cacheDir, false, cloneOpts)
	if err != nil {
		return nil, fmt.Errorf("cloning repository: %w", err)
	}

	return repo, nil
}

// checkout checks out the specified ref, tag, or branch.
func (g *GitSource) checkout(repo *git.Repository) error {
	w, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("getting worktree: %w", err)
	}

	// Determine what to checkout
	var checkoutRef string
	if g.revision != "" {
		checkoutRef = g.revision
	} else if g.tag != "" {
		checkoutRef = "refs/tags/" + g.tag
	} else if g.ref != "" {
		checkoutRef = g.ref
	} else if g.branch != "" {
		checkoutRef = "refs/heads/" + g.branch
	} else {
		// Default to master/main
		checkoutRef = "refs/heads/master"
	}

	// Try to resolve the reference
	hash, err := repo.ResolveRevision(plumbing.Revision(checkoutRef))
	if err != nil {
		// Try alternative branch name
		if checkoutRef == "refs/heads/master" {
			checkoutRef = "refs/heads/main"
			hash, err = repo.ResolveRevision(plumbing.Revision(checkoutRef))
		}
		if err != nil {
			return fmt.Errorf("resolving ref %s: %w", checkoutRef, err)
		}
	}

	// Checkout the specific commit
	err = w.Checkout(&git.CheckoutOptions{
		Hash: *hash,
	})
	if err != nil {
		return fmt.Errorf("checking out %s: %w", checkoutRef, err)
	}

	return nil
}

// ListVersions returns available versions (tags) from the Git repository.
func (g *GitSource) ListVersions(ctx context.Context, name string) ([]*berkshelf.Version, error) {
	repo, err := g.clone(ctx, name)
	if err != nil {
		return nil, err
	}

	// List all tags
	tags, err := repo.Tags()
	if err != nil {
		return nil, fmt.Errorf("listing tags: %w", err)
	}

	versions := make([]*berkshelf.Version, 0)
	err = tags.ForEach(func(ref *plumbing.Reference) error {
		tagName := ref.Name().Short()
		// Try to parse as version
		if v, err := berkshelf.NewVersion(tagName); err == nil {
			versions = append(versions, v)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("iterating tags: %w", err)
	}

	// If no version tags found but we have a specific ref, return a pseudo-version
	if len(versions) == 0 && (g.revision != "" || g.ref != "" || g.branch != "") {
		// Use a special version to indicate this is from a specific ref
		v, _ := berkshelf.NewVersion("0.0.0")
		versions = append(versions, v)
	}

	return versions, nil
}

// FetchMetadata reads the metadata from the cloned repository.
func (g *GitSource) FetchMetadata(ctx context.Context, name string, version *berkshelf.Version) (*berkshelf.Metadata, error) {
	repo, err := g.clone(ctx, name)
	if err != nil {
		return nil, err
	}

	// Checkout the appropriate version
	if version.String() != "0.0.0" {
		// This is a real version tag
		g.tag = version.String()
	}

	if err := g.checkout(repo); err != nil {
		return nil, err
	}

	// Find the repository root
	w, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("getting worktree: %w", err)
	}

	repoPath := w.Filesystem.Root()

	// Look for metadata.json or metadata.rb
	metadataPath := filepath.Join(repoPath, "metadata.json")
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		// Try metadata.rb
		metadataPath = filepath.Join(repoPath, "metadata.rb")
		if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
			return nil, &ErrInvalidMetadata{
				Name:   name,
				Reason: "no metadata.json or metadata.rb found",
			}
		}
		// For now, we don't parse metadata.rb
		// In a full implementation, we would need a Ruby parser
		return &berkshelf.Metadata{
			Name:    name,
			Version: version,
		}, nil
	}

	// Parse metadata.json
	_, err = os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("reading metadata: %w", err)
	}

	// TODO: Implement JSON parsing of metadata
	// For now, return a basic metadata
	return &berkshelf.Metadata{
		Name:    name,
		Version: version,
	}, nil
}

// FetchCookbook downloads the complete cookbook.
func (g *GitSource) FetchCookbook(ctx context.Context, name string, version *berkshelf.Version) (*berkshelf.Cookbook, error) {
	metadata, err := g.FetchMetadata(ctx, name, version)
	if err != nil {
		return nil, err
	}

	// The cookbook is already cloned in the cache directory
	cookbook := &berkshelf.Cookbook{
		Name:     name,
		Version:  version,
		Metadata: metadata,
		Path:     g.getCacheDir(name),
	}

	return cookbook, nil
}

// Search is not implemented for Git sources.
func (g *GitSource) Search(ctx context.Context, query string) ([]*berkshelf.Cookbook, error) {
	return nil, ErrNotImplemented
}
