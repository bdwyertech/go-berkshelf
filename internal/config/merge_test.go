package config

import (
	"reflect"
	"testing"
)

func TestMergeConfigs(t *testing.T) {
	tests := []struct {
		name     string
		base     *Config
		overlay  *Config
		expected *Config
	}{
		{
			name:     "nil base and overlay",
			base:     nil,
			overlay:  nil,
			expected: DefaultConfig(),
		},
		{
			name:     "nil base with overlay",
			base:     nil,
			overlay:  &Config{CachePath: StringPtr("/overlay/path")},
			expected: &Config{CachePath: StringPtr("/overlay/path")},
		},
		{
			name:     "base with nil overlay",
			base:     &Config{CachePath: StringPtr("/base/path"), SSLVerify: BoolPtr(true)},
			overlay:  nil,
			expected: &Config{CachePath: StringPtr("/base/path"), SSLVerify: BoolPtr(true)},
		},
		{
			name: "overlay cache path",
			base: &Config{
				CachePath:  StringPtr("/base/path"),
				SSLVerify:  BoolPtr(true),
				APITimeout: IntPtr(30),
			},
			overlay: &Config{
				CachePath: StringPtr("/overlay/path"),
			},
			expected: &Config{
				CachePath:  StringPtr("/overlay/path"),
				SSLVerify:  BoolPtr(true), // preserved from base
				APITimeout: IntPtr(30),    // preserved from base
			},
		},
		{
			name: "overlay boolean values",
			base: &Config{
				CachePath: StringPtr("/base/path"),
				SSLVerify: BoolPtr(true),
			},
			overlay: &Config{
				SSLVerify: BoolPtr(false), // explicitly set to false
			},
			expected: &Config{
				CachePath: StringPtr("/base/path"),
				SSLVerify: BoolPtr(false),
			},
		},
		{
			name: "overlay integer values",
			base: &Config{
				APITimeout:  IntPtr(30),
				RetryCount:  IntPtr(3),
				RetryDelay:  IntPtr(1),
				Concurrency: IntPtr(5),
			},
			overlay: &Config{
				APITimeout:  IntPtr(60),
				RetryCount:  IntPtr(0), // explicitly set to 0
				Concurrency: IntPtr(10),
				// RetryDelay not set (nil) - should preserve base value
			},
			expected: &Config{
				APITimeout:  IntPtr(60),
				RetryCount:  IntPtr(0),
				RetryDelay:  IntPtr(1), // preserved from base
				Concurrency: IntPtr(10),
			},
		},
		{
			name: "overlay string slices",
			base: &Config{
				DefaultSources: []string{"source1", "source2"},
				NoProxy:        []string{"localhost"},
			},
			overlay: &Config{
				DefaultSources: []string{"newsource1", "newsource2"},
			},
			expected: &Config{
				DefaultSources: []string{"newsource1", "newsource2"},
				NoProxy:        []string{"localhost"}, // preserved from base
			},
		},
		{
			name: "overlay proxy settings",
			base: &Config{
				Proxy:   StringPtr("http://old-proxy:8080"),
				NoProxy: []string{"localhost"},
			},
			overlay: &Config{
				Proxy:   StringPtr("http://new-proxy:3128"),
				NoProxy: []string{"localhost", "*.internal.com"},
			},
			expected: &Config{
				Proxy:   StringPtr("http://new-proxy:3128"),
				NoProxy: []string{"localhost", "*.internal.com"},
			},
		},
		{
			name: "overlay chef config",
			base: &Config{
				ChefConfig: &ChefConfig{
					NodeName:      StringPtr("base-node"),
					ClientKey:     StringPtr("/base/client.pem"),
					ChefServerURL: StringPtr("https://base-chef.example.com"),
				},
			},
			overlay: &Config{
				ChefConfig: &ChefConfig{
					NodeName:  StringPtr("overlay-node"),
					ClientKey: StringPtr("/overlay/client.pem"),
					// ChefServerURL not set - should preserve base value
				},
			},
			expected: &Config{
				ChefConfig: &ChefConfig{
					NodeName:      StringPtr("overlay-node"),
					ClientKey:     StringPtr("/overlay/client.pem"),
					ChefServerURL: StringPtr("https://base-chef.example.com"), // preserved from base
				},
			},
		},
		{
			name: "overlay chef config on nil base",
			base: &Config{
				ChefConfig: nil,
			},
			overlay: &Config{
				ChefConfig: &ChefConfig{
					NodeName:      StringPtr("new-node"),
					ClientKey:     StringPtr("/new/client.pem"),
					ChefServerURL: StringPtr("https://new-chef.example.com"),
				},
			},
			expected: &Config{
				ChefConfig: &ChefConfig{
					NodeName:      StringPtr("new-node"),
					ClientKey:     StringPtr("/new/client.pem"),
					ChefServerURL: StringPtr("https://new-chef.example.com"),
				},
			},
		},
		{
			name: "complete merge scenario",
			base: &Config{
				CachePath:      StringPtr("/base/cache"),
				DefaultSources: []string{"base-source"},
				SSLVerify:      BoolPtr(true),
				Proxy:          StringPtr("http://base-proxy:8080"),
				NoProxy:        []string{"localhost"},
				ChefConfig: &ChefConfig{
					NodeName:      StringPtr("base-node"),
					ClientKey:     StringPtr("/base/client.pem"),
					ChefServerURL: StringPtr("https://base-chef.example.com"),
					Organization:  StringPtr("base-org"),
				},
				APITimeout:  IntPtr(30),
				RetryCount:  IntPtr(3),
				RetryDelay:  IntPtr(1),
				Concurrency: IntPtr(5),
			},
			overlay: &Config{
				CachePath:   StringPtr("/overlay/cache"),
				SSLVerify:   BoolPtr(false),
				APITimeout:  IntPtr(60),
				RetryCount:  IntPtr(0),
				Concurrency: IntPtr(10),
				ChefConfig: &ChefConfig{
					NodeName:     StringPtr("overlay-node"),
					Organization: StringPtr("overlay-org"),
					Environment:  StringPtr("production"),
				},
				// RetryDelay not set - should preserve base value
			},
			expected: &Config{
				CachePath:      StringPtr("/overlay/cache"),
				DefaultSources: []string{"base-source"}, // preserved from base
				SSLVerify:      BoolPtr(false),
				Proxy:          StringPtr("http://base-proxy:8080"), // preserved from base
				NoProxy:        []string{"localhost"},               // preserved from base
				ChefConfig: &ChefConfig{
					NodeName:      StringPtr("overlay-node"),
					ClientKey:     StringPtr("/base/client.pem"),                // preserved from base
					ChefServerURL: StringPtr("https://base-chef.example.com"),   // preserved from base
					Organization:  StringPtr("overlay-org"),
					Environment:   StringPtr("production"),
				},
				APITimeout:  IntPtr(60),
				RetryCount:  IntPtr(0),
				RetryDelay:  IntPtr(1), // preserved from base
				Concurrency: IntPtr(10),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeConfigs(tt.base, tt.overlay)

			if !configsEqual(result, tt.expected) {
				t.Errorf("MergeConfigs() mismatch:\ngot:  %+v\nwant: %+v", result, tt.expected)
			}
		})
	}
}

// configsEqual compares two Config structs for equality
// Updated to handle pointer fields properly
func configsEqual(a, b *Config) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Compare pointer fields
	if !stringPtrEqual(a.CachePath, b.CachePath) ||
		!boolPtrEqual(a.SSLVerify, b.SSLVerify) ||
		!stringPtrEqual(a.Proxy, b.Proxy) ||
		!intPtrEqual(a.APITimeout, b.APITimeout) ||
		!intPtrEqual(a.RetryCount, b.RetryCount) ||
		!intPtrEqual(a.RetryDelay, b.RetryDelay) ||
		!intPtrEqual(a.Concurrency, b.Concurrency) {
		return false
	}

	// Compare slices
	if !reflect.DeepEqual(a.DefaultSources, b.DefaultSources) ||
		!reflect.DeepEqual(a.NoProxy, b.NoProxy) {
		return false
	}

	// Compare ChefConfig
	if !chefConfigEqual(a.ChefConfig, b.ChefConfig) {
		return false
	}

	return true
}

// Helper functions for pointer comparison
func stringPtrEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func boolPtrEqual(a, b *bool) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func intPtrEqual(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func chefConfigEqual(a, b *ChefConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	return stringPtrEqual(a.NodeName, b.NodeName) &&
		stringPtrEqual(a.ClientKey, b.ClientKey) &&
		stringPtrEqual(a.ChefServerURL, b.ChefServerURL) &&
		stringPtrEqual(a.Organization, b.Organization) &&
		stringPtrEqual(a.Environment, b.Environment)
}
