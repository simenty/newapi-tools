// NewAPI Tools - Docker management platform for newapi
package plugin

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

// ShellPlugin implements the Plugin interface by delegating to shell scripts.
// Each command maps to a .sh file in the plugin's scripts/ directory.
type ShellPlugin struct {
	dir   string
	meta  *Metadata
	ctx   Context
	ready bool
}

// NewShellPlugin creates a ShellPlugin from a plugin directory containing
// a metadata.yml and a scripts/ subdirectory.
func NewShellPlugin(dir string) (*ShellPlugin, error) {
	meta, err := ParseMetadata(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metadata for %s: %w", dir, err)
	}

	return &ShellPlugin{
		dir:   dir,
		meta:  meta,
		ready: false,
	}, nil
}

// Name returns the plugin identifier.
func (p *ShellPlugin) Name() string {
	return p.meta.Name
}

// Version returns the plugin version.
func (p *ShellPlugin) Version() string {
	return p.meta.Version
}

// Commands returns the list of commands this shell plugin provides.
func (p *ShellPlugin) Commands() []Command {
	cmds := make([]Command, 0, len(p.meta.Commands))
	for _, c := range p.meta.Commands {
		cmds = append(cmds, Command{
			Name:        c.Name,
			Description: c.Desc,
			Usage:       c.Help,
		})
	}
	return cmds
}

// Init initializes the shell plugin with the given context.
func (p *ShellPlugin) Init(ctx Context) error {
	p.ctx = ctx
	p.ready = true
	if p.ctx.Logger != nil {
		p.ctx.Logger.Debug("shell plugin initialized", "plugin", p.meta.Name, "dir", p.dir)
	}
	return nil
}

// Execute runs the named command by invoking the corresponding shell script.
func (p *ShellPlugin) Execute(cmd string, args []string) error {
	if !p.ready {
		return fmt.Errorf("plugin %s is not initialized", p.meta.Name)
	}

	scriptPath := p.scriptPath(cmd)

	if _, err := exec.LookPath("bash"); err != nil {
		return fmt.Errorf("bash is required for shell plugin execution: %w", err)
	}

	if p.ctx.Logger != nil {
		p.ctx.Logger.Debug("executing shell command",
			"plugin", p.meta.Name,
			"command", cmd,
			"script", scriptPath,
			"args", args,
		)
	}

	shellCmd := exec.Command("bash", scriptPath)
	shellCmd.Args = append(shellCmd.Args, args...)

	// Pass configuration via environment variables
	shellCmd.Env = append(shellCmd.Environ(),
		fmt.Sprintf("NEWAPI_HOME=%s", p.ctx.NewAPIHome),
		fmt.Sprintf("NEWAPI_PORT=%d", p.ctx.NewAPIPort),
		fmt.Sprintf("NEWAPI_DOCKER_IMAGE=%s", p.ctx.NewAPIDockerImg),
		fmt.Sprintf("NEWAPI_BACKUP_DIR=%s", p.ctx.NewAPIBackupDir),
		fmt.Sprintf("NEWAPI_DOCKER_COMPOSE_CMD=%s", p.ctx.DockerComposeCmd),
	)

	output, err := shellCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("shell command %s/%s failed: %w\noutput: %s",
			p.meta.Name, cmd, err, string(output))
	}

	return nil
}

// Shutdown performs cleanup for the shell plugin (no-op for shell plugins).
func (p *ShellPlugin) Shutdown() error {
	if p.ctx.Logger != nil {
		p.ctx.Logger.Debug("shell plugin shutdown", "plugin", p.meta.Name)
	}
	p.ready = false
	return nil
}

// scriptPath returns the full path to the script file for a given command.
func (p *ShellPlugin) scriptPath(cmd string) string {
	for _, c := range p.meta.Commands {
		if c.Name == cmd {
			if c.Script != "" {
				return filepath.Join(p.dir, c.Script)
			}
		}
	}
	return filepath.Join(p.dir, "scripts", cmd+".sh")
}
