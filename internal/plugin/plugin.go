// NewAPI Tools - Docker management platform for newapi
// Package plugin provides the Plugin interface, its implementations
// (ShellPlugin, GoPlugin), and plugin discovery mechanisms.
package plugin

import (
	"log/slog"
	"sync"
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

// ---------------------------------------------------------------------------
// Compile-time plugin registry
// ---------------------------------------------------------------------------

var (
	registryMu sync.RWMutex
	registry   = map[string]Plugin{}
)

// Register adds a plugin to the global registry.
// Intended to be called from a package-level init() function so that
// simply importing the plugin package (with a blank import) is enough
// to make the plugin available at startup.
//
//	If a plugin with the same name is already registered, Register panics.
//	This is a deliberate design choice: duplicate registrations indicate a
//	programming error that should be caught early.
func Register(p Plugin) {
	registryMu.Lock()
	defer registryMu.Unlock()

	name := p.Name()
	if _, exists := registry[name]; exists {
		panic("plugin: Register called twice for " + name)
	}
	registry[name] = p
}

// All returns a shallow copy of the global plugin registry.
// The returned map is safe to iterate without holding the lock.
func All() map[string]Plugin {
	registryMu.RLock()
	defer registryMu.RUnlock()

	out := make(map[string]Plugin, len(registry))
	for k, v := range registry {
		out[k] = v
	}
	return out
}

// Get retrieves a single plugin by name from the global registry.
func Get(name string) (Plugin, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	p, ok := registry[name]
	return p, ok
}
