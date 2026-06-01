package registry

import (
	"log/slog"
	"testing"

	"github.com/simenty/newapi-tools/internal/plugin"
)

// mockPlugin implements plugin.Plugin for testing.
type mockPlugin struct {
	name     string
	version  string
	commands []plugin.Command
	initErr  error
	executeFn func(cmd string, args []string) error
}

func (m *mockPlugin) Name() string                       { return m.name }
func (m *mockPlugin) Version() string                    { return m.version }
func (m *mockPlugin) Commands() []plugin.Command         { return m.commands }
func (m *mockPlugin) Init(ctx plugin.Context) error       { return m.initErr }
func (m *mockPlugin) Execute(cmd string, args []string) error {
	if m.executeFn != nil {
		return m.executeFn(cmd, args)
	}
	return nil
}
func (m *mockPlugin) Shutdown() error                    { return nil }

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry() returned nil")
	}
	cmds := r.ListCommands()
	if len(cmds) != 0 {
		t.Errorf("expected 0 commands, got %d", len(cmds))
	}
	plugins := r.ListPlugins()
	if len(plugins) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(plugins))
	}
}

func TestRegisterPlugin(t *testing.T) {
	logger := slog.Default()
	r := NewRegistry()

	p := &mockPlugin{
		name:    "test-plugin",
		version: "1.0.0",
		commands: []plugin.Command{
			{Name: "hello", Description: "says hello", Usage: "hello"},
			{Name: "ping", Description: "pings", Usage: "ping"},
		},
	}

	if err := r.RegisterPlugin(p, logger); err != nil {
		t.Fatalf("RegisterPlugin failed: %v", err)
	}

	// Check plugin is registered
	retrieved, ok := r.GetPlugin("test-plugin")
	if !ok {
		t.Fatal("plugin not found after registration")
	}
	if retrieved.Name() != "test-plugin" {
		t.Errorf("expected name 'test-plugin', got %q", retrieved.Name())
	}

	// Check commands
	cmds := r.ListCommands()
	if len(cmds) != 2 {
		t.Errorf("expected 2 commands, got %d", len(cmds))
	}

	// Check GetCommand
	cmd, p2, ok := r.GetCommand("hello")
	if !ok {
		t.Fatal("command 'hello' not found")
	}
	if cmd.Name != "hello" {
		t.Errorf("expected command name 'hello', got %q", cmd.Name)
	}
	if p2.Name() != "test-plugin" {
		t.Errorf("expected plugin name 'test-plugin', got %q", p2.Name())
	}
}

func TestRegisterPluginDuplicate(t *testing.T) {
	logger := slog.Default()
	r := NewRegistry()

	p1 := &mockPlugin{name: "dup-plugin", version: "1.0.0"}
	if err := r.RegisterPlugin(p1, logger); err != nil {
		t.Fatalf("first RegisterPlugin failed: %v", err)
	}

	p2 := &mockPlugin{name: "dup-plugin", version: "2.0.0"}
	err := r.RegisterPlugin(p2, logger)
	if err == nil {
		t.Error("expected error when registering duplicate plugin")
	} else if err.Error() != `plugin "dup-plugin" already registered` {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRegisterPluginDuplicateCommand(t *testing.T) {
	logger := slog.Default()
	r := NewRegistry()

	p1 := &mockPlugin{
		name:    "plugin-a",
		version: "1.0.0",
		commands: []plugin.Command{
			{Name: "shared-cmd", Description: "from plugin-a"},
		},
	}
	if err := r.RegisterPlugin(p1, logger); err != nil {
		t.Fatalf("RegisterPlugin plugin-a failed: %v", err)
	}

	p2 := &mockPlugin{
		name:    "plugin-b",
		version: "1.0.0",
		commands: []plugin.Command{
			{Name: "shared-cmd", Description: "from plugin-b (should be skipped)"},
			{Name: "unique-cmd", Description: "from plugin-b"},
		},
	}
	if err := r.RegisterPlugin(p2, logger); err != nil {
		t.Fatalf("RegisterPlugin plugin-b failed: %v", err)
	}

	// Only the first registration of shared-cmd should be retained
	cmd, p, ok := r.GetCommand("shared-cmd")
	if !ok {
		t.Fatal("shared-cmd not found")
	}
	if p.Name() != "plugin-a" {
		t.Errorf("expected owner 'plugin-a', got %q", p.Name())
	}
	if cmd.Description != "from plugin-a" {
		t.Errorf("expected description 'from plugin-a', got %q", cmd.Description)
	}

	// unique-cmd should be present
	if _, _, ok := r.GetCommand("unique-cmd"); !ok {
		t.Error("unique-cmd should be registered")
	}
}

func TestGetCommandNotFound(t *testing.T) {
	r := NewRegistry()
	_, _, ok := r.GetCommand("nonexistent")
	if ok {
		t.Error("expected false for nonexistent command")
	}
}

func TestGetPluginNotFound(t *testing.T) {
	r := NewRegistry()
	_, ok := r.GetPlugin("nonexistent")
	if ok {
		t.Error("expected false for nonexistent plugin")
	}
}

func TestListPlugins(t *testing.T) {
	logger := slog.Default()
	r := NewRegistry()

	r.RegisterPlugin(&mockPlugin{name: "alpha", version: "1.0.0"}, logger)
	r.RegisterPlugin(&mockPlugin{name: "beta", version: "2.0.0"}, logger)

	names := r.ListPlugins()
	if len(names) != 2 {
		t.Errorf("expected 2 plugins, got %d", len(names))
	}
	seen := make(map[string]bool)
	for _, n := range names {
		seen[n] = true
	}
	if !seen["alpha"] || !seen["beta"] {
		t.Errorf("unexpected plugin list: %v", names)
	}
}

func TestExecuteCommand(t *testing.T) {
	logger := slog.Default()
	r := NewRegistry()

	executed := false
	p := &mockPlugin{
		name:    "exec-plugin",
		version: "1.0.0",
		commands: []plugin.Command{
			{Name: "run", Description: "runs"},
		},
		executeFn: func(cmd string, args []string) error {
			executed = true
			if cmd != "run" {
				t.Errorf("expected cmd 'run', got %q", cmd)
			}
			if len(args) != 1 || args[0] != "arg1" {
				t.Errorf("expected args ['arg1'], got %v", args)
			}
			return nil
		},
	}

	r.RegisterPlugin(p, logger)

	if err := r.ExecuteCommand("run", []string{"arg1"}); err != nil {
		t.Fatalf("ExecuteCommand failed: %v", err)
	}
	if !executed {
		t.Error("Execute was not called")
	}
}

func TestExecuteCommandUnknown(t *testing.T) {
	r := NewRegistry()
	err := r.ExecuteCommand("unknown", nil)
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
	if err.Error() != "unknown command: unknown" {
		t.Errorf("unexpected error: %v", err)
	}
}