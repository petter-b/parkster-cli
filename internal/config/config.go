package config

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	// Output settings
	OutputFormat string `yaml:"output_format"` // human, json

	// Default timeout for operations
	Timeout Duration `yaml:"timeout"`

	// Debug mode
	Debug bool `yaml:"debug"`

	// Default account/service for commands that need one
	DefaultAccount string `yaml:"default_account"`

	// Service-specific settings
	Services map[string]ServiceConfig `yaml:"services"`
}

// ServiceConfig holds per-service configuration
type ServiceConfig struct {
	BaseURL string   `yaml:"base_url,omitempty"`
	Timeout Duration `yaml:"timeout,omitempty"`
}

// Duration wraps time.Duration for YAML unmarshaling
type Duration time.Duration

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(parsed)
	return nil
}

func (d Duration) MarshalYAML() (interface{}, error) {
	return time.Duration(d).String(), nil
}

func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}

// Default returns default configuration
func Default() *Config {
	return &Config{
		OutputFormat: "human",
		Timeout:      Duration(30 * time.Second),
		Debug:        false,
		Services:     make(map[string]ServiceConfig),
	}
}

// Load reads configuration from file
func Load(path string) (*Config, error) {
	if path == "" {
		path = DefaultPath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return nil, err
	}

	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save writes configuration to file
func (c *Config) Save(path string) error {
	if path == "" {
		path = DefaultPath()
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// DefaultPath returns the default config file path
func DefaultPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "parkster", "config.yaml")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	return filepath.Join(home, ".config", "parkster", "config.yaml")
}

// GetServiceConfig returns config for a specific service with defaults
func (c *Config) GetServiceConfig(service string) ServiceConfig {
	if cfg, ok := c.Services[service]; ok {
		return cfg
	}
	return ServiceConfig{
		Timeout: c.Timeout,
	}
}
