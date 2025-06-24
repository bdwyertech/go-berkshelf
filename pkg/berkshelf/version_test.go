package berkshelf

import (
	"testing"
)

func TestNewVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
		wantErr bool
	}{
		{
			name:    "simple version",
			version: "1.0.0",
			want:    "1.0.0",
			wantErr: false,
		},
		{
			name:    "version with v prefix",
			version: "v2.1.3",
			want:    "2.1.3",
			wantErr: false,
		},
		{
			name:    "version with prerelease",
			version: "1.0.0-alpha.1",
			want:    "1.0.0-alpha.1",
			wantErr: false,
		},
		{
			name:    "version with build metadata",
			version: "1.0.0+20130313144700",
			want:    "1.0.0+20130313144700",
			wantErr: false,
		},
		{
			name:    "invalid version",
			version: "not.a.version",
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty version",
			version: "",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewVersion(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.String() != tt.want {
				t.Errorf("NewVersion() = %v, want %v", got.String(), tt.want)
			}
		})
	}
}

func TestVersionCompare(t *testing.T) {
	tests := []struct {
		name string
		v1   string
		v2   string
		want int
	}{
		{
			name: "equal versions",
			v1:   "1.0.0",
			v2:   "1.0.0",
			want: 0,
		},
		{
			name: "v1 less than v2",
			v1:   "1.0.0",
			v2:   "2.0.0",
			want: -1,
		},
		{
			name: "v1 greater than v2",
			v1:   "2.0.0",
			v2:   "1.0.0",
			want: 1,
		},
		{
			name: "patch version difference",
			v1:   "1.0.1",
			v2:   "1.0.2",
			want: -1,
		},
		{
			name: "prerelease vs release",
			v1:   "1.0.0-alpha",
			v2:   "1.0.0",
			want: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v1, err := NewVersion(tt.v1)
			if err != nil {
				t.Fatalf("NewVersion(%s) failed: %v", tt.v1, err)
			}
			v2, err := NewVersion(tt.v2)
			if err != nil {
				t.Fatalf("NewVersion(%s) failed: %v", tt.v2, err)
			}

			got := v1.Compare(v2)
			if got != tt.want {
				t.Errorf("Compare() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVersionHelpers(t *testing.T) {
	v1 := MustVersion("1.0.0")
	v2 := MustVersion("2.0.0")

	if !v1.LessThan(v2) {
		t.Error("1.0.0 should be less than 2.0.0")
	}

	if !v2.GreaterThan(v1) {
		t.Error("2.0.0 should be greater than 1.0.0")
	}

	if !v1.Equal(MustVersion("1.0.0")) {
		t.Error("1.0.0 should equal 1.0.0")
	}
}

func TestMustVersionPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustVersion should panic on invalid version")
		}
	}()

	MustVersion("invalid.version")
}
