package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Global   GlobalConfig         `yaml:"global" mapstructure:"global"`
	Defaults DefaultConfig        `yaml:"defaults" mapstructure:"defaults"`
	Env      map[string]EnvConfig `yaml:"environments" mapstructure:"environments"`
	Secrets  SecretsConfig        `yaml:"secrets" mapstructure:"secrets"`
	Plugins  []PluginConfig       `yaml:"plugins" mapstructure:"plugins"`
}

type GlobalConfig struct {
	BaseURL   string            `yaml:"base_url" mapstructure:"base_url"`
	Headers   map[string]string `yaml:"headers" mapstructure:"headers"`
	Timeout   time.Duration     `yaml:"timeout" mapstructure:"timeout"`
	Retries   int               `yaml:"retries" mapstructure:"retries"`
	Variables map[string]any    `yaml:"variables" mapstructure:"variables"`
	Setup     []string          `yaml:"setup" mapstructure:"setup"`
	Teardown  []string          `yaml:"teardown" mapstructure:"teardown"`
}

type DefaultConfig struct {
	HTTPTimeout    time.Duration `yaml:"http_timeout" mapstructure:"http_timeout"`
	MaxRetries     int           `yaml:"max_retries" mapstructure:"max_retries"`
	RetryDelay     time.Duration `yaml:"retry_delay" mapstructure:"retry_delay"`
	FollowRedirect bool          `yaml:"follow_redirect" mapstructure:"follow_redirect"`
	VerifySSL      bool          `yaml:"verify_ssl" mapstructure:"verify_ssl"`
}

type EnvConfig struct {
	BaseURL   string            `yaml:"base_url" mapstructure:"base_url"`
	Headers   map[string]string `yaml:"headers" mapstructure:"headers"`
	Variables map[string]any    `yaml:"variables" mapstructure:"variables"`
}

type SecretsConfig struct {
	Provider string                 `yaml:"provider" mapstructure:"provider"` // vault, aws, env
	Config   map[string]interface{} `yaml:"config" mapstructure:"config"`
}

type PluginConfig struct {
	Name    string                 `yaml:"name" mapstructure:"name"`
	Path    string                 `yaml:"path" mapstructure:"path"`
	Config  map[string]interface{} `yaml:"config" mapstructure:"config"`
	Enabled bool                   `yaml:"enabled" mapstructure:"enabled"`
}

func LoadConfig(configFile string) (*Config, error) {
	config := &Config{
		Defaults: DefaultConfig{
			HTTPTimeout:    30 * time.Second,
			MaxRetries:     3,
			RetryDelay:     1 * time.Second,
			FollowRedirect: true,
			VerifySSL:      true,
		},
		Env: make(map[string]EnvConfig),
	}

	if configFile != "" {
		if err := loadConfigFile(configFile, config); err != nil {
			return nil, fmt.Errorf("failed to load config file %s: %w", configFile, err)
		}
	}

	// Override with environment variables
	if err := loadEnvOverrides(config); err != nil {
		return nil, fmt.Errorf("failed to load environment overrides: %w", err)
	}

	return config, nil
}

func loadConfigFile(configFile string, config *Config) error {
	if !filepath.IsAbs(configFile) {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		configFile = filepath.Join(wd, configFile)
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, config)
}

func loadEnvOverrides(config *Config) error {
	viper.SetEnvPrefix("FUEGO")
	viper.AutomaticEnv()

	// Override specific values from environment
	if baseURL := viper.GetString("BASE_URL"); baseURL != "" {
		config.Global.BaseURL = baseURL
	}

	if timeout := viper.GetDuration("TIMEOUT"); timeout > 0 {
		config.Global.Timeout = timeout
	}

	return nil
}

func (c *Config) GetEnvironment(env string) (EnvConfig, bool) {
	envConfig, exists := c.Env[env]
	return envConfig, exists
}

func (c *Config) MergeEnvironment(env string) *Config {
	merged := *c

	if envConfig, exists := c.Env[env]; exists {
		if envConfig.BaseURL != "" {
			merged.Global.BaseURL = envConfig.BaseURL
		}

		if merged.Global.Headers == nil {
			merged.Global.Headers = make(map[string]string)
		}
		for k, v := range envConfig.Headers {
			merged.Global.Headers[k] = v
		}

		if merged.Global.Variables == nil {
			merged.Global.Variables = make(map[string]any)
		}
		for k, v := range envConfig.Variables {
			merged.Global.Variables[k] = v
		}
	}

	return &merged
}
