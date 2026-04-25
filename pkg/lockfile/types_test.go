package lockfile_test

import (
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bdwyertech/go-berkshelf/pkg/berkshelf"
	"github.com/bdwyertech/go-berkshelf/pkg/lockfile"
)

var _ = Describe("LockFile Types", func() {
	Describe("NewLockFile", func() {
		It("should create lock file with defaults", func() {
			lf := lockfile.NewLockFile()
			Expect(lf.Revision).To(Equal(7))
			Expect(lf.Sources).NotTo(BeNil())
			Expect(lf.Sources).To(BeEmpty())
		})
	})

	Describe("AddCookbook", func() {
		It("should add cookbook with dependencies", func() {
			lf := lockfile.NewLockFile()

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
				URL:  "https://supermarket.chef.io",
			}

			sourceURL := "https://supermarket.chef.io"
			lf.AddCookbook(sourceURL, cookbook, sourceInfo)

			Expect(lf.Sources).To(HaveLen(1))

			src, exists := lf.Sources[sourceURL]
			Expect(exists).To(BeTrue())
			Expect(src.Type).To(Equal("supermarket"))

			Expect(src.Cookbooks).To(HaveLen(1))

			cookbookLock, exists := src.Cookbooks["nginx"]
			Expect(exists).To(BeTrue())
			Expect(cookbookLock.Version).To(Equal("1.2.3"))

			Expect(cookbookLock.Dependencies).To(HaveLen(1))
			aptConstraint, exists := cookbookLock.Dependencies["apt"]
			Expect(exists).To(BeTrue())
			Expect(aptConstraint).To(Equal("~> 1.0"))
		})
	})

	Describe("GetCookbook", func() {
		var lf *lockfile.LockFile

		BeforeEach(func() {
			lf = lockfile.NewLockFile()
			version, _ := berkshelf.NewVersion("1.2.3")
			cookbook := &berkshelf.Cookbook{
				Name:         "nginx",
				Version:      version,
				Dependencies: make(map[string]*berkshelf.Constraint),
			}
			sourceInfo := &lockfile.SourceInfo{Type: "supermarket"}
			lf.AddCookbook("https://supermarket.chef.io", cookbook, sourceInfo)
		})

		It("should return existing cookbook", func() {
			cookbookLock, src, exists := lf.GetCookbook("nginx")
			Expect(exists).To(BeTrue())
			Expect(cookbookLock.Version).To(Equal("1.2.3"))
			Expect(src).To(Equal("https://supermarket.chef.io"))
		})

		It("should return false for non-existing cookbook", func() {
			_, _, exists := lf.GetCookbook("nonexistent")
			Expect(exists).To(BeFalse())
		})
	})

	Describe("HasCookbook", func() {
		var lf *lockfile.LockFile

		BeforeEach(func() {
			lf = lockfile.NewLockFile()
			version, _ := berkshelf.NewVersion("1.2.3")
			cookbook := &berkshelf.Cookbook{
				Name:         "nginx",
				Version:      version,
				Dependencies: make(map[string]*berkshelf.Constraint),
			}
			sourceInfo := &lockfile.SourceInfo{Type: "supermarket"}
			lf.AddCookbook("https://supermarket.chef.io", cookbook, sourceInfo)
		})

		It("should return true for existing cookbook", func() {
			Expect(lf.HasCookbook("nginx")).To(BeTrue())
		})

		It("should return false for non-existing cookbook", func() {
			Expect(lf.HasCookbook("nonexistent")).To(BeFalse())
		})
	})

	Describe("ListCookbooks", func() {
		It("should list cookbooks from multiple sources", func() {
			lf := lockfile.NewLockFile()

			version1, _ := berkshelf.NewVersion("1.2.3")
			cookbook1 := &berkshelf.Cookbook{
				Name:         "nginx",
				Version:      version1,
				Dependencies: make(map[string]*berkshelf.Constraint),
			}
			sourceInfo1 := &lockfile.SourceInfo{Type: "supermarket"}
			lf.AddCookbook("https://supermarket.chef.io", cookbook1, sourceInfo1)

			version2, _ := berkshelf.NewVersion("2.0.0")
			cookbook2 := &berkshelf.Cookbook{
				Name:         "apache",
				Version:      version2,
				Dependencies: make(map[string]*berkshelf.Constraint),
			}
			sourceInfo2 := &lockfile.SourceInfo{Type: "git"}
			lf.AddCookbook("https://github.com/example/apache", cookbook2, sourceInfo2)

			cookbooks := lf.ListCookbooks()
			Expect(cookbooks).To(HaveLen(2))
			Expect(cookbooks).To(HaveKey("nginx"))
			Expect(cookbooks).To(HaveKey("apache"))
		})
	})

	Describe("ToJSON", func() {
		It("should serialize to valid JSON", func() {
			lf := lockfile.NewLockFile()
			lf.GeneratedAt = time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

			version, _ := berkshelf.NewVersion("1.2.3")
			constraint, _ := berkshelf.NewConstraint("~> 1.0")
			cookbook := &berkshelf.Cookbook{
				Name:    "nginx",
				Version: version,
				Dependencies: map[string]*berkshelf.Constraint{
					"apt": constraint,
				},
			}

			sourceInfo := &lockfile.SourceInfo{
				Type: "supermarket",
				URL:  "https://supermarket.chef.io",
			}

			lf.AddCookbook("https://supermarket.chef.io", cookbook, sourceInfo)

			data, err := lf.ToJSON()
			Expect(err).NotTo(HaveOccurred())

			var parsed map[string]interface{}
			err = json.Unmarshal(data, &parsed)
			Expect(err).NotTo(HaveOccurred())

			revision, ok := parsed["revision"].(float64)
			Expect(ok).To(BeTrue())
			Expect(revision).To(Equal(float64(7)))
		})
	})

	Describe("FromJSON", func() {
		It("should deserialize from JSON", func() {
			jsonData := `{
				"revision": 7,
				"generated_at": "2023-01-01T12:00:00Z",
				"sources": {
					"https://supermarket.chef.io": {
						"type": "supermarket",
						"url": "https://supermarket.chef.io",
						"cookbooks": {
							"nginx": {
								"version": "1.2.3",
								"dependencies": {
									"apt": "~> 1.0"
								},
								"source": {
									"type": "supermarket",
									"url": "https://supermarket.chef.io"
								}
							}
						}
					}
				}
			}`

			lf, err := lockfile.FromJSON([]byte(jsonData))
			Expect(err).NotTo(HaveOccurred())

			Expect(lf.Revision).To(Equal(7))
			Expect(lf.HasCookbook("nginx")).To(BeTrue())

			cookbook, _, exists := lf.GetCookbook("nginx")
			Expect(exists).To(BeTrue())
			Expect(cookbook.Version).To(Equal("1.2.3"))
			Expect(cookbook.Dependencies).To(HaveLen(1))
		})
	})

	Describe("IsOutdated", func() {
		It("should be outdated when older than max age", func() {
			lf := lockfile.NewLockFile()
			lf.GeneratedAt = time.Now().Add(-2 * time.Hour)
			Expect(lf.IsOutdated(1 * time.Hour)).To(BeTrue())
		})

		It("should not be outdated when newer than max age", func() {
			lf := lockfile.NewLockFile()
			lf.GeneratedAt = time.Now().Add(-2 * time.Hour)
			Expect(lf.IsOutdated(3 * time.Hour)).To(BeFalse())
		})
	})

	Describe("UpdateGeneratedAt", func() {
		It("should update generated timestamp", func() {
			lf := lockfile.NewLockFile()
			oldTime := lf.GeneratedAt

			time.Sleep(1 * time.Millisecond)
			lf.UpdateGeneratedAt()

			Expect(lf.GeneratedAt.After(oldTime)).To(BeTrue())
		})
	})
})
