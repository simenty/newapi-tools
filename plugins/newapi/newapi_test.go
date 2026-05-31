// NewAPI Tools - Docker management platform for newapi
package newapi

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/simenty/newapi-tools/internal/core"
	"github.com/simenty/newapi-tools/internal/plugin"
)

// TestNewAPIMetadataParses verifies that the newapi metadata.yml is valid.
func TestNewAPIMetadataParses(t *testing.T) {
	// Locate the metadata.yml relative to this test file
	metaPath := filepath.Join("..", "..", "plugins", "newapi")
	// Use the project root as base if running from project root
	if _, err := os.Stat(metaPath); err != nil {
		// Try relative to working directory
		metaPath = "plugins/newapi"
	}

	meta, err := plugin.ParseMetadata(metaPath)
	if err != nil {
		t.Fatalf("ParseMetadata failed: %v", err)
	}

	if meta.Name != "newapi" {
		t.Errorf("expected name 'newapi', got '%s'", meta.Name)
	}
	if meta.Version != "3.0.0" {
		t.Errorf("expected version '3.0.0', got '%s'", meta.Version)
	}
}

// TestNewAPICommandsCount verifies that all expected commands are declared.
func TestNewAPICommandsCount(t *testing.T) {
	metaPath := "plugins/newapi"
	if _, err := os.Stat(metaPath); err != nil {
		metaPath = filepath.Join("..", "..", "plugins", "newapi")
	}

	meta, err := plugin.ParseMetadata(metaPath)
	if err != nil {
		t.Fatalf("ParseMetadata failed: %v", err)
	}

	expectedCommands := []string{"install", "backup", "restore", "update", "status", "config", "doctor"}
	if len(meta.Commands) != len(expectedCommands) {
		t.Fatalf("expected %d commands, got %d", len(expectedCommands), len(meta.Commands))
	}

	for i, expected := range expectedCommands {
		if meta.Commands[i].Name != expected {
			t.Errorf("command[%d]: expected '%s', got '%s'", i, expected, meta.Commands[i].Name)
		}
	}
}

// TestNewAPIShellPluginCreation verifies that a ShellPlugin can be created from the newapi directory.
func TestNewAPIShellPluginCreation(t *testing.T) {
	pluginDir := "plugins/newapi"
	if _, err := os.Stat(pluginDir); err != nil {
		pluginDir = filepath.Join("..", "..", "plugins", "newapi")
	}

	p, err := plugin.NewShellPlugin(pluginDir)
	if err != nil {
		t.Fatalf("NewShellPlugin failed: %v", err)
	}

	if p.Name() != "newapi" {
		t.Errorf("expected plugin name 'newapi', got '%s'", p.Name())
	}
	if p.Version() != "3.0.0" {
		t.Errorf("expected plugin version '3.0.0', got '%s'", p.Version())
	}

	cmds := p.Commands()
	if len(cmds) != 7 {
		t.Fatalf("expected 7 commands, got %d", len(cmds))
	}
}

// TestNewAPIShellPluginInit verifies that the newapi ShellPlugin can be initialized.
func TestNewAPIShellPluginInit(t *testing.T) {
	pluginDir := "plugins/newapi"
	if _, err := os.Stat(pluginDir); err != nil {
		pluginDir = filepath.Join("..", "..", "plugins", "newapi")
	}

	p, err := plugin.NewShellPlugin(pluginDir)
	if err != nil {
		t.Fatalf("NewShellPlugin failed: %v", err)
	}

	ctx := plugin.NewContext(
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

// TestNewAPILoaderIntegration verifies that the Loader can discover the newapi plugin.
func TestNewAPILoaderIntegration(t *testing.T) {
	pluginsDir := "plugins"
	if _, err := os.Stat(pluginsDir); err != nil {
		pluginsDir = filepath.Join("..", "..", "plugins")
	}

	ctx := plugin.NewContext(
		"/opt/newapi",
		3000,
		"calciumion/new-api:latest",
		"/opt/newapi/backups",
		"docker compose",
		slog.Default(),
		"/tmp",
	)

	loader := plugin.NewLoader(pluginsDir, ctx)
	if err := loader.LoadAll(); err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	p, ok := loader.GetPlugin("newapi")
	if !ok {
		t.Fatal("expected to find 'newapi' plugin after LoadAll")
	}
	if p.Name() != "newapi" {
		t.Errorf("expected plugin name 'newapi', got '%s'", p.Name())
	}
	if len(p.Commands()) != 7 {
		t.Errorf("expected 7 commands, got %d", len(p.Commands()))
	}
}

// ---------------------------------------------------------------------------
// Go plugin (compile-time registry) tests
// ---------------------------------------------------------------------------

// TestNewAPIPluginImplementsInterface verifies NewAPIPlugin satisfies plugin.Plugin.
func TestNewAPIPluginImplementsInterface(t *testing.T) {
	var _ plugin.Plugin = &NewAPIPlugin{}
}

// TestNewAPIPluginName verifies the plugin name.
func TestNewAPIPluginName(t *testing.T) {
	p := &NewAPIPlugin{}
	if p.Name() != "newapi" {
		t.Errorf("expected name 'newapi', got '%s'", p.Name())
	}
}

// TestNewAPIPluginVersion verifies the plugin version matches core.Version.
func TestNewAPIPluginVersion(t *testing.T) {
	p := &NewAPIPlugin{}
	if p.Version() != core.Version {
		t.Errorf("expected version '%s', got '%s'", core.Version, p.Version())
	}
}

// TestNewAPIPluginCommands verifies that all 8 commands are registered.
func TestNewAPIPluginCommands(t *testing.T) {
	p := &NewAPIPlugin{}
	cmds := p.Commands()

	expectedNames := []string{"install", "status", "backup", "restore", "update", "doctor", "config", "mirror"}
	if len(cmds) != len(expectedNames) {
		t.Fatalf("expected %d commands, got %d", len(expectedNames), len(cmds))
	}

	for i, expected := range expectedNames {
		if cmds[i].Name != expected {
			t.Errorf("command[%d]: expected '%s', got '%s'", i, expected, cmds[i].Name)
		}
	}
}

// TestNewAPIPluginInit verifies Init returns nil.
func TestNewAPIPluginInit(t *testing.T) {
	p := &NewAPIPlugin{}
	ctx := plugin.NewContext(
		"/opt/newapi", 3000,
		"calciumion/new-api:latest",
		"/opt/newapi/backups",
		"docker compose",
		slog.Default(),
		"/tmp",
	)
	if err := p.Init(ctx); err != nil {
		t.Fatalf("Init should not fail, got: %v", err)
	}
}

// TestNewAPIPluginExecuteReturnsError verifies that Execute returns an error
// (built-in commands are dispatched by the CLI layer, not by Execute).
func TestNewAPIPluginExecuteReturnsError(t *testing.T) {
	p := &NewAPIPlugin{}
	err := p.Execute("install", nil)
	if err == nil {
		t.Error("expected Execute to return an error for built-in plugin")
	}
}

// TestNewAPIPluginShutdown verifies Shutdown returns nil.
func TestNewAPIPluginShutdown(t *testing.T) {
	p := &NewAPIPlugin{}
	if err := p.Shutdown(); err != nil {
		t.Fatalf("Shutdown should not fail, got: %v", err)
	}
}

// TestNewAPIPluginRegisteredInGlobalRegistry verifies that init() has
// registered the plugin in the global compile-time registry.
func TestNewAPIPluginRegisteredInGlobalRegistry(t *testing.T) {
	p, ok := plugin.Get("newapi")
	if !ok {
		t.Fatal("expected 'newapi' to be registered in global plugin registry")
	}
	if p.Name() != "newapi" {
		t.Errorf("expected registered plugin name 'newapi', got '%s'", p.Name())
	}
	if p.Version() != core.Version {
		t.Errorf("expected registered plugin version '%s', got '%s'", core.Version, p.Version())
	}
}

// TestNewAPIPluginAllRegistry verifies that All() includes the newapi plugin.
func TestNewAPIPluginAllRegistry(t *testing.T) {
	all := plugin.All()
	p, ok := all["newapi"]
	if !ok {
		t.Fatal("expected 'newapi' in plugin.All()")
	}
	if p.Version() != core.Version {
		t.Errorf("expected version '%s', got '%s'", core.Version, p.Version())
	}
}
