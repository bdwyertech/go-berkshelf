package types

import (
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *Version
		wantErr bool
	}{
		{
			name:  "simple version",
			input: "1.2.3",
			want: &Version{
				Major: 1,
				Minor: 2,
				Patch: 3,
			},
			wantErr: false,
		},
		{
			name:  "version with prerelease",
			input: "1.2.3-alpha.1",
			want: &Version{
				Major:      1,
				Minor:      2,
				Patch:      3,
				Prerelease: "alpha.1",
			},
			wantErr: false,
		},
		{
			name:  "version with build",
			input: "1.2.3+build.123",
			want: &Version{
				Major: 1,
				Minor: 2,
				Patch: 3,
				Build: "build.123",
			},
			wantErr: false,
		},
		{
			name:  "version with prerelease and build",
			input: "1.2.3-beta.2+build.456",
			want: &Version{
				Major:      1,
				Minor:      2,
				Patch:      3,
				Prerelease: "beta.2",
				Build:      "build.456",
			},
			wantErr: false,
		},
		{
			name:    "invalid version",
			input:   "invalid",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "missing patch",
			input:   "1.2",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !versionsEqual(got, tt.want) {
				t.Errorf("ParseVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVersionCompare(t *testing.T) {
	tests := []struct {
		name string
		v1   *Version
		v2   *Version
		want int
	}{
		{
			name: "equal versions",
			v1:   &Version{Major: 1, Minor: 2, Patch: 3},
			v2:   &Version{Major: 1, Minor: 2, Patch: 3},
			want: 0,
		},
		{
			name: "major version difference",
			v1:   &Version{Major: 1, Minor: 2, Patch: 3},
			v2:   &Version{Major: 2, Minor: 2, Patch: 3},
			want: -1,
		},
		{
			name: "minor version difference",
			v1:   &Version{Major: 1, Minor: 3, Patch: 3},
			v2:   &Version{Major: 1, Minor: 2, Patch: 3},
			want: 1,
		},
		{
			name: "patch version difference",
			v1:   &Version{Major: 1, Minor: 2, Patch: 4},
			v2:   &Version{Major: 1, Minor: 2, Patch: 3},
			want: 1,
		},
		{
			name: "prerelease vs release",
			v1:   &Version{Major: 1, Minor: 2, Patch: 3, Prerelease: "alpha"},
			v2:   &Version{Major: 1, Minor: 2, Patch: 3},
			want: -1,
		},
		{
			name: "release vs prerelease",
			v1:   &Version{Major: 1, Minor: 2, Patch: 3},
			v2:   &Version{Major: 1, Minor: 2, Patch: 3, Prerelease: "alpha"},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.v1.Compare(*tt.v2)
			if got != tt.want {
				t.Errorf("Version.Compare() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVersionString(t *testing.T) {
	tests := []struct {
		name string
		v    *Version
		want string
	}{
		{
			name: "simple version",
			v:    &Version{Major: 1, Minor: 2, Patch: 3},
			want: "1.2.3",
		},
		{
			name: "version with prerelease",
			v:    &Version{Major: 1, Minor: 2, Patch: 3, Prerelease: "alpha.1"},
			want: "1.2.3-alpha.1",
		},
		{
			name: "version with build",
			v:    &Version{Major: 1, Minor: 2, Patch: 3, Build: "build.123"},
			want: "1.2.3+build.123",
		},
		{
			name: "version with prerelease and build",
			v:    &Version{Major: 1, Minor: 2, Patch: 3, Prerelease: "beta.2", Build: "build.456"},
			want: "1.2.3-beta.2+build.456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.v.String()
			if got != tt.want {
				t.Errorf("Version.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function to compare versions
func versionsEqual(v1, v2 *Version) bool {
	if v1 == nil && v2 == nil {
		return true
	}
	if v1 == nil || v2 == nil {
		return false
	}
	return v1.Major == v2.Major &&
		v1.Minor == v2.Minor &&
		v1.Patch == v2.Patch &&
		v1.Prerelease == v2.Prerelease &&
		v1.Build == v2.Build
}
