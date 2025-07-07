package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"dario.cat/mergo"

	"github.com/bdwyer/go-berkshelf/pkg/source"
)

// Config represents berkshelf configuration with pointer fields for optional values
// and envconfig tags for automatic environment variable loading
type Config struct {
	CachePath      *string     `json:"cache_path,omitempty" env:"BERKSHELF_CACHE_PATH"`
	DefaultSources []string    `json:"default_sources,omitempty" env:"BERKSHELF_DEFAULT_SOURCES" env-separator:","`
	SSLVerify      *bool       `json:"ssl_verify,omitempty" env:"BERKSHELF_SSL_VERIFY"`
	Proxy          *string     `json:"proxy,omitempty" env:"BERKSHELF_PROXY"`
	NoProxy        []string    `json:"no_proxy,omitempty" env:"BERKSHELF_NO_PROXY" env-separator:","`
	ChefConfig     *ChefConfig `json:"chef,omitempty"`
	APITimeout     *int        `json:"api_timeout,omitempty" env:"BERKSHELF_API_TIMEOUT"`
	RetryCount     *int        `json:"retry_count,omitempty" env:"BERKSHELF_RETRY_COUNT"`
	RetryDelay     *int        `json:"retry_delay,omitempty" env:"BERKSHELF_RETRY_DELAY"`
	Concurrency    *int        `json:"concurrency,omitempty" env:"BERKSHELF_CONCURRENCY"`
}

// ChefConfig contains Chef-specific configuration with envconfig tags
type ChefConfig struct {
	NodeName      *string `json:"node_name,omitempty" env:"CHEF_NODE_NAME"`
	ClientKey     *string `json:"client_key,omitempty" env:"CHEF_CLIENT_KEY"`
	ChefServerURL *string `json:"chef_server_url,omitempty" env:"CHEF_SERVER_URL"`
	Organization  *string `json:"organization,omitempty" env:"CHEF_ORGANIZATION"`
	Environment   *string `json:"environment,omitempty" env:"CHEF_ENVIRONMENT"`
}

// Helper functions for creating pointers
func StringPtr(s string) *string { return &s }
func BoolPtr(b bool) *bool       { return &b }
func IntPtr(i int) *int          { return &i }

// =============================================================================
// GETTER METHODS WITH DEFAULTS
// =============================================================================

func (c *Config) GetCachePath() string {
	if c.CachePath != nil {
		return *c.CachePath
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".berkshelf", "cookbooks")
}

func (c *Config) GetDefaultSources() []string {
	if len(c.DefaultSources) > 0 {
		return c.DefaultSources
	}
	return []string{source.PUBLIC_SUPERMARKET}
}

func (c *Config) GetSSLVerify() bool {
	if c.SSLVerify != nil {
		return *c.SSLVerify
	}
	return true // default to secure
}

func (c *Config) GetProxy() string {
	if c.Proxy != nil {
		return *c.Proxy
	}
	return ""
}

func (c *Config) GetNoProxy() []string {
	return c.NoProxy // slices can be nil/empty naturally
}

func (c *Config) GetAPITimeout() int {
	if c.APITimeout != nil {
		return *c.APITimeout
	}
	return 30 // default 30 seconds
}

func (c *Config) GetRetryCount() int {
	if c.RetryCount != nil {
		return *c.RetryCount
	}
	return 3 // default 3 retries
}

func (c *Config) GetRetryDelay() int {
	if c.RetryDelay != nil {
		return *c.RetryDelay
	}
	return 1 // default 1 second
}

func (c *Config) GetConcurrency() int {
	if c.Concurrency != nil {
		return *c.Concurrency
	}
	return 5 // default 5 concurrent operations
}

// ChefConfig getter methods
func (c *ChefConfig) GetNodeName() string {
	if c != nil && c.NodeName != nil {
		return *c.NodeName
	}
	return ""
}

func (c *ChefConfig) GetClientKey() string {
	if c != nil && c.ClientKey != nil {
		return *c.ClientKey
	}
	return ""
}

func (c *ChefConfig) GetChefServerURL() string {
	if c != nil && c.ChefServerURL != nil {
		return *c.ChefServerURL
	}
	return ""
}

func (c *ChefConfig) GetOrganization() string {
	if c != nil && c.Organization != nil {
		return *c.Organization
	}
	return ""
}

func (c *ChefConfig) GetEnvironment() string {
	if c != nil && c.Environment != nil {
		return *c.Environment
	}
	return ""
}

// =============================================================================
// CONFIGURATION LOADING
// =============================================================================

// Load reads configuration from standard locations and environment variables
func Load() (*Config, error) {
	// Start with defaults
	config := DefaultConfig()

	// Try to load from file
	configPaths := getConfigPaths()
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			fileConfig, err := loadFromFile(path)
			if err != nil {
				return nil, fmt.Errorf("failed to load config from %s: %w", path, err)
			}
			// Merge file config over defaults
			config = MergeConfigs(config, fileConfig)
			break
		}
	}

	// Load environment variables and merge them over file/defaults
	envConfig := loadFromEnvironment()
	if envConfig != nil {
		config = MergeConfigs(config, envConfig)
	}

	return config, nil
}

// LoadFromFile loads configuration from a specific file
func LoadFromFile(path string) (*Config, error) {
	return loadFromFile(path)
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()

	return &Config{
		CachePath: StringPtr(filepath.Join(home, ".berkshelf", "cookbooks")),
		DefaultSources: []string{
			source.PUBLIC_SUPERMARKET,
		},
		SSLVerify:   BoolPtr(true),
		APITimeout:  IntPtr(30),
		RetryCount:  IntPtr(3),
		RetryDelay:  IntPtr(1),
		Concurrency: IntPtr(5),
	}
}

// =============================================================================
// ENVIRONMENT VARIABLE LOADING
// =============================================================================

// loadFromEnvironment loads configuration from environment variables
// Returns nil if no environment variables are set
// Note: struct tags are ready for go-envconfig integration in the future
func loadFromEnvironment() *Config {
	config := &Config{}
	hasValues := false

	// BERKSHELF_CACHE_PATH
	if val := os.Getenv("BERKSHELF_CACHE_PATH"); val != "" {
		config.CachePath = StringPtr(val)
		hasValues = true
	}

	// BERKSHELF_DEFAULT_SOURCES (comma-separated)
	if val := os.Getenv("BERKSHELF_DEFAULT_SOURCES"); val != "" {
		sources := strings.Split(val, ",")
		var cleanSources []string
		for _, source := range sources {
			if trimmed := strings.TrimSpace(source); trimmed != "" {
				cleanSources = append(cleanSources, trimmed)
			}
		}
		if len(cleanSources) > 0 {
			config.DefaultSources = cleanSources
			hasValues = true
		}
	}

	// BERKSHELF_SSL_VERIFY
	if val := os.Getenv("BERKSHELF_SSL_VERIFY"); val != "" {
		if parsed, err := strconv.ParseBool(val); err == nil {
			config.SSLVerify = BoolPtr(parsed)
			hasValues = true
		}
	}

	// BERKSHELF_PROXY
	if val := os.Getenv("BERKSHELF_PROXY"); val != "" {
		config.Proxy = StringPtr(val)
		hasValues = true
	}

	// BERKSHELF_NO_PROXY (comma-separated)
	if val := os.Getenv("BERKSHELF_NO_PROXY"); val != "" {
		noProxy := strings.Split(val, ",")
		var cleanNoProxy []string
		for _, entry := range noProxy {
			if trimmed := strings.TrimSpace(entry); trimmed != "" {
				cleanNoProxy = append(cleanNoProxy, trimmed)
			}
		}
		if len(cleanNoProxy) > 0 {
			config.NoProxy = cleanNoProxy
			hasValues = true
		}
	}

	// BERKSHELF_API_TIMEOUT
	if val := os.Getenv("BERKSHELF_API_TIMEOUT"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			config.APITimeout = IntPtr(parsed)
			hasValues = true
		}
	}

	// BERKSHELF_RETRY_COUNT
	if val := os.Getenv("BERKSHELF_RETRY_COUNT"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed >= 0 {
			config.RetryCount = IntPtr(parsed)
			hasValues = true
		}
	}

	// BERKSHELF_RETRY_DELAY
	if val := os.Getenv("BERKSHELF_RETRY_DELAY"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed >= 0 {
			config.RetryDelay = IntPtr(parsed)
			hasValues = true
		}
	}

	// BERKSHELF_CONCURRENCY
	if val := os.Getenv("BERKSHELF_CONCURRENCY"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			config.Concurrency = IntPtr(parsed)
			hasValues = true
		}
	}

	// Chef configuration
	chefConfig := loadChefConfigFromEnvironment()
	if chefConfig != nil {
		config.ChefConfig = chefConfig
		hasValues = true
	}

	if !hasValues {
		return nil
	}

	return config
}

// loadChefConfigFromEnvironment loads Chef configuration from environment variables
func loadChefConfigFromEnvironment() *ChefConfig {
	chefConfig := &ChefConfig{}
	hasValues := false

	if val := os.Getenv("CHEF_NODE_NAME"); val != "" {
		chefConfig.NodeName = StringPtr(val)
		hasValues = true
	}

	if val := os.Getenv("CHEF_CLIENT_KEY"); val != "" {
		chefConfig.ClientKey = StringPtr(val)
		hasValues = true
	}

	if val := os.Getenv("CHEF_SERVER_URL"); val != "" {
		chefConfig.ChefServerURL = StringPtr(val)
		hasValues = true
	}

	if val := os.Getenv("CHEF_ORGANIZATION"); val != "" {
		chefConfig.Organization = StringPtr(val)
		hasValues = true
	}

	if val := os.Getenv("CHEF_ENVIRONMENT"); val != "" {
		chefConfig.Environment = StringPtr(val)
		hasValues = true
	}

	if !hasValues {
		return nil
	}

	return chefConfig
}

// =============================================================================
// CONFIGURATION MERGING
// =============================================================================

// MergeConfigs merges multiple configurations with later configs taking precedence
// Uses pointer semantics to distinguish between "not set" (nil) and "set to zero value"
func MergeConfigs(base *Config, overlay *Config) *Config {
	if base == nil {
		if overlay == nil {
			return DefaultConfig()
		}
		return overlay
	}

	if overlay == nil {
		return base
	}

	// Create a deep copy of the base config to avoid modifying the original
	merged := &Config{}

	// First, copy all fields from base
	if err := mergo.Merge(merged, base, mergo.WithOverride); err != nil {
		// If mergo fails, fall back to manual copy
		*merged = *base
		// Deep copy slices
		if base.DefaultSources != nil {
			merged.DefaultSources = make([]string, len(base.DefaultSources))
			copy(merged.DefaultSources, base.DefaultSources)
		}
		if base.NoProxy != nil {
			merged.NoProxy = make([]string, len(base.NoProxy))
			copy(merged.NoProxy, base.NoProxy)
		}
		// Deep copy ChefConfig
		if base.ChefConfig != nil {
			merged.ChefConfig = &ChefConfig{
				NodeName:      base.ChefConfig.NodeName,
				ClientKey:     base.ChefConfig.ClientKey,
				ChefServerURL: base.ChefConfig.ChefServerURL,
				Organization:  base.ChefConfig.Organization,
				Environment:   base.ChefConfig.Environment,
			}
		}
	}

	// Now selectively merge overlay fields
	// The key insight: only merge fields that are explicitly set (non-nil) in overlay

	// Pointer fields: only override if overlay has non-nil value
	if overlay.CachePath != nil {
		merged.CachePath = overlay.CachePath
	}

	if overlay.SSLVerify != nil {
		merged.SSLVerify = overlay.SSLVerify
	}

	if overlay.Proxy != nil {
		merged.Proxy = overlay.Proxy
	}

	if overlay.APITimeout != nil {
		merged.APITimeout = overlay.APITimeout
	}

	if overlay.RetryCount != nil {
		merged.RetryCount = overlay.RetryCount
	}

	if overlay.RetryDelay != nil {
		merged.RetryDelay = overlay.RetryDelay
	}

	if overlay.Concurrency != nil {
		merged.Concurrency = overlay.Concurrency
	}

	// Slice fields: only override if overlay has non-empty slice
	if len(overlay.DefaultSources) > 0 {
		merged.DefaultSources = make([]string, len(overlay.DefaultSources))
		copy(merged.DefaultSources, overlay.DefaultSources)
	}

	if len(overlay.NoProxy) > 0 {
		merged.NoProxy = make([]string, len(overlay.NoProxy))
		copy(merged.NoProxy, overlay.NoProxy)
	}

	// ChefConfig: merge individual fields if overlay ChefConfig exists
	if overlay.ChefConfig != nil {
		if merged.ChefConfig == nil {
			merged.ChefConfig = &ChefConfig{}
		}

		// Only override non-nil fields in overlay ChefConfig
		if overlay.ChefConfig.NodeName != nil {
			merged.ChefConfig.NodeName = overlay.ChefConfig.NodeName
		}
		if overlay.ChefConfig.ClientKey != nil {
			merged.ChefConfig.ClientKey = overlay.ChefConfig.ClientKey
		}
		if overlay.ChefConfig.ChefServerURL != nil {
			merged.ChefConfig.ChefServerURL = overlay.ChefConfig.ChefServerURL
		}
		if overlay.ChefConfig.Organization != nil {
			merged.ChefConfig.Organization = overlay.ChefConfig.Organization
		}
		if overlay.ChefConfig.Environment != nil {
			merged.ChefConfig.Environment = overlay.ChefConfig.Environment
		}
	}

	return merged
}

// =============================================================================
// FILE OPERATIONS AND VALIDATION
// =============================================================================

// Save writes configuration to disk
func (c *Config) Save(path string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal configuration to JSON
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	cachePath := c.GetCachePath()
	if cachePath == "" {
		return fmt.Errorf("cache_path cannot be empty")
	}

	defaultSources := c.GetDefaultSources()
	if len(defaultSources) == 0 {
		return fmt.Errorf("at least one default source must be configured")
	}

	apiTimeout := c.GetAPITimeout()
	if apiTimeout <= 0 {
		return fmt.Errorf("api_timeout must be positive")
	}

	retryCount := c.GetRetryCount()
	if retryCount < 0 {
		return fmt.Errorf("retry_count cannot be negative")
	}

	retryDelay := c.GetRetryDelay()
	if retryDelay < 0 {
		return fmt.Errorf("retry_delay cannot be negative")
	}

	concurrency := c.GetConcurrency()
	if concurrency <= 0 {
		return fmt.Errorf("concurrency must be positive")
	}

	// Validate Chef config if present
	if c.ChefConfig != nil {
		if err := c.ChefConfig.validate(); err != nil {
			return fmt.Errorf("chef config validation failed: %w", err)
		}
	}

	return nil
}

// GetCachePathResolved returns the resolved cache path (expanding ~ if needed)
func (c *Config) GetCachePathResolved() string {
	cachePath := c.GetCachePath()
	// Expand ~ to home directory if needed
	if filepath.HasPrefix(cachePath, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, cachePath[2:])
	}
	return cachePath
}

// =============================================================================
// UTILITY FUNCTIONS
// =============================================================================

// GetConfigDir returns the berkshelf config directory
func GetConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".berkshelf")
}

// GetDefaultConfigPath returns the default config file path
func GetDefaultConfigPath() string {
	return filepath.Join(GetConfigDir(), "config.json")
}

// validate validates Chef configuration
func (c *ChefConfig) validate() error {
	nodeName := c.GetNodeName()
	if nodeName == "" {
		return fmt.Errorf("node_name cannot be empty")
	}

	clientKey := c.GetClientKey()
	if clientKey == "" {
		return fmt.Errorf("client_key cannot be empty")
	}

	chefServerURL := c.GetChefServerURL()
	if chefServerURL == "" {
		return fmt.Errorf("chef_server_url cannot be empty")
	}

	// Check if client key file exists
	if _, err := os.Stat(clientKey); err != nil {
		return fmt.Errorf("client key file %s does not exist", clientKey)
	}

	return nil
}

// getConfigPaths returns possible configuration file paths in order of precedence
func getConfigPaths() []string {
	home, _ := os.UserHomeDir()

	return []string{
		// Local project config (highest precedence)
		"./.berkshelf/config.json",
		"./config.json",

		// User-specific config
		filepath.Join(home, ".berkshelf", "config.json"),

		// Global config (lowest precedence)
		"/etc/berkshelf/config.json",
	}
}

// loadFromFile loads configuration from a JSON file
func loadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := &Config{}
	if err := json.Unmarshal(data, config); err != nil {
		return nil, err
	}

	// Validate the loaded configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}
