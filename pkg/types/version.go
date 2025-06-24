package types

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Version represents a semantic version
type Version struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
	Build      string
}

// String returns the string representation of the version
func (v Version) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Prerelease != "" {
		s += "-" + v.Prerelease
	}
	if v.Build != "" {
		s += "+" + v.Build
	}
	return s
}

// Compare compares two versions
// Returns:
//
//	-1 if v < other
//	 0 if v == other
//	 1 if v > other
func (v Version) Compare(other Version) int {
	if v.Major != other.Major {
		if v.Major < other.Major {
			return -1
		}
		return 1
	}

	if v.Minor != other.Minor {
		if v.Minor < other.Minor {
			return -1
		}
		return 1
	}

	if v.Patch != other.Patch {
		if v.Patch < other.Patch {
			return -1
		}
		return 1
	}

	// Handle prerelease versions
	if v.Prerelease != "" && other.Prerelease == "" {
		return -1 // v is prerelease, other is not
	}
	if v.Prerelease == "" && other.Prerelease != "" {
		return 1 // v is not prerelease, other is
	}
	if v.Prerelease != "" && other.Prerelease != "" {
		return strings.Compare(v.Prerelease, other.Prerelease)
	}

	return 0
}

// ParseVersion parses a version string into a Version struct
func ParseVersion(s string) (*Version, error) {
	// Regex for semantic versioning
	re := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z\-\.]+))?(?:\+([0-9A-Za-z\-\.]+))?$`)
	matches := re.FindStringSubmatch(s)

	if matches == nil {
		return nil, fmt.Errorf("invalid version format: %s", s)
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])

	return &Version{
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		Prerelease: matches[4],
		Build:      matches[5],
	}, nil
}

// MustParseVersion parses a version string and panics on error
func MustParseVersion(s string) *Version {
	v, err := ParseVersion(s)
	if err != nil {
		panic(err)
	}
	return v
}
