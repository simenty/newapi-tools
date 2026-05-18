// NewAPI Tools - Docker management platform for newapi
// Package plugin provides the Plugin interface, its implementations
// (ShellPlugin, GoPlugin), and plugin discovery mechanisms.
package plugin

import (
	"log/slog"
)

// Command describes a command provided by a plugin.
type Command struct {
	Name        string
	Description string
	Usage       string
}

// Plugin is the interface that all plugins must implement.
type Plugin interface {
	Name() string
	Version() string
	Commands() []Command
	Init(ctx Context) error
	Execute(cmd string, args []string) error
	Shutdown() error
}

// Context provides runtime context to plugins.
// Self-contained to avoid circular imports.
type Context struct {
	NewAPIHome       string
	NewAPIPort       int
	NewAPIDockerImg  string
	NewAPIBackupDir  string
	DockerComposeCmd string
	Logger           *slog.Logger
	HomeDir          string
}

// NewContext creates a plugin Context from individual config values.
func NewContext(home string, port int, dockerImg string, backupDir string, composeCmd string, logger *slog.Logger, homeDir string) Context {
	return Context{
		NewAPIHome:       home,
		NewAPIPort:       port,
		NewAPIDockerImg:  dockerImg,
		NewAPIBackupDir:  backupDir,
		DockerComposeCmd: composeCmd,
		Logger:           logger,
		HomeDir:          homeDir,
	}
}
