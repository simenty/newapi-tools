// NewAPI Tools - Docker management platform for newapi
// Package registry manages plugin registration and command routing.
package registry

import (
	"fmt"
	"log/slog"

	"github.com/simenty/newapi-tools/internal/plugin"
)

// Registry manages plugin registration and command routing.
type Registry struct {
	plugins  map[string]plugin.Plugin
	commands map[string]*registeredCmd
}

// registeredCmd holds a command and its owning plugin.
type registeredCmd struct {
	cmd    plugin.Command
	plugin plugin.Plugin
}

// NewRegistry creates a new empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		plugins:  make(map[string]plugin.Plugin),
		commands: make(map[string]*registeredCmd),
	}
}

// RegisterPlugin registers a plugin and all its commands.
// Returns an error if a plugin with the same name is already registered.
func (r *Registry) RegisterPlugin(p plugin.Plugin, logger *slog.Logger) error {
	name := p.Name()
	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("plugin %q already registered", name)
	}

	r.plugins[name] = p

	// Register all commands from this plugin
	for _, cmd := range p.Commands() {
		if existing, ok := r.commands[cmd.Name]; ok {
			logger.Warn("command already registered, skipping",
				"command", cmd.Name,
				"existing_plugin", existing.plugin.Name(),
				"new_plugin", name,
			)
			continue
		}
		r.commands[cmd.Name] = &registeredCmd{
			cmd:    cmd,
			plugin: p,
		}
		logger.Debug("registered command", "command", cmd.Name, "plugin", name)
	}

	return nil
}

// GetCommand returns the registered command and its owning plugin.
func (r *Registry) GetCommand(name string) (plugin.Command, plugin.Plugin, bool) {
	rc, ok := r.commands[name]
	if !ok {
		return plugin.Command{}, nil, false
	}
	return rc.cmd, rc.plugin, true
}

// ListCommands returns all registered commands.
func (r *Registry) ListCommands() []plugin.Command {
	cmds := make([]plugin.Command, 0, len(r.commands))
	for _, rc := range r.commands {
		cmds = append(cmds, rc.cmd)
	}
	return cmds
}

// GetPlugin returns a plugin by name.
func (r *Registry) GetPlugin(name string) (plugin.Plugin, bool) {
	p, ok := r.plugins[name]
	return p, ok
}

// ListPlugins returns all registered plugin names.
func (r *Registry) ListPlugins() []string {
	names := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		names = append(names, name)
	}
	return names
}

// ExecuteCommand looks up a command by name and executes it via the owning plugin.
func (r *Registry) ExecuteCommand(name string, args []string) error {
	rc, ok := r.commands[name]
	if !ok {
		return fmt.Errorf("unknown command: %s", name)
	}
	return rc.plugin.Execute(name, args)
}
