// NewAPI Tools - Docker management platform for newapi
package cli

import (
	"fmt"
	"os"

	"github.com/Bonus520/newapi-tools/internal/core"
	"github.com/Bonus520/newapi-tools/internal/plugin"
	"github.com/Bonus520/newapi-tools/internal/registry"
	"github.com/Bonus520/newapi-tools/internal/ui"
	"github.com/spf13/cobra"
)

// globalRegistry holds the command registry for the application.
var globalRegistry *registry.Registry

// rootCmd is the base command for newapi-tools CLI.
var rootCmd = &cobra.Command{
	Use:           "newapi-tools",
	Short:         "Docker management platform for new-api",
	Long:          `NewAPI Tools is a Docker management platform for new-api.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command.
func Execute() {
	cobra.OnInitialize(initConfig, initPlugins)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
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

	rootCmd.Version = core.Version
	rootCmd.SetVersionTemplate(fmt.Sprintf("newapi-tools %s (commit: %s, built: %s)\n", core.Version, core.GitCommit, core.BuildDate))
}

// initConfig loads configuration and initializes the logger.
func initConfig() {
	configFile, _ := rootCmd.PersistentFlags().GetString("config")
	_, err := core.LoadConfig(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		return
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
