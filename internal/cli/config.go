// NewAPI Tools - Docker management platform for newapi
package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Bonus520/newapi-tools/internal/apperr"
	"github.com/Bonus520/newapi-tools/internal/core"
	"github.com/Bonus520/newapi-tools/internal/security"
	"github.com/Bonus520/newapi-tools/internal/ui"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage new-api configuration",
	Long:  `View or modify new-api configuration. Without subcommands, shows current config.`,
	RunE:  runConfigShow,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration key to the given value and persist to config file.

Valid keys:
  newapi.home          Installation directory
  newapi.port          Host port for new-api
  newapi.docker_image  Docker image to use
  newapi.backup_dir    Backup storage directory
  docker.compose_cmd   Docker compose command
  log.level             Log level (debug|info|warn|error)
  log.format            Log format (text|json)
  instance.active      Current active instance name`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive configuration wizard",
	Long:  `Run an interactive wizard to set up your new-api configuration.`,
	RunE:  runConfigInit,
}

var configChmodCmd = &cobra.Command{
	Use:   "chmod",
	Short: "Fix configuration file permissions (chmod 600)",
	Long: `Fix permissions on sensitive configuration files to 600 (owner read/write only).
Files fixed: .env, docker-compose.yml in the new-api home directory, and the active config file.
This is a no-op on Windows.`,
	RunE: runConfigChmod,
}

func init() {
	configCmd.Flags().Bool("json", false, "output in JSON format")

	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configChmodCmd)
	rootCmd.AddCommand(configCmd)
}

// ---- config show ----

func runConfigShow(cmd *cobra.Command, args []string) error {
	cfg := core.GetConfig()
	if cfg == nil {
		return fmt.Errorf("configuration not loaded")
	}

	jsonOut, _ := cmd.Flags().GetBool("json")
	if jsonOut {
		return printConfigJSON(cfg)
	}
	printConfigTable(cfg)
	return nil
}

func printConfigTable(cfg *core.Config) {
	fmt.Println("Current Configuration:")
	fmt.Println(strings.Repeat("-", 50))
	fmt.Printf("  %-20s %s\n", "newapi.home", cfg.NewAPI.Home)
	fmt.Printf("  %-20s %d\n", "newapi.port", cfg.NewAPI.Port)
	fmt.Printf("  %-20s %s\n", "newapi.docker_image", cfg.NewAPI.DockerImage)
	fmt.Printf("  %-20s %s\n", "newapi.backup_dir", cfg.NewAPI.BackupDir)
	fmt.Printf("  %-20s %s\n", "docker.compose_cmd", cfg.Docker.ComposeCmd)
	fmt.Printf("  %-20s %s\n", "log.level", cfg.Log.Level)
	fmt.Printf("  %-20s %s\n", "log.format", cfg.Log.Format)
	fmt.Printf("  %-20s %s\n", "instance.active", cfg.Instance.Active)
	fmt.Println(strings.Repeat("-", 50))

	configFile := core.ConfigFileUsed()
	if configFile != "" {
		fmt.Printf("  Config file: %s\n", configFile)
	} else {
		fmt.Printf("  Config file: (none — using defaults)\n")
	}
}

func printConfigJSON(cfg *core.Config) error {
	fmt.Printf(`{
  "newapi": {
    "home": %q,
    "port": %d,
    "docker_image": %q,
    "backup_dir": %q
  },
  "docker": {
    "compose_cmd": %q
  },
  "log": {
    "level": %q,
    "format": %q
  },
  "instance": {
    "active": %q
  }
}
`,
		cfg.NewAPI.Home,
		cfg.NewAPI.Port,
		cfg.NewAPI.DockerImage,
		cfg.NewAPI.BackupDir,
		cfg.Docker.ComposeCmd,
		cfg.Log.Level,
		cfg.Log.Format,
		cfg.Instance.Active,
	)
	return nil
}

// ---- config set ----

// validKeys lists all settable config keys and their type (string/int).
var validKeys = map[string]string{
	"newapi.home":         "string",
	"newapi.port":         "int",
	"newapi.docker_image": "string",
	"newapi.backup_dir":   "string",
	"docker.compose_cmd":  "string",
	"log.level":           "string",
	"log.format":          "string",
	"instance.active":     "string",
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key, value := args[0], args[1]

	keyType, ok := validKeys[key]
	if !ok {
		fmt.Fprintf(os.Stderr, "Unknown key: %q\n", key)
		fmt.Fprintln(os.Stderr, "Valid keys:")
		for k := range validKeys {
			fmt.Fprintf(os.Stderr, "  %s\n", k)
		}
		return fmt.Errorf("invalid config key: %s", key)
	}

	cfg := core.GetConfig()
	if cfg == nil {
		return fmt.Errorf("configuration not loaded")
	}

	// Apply the value
	if err := applyConfigValue(cfg, key, value, keyType); err != nil {
		return err
	}

	// Determine config file path
	configFile := core.ConfigFileUsed()
	if configFile == "" {
		configFile = core.ConfigFilePath()
	}

	// Persist
	if err := core.WriteConfig(cfg, configFile); err != nil {
		return apperr.Wrap(apperr.CodeConfigLoad, "", err)
	}

	fmt.Printf("Set %s = %s\n", key, value)
	fmt.Printf("Config saved to %s\n", configFile)

	ui.L().Info("config updated", "key", key, "value", value, "file", configFile)
	return nil
}

func applyConfigValue(cfg *core.Config, key, value, keyType string) error {
	switch key {
	case "newapi.home":
		cfg.NewAPI.Home = value
	case "newapi.port":
		port, err := strconv.Atoi(value)
		if err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("invalid port: %s (must be 1-65535)", value)
		}
		cfg.NewAPI.Port = port
	case "newapi.docker_image":
		cfg.NewAPI.DockerImage = value
	case "newapi.backup_dir":
		cfg.NewAPI.BackupDir = value
	case "docker.compose_cmd":
		cfg.Docker.ComposeCmd = value
	case "log.level":
		validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
		if !validLevels[value] {
			return fmt.Errorf("invalid log level: %s (must be debug|info|warn|error)", value)
		}
		cfg.Log.Level = value
	case "log.format":
		validFormats := map[string]bool{"text": true, "json": true}
		if !validFormats[value] {
			return fmt.Errorf("invalid log format: %s (must be text|json)", value)
		}
		cfg.Log.Format = value
	case "instance.active":
		cfg.Instance.Active = value
	default:
		return fmt.Errorf("unknown key: %s", key)
	}
	return nil
}

// ---- config init ----

func runConfigInit(cmd *cobra.Command, args []string) error {
	cfg := core.GetConfig()
	if cfg == nil {
		cfg = core.DefaultConfig()
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("=== NewAPI Tools Configuration Wizard ===")
	fmt.Println("Press Enter to accept the default value in [brackets].")
	fmt.Println()

	// newapi.home
	cfg.NewAPI.Home = prompt(reader, "Installation directory", cfg.NewAPI.Home)

	// newapi.port
	portStr := prompt(reader, "Host port for new-api", fmt.Sprintf("%d", cfg.NewAPI.Port))
	if p, err := strconv.Atoi(portStr); err == nil && p > 0 && p <= 65535 {
		cfg.NewAPI.Port = p
	} else {
		fmt.Fprintf(os.Stderr, "Invalid port, keeping default: %d\n", cfg.NewAPI.Port)
	}

	// newapi.docker_image
	cfg.NewAPI.DockerImage = prompt(reader, "Docker image", cfg.NewAPI.DockerImage)

	// newapi.backup_dir
	cfg.NewAPI.BackupDir = prompt(reader, "Backup directory", cfg.NewAPI.BackupDir)

	// docker.compose_cmd
	cfg.Docker.ComposeCmd = prompt(reader, "Docker compose command", cfg.Docker.ComposeCmd)

	// log.level
	cfg.Log.Level = prompt(reader, "Log level (debug|info|warn|error)", cfg.Log.Level)

	// log.format
	cfg.Log.Format = prompt(reader, "Log format (text|json)", cfg.Log.Format)

	// Confirm
	fmt.Println()
	fmt.Println("=== Configuration Summary ===")
	printConfigTable(cfg)
	fmt.Println()

	confirm := prompt(reader, "Save this configuration? (y/n)", "y")
	if strings.ToLower(confirm) != "y" && strings.ToLower(confirm) != "yes" {
		fmt.Println("Configuration not saved.")
		return nil
	}

	// Write
	configFile := core.ConfigFileUsed()
	if configFile == "" {
		configFile = core.ConfigFilePath()
	}

	if err := core.WriteConfig(cfg, configFile); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Configuration saved to %s\n", configFile)
	ui.L().Info("config initialized", "file", configFile)
	return nil
}

// ---- config chmod ----

// runConfigChmod fixes permissions (chmod 600) on sensitive config files.
func runConfigChmod(cmd *cobra.Command, args []string) error {
	cfg := core.GetConfig()
	if cfg == nil {
		return fmt.Errorf("configuration not loaded")
	}

	// Build list of files to fix.
	targets := []string{
		filepath.Join(cfg.NewAPI.Home, ".env"),
		filepath.Join(cfg.NewAPI.Home, "docker-compose.yml"),
	}
	if cf := core.ConfigFileUsed(); cf != "" {
		targets = append(targets, cf)
	}

	var hadError bool
	for _, path := range targets {
		if err := security.FixConfigPerm(path); err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR: %v\n", err)
			hadError = true
		} else {
			fmt.Printf("  chmod 600 %s\n", path)
		}
	}

	if hadError {
		return fmt.Errorf("one or more files could not be fixed")
	}
	fmt.Println("Done. All sensitive config files have been secured.")
	return nil
}

// prompt displays a prompt with a default value and returns the user's input.
// If the user presses Enter without typing, the default is returned.
func prompt(reader *bufio.Reader, label, defaultVal string) string {
	fmt.Printf("  %s [%s]: ", label, defaultVal)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}
