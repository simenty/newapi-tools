// NewAPI Tools - Docker management platform for newapi
package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Config holds the full application configuration.
type Config struct {
	NewAPI NewAPIConfig `mapstructure:"newapi"`
	Docker DockerConfig `mapstructure:"docker"`
	Log    LogConfig    `mapstructure:"log"`
}

// NewAPIConfig holds new-api specific configuration.
type NewAPIConfig struct {
	Home        string `mapstructure:"home"`
	Port        int    `mapstructure:"port"`
	DockerImage string `mapstructure:"docker_image"`
	BackupDir   string `mapstructure:"backup_dir"`
}

// DockerConfig holds Docker-related configuration.
type DockerConfig struct {
	ComposeCmd string `mapstructure:"compose_cmd"`
}

// LogConfig holds logging configuration.
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// DefaultConfig returns a Config populated with default values.
func DefaultConfig() *Config {
	return &Config{
		NewAPI: NewAPIConfig{
			Home:        "/opt/newapi",
			Port:        3000,
			DockerImage: "calciumion/new-api:latest",
			BackupDir:   "/opt/newapi/backups",
		},
		Docker: DockerConfig{
			ComposeCmd: "docker compose",
		},
		Log: LogConfig{
			Level:  "info",
			Format: "text",
		},
	}
}

// configInstance holds the loaded configuration.
var configInstance *Config

// LoadConfig initializes Viper, reads config file, and returns the merged Config.
// configFile is the explicit path from --config flag (may be empty).
func LoadConfig(configFile string) (*Config, error) {
	v := viper.New()

	// Set defaults from DefaultConfig
	setDefaults(v)

	// Configure Viper
	v.SetConfigName("newapi-tools")
	v.SetConfigType("yaml")

	// Config file search paths
	if configFile != "" {
		v.SetConfigFile(configFile)
	} else {
		// Search in standard locations
		v.AddConfigPath("/etc/newapi-tools/")
		v.AddConfigPath(filepath.Join(configDir(), "newapi-tools"))
		v.AddConfigPath(".")
	}

	// Environment variable support
	v.SetEnvPrefix("NEWAPI")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read config file (it's okay if it doesn't exist)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is acceptable — use defaults
	}

	// Unmarshal into Config struct
	cfg := DefaultConfig()
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	configInstance = cfg
	return cfg, nil
}

// GetConfig returns the currently loaded configuration.
// Returns nil if LoadConfig has not been called.
func GetConfig() *Config {
	return configInstance
}

// setDefaults registers default values with Viper.
func setDefaults(v *viper.Viper) {
	defaults := DefaultConfig()
	v.SetDefault("newapi.home", defaults.NewAPI.Home)
	v.SetDefault("newapi.port", defaults.NewAPI.Port)
	v.SetDefault("newapi.docker_image", defaults.NewAPI.DockerImage)
	v.SetDefault("newapi.backup_dir", defaults.NewAPI.BackupDir)
	v.SetDefault("docker.compose_cmd", defaults.Docker.ComposeCmd)
	v.SetDefault("log.level", defaults.Log.Level)
	v.SetDefault("log.format", defaults.Log.Format)
}

// configDir returns the user's config directory path.
func configDir() string {
	if cfgDir := os.Getenv("XDG_CONFIG_HOME"); cfgDir != "" {
		return cfgDir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".config")
}

// ConfigFileUsed returns the path of the config file that was loaded.
// Must be called after LoadConfig.
func ConfigFileUsed() string {
	return viper.ConfigFileUsed()
}

// WriteConfig writes the given Config to the specified YAML file.
// Parent directories are created automatically.
func WriteConfig(cfg *Config, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data := map[string]interface{}{
		"newapi": map[string]interface{}{
			"home":         cfg.NewAPI.Home,
			"port":         cfg.NewAPI.Port,
			"docker_image": cfg.NewAPI.DockerImage,
			"backup_dir":   cfg.NewAPI.BackupDir,
		},
		"docker": map[string]interface{}{
			"compose_cmd": cfg.Docker.ComposeCmd,
		},
		"log": map[string]interface{}{
			"level":  cfg.Log.Level,
			"format": cfg.Log.Format,
		},
	}

	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}

// ConfigFilePath returns the default config file path.
// Priority: --config flag > XDG_CONFIG_HOME > ~/.config/newapi-tools/newapi-tools.yml
func ConfigFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "newapi-tools.yml"
	}
	return filepath.Join(home, ".config", "newapi-tools", "newapi-tools.yml")
}
