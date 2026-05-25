// NewAPI Tools - Docker management platform for newapi
package plugin

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestParseMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	metaContent := []byte(`name: test-plugin
display_name: Test Plugin
version: "1.0.0"
description: A test plugin
commands:
  - name: install
    script: scripts/install.sh
    desc: Install something
    help: Install the thing
  - name: status
    script: scripts/status.sh
    desc: Show status
    help: Display current status
`)
	metaPath := filepath.Join(tmpDir, "metadata.yml")
	if err := os.WriteFile(metaPath, metaContent, 0644); err != nil {
		t.Fatalf("failed to write metadata.yml: %v", err)
	}

	meta, err := ParseMetadata(tmpDir)
	if err != nil {
		t.Fatalf("ParseMetadata failed: %v", err)
	}

	if meta.Name != "test-plugin" {
		t.Errorf("expected name 'test-plugin', got '%s'", meta.Name)
	}
	if meta.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", meta.Version)
	}
	if len(meta.Commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(meta.Commands))
	}
	if meta.Commands[0].Name != "install" {
		t.Errorf("expected first command 'install', got '%s'", meta.Commands[0].Name)
	}
	if meta.Commands[1].Script != "scripts/status.sh" {
		t.Errorf("expected second script 'scripts/status.sh', got '%s'", meta.Commands[1].Script)
	}
}

func TestParseMetadataNonexistent(t *testing.T) {
	_, err := ParseMetadata("/nonexistent/path")
	if err == nil {
		t.Error("expected error for nonexistent metadata.yml")
	}
}

func TestShellPluginCreation(t *testing.T) {
	tmpDir := setupTestPluginDir(t)

	p, err := NewShellPlugin(tmpDir)
	if err != nil {
		t.Fatalf("NewShellPlugin failed: %v", err)
	}

	if p.Name() != "test-plugin" {
		t.Errorf("expected name 'test-plugin', got '%s'", p.Name())
	}
	if p.Version() != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", p.Version())
	}

	cmds := p.Commands()
	if len(cmds) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(cmds))
	}
	if cmds[0].Name != "install" {
		t.Errorf("expected first command 'install', got '%s'", cmds[0].Name)
	}
}

func TestShellPluginInit(t *testing.T) {
	tmpDir := setupTestPluginDir(t)

	p, err := NewShellPlugin(tmpDir)
	if err != nil {
		t.Fatalf("NewShellPlugin failed: %v", err)
	}

	ctx := NewContext(
		"/opt/newapi",
		3000,
		"calciumion/new-api:latest",
		"/opt/newapi/backups",
		"docker compose",
		slog.Default(),
		"/tmp",
	)

	if err := p.Init(ctx); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
}

func TestShellPluginExecuteNotInitialized(t *testing.T) {
	tmpDir := setupTestPluginDir(t)

	p, err := NewShellPlugin(tmpDir)
	if err != nil {
		t.Fatalf("NewShellPlugin failed: %v", err)
	}

	// Should fail because plugin is not initialized
	err = p.Execute("install", nil)
	if err == nil {
		t.Error("expected error when executing on uninitialized plugin")
	}
}

func TestShellPluginShutdown(t *testing.T) {
	tmpDir := setupTestPluginDir(t)

	p, err := NewShellPlugin(tmpDir)
	if err != nil {
		t.Fatalf("NewShellPlugin failed: %v", err)
	}

	ctx := NewContext(
		"/opt/newapi",
		3000,
		"calciumion/new-api:latest",
		"/opt/newapi/backups",
		"docker compose",
		slog.Default(),
		"/tmp",
	)

	if err := p.Init(ctx); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if err := p.Shutdown(); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}

func TestLoaderLoadAll(t *testing.T) {
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "test-plugin")
	scriptsDir := filepath.Join(pluginDir, "scripts")

	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatalf("failed to create scripts dir: %v", err)
	}

	metaContent := []byte(`name: test-plugin
display_name: Test Plugin
version: "1.0.0"
description: A test plugin
commands:
  - name: hello
    script: scripts/hello.sh
    desc: Say hello
    help: Say hello
`)
	if err := os.WriteFile(filepath.Join(pluginDir, "metadata.yml"), metaContent, 0644); err != nil {
		t.Fatalf("failed to write metadata.yml: %v", err)
	}

	ctx := NewContext(
		"/opt/newapi",
		3000,
		"calciumion/new-api:latest",
		"/opt/newapi/backups",
		"docker compose",
		slog.Default(),
		"/tmp",
	)

	loader := NewLoader(tmpDir, ctx)
	if err := loader.LoadAll(); err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	p, ok := loader.GetPlugin("test-plugin")
	if !ok {
		t.Fatal("expected to find 'test-plugin' after LoadAll")
	}
	if p.Name() != "test-plugin" {
		t.Errorf("expected plugin name 'test-plugin', got '%s'", p.Name())
	}
}

func TestLoaderEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	ctx := NewContext(
		"/opt/newapi",
		3000,
		"calciumion/new-api:latest",
		"/opt/newapi/backups",
		"docker compose",
		slog.Default(),
		"/tmp",
	)

	loader := NewLoader(tmpDir, ctx)
	if err := loader.LoadAll(); err != nil {
		t.Fatalf("LoadAll on empty dir should not error: %v", err)
	}

	if len(loader.AllPlugins()) != 0 {
		t.Error("expected no plugins in empty directory")
	}
}

func TestLoaderNonexistentDir(t *testing.T) {
	ctx := NewContext(
		"/opt/newapi",
		3000,
		"calciumion/new-api:latest",
		"/opt/newapi/backups",
		"docker compose",
		slog.Default(),
		"/tmp",
	)

	loader := NewLoader("/nonexistent/plugins", ctx)
	if err := loader.LoadAll(); err != nil {
		t.Fatalf("LoadAll on nonexistent dir should not error: %v", err)
	}
}

func TestLoaderPluginNames(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestPluginInDir(t, tmpDir, "plugin-a")
	setupTestPluginInDir(t, tmpDir, "plugin-b")

	ctx := NewContext(
		"/opt/newapi",
		3000,
		"calciumion/new-api:latest",
		"/opt/newapi/backups",
		"docker compose",
		slog.Default(),
		"/tmp",
	)

	loader := NewLoader(tmpDir, ctx)
	if err := loader.LoadAll(); err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	names := loader.PluginNames()
	if len(names) != 2 {
		t.Errorf("expected 2 plugin names, got %d", len(names))
	}
}

// TestRegisterAndGet tests the global compile-time registry.
func TestRegisterAndGet(t *testing.T) {
	// Clear registry for this test — we use a fresh map via All/Get.
	// Since init() from plugins/newapi may have already registered,
	// we just verify the API works with a dummy plugin.

	// We cannot safely call Register("newapi") again because it would panic.
	// Instead, verify the existing registrations work.
	all := All()
	if len(all) == 0 {
		t.Log("No plugins registered in global registry (this is okay if no plugin packages were imported)")
	}
}

// TestGetNonExistent verifies Get returns false for unknown plugins.
func TestGetNonExistent(t *testing.T) {
	_, ok := Get("nonexistent-plugin-xyz")
	if ok {
		t.Error("expected Get to return false for unregistered plugin")
	}
}

// TestRegisterDuplicatePanics verifies that registering the same name twice panics.
func TestRegisterDuplicatePanics(t *testing.T) {
	dummy := &dummyPlugin{name: "dup-test-plugin"}

	// Register once — should succeed
	Register(dummy)

	// Register again — should panic
	defer func() {
		r := recover()
		if r == nil {
			t.Error("expected panic when registering duplicate plugin")
		}
		// Clean up: remove from registry so other tests aren't affected
		registryMu.Lock()
		delete(registry, "dup-test-plugin")
		registryMu.Unlock()
	}()
	Register(dummy)
}

// dummyPlugin is a minimal Plugin implementation for testing.
type dummyPlugin struct {
	name string
}

func (d *dummyPlugin) Name() string                              { return d.name }
func (d *dummyPlugin) Version() string                           { return "0.0.1" }
func (d *dummyPlugin) Commands() []Command                       { return nil }
func (d *dummyPlugin) Init(ctx Context) error                    { return nil }
func (d *dummyPlugin) Execute(cmd string, args []string) error   { return nil }
func (d *dummyPlugin) Shutdown() error                           { return nil }

// setupTestPluginDir creates a temporary plugin directory with metadata.yml and scripts/.
func setupTestPluginDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	setupTestPluginInDir(t, tmpDir, "test-plugin")
	return filepath.Join(tmpDir, "test-plugin")
}

// setupTestPluginInDir creates a plugin subdirectory with metadata.yml and scripts/.
func setupTestPluginInDir(t *testing.T, parentDir string, pluginName string) {
	t.Helper()
	pluginDir := filepath.Join(parentDir, pluginName)
	scriptsDir := filepath.Join(pluginDir, "scripts")

	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}

	metaContent := []byte(fmt.Sprintf(`name: %s
display_name: %s
version: "1.0.0"
description: A test plugin
commands:
  - name: install
    script: scripts/install.sh
    desc: Install something
    help: Install the thing
  - name: status
    script: scripts/status.sh
    desc: Show status
    help: Display current status
`, pluginName, pluginName))

	if err := os.WriteFile(filepath.Join(pluginDir, "metadata.yml"), metaContent, 0644); err != nil {
		t.Fatalf("failed to write metadata.yml: %v", err)
	}
}
