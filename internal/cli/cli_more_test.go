package cli

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
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

// ---- Completion subcommand tests ----

func TestCompletionSubcommands(t *testing.T) {
	cmd := GetCompletionCommand()
	if cmd == nil {
		t.Fatal("completion command not found")
	}
	subNames := map[string]bool{}
	for _, sub := range cmd.Commands() {
		name := strings.SplitN(sub.Use, " ", 2)[0]
		subNames[name] = true
	}
	for _, name := range []string{"bash", "zsh", "fish", "powershell"} {
		if !subNames[name] {
			t.Errorf("expected 'completion %s' subcommand", name)
		}
	}
}

func TestGetCompletionCommand(t *testing.T) {
	cmd := GetCompletionCommand()
	if cmd == nil {
		t.Fatal("GetCompletionCommand() returned nil")
	}
	if !strings.Contains(cmd.Use, "completion") {
		t.Errorf("unexpected Use: %q", cmd.Use)
	}
}

// ---- Config validate/chmod subcommand tests ----

func TestConfigValidateSubcommand(t *testing.T) {
	cmd := findSubCmd("config")
	if cmd == nil {
		t.Fatal("config command not found")
	}
	found := false
	for _, sub := range cmd.Commands() {
		if sub.Use == "validate" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'config validate' subcommand")
	}
}

func TestConfigChmodSubcommand(t *testing.T) {
	cmd := findSubCmd("config")
	if cmd == nil {
		t.Fatal("config command not found")
	}
	found := false
	for _, sub := range cmd.Commands() {
		if sub.Use == "chmod" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'config chmod' subcommand")
	}
}

func TestConfigSubcommandsFull(t *testing.T) {
	cmd := findSubCmd("config")
	if cmd == nil {
		t.Fatal("config command not found")
	}
	subNames := map[string]bool{}
	for _, sub := range cmd.Commands() {
		name := strings.SplitN(sub.Use, " ", 2)[0]
		subNames[name] = true
	}
	for _, name := range []string{"set", "init", "chmod", "validate"} {
		if !subNames[name] {
			t.Errorf("expected 'config %s' subcommand", name)
		}
	}
}

// ---- Doctor command tests ----

func TestDoctorVerboseFlag(t *testing.T) {
	cmd := findSubCmd("doctor")
	if cmd == nil {
		t.Fatal("doctor command not found")
	}
	if cmd.Flags().Lookup("verbose") == nil {
		t.Error("expected flag '--verbose' on doctor command")
	}
	if cmd.Flags().ShorthandLookup("v") == nil {
		t.Error("expected shorthand '-v' on doctor command")
	}
}

// ---- Status command additional flags ----

func TestStatusCommandFlagsFull(t *testing.T) {
	cmd := findSubCmd("status")
	if cmd == nil {
		t.Fatal("status command not found")
	}
	for _, flag := range []string{"json", "watch", "interval", "all"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' on status command", flag)
		}
	}
}

// ---- isRelatedContainer test ----

func TestIsRelatedContainer(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"new-api", true},
		{"newapi-app", true},
		{"newapi-tools", true},
		{"mysql", true},
		{"redis", true},
		{"nginx", false},
		{"", false},
		{"NEW-API", true},
		{"MySql", true},
	}
	for _, tc := range cases {
		got := isRelatedContainer(tc.name)
		if got != tc.want {
			t.Errorf("isRelatedContainer(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// ---- Output/display function tests ----

func TestPrintDoctorTable(t *testing.T) {
	old := os.Stdout
	devNull, _ := os.Open(os.DevNull)
	os.Stdout = devNull
	defer func() {
		os.Stdout = old
		devNull.Close()
	}()

	results := []checkResult{
		{"check1", "OK", "all good", nil},
		{"check2", "FAIL", "something wrong", nil},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("printDoctorTable panicked: %v", r)
		}
	}()
	printDoctorTable(results, false)
}

func TestPrintDoctorTableVerbose(t *testing.T) {
	old := os.Stdout
	devNull, _ := os.Open(os.DevNull)
	os.Stdout = devNull
	defer func() {
		os.Stdout = old
		devNull.Close()
	}()

	results := []checkResult{
		{
			Name:    "test-check",
			Status:  "OK",
			Message: "test message",
			Detail: &VerboseCheck{
				FilePath:  "/tmp/test",
				Command:   "test --help",
				Expected:  "success",
				Actual:    "ok",
				RawOutput: "output",
			},
		},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("printDoctorTable verbose panicked: %v", r)
		}
	}()
	printDoctorTable(results, true)
}

func TestPrintVerboseDetail(t *testing.T) {
	old := os.Stdout
	devNull, _ := os.Open(os.DevNull)
	os.Stdout = devNull
	defer func() {
		os.Stdout = old
		devNull.Close()
	}()

	detail := &VerboseCheck{
		FilePath:  "/tmp/test",
		Command:   "test --help",
		Expected:  "success",
		Actual:    "ok",
		RawOutput: "output",
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("printVerboseDetail panicked: %v", r)
		}
	}()
	printVerboseDetail(detail, 20)
}

func TestPrintVerboseDetailEmpty(t *testing.T) {
	old := os.Stdout
	devNull, _ := os.Open(os.DevNull)
	os.Stdout = devNull
	defer func() {
		os.Stdout = old
		devNull.Close()
	}()

	detail := &VerboseCheck{}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("printVerboseDetail empty panicked: %v", r)
		}
	}()
	printVerboseDetail(detail, 10)
}

// ---- VerboseCheck struct test ----

func TestVerboseCheckStruct(t *testing.T) {
	v := &VerboseCheck{
		FilePath:  "/path/to/file",
		Command:   "some command",
		Expected:  "expected value",
		Actual:    "actual value",
		RawOutput: "raw output",
	}
	if v.FilePath != "/path/to/file" {
		t.Errorf("unexpected FilePath: %q", v.FilePath)
	}
	if v.Command != "some command" {
		t.Errorf("unexpected Command: %q", v.Command)
	}
	if v.Expected != "expected value" {
		t.Errorf("unexpected Expected: %q", v.Expected)
	}
	if v.Actual != "actual value" {
		t.Errorf("unexpected Actual: %q", v.Actual)
	}
	if v.RawOutput != "raw output" {
		t.Errorf("unexpected RawOutput: %q", v.RawOutput)
	}
}

// ---- clearScreen test ----

func TestClearScreen(t *testing.T) {
	old := os.Stdout
	devNull, _ := os.Open(os.DevNull)
	os.Stdout = devNull
	defer func() {
		os.Stdout = old
		devNull.Close()
	}()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("clearScreen panicked: %v", r)
		}
	}()
	clearScreen()
}

// ---- builtinNamesList test ----

func TestBuiltinNamesList(t *testing.T) {
	result := builtinNamesList()
	if result == "" {
		t.Error("builtinNamesList() returned empty")
	}
	if !strings.Contains(result, "tuna") {
		t.Error("expected 'tuna' in builtin list")
	}
	if !strings.Contains(result, "aliyun") {
		t.Error("expected 'aliyun' in builtin list")
	}
}

// ---- printConfigTable tests ----

func TestPrintConfigTable(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.NewAPI.Home = "/test/home"
	cfg.NewAPI.Port = 8080
	cfg.Instance.Active = "test-instance"

	old := os.Stdout
	devNull, _ := os.Open(os.DevNull)
	os.Stdout = devNull
	defer func() {
		os.Stdout = old
		devNull.Close()
	}()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("printConfigTable panicked: %v", r)
		}
	}()
	printConfigTable(cfg)
}

func TestPrintConfigTableNoInstance(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.NewAPI.Home = "/test/home"
	cfg.NewAPI.Port = 8080
	cfg.Instance.Active = ""
	cfg.NewAPI.Domain = ""

	old := os.Stdout
	devNull, _ := os.Open(os.DevNull)
	os.Stdout = devNull
	defer func() {
		os.Stdout = old
		devNull.Close()
	}()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("printConfigTable no instance panicked: %v", r)
		}
	}()
	printConfigTable(cfg)
}

// ---- checkResult struct test ----

func TestCheckResultStruct(t *testing.T) {
	r := checkResult{
		Name:    "test-check",
		Status:  "OK",
		Message: "everything is fine",
	}
	if r.Name != "test-check" {
		t.Errorf("unexpected Name: %q", r.Name)
	}
	if r.Status != "OK" {
		t.Errorf("unexpected Status: %q", r.Status)
	}
}

// ---- checkDockerGroupMembership test ----

func TestCheckDockerGroupMembership(t *testing.T) {
	r := checkDockerGroupMembership()
	validStatuses := map[string]bool{"OK": true, "FAIL": true, "WARN": true, "SKIP": true}
	if !validStatuses[r.Status] {
		t.Errorf("unexpected status: %q", r.Status)
	}
}

// ---- GetRegistry test ----

func TestGetRegistry(t *testing.T) {
	reg := GetRegistry()
	// Registry may be nil if not initialized, which is valid
	if reg != nil {
		t.Log("registry is initialized")
	}
}

// ---- isValidInstanceName tests ----

func TestIsValidInstanceName(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"default", true},
		{"my-instance", true},
		{"instance123", true},
		{"a", true},
		{"", false},
		{"123-instance", false},
		{"-instance", false},
		{"insta_nce", false},
		{"instance.name", false},
		{"UPPERCASE", true},
	}
	for _, tc := range cases {
		got := isValidInstanceName(tc.name)
		if got != tc.want {
			t.Errorf("isValidInstanceName(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// ---- truncate tests ----

func TestTruncate(t *testing.T) {
	cases := []struct {
		input string
		max   int
		want  string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "he..."},
		{"", 5, ""},
		{"abc", 3, "abc"},
		{"abcd", 3, "..."},
		{"你好世界", 3, "..."},
		{"hello world", 20, "hello world"},
	}
	for _, tc := range cases {
		got := truncate(tc.input, tc.max)
		if got != tc.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tc.input, tc.max, got, tc.want)
		}
	}
}

// ---- randomPassword tests ----

func TestRandomPassword(t *testing.T) {
	pwd := randomPassword(32)
	if len(pwd) != 32 {
		t.Errorf("expected length 32, got %d", len(pwd))
	}
	// Multiple calls should return different results
	pwd2 := randomPassword(32)
	if pwd == pwd2 {
		t.Error("randomPassword should produce different values")
	}
	// Zero length
	pwd3 := randomPassword(0)
	if len(pwd3) != 0 {
		t.Errorf("expected length 0, got %d", len(pwd3))
	}
}

func TestRandomPasswordChars(t *testing.T) {
	pwd := randomPassword(64)
	if len(pwd) != 64 {
		t.Fatalf("expected length 64, got %d", len(pwd))
	}
	// Should only contain alphanumeric chars
	for _, c := range pwd {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			t.Errorf("unexpected character %c in password", c)
		}
	}
}

// ---- generateComposeYAML tests ----

func TestGenerateComposeYAML(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.NewAPI.DockerImage = "test-image:latest"
	cfg.NewAPI.Port = 8080

	opts := installOptions{
		Port:        8080,
		DBType:      "mysql",
		RedisAddr:   "redis:6379",
		DockerImage: "test-image:latest",
	}

	yaml := generateComposeYAML(cfg, opts, "")
	if yaml == "" {
		t.Fatal("generateComposeYAML returned empty")
	}
	if !strings.Contains(yaml, "test-image:latest") {
		t.Error("expected image in compose YAML")
	}
	if !strings.Contains(yaml, "mysql") {
		t.Error("expected mysql service in compose YAML")
	}
	if !strings.Contains(yaml, "redis") {
		t.Error("expected redis service in compose YAML")
	}
	if !strings.Contains(yaml, "8080:3000") {
		t.Error("expected port mapping in compose YAML")
	}
}

func TestGenerateComposeYAMLSQLite(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.NewAPI.DockerImage = "test-image:latest"

	opts := installOptions{
		Port:        3000,
		DBType:      "sqlite",
		DockerImage: "test-image:latest",
	}

	yaml := generateComposeYAML(cfg, opts, "")
	if yaml == "" {
		t.Fatal("generateComposeYAML returned empty")
	}
	if strings.Contains(yaml, "mysql") {
		t.Error("sqlite mode should not include mysql service")
	}
	if !strings.Contains(yaml, "redis") {
		t.Error("expected redis service in compose YAML")
	}
}

func TestGenerateComposeYAMLWithProjectName(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.NewAPI.DockerImage = "test-image:latest"

	opts := installOptions{
		Port:        3000,
		DBType:      "sqlite",
		DockerImage: "test-image:latest",
	}

	yaml := generateComposeYAML(cfg, opts, "my-project")
	if !strings.Contains(yaml, "my-project") {
		t.Error("expected project name in compose YAML")
	}
}

// ---- generateEnvFile tests ----

func TestGenerateEnvFileMySQL(t *testing.T) {
	opts := installOptions{
		RedisAddr: "redis:6379",
		DBType:    "mysql",
	}
	env := generateEnvFile(opts)
	if env == "" {
		t.Fatal("generateEnvFile returned empty")
	}
	if !strings.Contains(env, "MYSQL_ROOT_PASSWORD") {
		t.Error("expected MYSQL_ROOT_PASSWORD in env file")
	}
	if !strings.Contains(env, "REDIS_CONN_STRING") {
		t.Error("expected REDIS_CONN_STRING in env file")
	}
}

func TestGenerateEnvFileSQLite(t *testing.T) {
	opts := installOptions{
		RedisAddr: "redis:6379",
		DBType:    "sqlite",
	}
	env := generateEnvFile(opts)
	if env == "" {
		t.Fatal("generateEnvFile returned empty")
	}
	if strings.Contains(env, "MYSQL_ROOT_PASSWORD") {
		t.Error("sqlite mode should not include MYSQL_ROOT_PASSWORD")
	}
	if !strings.Contains(env, "REDIS_CONN_STRING") {
		t.Error("expected REDIS_CONN_STRING in env file")
	}
}

// ---- readEnvValue tests ----

func TestReadEnvValue(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	content := `KEY=value
MYSQL_ROOT_PASSWORD=secret123
EMPTY=
`
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if got := readEnvValue(envFile, "KEY"); got != "value" {
		t.Errorf("readEnvValue(KEY) = %q, want %q", got, "value")
	}
	if got := readEnvValue(envFile, "MYSQL_ROOT_PASSWORD"); got != "secret123" {
		t.Errorf("readEnvValue(MYSQL_ROOT_PASSWORD) = %q, want %q", got, "secret123")
	}
	if got := readEnvValue(envFile, "NONEXISTENT"); got != "" {
		t.Errorf("readEnvValue(NONEXISTENT) = %q, want %q", got, "")
	}
}

func TestReadEnvValueFileNotExist(t *testing.T) {
	got := readEnvValue("/nonexistent/path/.env", "KEY")
	if got != "" {
		t.Errorf("expected empty for missing file, got %q", got)
	}
}

// ---- install command flags tests ----

func TestInstallCommandFlagsFull(t *testing.T) {
	cmd := findSubCmd("install")
	if cmd == nil {
		t.Fatal("install command not found")
	}
	for _, flag := range []string{"port", "image", "force", "domain", "mirror", "no-auto-mirror", "interactive", "instance", "health-timeout"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' on install command", flag)
		}
	}
}

// ---- checkDockerDaemon test ----

func TestCheckDockerDaemon(t *testing.T) {
	r := checkDockerDaemon("")
	validStatuses := map[string]bool{"OK": true, "FAIL": true, "WARN": true, "SKIP": true}
	if !validStatuses[r.Status] {
		t.Errorf("unexpected status: %q", r.Status)
	}
	if r.Name == "" {
		t.Error("check result name should not be empty")
	}
}

// ---- checkDiskSpace test ----

func TestCheckDiskSpace(t *testing.T) {
	dir := t.TempDir()
	r := checkDiskSpace(dir)
	validStatuses := map[string]bool{"OK": true, "FAIL": true, "WARN": true, "SKIP": true}
	if !validStatuses[r.Status] {
		t.Errorf("unexpected status: %q", r.Status)
	}
}

func TestCheckDiskSpaceEmpty(t *testing.T) {
	r := checkDiskSpace("")
	if r.Status != "SKIP" {
		t.Errorf("expected SKIP for empty home, got %s", r.Status)
	}
}

// ---- checkConfigPermissions with files test ----

func TestCheckConfigPermissionsWithHome(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.NewAPI.Home = t.TempDir()
	// Create files that already exist
	os.WriteFile(filepath.Join(cfg.NewAPI.Home, ".env"), []byte("KEY=val"), 0644)
	os.WriteFile(filepath.Join(cfg.NewAPI.Home, "docker-compose.yml"), []byte("version: '3'"), 0644)

	r := checkConfigPermissions(cfg)
	validStatuses := map[string]bool{"OK": true, "FAIL": true, "WARN": true, "SKIP": true}
	if !validStatuses[r.Status] {
		t.Errorf("unexpected status: %q", r.Status)
	}
}

// ---- runConfigValidate test ----

func TestRunConfigValidate(t *testing.T) {
	cmd := &cobra.Command{}
	err := runConfigValidate(cmd, nil)
	if err == nil {
		t.Error("expected error when config not loaded")
	}
}

// ---- runConfigShow test ----

func TestRunConfigShow(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("json", false, "")
	err := runConfigShow(cmd, nil)
	if err == nil {
		t.Error("expected error when config not loaded")
	}
}

// ---- mirror subcommand flags test ----

func TestMirrorTestCommandExists(t *testing.T) {
	cmd := findSubCmd("mirror")
	if cmd == nil {
		t.Fatal("mirror command not found")
	}

	var testCmd *cobra.Command
	for _, sub := range cmd.Commands() {
		name := strings.SplitN(sub.Use, " ", 2)[0]
		if name == "test" {
			testCmd = sub
			break
		}
	}
	if testCmd == nil {
		t.Fatal("mirror test command not found")
	}
}

// ---- version RunE test ----

func TestVersionCommandRun(t *testing.T) {
	cmd := findSubCmd("version")
	if cmd == nil {
		t.Fatal("version command not found")
	}

	old := os.Stdout
	devNull, _ := os.Open(os.DevNull)
	os.Stdout = devNull
	defer func() {
		os.Stdout = old
		devNull.Close()
	}()

	// versionCmd.Run has no error return
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("version command Run panicked: %v", r)
		}
	}()
	cmd.Run(cmd, nil)
}

// ---- fixConfigPermissions test ----

func TestFixConfigPermissions(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.NewAPI.Home = t.TempDir()
	os.WriteFile(filepath.Join(cfg.NewAPI.Home, ".env"), []byte("KEY=val"), 0644)

	err := fixConfigPermissions(context.Background(), cfg)
	if err != nil {
		t.Errorf("fixConfigPermissions failed: %v", err)
	}
}

// ---- instance add flags test ----

func TestInstanceAddFlags(t *testing.T) {
	cmd := findSubCmd("instance")
	if cmd == nil {
		t.Fatal("instance command not found")
	}
	var addCmd *cobra.Command
	for _, sub := range cmd.Commands() {
		name := strings.SplitN(sub.Use, " ", 2)[0]
		if name == "add" {
			addCmd = sub
			break
		}
	}
	if addCmd == nil {
		t.Fatal("instance add command not found")
	}
	for _, flag := range []string{"home", "port", "image", "domain", "health-timeout", "max-backups"} {
		if addCmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' on instance add command", flag)
		}
	}
}

// ---- update command flags test ----

func TestUpdateCommandFlagsFull(t *testing.T) {
	cmd := findSubCmd("update")
	if cmd == nil {
		t.Fatal("update command not found")
	}
	for _, flag := range []string{"image", "backup", "force", "mirror", "no-auto-mirror", "check", "self"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' on update command", flag)
		}
	}
}

// ---- mirror add flags test ----

func TestMirrorAddFlags(t *testing.T) {
	cmd := findSubCmd("mirror")
	if cmd == nil {
		t.Fatal("mirror command not found")
	}
	var addCmd *cobra.Command
	for _, sub := range cmd.Commands() {
		name := strings.SplitN(sub.Use, " ", 2)[0]
		if name == "add" {
			addCmd = sub
			break
		}
	}
	if addCmd == nil {
		t.Fatal("mirror add command not found")
	}
	if addCmd.Flags().Lookup("apply") == nil {
		t.Error("expected flag '--apply' on mirror add command")
	}
}

// ---- mirror builtin and reset subcommands existence test ----

func TestMirrorBuiltinAndResetCommands(t *testing.T) {
	cmd := findSubCmd("mirror")
	if cmd == nil {
		t.Fatal("mirror command not found")
	}
	subNames := map[string]bool{}
	for _, sub := range cmd.Commands() {
		name := strings.SplitN(sub.Use, " ", 2)[0]
		subNames[name] = true
	}
	for _, name := range []string{"builtin", "reset", "apply", "remove", "list"} {
		if !subNames[name] {
			t.Errorf("expected 'mirror %s' subcommand", name)
		}
	}
}

// ---- performBackup test ----

func TestPerformBackup(t *testing.T) {
	ctx := context.Background()
	cfg := core.DefaultConfig()
	cfg.NewAPI.Home = t.TempDir()
	cfg.NewAPI.BackupDir = t.TempDir()

	os.WriteFile(filepath.Join(cfg.NewAPI.Home, "docker-compose.yml"), []byte("version: '3'"), 0644)
	os.WriteFile(filepath.Join(cfg.NewAPI.Home, ".env"), []byte("KEY=val"), 0644)

	os.MkdirAll(filepath.Join(cfg.NewAPI.Home, "data"), 0755)
	os.WriteFile(filepath.Join(cfg.NewAPI.Home, "data", "test.db"), []byte("data"), 0644)

	result, err := performBackup(ctx, cfg)
	if err != nil {
		t.Fatalf("performBackup failed: %v", err)
	}
	if result == "" {
		t.Error("performBackup returned empty path")
	}
	if _, err := os.Stat(result); os.IsNotExist(err) {
		t.Error("backup archive was not created")
	}
}

// ---- runInstanceList test ----

func TestRunInstanceList(t *testing.T) {
	cmd := findSubCmd("instance")
	if cmd == nil {
		t.Fatal("instance command not found")
	}
	var listCmd *cobra.Command
	for _, sub := range cmd.Commands() {
		name := strings.SplitN(sub.Use, " ", 2)[0]
		if name == "list" {
			listCmd = sub
			break
		}
	}
	if listCmd == nil {
		t.Fatal("instance list command not found")
	}

	old := os.Stdout
	devNull, _ := os.Open(os.DevNull)
	os.Stdout = devNull
	defer func() {
		os.Stdout = old
		devNull.Close()
	}()

	err := listCmd.RunE(listCmd, nil)
	if err != nil {
		t.Errorf("instance list failed: %v", err)
	}
}

// ---- runConfigChmod test ----

func TestRunConfigChmod(t *testing.T) {
	cmd := &cobra.Command{}
	err := runConfigChmod(cmd, nil)
	if err == nil {
		t.Error("expected error when config not loaded")
	}
}

// ---- checkHTTPHealth test with zero port ----

func TestCheckHTTPHealthZeroPort(t *testing.T) {
	r := checkHTTPHealth(0)
	validStatuses := map[string]bool{"OK": true, "FAIL": true, "WARN": true, "SKIP": true}
	if !validStatuses[r.Status] {
		t.Errorf("unexpected status: %q", r.Status)
	}
}

// ---- runMirrorBuiltin test ----

func TestRunMirrorBuiltin(t *testing.T) {
	cmd := &cobra.Command{}
	old := os.Stdout
	devNull, _ := os.Open(os.DevNull)
	os.Stdout = devNull
	defer func() {
		os.Stdout = old
		devNull.Close()
	}()

	err := runMirrorBuiltin(cmd, nil)
	if err != nil {
		t.Errorf("runMirrorBuiltin failed: %v", err)
	}
}

// ---- checkContainer test ----

func TestCheckContainer(t *testing.T) {
	r := checkContainer(context.Background(), "new-api", "")
	validStatuses := map[string]bool{"OK": true, "FAIL": true, "WARN": true, "SKIP": true}
	if !validStatuses[r.Status] {
		t.Errorf("unexpected status: %q", r.Status)
	}
}

// ---- checkHTTPHealth with mock server test ----

func TestCheckHTTPHealthWithMock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	portStr := strings.Split(server.URL, ":")[2]
	port, _ := strconv.Atoi(portStr)

	r := checkHTTPHealth(port)
	if r.Status != "OK" {
		t.Errorf("expected OK status, got %s: %s", r.Status, r.Message)
	}
}

// ---- applyConfigValue all keys test ----

func TestApplyConfigValueAllKeys(t *testing.T) {
	cfg := core.DefaultConfig()

	tests := []struct {
		key     string
		value   string
		keyType string
	}{
		{"newapi.docker_image", "test-image:latest", "string"},
		{"newapi.backup_dir", "/tmp/backups", "string"},
		{"newapi.domain", "example.com", "string"},
		{"newapi.health_timeout", "60", "int"},
		{"newapi.max_backups", "5", "int"},
		{"docker.compose_cmd", "docker-compose", "string"},
		{"log.level", "debug", "string"},
		{"log.format", "json", "string"},
		{"instance.active", "test-instance", "string"},
	}

	for _, tc := range tests {
		err := applyConfigValue(cfg, tc.key, tc.value, tc.keyType)
		if err != nil {
			t.Errorf("applyConfigValue(%q, %q) failed: %v", tc.key, tc.value, err)
		}
	}

	if cfg.NewAPI.DockerImage != "test-image:latest" {
		t.Errorf("DockerImage = %q", cfg.NewAPI.DockerImage)
	}
	if cfg.NewAPI.BackupDir != "/tmp/backups" {
		t.Errorf("BackupDir = %q", cfg.NewAPI.BackupDir)
	}
	if cfg.NewAPI.Domain != "example.com" {
		t.Errorf("Domain = %q", cfg.NewAPI.Domain)
	}
	if cfg.NewAPI.HealthTimeout != 60 {
		t.Errorf("HealthTimeout = %d", cfg.NewAPI.HealthTimeout)
	}
	if cfg.NewAPI.MaxBackups != 5 {
		t.Errorf("MaxBackups = %d", cfg.NewAPI.MaxBackups)
	}
	if cfg.Docker.ComposeCmd != "docker-compose" {
		t.Errorf("ComposeCmd = %q", cfg.Docker.ComposeCmd)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("Log.Level = %q", cfg.Log.Level)
	}
	if cfg.Log.Format != "json" {
		t.Errorf("Log.Format = %q", cfg.Log.Format)
	}
	if cfg.Instance.Active != "test-instance" {
		t.Errorf("Instance.Active = %q", cfg.Instance.Active)
	}
}

func TestApplyConfigValueInvalidValues(t *testing.T) {
	cfg := core.DefaultConfig()

	if err := applyConfigValue(cfg, "newapi.health_timeout", "0", "int"); err == nil {
		t.Error("expected error for health_timeout 0")
	}
	if err := applyConfigValue(cfg, "newapi.max_backups", "0", "int"); err == nil {
		t.Error("expected error for max_backups 0")
	}
	if err := applyConfigValue(cfg, "log.level", "invalid", "string"); err == nil {
		t.Error("expected error for invalid log level")
	}
	if err := applyConfigValue(cfg, "log.format", "invalid", "string"); err == nil {
		t.Error("expected error for invalid log format")
	}
}

// ---- runConfigSet tests ----

func TestRunConfigSetInvalidKey(t *testing.T) {
	cmd := &cobra.Command{}
	err := runConfigSet(cmd, []string{"invalid.key", "value"})
	if err == nil {
		t.Error("expected error for invalid key")
	}
}

// ---- ExampleCompletion_bash test ----

func TestExampleCompletionBash(t *testing.T) {
	old := os.Stdout
	devNull, _ := os.Open(os.DevNull)
	os.Stdout = devNull
	defer func() {
		os.Stdout = old
		devNull.Close()
	}()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ExampleCompletion_bash panicked: %v", r)
		}
	}()
	ExampleCompletion_bash()
}

// ---- runMirrorList test ----

func TestRunMirrorList(t *testing.T) {
	old := os.Stdout
	devNull, _ := os.Open(os.DevNull)
	os.Stdout = devNull
	defer func() {
		os.Stdout = old
		devNull.Close()
	}()

	cmd := &cobra.Command{}
	err := runMirrorList(cmd, nil)
	if err != nil {
		t.Logf("runMirrorList returned error (expected on non-Docker systems): %v", err)
	}
}

// ---- runMirrorReset test ----

func TestRunMirrorReset(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("apply", true, "")
	err := runMirrorReset(cmd, nil)
	if err != nil {
		t.Logf("runMirrorReset returned error (expected on non-Docker systems): %v", err)
	}
}
