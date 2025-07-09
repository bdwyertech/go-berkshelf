package source

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/bdwyertech/go-berkshelf/pkg/berkshelf"
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
func NewGitSource(uri string, opts *berkshelf.SourceLocation) (*GitSource, error) {
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
		branch:   getStringOption(opts.Options, "branch"),
		tag:      getStringOption(opts.Options, "tag"),
		ref:      opts.Ref,
		revision: getStringOption(opts.Options, "revision"),
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
func (g *GitSource) setupAuth(opts *berkshelf.SourceLocation) error {
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
		if user := getStringOption(opts.Options, "username"); user != "" {
			if pass := getStringOption(opts.Options, "password"); pass != "" {
				g.auth = &http.BasicAuth{
					Username: user,
					Password: pass,
				}
			}
		}
		// Could also check for token-based auth
		if token := getStringOption(opts.Options, "token"); token != "" {
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

// GetBranch returns the branch for this Git source.
func (g *GitSource) GetBranch() string {
	return g.branch
}

// GetTag returns the tag for this Git source.
func (g *GitSource) GetTag() string {
	return g.tag
}

// GetRef returns the ref for this Git source.
func (g *GitSource) GetRef() string {
	return g.ref
}

// GetRevision returns the revision for this Git source.
func (g *GitSource) GetRevision() string {
	return g.revision
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
		err = repo.Fetch(&git.FetchOptions{
			RemoteName: "origin",
			Auth:       g.auth,
		})
		if err != nil && err != git.NoErrAlreadyUpToDate {
			// If fetch fails, continue with existing clone
			log.Debugf("Failed to fetch updates for %s: %v", name, err)
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
	} else if g.ref != "" {
		checkoutRef = g.ref
	} else if g.tag != "" {
		checkoutRef = "refs/tags/" + g.tag
	} else if g.branch != "" {
		checkoutRef = "refs/heads/" + g.branch
	} else {
		// Default to master/main
		checkoutRef = "refs/heads/master"
	}

	// Try to resolve the reference
	hash, err := repo.ResolveRevision(plumbing.Revision(checkoutRef))
	if err != nil {
		// If it's a branch that doesn't exist locally, try remote reference
		if g.branch != "" {
			remoteRef := "refs/remotes/origin/" + g.branch
			hash, err = repo.ResolveRevision(plumbing.Revision(remoteRef))
			if err == nil {
				// Create local branch tracking the remote branch
				branchRef := plumbing.NewBranchReferenceName(g.branch)
				ref := plumbing.NewHashReference(branchRef, *hash)
				err = repo.Storer.SetReference(ref)
				if err != nil {
					log.Debugf("Failed to create local branch %s: %v", g.branch, err)
				}
				checkoutRef = "refs/heads/" + g.branch
			}
		}

		// Try alternative default branch names
		if checkoutRef == "refs/heads/master" {
			checkoutRef = "refs/heads/main"
			hash, err = repo.ResolveRevision(plumbing.Revision(checkoutRef))
			if err != nil {
				// Try remote main
				hash, err = repo.ResolveRevision(plumbing.Revision("refs/remotes/origin/main"))
			}
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
	g.revision = hash.String()

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

// DownloadAndExtractCookbook copies the cookbook files from the Git cache to the target directory.
func (g *GitSource) DownloadAndExtractCookbook(ctx context.Context, cookbook *berkshelf.Cookbook, targetDir string) error {
	// Ensure the cookbook is cloned and at the right version
	repo, err := g.clone(ctx, cookbook.Name)
	if err != nil {
		return fmt.Errorf("cloning repository: %w", err)
	}

	if err := g.checkout(repo); err != nil {
		return fmt.Errorf("checking out version: %w", err)
	}

	// Get the source directory (repository root)
	w, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("getting worktree: %w", err)
	}

	sourceDir := w.Filesystem.Root()

	// Create target directory
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("creating target directory: %w", err)
	}

	// Copy all files from source to target
	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .git directory
		if strings.Contains(path, ".git") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Calculate relative path
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(targetDir, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		// Copy file
		return copyFile(path, targetPath, info.Mode())
	})

	if err != nil {
		return fmt.Errorf("copying cookbook files: %w", err)
	}

	// Update cookbook path
	cookbook.Path = targetDir

	return nil
}

// copyFile copies a file from src to dst with the given mode.
func copyFile(src, dst string, mode os.FileMode) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return os.Chmod(dst, mode)
}

// Search is not implemented for Git sources.
func (g *GitSource) Search(ctx context.Context, query string) ([]*berkshelf.Cookbook, error) {
	return nil, ErrNotImplemented
}

// GetSourceLocation returns the source location for this git source
func (g *GitSource) GetSourceLocation() *berkshelf.SourceLocation {
	location := &berkshelf.SourceLocation{
		Type: "git",
		URL:  g.uri,
		Ref:  g.ref,
	}

	// Add Git-specific options
	if g.branch != "" || g.tag != "" || g.revision != "" {
		location.Options = make(map[string]any)
		if g.branch != "" {
			location.Options["branch"] = g.branch
		}
		if g.tag != "" {
			location.Options["tag"] = g.tag
		}
		if g.revision != "" {
			location.Options["revision"] = g.revision
		}
	}

	return location
}

// GetSourceType returns the source type
func (g *GitSource) GetSourceType() string {
	return "git"
}

// GetSourceURL returns the source URL
func (g *GitSource) GetSourceURL() string {
	return g.uri
}
