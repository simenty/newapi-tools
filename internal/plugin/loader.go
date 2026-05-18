// NewAPI Tools - Docker management platform for newapi
package plugin

import (
	"fmt"
	"os"
	"path/filepath"
)

// Loader discovers and loads plugins from the filesystem.
type Loader struct {
	pluginDir string
	plugins   map[string]Plugin
	ctx       Context
}

// NewLoader creates a new Loader with the given plugins directory.
func NewLoader(pluginDir string, ctx Context) *Loader {
	return &Loader{
		pluginDir: pluginDir,
		plugins:   make(map[string]Plugin),
		ctx:       ctx,
	}
}

// LoadAll scans the plugins directory and loads all discoverable plugins.
// Discovery rules:
//  1. Each subdirectory of pluginDir is a potential plugin
//  2. If it has a metadata.yml and scripts/ directory → ShellPlugin
//  3. Future: if it has a compiled Go plugin → GoPlugin
//  4. GoPlugin takes priority if both exist (future)
func (l *Loader) LoadAll() error {
	entries, err := os.ReadDir(l.pluginDir)
	if err != nil {
		if os.IsNotExist(err) {
			if l.ctx.Logger != nil {
				l.ctx.Logger.Info("plugins directory not found, skipping plugin loading", "dir", l.pluginDir)
			}
			return nil
		}
		return fmt.Errorf("failed to read plugins directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginPath := filepath.Join(l.pluginDir, entry.Name())

		// Check for metadata.yml (required for all plugin types)
		metaPath := filepath.Join(pluginPath, "metadata.yml")
		if _, err := os.Stat(metaPath); os.IsNotExist(err) {
			if l.ctx.Logger != nil {
				l.ctx.Logger.Debug("skipping directory without metadata.yml", "dir", pluginPath)
			}
			continue
		}

		// Check for scripts/ directory → ShellPlugin
		scriptsDir := filepath.Join(pluginPath, "scripts")
		if info, err := os.Stat(scriptsDir); err == nil && info.IsDir() {
			p, err := NewShellPlugin(pluginPath)
			if err != nil {
				if l.ctx.Logger != nil {
					l.ctx.Logger.Error("failed to load shell plugin", "dir", pluginPath, "error", err)
				}
				continue
			}
			if err := p.Init(l.ctx); err != nil {
				if l.ctx.Logger != nil {
					l.ctx.Logger.Error("failed to initialize shell plugin", "name", p.Name(), "error", err)
				}
				continue
			}
			l.plugins[p.Name()] = p
			if l.ctx.Logger != nil {
				l.ctx.Logger.Info("loaded shell plugin", "name", p.Name(), "version", p.Version())
			}
			continue
		}

		// Future: Check for Go plugin binary

		if l.ctx.Logger != nil {
			l.ctx.Logger.Debug("skipping unrecognized plugin directory", "dir", pluginPath)
		}
	}

	return nil
}

// GetPlugin returns a loaded plugin by name.
func (l *Loader) GetPlugin(name string) (Plugin, bool) {
	p, ok := l.plugins[name]
	return p, ok
}

// AllPlugins returns all loaded plugins.
func (l *Loader) AllPlugins() map[string]Plugin {
	return l.plugins
}

// PluginNames returns the names of all loaded plugins.
func (l *Loader) PluginNames() []string {
	names := make([]string, 0, len(l.plugins))
	for name := range l.plugins {
		names = append(names, name)
	}
	return names
}
