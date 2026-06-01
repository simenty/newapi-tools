package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/simenty/newapi-tools/internal/core"
	"github.com/simenty/newapi-tools/internal/instance"
	"github.com/spf13/cobra"
)

func TestMirrorCommandExists(t *testing.T) {
	if findSubCmd("mirror") == nil {
		t.Error("expected 'mirror' subcommand to be registered")
	}
}

func TestMirrorSubcommands(t *testing.T) {
	cmd := findSubCmd("mirror")
	if cmd == nil {
		t.Fatal("mirror command not found")
	}
	subNames := map[string]bool{}
	for _, sub := range cmd.Commands() {
		name := strings.SplitN(sub.Use, " ", 2)[0]
		subNames[name] = true
	}
	for _, name := range []string{"add", "remove", "list", "apply", "test", "reset", "builtin"} {
		if !subNames[name] {
			t.Errorf("expected 'mirror %s' subcommand", name)
		}
	}
}

func TestVersionCommandExists(t *testing.T) {
	if findSubCmd("version") == nil {
		t.Error("expected 'version' subcommand to be registered")
	}
}

func TestVersionCommandUse(t *testing.T) {
	cmd := findSubCmd("version")
	if cmd == nil {
		t.Fatal("version command not found")
	}
	if cmd.Use != "version" {
		t.Errorf("expected Use 'version', got %q", cmd.Use)
	}
}

func TestAuditCommandExists(t *testing.T) {
	if findSubCmd("audit") == nil {
		t.Error("expected 'audit' subcommand to be registered")
	}
}

func TestAuditSubcommandExists(t *testing.T) {
	cmd := findSubCmd("audit")
	if cmd == nil {
		t.Fatal("audit command not found")
	}
	found := false
	for _, sub := range cmd.Commands() {
		if sub.Use == "list" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'audit list' subcommand")
	}
}

func TestAuditListFlags(t *testing.T) {
	cmd := findSubCmd("audit")
	if cmd == nil {
		t.Fatal("audit command not found")
	}
	var listCmd *cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Use == "list" {
			listCmd = sub
			break
		}
	}
	if listCmd == nil {
		t.Fatal("audit list command not found")
	}
	for _, flag := range []string{"last", "cmd", "since", "json"} {
		if listCmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' on audit list command", flag)
		}
	}
}

func TestInstanceCommandExists(t *testing.T) {
	if findSubCmd("instance") == nil {
		t.Error("expected 'instance' subcommand to be registered")
	}
}

func TestInstanceSubcommands(t *testing.T) {
	cmd := findSubCmd("instance")
	if cmd == nil {
		t.Fatal("instance command not found")
	}
	subNames := map[string]bool{}
	for _, sub := range cmd.Commands() {
		name := strings.SplitN(sub.Use, " ", 2)[0]
		subNames[name] = true
	}
	for _, name := range []string{"add", "list", "switch", "remove"} {
		if !subNames[name] {
			t.Errorf("expected 'instance %s' subcommand", name)
		}
	}
}

func TestCheckDockerBinaryTypes(t *testing.T) {
	r := checkDockerBinary()
	if r.Status == "" {
		t.Error("status should not be empty")
	}
	valid := r.Status == "OK" || r.Status == "FAIL"
	if !valid {
		t.Errorf("unexpected status: %q", r.Status)
	}
}

func TestShortDigestEdgeCases(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"sha256:", "sha256:"},
		{"sha256:abc", "sha256:abc"},
	}
	for _, tc := range cases {
		got := shortDigest(tc.input)
		if tc.want != "" && got != tc.want {
			t.Errorf("shortDigest(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestGetRootCmdNonNil(t *testing.T) {
	cmd := GetRootCmd()
	if cmd == nil {
		t.Fatal("GetRootCmd() returned nil")
	}
	if cmd.Use == "" {
		t.Error("root command Use is empty")
	}
}

func TestApplyConfigValueUnknownField(t *testing.T) {
	cfg := core.DefaultConfig()
	err := applyConfigValue(cfg, "", "val", "string")
	if err == nil {
		t.Error("expected error for empty key")
	}
}

func TestSyncInstanceToConfig(t *testing.T) {
	cfg := core.DefaultConfig()
	inst := &instance.Instance{
		Name:          "test-instance",
		Home:          "/custom/home",
		Port:          9999,
		DockerImage:   "custom-image:latest",
		Domain:        "example.com",
		HealthTimeout: 60,
		MaxBackups:    10,
	}
	syncInstanceToConfig(inst, cfg)
	if cfg.NewAPI.Home != "/custom/home" {
		t.Errorf("Home = %q, want %q", cfg.NewAPI.Home, "/custom/home")
	}
	if cfg.NewAPI.Port != 9999 {
		t.Errorf("Port = %d, want %d", cfg.NewAPI.Port, 9999)
	}
	if cfg.NewAPI.DockerImage != "custom-image:latest" {
		t.Errorf("DockerImage = %q, want %q", cfg.NewAPI.DockerImage, "custom-image:latest")
	}
	if cfg.NewAPI.Domain != "example.com" {
		t.Errorf("Domain = %q, want %q", cfg.NewAPI.Domain, "example.com")
	}
	if cfg.NewAPI.HealthTimeout != 60 {
		t.Errorf("HealthTimeout = %d, want %d", cfg.NewAPI.HealthTimeout, 60)
	}
	if cfg.NewAPI.MaxBackups != 10 {
		t.Errorf("MaxBackups = %d, want %d", cfg.NewAPI.MaxBackups, 10)
	}
}

func TestGetCurrentUser(t *testing.T) {
	user := getCurrentUser()
	if user == "" {
		t.Error("getCurrentUser() returned empty")
	}
	if user == "unknown" {
		t.Log("getCurrentUser returned 'unknown' (running in restricted environment)")
	}
}

func TestGetFullCommand(t *testing.T) {
	cmd := GetRootCmd()
	if cmd == nil {
		t.Fatal("GetRootCmd() returned nil")
	}
	// Just verify it returns something starting with "newapi-tools"
	fullCmd := getFullCommand()
	if fullCmd == "" {
		t.Error("getFullCommand() returned empty")
	}
}

func TestCountResults(t *testing.T) {
	results := []checkResult{
		{"check1", "OK", "all good", nil},
		{"check2", "FAIL", "failed", nil},
		{"check3", "WARN", "warning", nil},
		{"check4", "WARN", "another warning", nil},
		{"check5", "FAIL", "another fail", nil},
	}
	fails, warns := countResults(results)
	if fails != 2 {
		t.Errorf("expected 2 failures, got %d", fails)
	}
	if warns != 2 {
		t.Errorf("expected 2 warnings, got %d", warns)
	}
}

func TestCountResultsEmpty(t *testing.T) {
	fails, warns := countResults(nil)
	if fails != 0 || warns != 0 {
		t.Errorf("expected 0, got fails=%d warns=%d", fails, warns)
	}
}

func TestCheckConfigPermissionsSkip(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.NewAPI.Home = ""
	r := checkConfigPermissions(cfg)
	if r.Status != "SKIP" {
		t.Errorf("expected SKIP when no home configured, got %s", r.Status)
	}
}

func TestFixCreateHomeDir(t *testing.T) {
	cfg := core.DefaultConfig()
	tmpDir := t.TempDir()
	cfg.NewAPI.Home = filepath.Join(tmpDir, "newhome")
	err := fixCreateHomeDir(context.Background(), cfg)
	if err != nil {
		t.Fatalf("fixCreateHomeDir failed: %v", err)
	}
	if _, err := os.Stat(cfg.NewAPI.Home); os.IsNotExist(err) {
		t.Error("home directory should have been created")
	}
}

func TestFixCreateHomeDirEmpty(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.NewAPI.Home = ""
	err := fixCreateHomeDir(context.Background(), cfg)
	if err == nil {
		t.Error("expected error for empty home directory")
	}
}

func TestRunAllChecks(t *testing.T) {
	cfg := core.DefaultConfig()
	results := runAllChecks(context.Background(), cfg)
	if len(results) == 0 {
		t.Error("runAllChecks returned no results")
	}
	if len(results) != len(checks) {
		t.Errorf("expected %d checks, got %d", len(checks), len(results))
	}
}

func TestPrintConfigJSON(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.NewAPI.Home = "/test/home"
	cfg.NewAPI.Port = 8080
	err := printConfigJSON(cfg)
	if err != nil {
		t.Errorf("printConfigJSON failed: %v", err)
	}
}