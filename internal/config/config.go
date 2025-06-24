package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config represents berkshelf configuration
type Config struct {
	CachePath      string      `json:"cache_path"`
	DefaultSources []string    `json:"default_sources"`
	SSLVerify      bool        `json:"ssl_verify"`
	Proxy          string      `json:"proxy,omitempty"`
	NoProxy        []string    `json:"no_proxy,omitempty"`
	ChefConfig     *ChefConfig `json:"chef,omitempty"`
	APITimeout     int         `json:"api_timeout"` // seconds
	RetryCount     int         `json:"retry_count"`
	RetryDelay     int         `json:"retry_delay"` // seconds
	Concurrency    int         `json:"concurrency"` // max concurrent downloads
}

// ChefConfig contains Chef-specific configuration
type ChefConfig struct {
	NodeName      string `json:"node_name"`
	ClientKey     string `json:"client_key"`
	ChefServerURL string `json:"chef_server_url"`
	Organization  string `json:"organization,omitempty"`
	Environment   string `json:"environment,omitempty"`
}

// Load reads configuration from standard locations
func Load() (*Config, error) {
	// Try to load from various locations in order of precedence
	configPaths := getConfigPaths()

	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			config, err := loadFromFile(path)
			if err != nil {
				return nil, fmt.Errorf("failed to load config from %s: %w", path, err)
			}
			return config, nil
		}
	}

	// Return default config if no config file found
	return DefaultConfig(), nil
}

// LoadFromFile loads configuration from a specific file
func LoadFromFile(path string) (*Config, error) {
	return loadFromFile(path)
}

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
	if c.CachePath == "" {
		return fmt.Errorf("cache_path cannot be empty")
	}

	if len(c.DefaultSources) == 0 {
		return fmt.Errorf("at least one default source must be configured")
	}

	if c.APITimeout <= 0 {
		return fmt.Errorf("api_timeout must be positive")
	}

	if c.RetryCount < 0 {
		return fmt.Errorf("retry_count cannot be negative")
	}

	if c.RetryDelay < 0 {
		return fmt.Errorf("retry_delay cannot be negative")
	}

	if c.Concurrency <= 0 {
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

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()

	return &Config{
		CachePath: filepath.Join(home, ".berkshelf", "cookbooks"),
		DefaultSources: []string{
			"https://supermarket.chef.io",
		},
		SSLVerify:   true,
		APITimeout:  30,
		RetryCount:  3,
		RetryDelay:  1,
		Concurrency: 5,
	}
}

// GetCachePath returns the resolved cache path
func (c *Config) GetCachePath() string {
	// Expand ~ to home directory if needed
	if filepath.HasPrefix(c.CachePath, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, c.CachePath[2:])
	}
	return c.CachePath
}

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
	if c.NodeName == "" {
		return fmt.Errorf("node_name cannot be empty")
	}

	if c.ClientKey == "" {
		return fmt.Errorf("client_key cannot be empty")
	}

	if c.ChefServerURL == "" {
		return fmt.Errorf("chef_server_url cannot be empty")
	}

	// Check if client key file exists
	if _, err := os.Stat(c.ClientKey); err != nil {
		return fmt.Errorf("client key file %s does not exist", c.ClientKey)
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

	config := DefaultConfig()
	if err := json.Unmarshal(data, config); err != nil {
		return nil, err
	}

	// Validate the loaded configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}
