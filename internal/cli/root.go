// NewAPI Tools - Docker management platform for newapi
package cli

import (
	"fmt"
	"log/slog"
	"os"
	"os/user"
	"time"

	"github.com/simenty/newapi-tools/internal/audit"
	"github.com/simenty/newapi-tools/internal/core"
	"github.com/simenty/newapi-tools/internal/i18n"
	"github.com/simenty/newapi-tools/internal/instance"
	"github.com/simenty/newapi-tools/internal/plugin"
	"github.com/simenty/newapi-tools/internal/registry"
	"github.com/simenty/newapi-tools/internal/security"
	"github.com/simenty/newapi-tools/internal/ui"
	"github.com/spf13/cobra"
)

// globalRegistry holds the command registry for the application.
var globalRegistry *registry.Registry

// globalAuditLogger holds the audit logger for command tracking.
var globalAuditLogger *audit.AuditLogger

// rootCmd is the base command for newapi-tools CLI.
var rootCmd = &cobra.Command{
	Use:           "newapi-tools",
	Short:         "Docker management platform for new-api",
	Long:          `NewAPI Tools is a Docker management platform for new-api.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Run Docker group permission pre-check (non-blocking warning)
		inGroup, err := security.CheckDockerGroup()
		if err == nil && !inGroup {
			fmt.Fprintf(os.Stderr, "\033[33m⚠ Warning: current user is not in the 'docker' group. Some commands may require sudo.\033[0m\n")
			fmt.Fprintf(os.Stderr, "\033[33m  Fix: sudo usermod -aG docker $USER && newgrp docker\033[0m\n\n")
		}
		return nil
	},
}

// Execute runs the root command with audit logging.
func Execute() {
	cobra.OnInitialize(initConfig, initPlugins)

	// Initialize audit logger
	globalAuditLogger = audit.NewAuditLogger("")

	// Record start time before execution
	startTime := time.Now()

	err := rootCmd.Execute()

	// Build and write audit entry
	durationMs := time.Since(startTime).Milliseconds()
	result := "ok"
	errMsg := ""
	if err != nil {
		result = "error"
		errMsg = err.Error()
	}

	currentUser := getCurrentUser()
	entry := audit.AuditEntry{
		Timestamp:  startTime,
		Command:    getFullCommand(),
		User:       currentUser,
		Args:       os.Args[1:],
		Result:     result,
		Error:      errMsg,
		DurationMs: durationMs,
	}

	// Write audit log — failure should not affect command exit
	if auditErr := globalAuditLogger.Log(entry); auditErr != nil {
		slog.Warn("audit log write failed", "error", auditErr)
	}

	if err != nil {
		ui.PrintError(err)
		os.Exit(1)
	}
}

// GetRootCmd returns the root cobra command, useful for testing.
func GetRootCmd() *cobra.Command {
	return rootCmd
}

// GetRegistry returns the global command registry.
func GetRegistry() *registry.Registry {
	return globalRegistry
}

func init() {
	rootCmd.PersistentFlags().String("config", "", "config file (default is $HOME/.newapi-tools.yml)")
	rootCmd.PersistentFlags().String("log-level", "", "log level (debug|info|warn|error)")
	rootCmd.PersistentFlags().String("log-format", "", "log format (text|json)")
	rootCmd.PersistentFlags().String("plugins-dir", "", "plugins directory (default is ./plugins/)")
	rootCmd.PersistentFlags().String("lang", "", "language for output messages (e.g. zh-CN, en)")
	rootCmd.PersistentFlags().Bool("debug", false, "enable debug logging (most verbose)")
	rootCmd.PersistentFlags().Bool("verbose", false, "enable verbose (info-level) logging")
	rootCmd.PersistentFlags().String("instance", "", "instance name to operate on (default is the active instance)")

	rootCmd.Version = core.Version
	rootCmd.SetVersionTemplate(fmt.Sprintf("newapi-tools %s (commit: %s, built: %s)\n", core.Version, core.GitCommit, core.BuildDate))
}

// initConfig loads configuration and initializes the logger.
func initConfig() {
	// Initialize i18n based on --lang flag or environment variables
	lang, _ := rootCmd.PersistentFlags().GetString("lang")
	if err := i18n.Init(lang); err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize i18n: %v\n", err)
		// Non-fatal: i18n.T() will return keys as fallback
	}

	configFile, _ := rootCmd.PersistentFlags().GetString("config")
	cfg, err := core.LoadConfig(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		return
	}

	// Load active instance configuration
	if cfg.Instance.Active != "" {
		store := instance.NewStore("")
		activeInstance, err := store.Get(cfg.Instance.Active)
		if err == nil && activeInstance != nil {
			syncInstanceToConfig(activeInstance, cfg)
		}
	}

	// Apply --instance flag override
	instanceName, _ := rootCmd.PersistentFlags().GetString("instance")
	if instanceName != "" {
		store := instance.NewStore("")
		if inst, err := store.Get(instanceName); err == nil && inst != nil {
			syncInstanceToConfig(inst, cfg)
		}
	}

	// Apply --debug/--verbose flag overrides for log level
	debug, _ := rootCmd.PersistentFlags().GetBool("debug")
	verbose, _ := rootCmd.PersistentFlags().GetBool("verbose")
	if debug {
		core.GetConfig().Log.Level = "debug"
	} else if verbose {
		core.GetConfig().Log.Level = "info"
	}

	// Apply CLI flag overrides for log settings
	if logLevel, _ := rootCmd.PersistentFlags().GetString("log-level"); logLevel != "" {
		core.GetConfig().Log.Level = logLevel
	}
	if logFormat, _ := rootCmd.PersistentFlags().GetString("log-format"); logFormat != "" {
		core.GetConfig().Log.Format = logFormat
	}

	// Initialize structured logger
	ui.SetupLogger(&core.GetConfig().Log)
}

// syncInstanceToConfig synchronizes instance configuration to core.Config.
func syncInstanceToConfig(inst *instance.Instance, cfg *core.Config) {
	cfg.NewAPI.Home = inst.Home
	cfg.NewAPI.Port = inst.Port
	cfg.NewAPI.DockerImage = inst.DockerImage
	cfg.NewAPI.Domain = inst.Domain
	cfg.NewAPI.HealthTimeout = inst.HealthTimeout
	cfg.NewAPI.MaxBackups = inst.MaxBackups
}

// initPlugins discovers and loads plugins, then registers their commands
// as dynamic Cobra subcommands.
func initPlugins() {
	if globalRegistry != nil {
		return // Already initialized
	}

	globalRegistry = registry.NewRegistry()

	// Determine plugins directory
	pluginsDir, _ := rootCmd.PersistentFlags().GetString("plugins-dir")
	if pluginsDir == "" {
		pluginsDir = "./plugins"
	}

	// Check if plugins dir exists
	if _, err := os.Stat(pluginsDir); os.IsNotExist(err) {
		ui.L().Debug("plugins directory not found", "dir", pluginsDir)
		return
	}

	// Load plugins from filesystem
	cfg := core.GetConfig()
	ctx := plugin.NewContext(
		cfg.NewAPI.Home,
		cfg.NewAPI.Port,
		cfg.NewAPI.DockerImage,
		cfg.NewAPI.BackupDir,
		cfg.NewAPI.Domain,
		cfg.NewAPI.HealthTimeout,
		cfg.NewAPI.MaxBackups,
		cfg.Docker.ComposeCmd,
		ui.L(),
		"",
	)

	loader := plugin.NewLoader(pluginsDir, ctx)
	if err := loader.LoadAll(); err != nil {
		ui.L().Warn("failed to load plugins", "error", err)
		return
	}

	// Register all loaded plugins with the registry and create Cobra commands
	for name, p := range loader.AllPlugins() {
		if err := globalRegistry.RegisterPlugin(p, ui.L()); err != nil {
			ui.L().Error("failed to register plugin", "name", name, "error", err)
			continue
		}

		// Create Cobra subcommands for each plugin command
		for _, cmd := range p.Commands() {
			pluginCmd := createPluginCommand(p, cmd)
			rootCmd.AddCommand(pluginCmd)
		}

		ui.L().Debug("registered plugin commands", "plugin", name, "commands", len(p.Commands()))
	}
}

// createPluginCommand creates a Cobra command that delegates to a plugin.
func createPluginCommand(p plugin.Plugin, cmd plugin.Command) *cobra.Command {
	return &cobra.Command{
		Use:   cmd.Name,
		Short: cmd.Description,
		Long:  cmd.Usage,
		RunE: func(c *cobra.Command, args []string) error {
			return p.Execute(cmd.Name, args)
		},
	}
}

// getCurrentUser returns the current OS username, or "unknown" if it cannot be determined.
func getCurrentUser() string {
	current, err := user.Current()
	if err != nil {
		return "unknown"
	}
	return current.Username
}

// getFullCommand returns the full command path as invoked (e.g. "newapi-tools install").
func getFullCommand() string {
	if rootCmd.CalledAs() != "" {
		return "newapi-tools " + rootCmd.CalledAs()
	}
	return "newapi-tools"
}
