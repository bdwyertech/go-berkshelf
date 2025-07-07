package config

import (
	"os"
	"testing"
)

func TestLoadFromEnvironment(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected *Config
	}{
		{
			name:     "no environment variables set",
			envVars:  map[string]string{},
			expected: nil, // Should return nil when no env vars are set
		},
		{
			name: "cache path override",
			envVars: map[string]string{
				"BERKSHELF_CACHE_PATH": "/custom/cache/path",
			},
			expected: &Config{
				CachePath: StringPtr("/custom/cache/path"),
			},
		},
		{
			name: "ssl verify false",
			envVars: map[string]string{
				"BERKSHELF_SSL_VERIFY": "false",
			},
			expected: &Config{
				SSLVerify: BoolPtr(false),
			},
		},
		{
			name: "ssl verify true variations",
			envVars: map[string]string{
				"BERKSHELF_SSL_VERIFY": "1",
			},
			expected: &Config{
				SSLVerify: BoolPtr(true),
			},
		},
		{
			name: "api timeout",
			envVars: map[string]string{
				"BERKSHELF_API_TIMEOUT": "60",
			},
			expected: &Config{
				APITimeout: IntPtr(60),
			},
		},
		{
			name: "retry configuration",
			envVars: map[string]string{
				"BERKSHELF_RETRY_COUNT": "5",
				"BERKSHELF_RETRY_DELAY": "2",
			},
			expected: &Config{
				RetryCount: IntPtr(5),
				RetryDelay: IntPtr(2),
			},
		},
		{
			name: "concurrency setting",
			envVars: map[string]string{
				"BERKSHELF_CONCURRENCY": "10",
			},
			expected: &Config{
				Concurrency: IntPtr(10),
			},
		},
		{
			name: "proxy configuration",
			envVars: map[string]string{
				"BERKSHELF_PROXY": "http://proxy.example.com:8080",
			},
			expected: &Config{
				Proxy: StringPtr("http://proxy.example.com:8080"),
			},
		},
		{
			name: "no proxy list",
			envVars: map[string]string{
				"BERKSHELF_NO_PROXY": "localhost,127.0.0.1,*.internal.com",
			},
			expected: &Config{
				NoProxy: []string{"localhost", "127.0.0.1", "*.internal.com"},
			},
		},
		{
			name: "complete configuration",
			envVars: map[string]string{
				"BERKSHELF_CACHE_PATH":   "/opt/berkshelf/cache",
				"BERKSHELF_SSL_VERIFY":   "true",
				"BERKSHELF_API_TIMEOUT":  "45",
				"BERKSHELF_RETRY_COUNT":  "3",
				"BERKSHELF_RETRY_DELAY":  "1",
				"BERKSHELF_CONCURRENCY": "8",
				"BERKSHELF_PROXY":       "http://corporate-proxy:3128",
				"BERKSHELF_NO_PROXY":    "localhost,*.internal.com",
			},
			expected: &Config{
				CachePath:   StringPtr("/opt/berkshelf/cache"),
				SSLVerify:   BoolPtr(true),
				APITimeout:  IntPtr(45),
				RetryCount:  IntPtr(3),
				RetryDelay:  IntPtr(1),
				Concurrency: IntPtr(8),
				Proxy:       StringPtr("http://corporate-proxy:3128"),
				NoProxy:     []string{"localhost", "*.internal.com"},
			},
		},
		{
			name: "chef configuration",
			envVars: map[string]string{
				"CHEF_NODE_NAME":   "test-node",
				"CHEF_CLIENT_KEY":  "/path/to/client.pem",
				"CHEF_SERVER_URL":  "https://chef.example.com",
				"CHEF_ORGANIZATION": "test-org",
				"CHEF_ENVIRONMENT":  "production",
			},
			expected: &Config{
				ChefConfig: &ChefConfig{
					NodeName:      StringPtr("test-node"),
					ClientKey:     StringPtr("/path/to/client.pem"),
					ChefServerURL: StringPtr("https://chef.example.com"),
					Organization:  StringPtr("test-org"),
					Environment:   StringPtr("production"),
				},
			},
		},
		{
			name: "invalid values ignored",
			envVars: map[string]string{
				"BERKSHELF_API_TIMEOUT":  "invalid",
				"BERKSHELF_RETRY_COUNT":  "-1", // negative values should be ignored for retry count
				"BERKSHELF_CONCURRENCY": "0",  // zero/negative values should be ignored for concurrency
				"BERKSHELF_SSL_VERIFY":   "maybe",
			},
			expected: nil, // Should return nil when no valid env vars are set
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			clearEnv()

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// Load from environment
			result := loadFromEnvironment()

			// Compare results
			if !configsEqual(result, tt.expected) {
				t.Errorf("loadFromEnvironment() = %+v, want %+v", result, tt.expected)
			}

			// Clean up
			clearEnv()
		})
	}
}

// clearEnv clears all berkshelf-related environment variables
func clearEnv() {
	envVars := []string{
		"BERKSHELF_CACHE_PATH",
		"BERKSHELF_DEFAULT_SOURCES",
		"BERKSHELF_SSL_VERIFY",
		"BERKSHELF_PROXY",
		"BERKSHELF_NO_PROXY",
		"BERKSHELF_API_TIMEOUT",
		"BERKSHELF_RETRY_COUNT",
		"BERKSHELF_RETRY_DELAY",
		"BERKSHELF_CONCURRENCY",
		"CHEF_NODE_NAME",
		"CHEF_CLIENT_KEY",
		"CHEF_SERVER_URL",
		"CHEF_ORGANIZATION",
		"CHEF_ENVIRONMENT",
	}

	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}
}
