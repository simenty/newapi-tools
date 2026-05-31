// NewAPI Tools - Docker management platform for newapi
// Package newapi provides the built-in newapi plugin.
//
// The actual command logic lives in internal/cli/ — this package only
// registers the plugin with the compile-time registry so that the
// Loader and Registry can discover it automatically.
package newapi

import (
	"fmt"

	"github.com/simenty/newapi-tools/internal/core"
	"github.com/simenty/newapi-tools/internal/plugin"
)

// NewAPIPlugin implements plugin.Plugin for the built-in newapi commands.
// Command execution is delegated to the CLI layer; this struct is mainly
// a registration vehicle that makes the plugin visible to the framework.
type NewAPIPlugin struct{}

// Verify the interface at compile time.
var _ plugin.Plugin = (*NewAPIPlugin)(nil)

func (p *NewAPIPlugin) Name() string    { return "newapi" }
func (p *NewAPIPlugin) Version() string { return core.Version }

func (p *NewAPIPlugin) Commands() []plugin.Command {
	return []plugin.Command{
		{Name: "install", Description: "安装 NewAPI", Usage: "newapi install [flags]"},
		{Name: "status", Description: "查看 NewAPI 运行状态", Usage: "newapi status"},
		{Name: "backup", Description: "备份 NewAPI 数据", Usage: "newapi backup [flags]"},
		{Name: "restore", Description: "从备份恢复 NewAPI", Usage: "newapi restore [flags]"},
		{Name: "update", Description: "更新 NewAPI 到最新版本", Usage: "newapi update [flags]"},
		{Name: "doctor", Description: "诊断 NewAPI 部署问题", Usage: "newapi doctor"},
		{Name: "config", Description: "管理 NewAPI 配置", Usage: "newapi config [key] [value]"},
		{Name: "mirror", Description: "管理 Docker 镜像源", Usage: "newapi mirror [command]"},
	}
}

// Init is called when the plugin is loaded into the runtime.
// The built-in newapi plugin does not need extra initialisation because
// the CLI layer handles flag parsing and dependency wiring independently.
func (p *NewAPIPlugin) Init(ctx plugin.Context) error {
	return nil
}

// Execute dispatches a command.  For the built-in plugin the real work is
// done by internal/cli, so this method returns a hint directing the caller
// to use the CLI path instead.  External (non-built-in) plugins that
// implement Execute with real logic will override this behaviour.
func (p *NewAPIPlugin) Execute(cmd string, args []string) error {
	return fmt.Errorf("builtin plugin %q: command %q is handled by the CLI layer", p.Name(), cmd)
}

// Shutdown is a no-op for the built-in plugin.
func (p *NewAPIPlugin) Shutdown() error { return nil }

// init registers the plugin with the global compile-time registry.
// Adding `_ "github.com/Bonus520/newapi-tools/plugins/newapi"` to the
// main package imports is sufficient to trigger this function.
func init() {
	plugin.Register(&NewAPIPlugin{})
}
