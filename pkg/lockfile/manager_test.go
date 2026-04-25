package lockfile_test

import (
	"os"
	"path/filepath"
	"sort"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bdwyertech/go-berkshelf/pkg/berkshelf"
	"github.com/bdwyertech/go-berkshelf/pkg/lockfile"
	"github.com/bdwyertech/go-berkshelf/pkg/resolver"
	"github.com/bdwyertech/go-berkshelf/pkg/source"
)

var _ = Describe("Manager", func() {
	var (
		tmpDir  string
		manager *lockfile.Manager
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "lockfile_test")
		Expect(err).NotTo(HaveOccurred())
		manager = lockfile.NewManager(tmpDir)
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	Describe("NewManager", func() {
		It("should create manager with default path", func() {
			workDir := "/tmp/test"
			m := lockfile.NewManager(workDir)
			expectedPath := filepath.Join(workDir, lockfile.DefaultLockFileName)
			Expect(m.GetPath()).To(Equal(expectedPath))
		})
	})

	Describe("NewManagerWithPath", func() {
		It("should create manager with custom path", func() {
			customPath := "/tmp/custom/Berksfile.lock"
			m := lockfile.NewManagerWithPath(customPath)
			Expect(m.GetPath()).To(Equal(customPath))
		})
	})

	Describe("Exists", func() {
		It("should not exist initially", func() {
			Expect(manager.Exists()).To(BeFalse())
		})

		It("should exist after saving", func() {
			lf := lockfile.NewLockFile()
			err := manager.Save(lf)
			Expect(err).NotTo(HaveOccurred())
			Expect(manager.Exists()).To(BeTrue())
		})
	})

	Describe("Save and Load", func() {
		It("should save and load lock file with data", func() {
			originalLockFile := lockfile.NewLockFile()

			version, err := berkshelf.NewVersion("1.2.3")
			Expect(err).NotTo(HaveOccurred())
			constraint, err := berkshelf.NewConstraint("~> 1.0")
			Expect(err).NotTo(HaveOccurred())

			cookbook := &berkshelf.Cookbook{
				Name:    "nginx",
				Version: version,
				Dependencies: map[string]*berkshelf.Constraint{
					"apt": constraint,
				},
			}

			sourceInfo := &lockfile.SourceInfo{
				Type: "supermarket",
				URL:  source.PUBLIC_SUPERMARKET,
			}

			originalLockFile.AddCookbook(source.PUBLIC_SUPERMARKET, cookbook, sourceInfo)

			err = manager.Save(originalLockFile)
			Expect(err).NotTo(HaveOccurred())

			loadedLockFile, err := manager.Load()
			Expect(err).NotTo(HaveOccurred())

			Expect(loadedLockFile.Revision).To(Equal(originalLockFile.Revision))
			Expect(loadedLockFile.HasCookbook("nginx")).To(BeTrue())

			loadedCookbook, _, exists := loadedLockFile.GetCookbook("nginx")
			Expect(exists).To(BeTrue())
			Expect(loadedCookbook.Version).To(Equal("1.2.3"))
		})
	})

	Describe("Load", func() {
		It("should return empty lock file for non-existent path", func() {
			lf, err := manager.Load()
			Expect(err).NotTo(HaveOccurred())
			Expect(lf).NotTo(BeNil())
			Expect(lf.Sources).To(BeEmpty())
		})
	})

	Describe("Generate", func() {
		It("should generate lock file from resolution", func() {
			resolution := resolver.NewResolution()

			version, err := berkshelf.NewVersion("1.2.3")
			Expect(err).NotTo(HaveOccurred())

			cookbook := &berkshelf.Cookbook{
				Name:         "nginx",
				Version:      version,
				Dependencies: make(map[string]*berkshelf.Constraint),
			}

			resolvedCookbook := &resolver.ResolvedCookbook{
				Name:         "nginx",
				Version:      version,
				Source:       nil,
				Dependencies: make(map[string]*berkshelf.Version),
				Cookbook:      cookbook,
			}

			resolution.AddCookbook(resolvedCookbook)

			lf, err := manager.Generate(resolution)
			Expect(err).NotTo(HaveOccurred())
			Expect(lf.HasCookbook("nginx")).To(BeTrue())
		})
	})

	Describe("Update", func() {
		It("should update lock file with new resolution", func() {
			initialLockFile := lockfile.NewLockFile()
			err := manager.Save(initialLockFile)
			Expect(err).NotTo(HaveOccurred())

			resolution := resolver.NewResolution()

			version, err := berkshelf.NewVersion("1.2.3")
			Expect(err).NotTo(HaveOccurred())

			cookbook := &berkshelf.Cookbook{
				Name:         "nginx",
				Version:      version,
				Dependencies: make(map[string]*berkshelf.Constraint),
			}

			resolvedCookbook := &resolver.ResolvedCookbook{
				Name:         "nginx",
				Version:      version,
				Source:       nil,
				Dependencies: make(map[string]*berkshelf.Version),
				Cookbook:      cookbook,
			}

			resolution.AddCookbook(resolvedCookbook)

			err = manager.Update(resolution)
			Expect(err).NotTo(HaveOccurred())

			updatedLockFile, err := manager.Load()
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedLockFile.HasCookbook("nginx")).To(BeTrue())
		})
	})

	Describe("IsOutdated", func() {
		It("should report non-existent lock file as outdated", func() {
			outdated, err := manager.IsOutdated()
			Expect(err).NotTo(HaveOccurred())
			Expect(outdated).To(BeTrue())
		})

		It("should not be outdated without Berksfile", func() {
			lf := lockfile.NewLockFile()
			err := manager.Save(lf)
			Expect(err).NotTo(HaveOccurred())

			outdated, err := manager.IsOutdated()
			Expect(err).NotTo(HaveOccurred())
			Expect(outdated).To(BeFalse())
		})
	})

	Describe("Validate", func() {
		It("should error for non-existent lock file", func() {
			err := manager.Validate()
			Expect(err).To(HaveOccurred())
		})

		It("should pass for valid lock file", func() {
			lf := lockfile.NewLockFile()
			version, err := berkshelf.NewVersion("1.2.3")
			Expect(err).NotTo(HaveOccurred())

			cookbook := &berkshelf.Cookbook{
				Name:         "nginx",
				Version:      version,
				Dependencies: make(map[string]*berkshelf.Constraint),
			}

			sourceInfo := &lockfile.SourceInfo{
				Type: "supermarket",
				URL:  "https://supermarket.chef.io",
			}

			lf.AddCookbook("https://supermarket.chef.io", cookbook, sourceInfo)

			err = manager.Save(lf)
			Expect(err).NotTo(HaveOccurred())

			err = manager.Validate()
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Remove", func() {
		It("should not error for non-existent lock file", func() {
			err := manager.Remove()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should remove existing lock file", func() {
			lf := lockfile.NewLockFile()
			err := manager.Save(lf)
			Expect(err).NotTo(HaveOccurred())
			Expect(manager.Exists()).To(BeTrue())

			err = manager.Remove()
			Expect(err).NotTo(HaveOccurred())
			Expect(manager.Exists()).To(BeFalse())
		})
	})

	Describe("Backup", func() {
		It("should error for non-existent lock file", func() {
			err := manager.Backup()
			Expect(err).To(HaveOccurred())
		})

		It("should create backup of existing lock file", func() {
			lf := lockfile.NewLockFile()
			version, err := berkshelf.NewVersion("1.2.3")
			Expect(err).NotTo(HaveOccurred())

			cookbook := &berkshelf.Cookbook{
				Name:         "nginx",
				Version:      version,
				Dependencies: make(map[string]*berkshelf.Constraint),
			}

			sourceInfo := &lockfile.SourceInfo{
				Type: "supermarket",
				URL:  "https://supermarket.chef.io",
			}

			lf.AddCookbook("https://supermarket.chef.io", cookbook, sourceInfo)

			err = manager.Save(lf)
			Expect(err).NotTo(HaveOccurred())

			err = manager.Backup()
			Expect(err).NotTo(HaveOccurred())

			backupPath := manager.GetPath() + ".backup"
			_, err = os.Stat(backupPath)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Source Key Grouping", func() {
		It("should group path sources under 'path' key", func() {
			resolution := resolver.NewResolution()

			version, err := berkshelf.NewVersion("0.0.1")
			Expect(err).NotTo(HaveOccurred())

			cookbook := &berkshelf.Cookbook{
				Name:         "test",
				Version:      version,
				Dependencies: make(map[string]*berkshelf.Constraint),
			}

			resolvedCookbook := &resolver.ResolvedCookbook{
				Name:    "test",
				Version: version,
				Cookbook: cookbook,
				Source: &berkshelf.SourceLocation{
					Type: "path",
					Path: "/some/path",
				},
			}

			resolution.AddCookbook(resolvedCookbook)

			lf, err := manager.Generate(resolution)
			Expect(err).NotTo(HaveOccurred())
			Expect(lf.Sources).To(HaveKey("path"))
		})

		It("should group git sources by URL", func() {
			resolution := resolver.NewResolution()

			version, err := berkshelf.NewVersion("1.0.0")
			Expect(err).NotTo(HaveOccurred())

			cookbook := &berkshelf.Cookbook{
				Name:         "git-cookbook",
				Version:      version,
				Dependencies: make(map[string]*berkshelf.Constraint),
			}

			resolvedCookbook := &resolver.ResolvedCookbook{
				Name:    "git-cookbook",
				Version: version,
				Cookbook: cookbook,
				Source: &berkshelf.SourceLocation{
					Type: "git",
					URL:  "https://github.com/example/repo.git",
				},
			}

			resolution.AddCookbook(resolvedCookbook)

			lf, err := manager.Generate(resolution)
			Expect(err).NotTo(HaveOccurred())
			Expect(lf.Sources).To(HaveKey("https://github.com/example/repo.git"))
		})

		It("should group supermarket sources by URL", func() {
			resolution := resolver.NewResolution()

			version, err := berkshelf.NewVersion("2.0.0")
			Expect(err).NotTo(HaveOccurred())

			cookbook := &berkshelf.Cookbook{
				Name:         "nginx",
				Version:      version,
				Dependencies: make(map[string]*berkshelf.Constraint),
			}

			resolvedCookbook := &resolver.ResolvedCookbook{
				Name:    "nginx",
				Version: version,
				Cookbook: cookbook,
				Source: &berkshelf.SourceLocation{
					Type: "supermarket",
					URL:  "https://supermarket.chef.io",
				},
			}

			resolution.AddCookbook(resolvedCookbook)

			lf, err := manager.Generate(resolution)
			Expect(err).NotTo(HaveOccurred())
			Expect(lf.Sources).To(HaveKey("https://supermarket.chef.io"))
		})

		It("should use default supermarket for nil source", func() {
			resolution := resolver.NewResolution()

			version, err := berkshelf.NewVersion("1.0.0")
			Expect(err).NotTo(HaveOccurred())

			cookbook := &berkshelf.Cookbook{
				Name:         "default-cookbook",
				Version:      version,
				Dependencies: make(map[string]*berkshelf.Constraint),
			}

			resolvedCookbook := &resolver.ResolvedCookbook{
				Name:         "default-cookbook",
				Version:      version,
				Source:       nil,
				Dependencies: make(map[string]*berkshelf.Version),
				Cookbook:      cookbook,
			}

			resolution.AddCookbook(resolvedCookbook)

			lf, err := manager.Generate(resolution)
			Expect(err).NotTo(HaveOccurred())
			Expect(lf.Sources).To(HaveKey(source.PUBLIC_SUPERMARKET))
		})
	})

	Describe("Multiple Source Types", func() {
		It("should handle multiple source types", func() {
			resolution := resolver.NewResolution()

			// Add path cookbook
			pathVersion, err := berkshelf.NewVersion("0.0.1")
			Expect(err).NotTo(HaveOccurred())
			pathCookbook := &berkshelf.Cookbook{
				Name:         "test",
				Version:      pathVersion,
				Dependencies: make(map[string]*berkshelf.Constraint),
			}
			pathResolved := &resolver.ResolvedCookbook{
				Name:    "test",
				Version: pathVersion,
				Cookbook: pathCookbook,
				Source: &berkshelf.SourceLocation{
					Type: "path",
					Path: "/path/to/test",
				},
			}
			resolution.AddCookbook(pathResolved)

			// Add git cookbook
			gitVersion, err := berkshelf.NewVersion("1.0.0")
			Expect(err).NotTo(HaveOccurred())
			gitCookbook := &berkshelf.Cookbook{
				Name:         "git-cookbook",
				Version:      gitVersion,
				Dependencies: make(map[string]*berkshelf.Constraint),
			}
			gitResolved := &resolver.ResolvedCookbook{
				Name:    "git-cookbook",
				Version: gitVersion,
				Cookbook: gitCookbook,
				Source: &berkshelf.SourceLocation{
					Type: "git",
					URL:  "https://github.com/example/cookbook.git",
				},
			}
			resolution.AddCookbook(gitResolved)

			// Add supermarket cookbook
			supermarketVersion, err := berkshelf.NewVersion("2.0.0")
			Expect(err).NotTo(HaveOccurred())
			supermarketCookbook := &berkshelf.Cookbook{
				Name:         "nginx",
				Version:      supermarketVersion,
				Dependencies: make(map[string]*berkshelf.Constraint),
			}
			supermarketResolved := &resolver.ResolvedCookbook{
				Name:    "nginx",
				Version: supermarketVersion,
				Cookbook: supermarketCookbook,
				Source: &berkshelf.SourceLocation{
					Type: "supermarket",
					URL:  source.PUBLIC_SUPERMARKET,
				},
			}
			resolution.AddCookbook(supermarketResolved)

			lf, err := manager.Generate(resolution)
			Expect(err).NotTo(HaveOccurred())

			Expect(lf.Sources).To(HaveLen(3))
			Expect(lf.Sources).To(HaveKey("path"))
			Expect(lf.Sources).To(HaveKey("https://github.com/example/cookbook.git"))
			Expect(lf.Sources).To(HaveKey(source.PUBLIC_SUPERMARKET))

			Expect(lf.HasCookbook("test")).To(BeTrue())
			Expect(lf.HasCookbook("git-cookbook")).To(BeTrue())
			Expect(lf.HasCookbook("nginx")).To(BeTrue())
		})
	})
})


var _ = Describe("ExtractDirectDependencies with constraints", func() {
	var (
		tmpDir        string
		berksfilePath string
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "lockfile-constraint-test")
		Expect(err).NotTo(HaveOccurred())
		berksfilePath = filepath.Join(tmpDir, "Berksfile")
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	// Validates: Requirements 1.1, 1.2
	It("should include constraint annotations for constrained cookbooks", func() {
		berksfileContent := `source 'https://supermarket.chef.io'

cookbook 'apt', '~> 7.5'
cookbook 'nginx', '>= 12.0'
`
		err := os.WriteFile(berksfilePath, []byte(berksfileContent), 0644)
		Expect(err).NotTo(HaveOccurred())

		deps, err := lockfile.ExtractDirectDependencies(berksfilePath, nil)
		Expect(err).NotTo(HaveOccurred())

		// The bug condition: constrained cookbooks should include their constraint
		// in the dependency string, e.g. "apt (~> 7.5)" not just "apt"
		Expect(deps).To(ContainElement("apt (~> 7.5)"))
		Expect(deps).To(ContainElement("nginx (>= 12.0.0)"))
	})

	It("should include exact version constraint annotation", func() {
		berksfileContent := `source 'https://supermarket.chef.io'

cookbook 'mysql', '= 5.0.0'
`
		err := os.WriteFile(berksfilePath, []byte(berksfileContent), 0644)
		Expect(err).NotTo(HaveOccurred())

		deps, err := lockfile.ExtractDirectDependencies(berksfilePath, nil)
		Expect(err).NotTo(HaveOccurred())

		Expect(deps).To(ContainElement("mysql (= 5.0.0)"))
	})
})

var _ = Describe("ExtractDirectDependencies preservation for unconstrained cookbooks", func() {
	// Validates: Requirements 3.1, 3.2, 3.3
	var (
		tmpDir        string
		berksfilePath string
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "lockfile-preservation-test")
		Expect(err).NotTo(HaveOccurred())
		berksfilePath = filepath.Join(tmpDir, "Berksfile")
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	It("should produce just the cookbook name for unconstrained cookbooks (no version specified)", func() {
		berksfileContent := `source 'https://supermarket.chef.io'

cookbook 'redis'
cookbook 'apt'
`
		err := os.WriteFile(berksfilePath, []byte(berksfileContent), 0644)
		Expect(err).NotTo(HaveOccurred())

		deps, err := lockfile.ExtractDirectDependencies(berksfilePath, nil)
		Expect(err).NotTo(HaveOccurred())

		// Unconstrained cookbooks should appear as just the name
		Expect(deps).To(ConsistOf("apt", "redis"))
		// Each entry should be exactly the cookbook name with no parentheses
		for _, dep := range deps {
			Expect(dep).NotTo(ContainSubstring("("))
			Expect(dep).NotTo(ContainSubstring(")"))
		}
	})

	It("should produce alphabetically sorted output for multiple unconstrained cookbooks", func() {
		berksfileContent := `source 'https://supermarket.chef.io'

cookbook 'zookeeper'
cookbook 'mysql'
cookbook 'apache'
cookbook 'nginx'
`
		err := os.WriteFile(berksfilePath, []byte(berksfileContent), 0644)
		Expect(err).NotTo(HaveOccurred())

		deps, err := lockfile.ExtractDirectDependencies(berksfilePath, nil)
		Expect(err).NotTo(HaveOccurred())

		Expect(deps).To(HaveLen(4))
		Expect(sort.StringsAreSorted(deps)).To(BeTrue(), "dependencies should be alphabetically sorted")
		Expect(deps).To(Equal([]string{"apache", "mysql", "nginx", "zookeeper"}))
	})

	It("should preserve unconstrained cookbooks as just the name in a mix of constrained and unconstrained", func() {
		berksfileContent := `source 'https://supermarket.chef.io'

cookbook 'apt', '~> 7.5'
cookbook 'redis'
cookbook 'nginx', '>= 12.0'
cookbook 'mysql'
`
		err := os.WriteFile(berksfilePath, []byte(berksfileContent), 0644)
		Expect(err).NotTo(HaveOccurred())

		deps, err := lockfile.ExtractDirectDependencies(berksfilePath, nil)
		Expect(err).NotTo(HaveOccurred())

		// On unfixed code, ALL cookbooks appear as just the name (the bug drops constraints).
		// The unconstrained ones (redis, mysql) should be just the name on both unfixed and fixed code.
		Expect(deps).To(ContainElement("redis"))
		Expect(deps).To(ContainElement("mysql"))

		// Verify unconstrained entries have no parentheses
		for _, dep := range deps {
			if dep == "redis" || dep == "mysql" {
				Expect(dep).NotTo(ContainSubstring("("))
			}
		}
	})

	It("should produce just the cookbook name for a path-sourced cookbook with no constraint", func() {
		berksfileContent := `source 'https://supermarket.chef.io'

cookbook 'test', path: 'test/fixtures/cookbook'
`
		err := os.WriteFile(berksfilePath, []byte(berksfileContent), 0644)
		Expect(err).NotTo(HaveOccurred())

		deps, err := lockfile.ExtractDirectDependencies(berksfilePath, nil)
		Expect(err).NotTo(HaveOccurred())

		Expect(deps).To(ConsistOf("test"))
		Expect(deps[0]).To(Equal("test"))
	})
})
