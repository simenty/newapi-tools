// NewAPI Tools - Docker management platform for newapi
package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.NewAPI.Home != "/opt/newapi" {
		t.Errorf("expected NewAPI.Home '/opt/newapi', got '%s'", cfg.NewAPI.Home)
	}
	if cfg.NewAPI.Port != 3000 {
		t.Errorf("expected NewAPI.Port 3000, got %d", cfg.NewAPI.Port)
	}
	if cfg.NewAPI.DockerImage != "calciumion/new-api:latest" {
		t.Errorf("expected NewAPI.DockerImage 'calciumion/new-api:latest', got '%s'", cfg.NewAPI.DockerImage)
	}
	if cfg.NewAPI.BackupDir != "/opt/newapi/backups" {
		t.Errorf("expected NewAPI.BackupDir '/opt/newapi/backups', got '%s'", cfg.NewAPI.BackupDir)
	}
	if cfg.Docker.ComposeCmd != "docker compose" {
		t.Errorf("expected Docker.ComposeCmd 'docker compose', got '%s'", cfg.Docker.ComposeCmd)
	}
	if cfg.Log.Level != "info" {
		t.Errorf("expected Log.Level 'info', got '%s'", cfg.Log.Level)
	}
	if cfg.Log.Format != "text" {
		t.Errorf("expected Log.Format 'text', got '%s'", cfg.Log.Format)
	}
}

func TestLoadConfigWithoutFile(t *testing.T) {
	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig with no file should not error: %v", err)
	}
	if cfg.NewAPI.Port != 3000 {
		t.Errorf("expected default port 3000, got %d", cfg.NewAPI.Port)
	}
	if cfg.NewAPI.Home != "/opt/newapi" {
		t.Errorf("expected default home '/opt/newapi', got '%s'", cfg.NewAPI.Home)
	}
}

func TestLoadConfigWithExplicitFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "test-config.yml")
	content := []byte(`newapi:
  home: "/custom/path"
  port: 8080
  docker_image: "custom/image:v2"
  backup_dir: "/custom/backups"
docker:
  compose_cmd: "docker-compose"
log:
  level: "debug"
  format: "json"
`)
	if err := os.WriteFile(cfgPath, content, 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig with explicit file should not error: %v", err)
	}

	if cfg.NewAPI.Home != "/custom/path" {
		t.Errorf("expected NewAPI.Home '/custom/path', got '%s'", cfg.NewAPI.Home)
	}
	if cfg.NewAPI.Port != 8080 {
		t.Errorf("expected NewAPI.Port 8080, got %d", cfg.NewAPI.Port)
	}
	if cfg.NewAPI.DockerImage != "custom/image:v2" {
		t.Errorf("expected NewAPI.DockerImage 'custom/image:v2', got '%s'", cfg.NewAPI.DockerImage)
	}
	if cfg.Docker.ComposeCmd != "docker-compose" {
		t.Errorf("expected Docker.ComposeCmd 'docker-compose', got '%s'", cfg.Docker.ComposeCmd)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("expected Log.Level 'debug', got '%s'", cfg.Log.Level)
	}
	if cfg.Log.Format != "json" {
		t.Errorf("expected Log.Format 'json', got '%s'", cfg.Log.Format)
	}
}

func TestLoadConfigNonexistentFile(t *testing.T) {
	// When an explicit config file path is given and doesn't exist,
	// Viper returns a file-not-found error (not ConfigFileNotFoundError).
	// This is expected behavior — the caller should validate the path.
	_, err := LoadConfig("/nonexistent/path/config.yml")
	if err == nil {
		t.Fatal("LoadConfig with nonexistent explicit file should return an error")
	}
}

func TestGetConfig(t *testing.T) {
	// Before LoadConfig, GetConfig may be nil or previous value
	// After LoadConfig, it should return the loaded config
	_, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	cfg := GetConfig()
	if cfg == nil {
		t.Fatal("GetConfig should not return nil after LoadConfig")
	}
	if cfg.NewAPI.Port != 3000 {
		t.Errorf("expected default port 3000, got %d", cfg.NewAPI.Port)
	}
}

func TestConfigDir(t *testing.T) {
	dir := configDir()
	if dir == "" {
		t.Error("configDir should not return empty string")
	}
}

func TestLoadConfigPartialOverride(t *testing.T) {
	// Create a config file that only overrides some values
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "partial.yml")
	content := []byte(`newapi:
  port: 9000
log:
  level: "debug"
`)
	if err := os.WriteFile(cfgPath, content, 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig with partial config should not error: %v", err)
	}

	// Overridden values
	if cfg.NewAPI.Port != 9000 {
		t.Errorf("expected overridden port 9000, got %d", cfg.NewAPI.Port)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("expected overridden log level 'debug', got '%s'", cfg.Log.Level)
	}

	// Default values should be preserved for non-overridden fields
	if cfg.Docker.ComposeCmd != "docker compose" {
		t.Errorf("expected default Docker.ComposeCmd 'docker compose', got '%s'", cfg.Docker.ComposeCmd)
	}
}

func TestValidatePort(t *testing.T) {
	cfg := DefaultConfig()
	cfg.NewAPI.Port = 0
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for port 0")
	}
	cfg.NewAPI.Port = 70000
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for port 70000")
	}
}

func TestValidateHome(t *testing.T) {
	cfg := DefaultConfig()
	cfg.NewAPI.Home = ""
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty home")
	}
}

func TestValidateDockerImage(t *testing.T) {
	cfg := DefaultConfig()
	cfg.NewAPI.DockerImage = ""
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty docker_image")
	}
}

func TestValidateBackupDir(t *testing.T) {
	cfg := DefaultConfig()
	cfg.NewAPI.BackupDir = ""
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty backup_dir")
	}
}

func TestValidateComposeCmd(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Docker.ComposeCmd = ""
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty compose_cmd")
	}
}

func TestValidateHealthTimeout(t *testing.T) {
	cfg := DefaultConfig()
	cfg.NewAPI.HealthTimeout = -1
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for negative health_timeout")
	}
}

func TestValidateMaxBackups(t *testing.T) {
	cfg := DefaultConfig()
	cfg.NewAPI.MaxBackups = -1
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for negative max_backups")
	}
}

func TestValidConfigPasses(t *testing.T) {
	cfg := DefaultConfig()
	if err := cfg.Validate(); err != nil {
		t.Errorf("default config should be valid, got: %v", err)
	}
}
