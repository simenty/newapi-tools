// NewAPI Tools - Docker management platform for newapi
package cli

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/simenty/newapi-tools/internal/core"
	"github.com/spf13/cobra"
)

// ---- Existing tests (preserved) ----

func TestInstallCommandExists(t *testing.T) {
	cmd := GetRootCmd()
	found := false
	for _, sub := range cmd.Commands() {
		if sub.Use == "install" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'install' subcommand to be registered")
	}
}

func TestStatusCommandExists(t *testing.T) {
	cmd := GetRootCmd()
	found := false
	for _, sub := range cmd.Commands() {
		if sub.Use == "status" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'status' subcommand to be registered")
	}
}

func TestInstallCommandFlags(t *testing.T) {
	cmd := GetRootCmd()
	var installCmd *cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Use == "install" {
			installCmd = sub
			break
		}
	}
	if installCmd == nil {
		t.Fatal("install command not found")
	}

	// Check expected flags exist
	expectedFlags := []string{"port", "image", "force"}
	for _, name := range expectedFlags {
		f := installCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("expected flag '--%s' on install command", name)
		}
	}
}

func TestStatusCommandFlags(t *testing.T) {
	cmd := GetRootCmd()
	var statusCmd *cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Use == "status" {
			statusCmd = sub
			break
		}
	}
	if statusCmd == nil {
		t.Fatal("status command not found")
	}

	f := statusCmd.Flags().Lookup("json")
	if f == nil {
		t.Error("expected flag '--json' on status command")
	}
}

// ---- New command registration tests ----

func findSubCmd(use string) *cobra.Command {
	for _, sub := range GetRootCmd().Commands() {
		if sub.Use == use {
			return sub
		}
	}
	return nil
}

func TestBackupCommandExists(t *testing.T) {
	if findSubCmd("backup") == nil {
		t.Error("expected 'backup' subcommand to be registered")
	}
}

func TestRestoreCommandExists(t *testing.T) {
	if findSubCmd("restore") == nil {
		t.Error("expected 'restore' subcommand to be registered")
	}
}

func TestUpdateCommandExists(t *testing.T) {
	if findSubCmd("update") == nil {
		t.Error("expected 'update' subcommand to be registered")
	}
}

func TestDoctorCommandExists(t *testing.T) {
	if findSubCmd("doctor") == nil {
		t.Error("expected 'doctor' subcommand to be registered")
	}
}

func TestBackupCommandFlags(t *testing.T) {
	cmd := findSubCmd("backup")
	if cmd == nil {
		t.Fatal("backup command not found")
	}
	for _, flag := range []string{"output", "compress"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' on backup command", flag)
		}
	}
}

func TestRestoreCommandFlags(t *testing.T) {
	cmd := findSubCmd("restore")
	if cmd == nil {
		t.Fatal("restore command not found")
	}
	for _, flag := range []string{"file", "force"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' on restore command", flag)
		}
	}
}

func TestUpdateCommandFlags(t *testing.T) {
	cmd := findSubCmd("update")
	if cmd == nil {
		t.Fatal("update command not found")
	}
	for _, flag := range []string{"image", "backup", "force"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' on update command", flag)
		}
	}
}

func TestDoctorCommandFlags(t *testing.T) {
	cmd := findSubCmd("doctor")
	if cmd == nil {
		t.Fatal("doctor command not found")
	}
	for _, flag := range []string{"fix", "json"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag '--%s' on doctor command", flag)
		}
	}
}

// ---- Backup helper function tests ----

func TestCreateTarArchiveAndExtract(t *testing.T) {
	// Create a temp staging dir with test files
	stageDir := t.TempDir()
	testContent := "hello from newapi backup test"
	testFile := filepath.Join(stageDir, "test.txt")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	subDir := filepath.Join(stageDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "nested.txt"), []byte("nested"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create archive
	archivePath := filepath.Join(t.TempDir(), "backup.tar.gz")
	if err := createTarArchive(archivePath, stageDir, true); err != nil {
		t.Fatalf("createTarArchive failed: %v", err)
	}

	// Verify archive exists and is non-empty
	info, err := os.Stat(archivePath)
	if err != nil {
		t.Fatalf("archive not created: %v", err)
	}
	if info.Size() == 0 {
		t.Error("archive is empty")
	}

	// Extract and verify
	dstDir := t.TempDir()
	if err := extractTarArchive(archivePath, dstDir); err != nil {
		t.Fatalf("extractTarArchive failed: %v", err)
	}

	extracted := filepath.Join(dstDir, "test.txt")
	data, err := os.ReadFile(extracted)
	if err != nil {
		t.Fatalf("extracted file not found: %v", err)
	}
	if string(data) != testContent {
		t.Errorf("content mismatch: got %q, want %q", string(data), testContent)
	}

	// Check nested file
	nestedExtracted := filepath.Join(dstDir, "subdir", "nested.txt")
	if _, err := os.Stat(nestedExtracted); os.IsNotExist(err) {
		t.Error("nested file not extracted")
	}
}

func TestCreateTarArchiveUncompressed(t *testing.T) {
	stageDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(stageDir, "data.txt"), []byte("data"), 0644)

	archivePath := filepath.Join(t.TempDir(), "backup.tar")
	if err := createTarArchive(archivePath, stageDir, false); err != nil {
		t.Fatalf("createTarArchive (no compress) failed: %v", err)
	}

	info, _ := os.Stat(archivePath)
	if info == nil || info.Size() == 0 {
		t.Error("uncompressed archive is empty or missing")
	}
}

func TestCopyFileIfExists(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	src := filepath.Join(srcDir, "file.txt")
	dst := filepath.Join(dstDir, "file.txt")

	// Non-existent source should not error
	if err := copyFileIfExists(src+"_missing", dst); err != nil {
		t.Errorf("copyFileIfExists should skip missing files, got: %v", err)
	}

	// Write source and copy
	content := "test content"
	if err := os.WriteFile(src, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := copyFileIfExists(src, dst); err != nil {
		t.Fatalf("copyFileIfExists failed: %v", err)
	}

	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("copied file not found: %v", err)
	}
	if string(data) != content {
		t.Errorf("content mismatch: got %q want %q", string(data), content)
	}
}

func TestCopyDir(t *testing.T) {
	src := t.TempDir()
	_ = os.WriteFile(filepath.Join(src, "a.txt"), []byte("a"), 0644)
	sub := filepath.Join(src, "sub")
	_ = os.MkdirAll(sub, 0755)
	_ = os.WriteFile(filepath.Join(sub, "b.txt"), []byte("b"), 0644)

	dst := t.TempDir()
	if err := copyDir(src, dst); err != nil {
		t.Fatalf("copyDir failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dst, "a.txt")); os.IsNotExist(err) {
		t.Error("a.txt not copied")
	}
	if _, err := os.Stat(filepath.Join(dst, "sub", "b.txt")); os.IsNotExist(err) {
		t.Error("sub/b.txt not copied")
	}
}

// ---- Restore helper tests ----

func TestFindLatestBackup(t *testing.T) {
	dir := t.TempDir()

	// Create fake backup files
	files := []string{
		"newapi-backup-20260101-120000.tar.gz",
		"newapi-backup-20260102-090000.tar.gz",
		"newapi-backup-20260102-150000.tar.gz", // <- should be latest
		"other-file.txt",                       // not a backup
	}
	for _, f := range files {
		_ = os.WriteFile(filepath.Join(dir, f), []byte("x"), 0644)
	}

	latest, err := findLatestBackup(dir)
	if err != nil {
		t.Fatalf("findLatestBackup failed: %v", err)
	}
	if !strings.Contains(latest, "20260102-150000") {
		t.Errorf("expected latest backup 20260102-150000, got %s", latest)
	}
}

func TestFindLatestBackupEmpty(t *testing.T) {
	dir := t.TempDir()
	// No backup files
	_, err := findLatestBackup(dir)
	if err == nil {
		t.Error("expected error for empty backup dir")
	}
}

func TestExtractTarArchivePathTraversal(t *testing.T) {
	// Create a malicious tar with path traversal entry
	archivePath := filepath.Join(t.TempDir(), "malicious.tar.gz")
	f, _ := os.Create(archivePath)
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	// Attempt path traversal
	hdr := &tar.Header{
		Name:     "../../etc/passwd",
		Typeflag: tar.TypeReg,
		Size:     5,
	}
	_ = tw.WriteHeader(hdr)
	_, _ = tw.Write([]byte("EVIL"))
	tw.Close()
	gw.Close()
	f.Close()

	dstDir := t.TempDir()
	err := extractTarArchive(archivePath, dstDir)
	if err == nil {
		t.Error("expected path traversal to be rejected")
	}
}

// ---- Doctor helper tests ----

func TestCheckDockerBinary(t *testing.T) {
	r := checkDockerBinary()
	// Just verify structure — docker may or may not be on PATH in CI
	if r.Name == "" || r.Status == "" {
		t.Error("checkDockerBinary returned empty result")
	}
	validStatuses := map[string]bool{"OK": true, "FAIL": true, "WARN": true, "SKIP": true}
	if !validStatuses[r.Status] {
		t.Errorf("unexpected status: %q", r.Status)
	}
}

func TestCheckHomeDirMissing(t *testing.T) {
	r := checkHomeDir("/nonexistent/path/to/home/xyz")
	if r.Status != "FAIL" {
		t.Errorf("expected FAIL for missing home dir, got %s", r.Status)
	}
}

func TestCheckHomeDirEmpty(t *testing.T) {
	r := checkHomeDir("")
	if r.Status != "WARN" {
		t.Errorf("expected WARN for empty home, got %s", r.Status)
	}
}

func TestCheckHomeDirExists(t *testing.T) {
	dir := t.TempDir()
	r := checkHomeDir(dir)
	if r.Status != "OK" {
		t.Errorf("expected OK for existing dir, got %s: %s", r.Status, r.Message)
	}
}

func TestCheckComposeFileMissing(t *testing.T) {
	dir := t.TempDir()
	r := checkComposeFile(dir)
	if r.Status != "FAIL" {
		t.Errorf("expected FAIL when compose file missing, got %s", r.Status)
	}
}

func TestCheckComposeFileExists(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte("version: '3'"), 0644)
	r := checkComposeFile(dir)
	if r.Status != "OK" {
		t.Errorf("expected OK when compose file exists, got %s", r.Status)
	}
}

func TestCheckEnvFileMissing(t *testing.T) {
	dir := t.TempDir()
	r := checkEnvFile(dir)
	if r.Status != "WARN" {
		t.Errorf("expected WARN when .env missing, got %s", r.Status)
	}
}

func TestCheckEnvFileExists(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, ".env"), []byte("KEY=val"), 0644)
	r := checkEnvFile(dir)
	if r.Status != "OK" {
		t.Errorf("expected OK when .env exists, got %s", r.Status)
	}
}

func TestPrintDoctorJSON(t *testing.T) {
	// Just verify it doesn't panic
	results := []checkResult{
		{"docker binary", "OK", "/usr/bin/docker", nil},
		{"docker daemon", "FAIL", "not running", nil},
	}
	// Capture to /dev/null equivalent — just ensure no panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("printDoctorJSON panicked: %v", r)
		}
	}()
	// Redirect stdout temporarily
	old := os.Stdout
	devNull, _ := os.Open(os.DevNull)
	os.Stdout = devNull
	printDoctorJSON(results, false)
	os.Stdout = old
	devNull.Close()
}

func TestShortDigest(t *testing.T) {
	cases := []struct {
		input string
		check func(string) bool
	}{
		{"sha256:abc123def456789012345678901234567890", func(s string) bool { return strings.HasSuffix(s, "...") }},
		{"short", func(s string) bool { return s == "short" }},
		{"sha256:abc", func(s string) bool { return s == "sha256:abc" }},
	}
	for _, c := range cases {
		got := shortDigest(c.input)
		if !c.check(got) {
			t.Errorf("shortDigest(%q) = %q, unexpected result", c.input, got)
		}
	}
}

// ---- Config command tests ----

func TestConfigCommandExists(t *testing.T) {
	if findSubCmd("config") == nil {
		t.Error("expected 'config' subcommand to be registered")
	}
}

func TestConfigCommandFlags(t *testing.T) {
	cmd := findSubCmd("config")
	if cmd == nil {
		t.Fatal("config command not found")
	}
	if cmd.Flags().Lookup("json") == nil {
		t.Error("expected flag '--json' on config command")
	}
}

func TestConfigSubcommands(t *testing.T) {
	cmd := findSubCmd("config")
	if cmd == nil {
		t.Fatal("config command not found")
	}
	subNames := map[string]bool{}
	for _, sub := range cmd.Commands() {
		// Use first word of Use field (e.g., "set <key> <value>" → "set")
		name := strings.SplitN(sub.Use, " ", 2)[0]
		subNames[name] = true
	}
	for _, name := range []string{"set", "init"} {
		if !subNames[name] {
			t.Errorf("expected 'config %s' subcommand", name)
		}
	}
}

func TestApplyConfigValue(t *testing.T) {
	cfg := core.DefaultConfig()

	// Test string key
	if err := applyConfigValue(cfg, "newapi.home", "/custom/home", "string"); err != nil {
		t.Fatalf("applyConfigValue string: %v", err)
	}
	if cfg.NewAPI.Home != "/custom/home" {
		t.Errorf("expected home '/custom/home', got %q", cfg.NewAPI.Home)
	}

	// Test int key
	if err := applyConfigValue(cfg, "newapi.port", "8080", "int"); err != nil {
		t.Fatalf("applyConfigValue int: %v", err)
	}
	if cfg.NewAPI.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.NewAPI.Port)
	}

	// Test invalid port
	if err := applyConfigValue(cfg, "newapi.port", "notanumber", "int"); err == nil {
		t.Error("expected error for invalid port")
	}

	// Test invalid key
	if err := applyConfigValue(cfg, "invalid.key", "x", "string"); err == nil {
		t.Error("expected error for invalid key")
	}
}

func TestPrompt(t *testing.T) {
	// Redirect stdout to suppress prompt output
	old := os.Stdout
	devNull, _ := os.Open(os.DevNull)
	os.Stdout = devNull
	defer func() {
		os.Stdout = old
		devNull.Close()
	}()

	input := "\n" // just press enter
	reader := bufio.NewReader(strings.NewReader(input))
	result := prompt(reader, "test label", "default_val")
	if result != "default_val" {
		t.Errorf("expected default, got %q", result)
	}

	input2 := "custom\n"
	reader2 := bufio.NewReader(strings.NewReader(input2))
	result2 := prompt(reader2, "test label", "default_val")
	if result2 != "custom" {
		t.Errorf("expected 'custom', got %q", result2)
	}
}

// ---- Doctor --fix tests ----

func TestRunAutoFixHomeDir(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "newapi-home")

	cfg := core.DefaultConfig()
	cfg.NewAPI.Home = homeDir

	// Home dir doesn't exist → should be created
	results := []checkResult{
		{"home-dir", "FAIL", "does not exist", nil},
	}

	// Redirect stdout
	old := os.Stdout
	devNull, _ := os.Open(os.DevNull)
	os.Stdout = devNull

	fixCount := runAutoFix(context.Background(), results, cfg)

	os.Stdout = old
	devNull.Close()

	if fixCount < 1 {
		t.Errorf("expected at least 1 fix, got %d", fixCount)
	}
	if _, err := os.Stat(homeDir); os.IsNotExist(err) {
		t.Error("home directory should have been created by auto-fix")
	}
}

func TestRunAutoFixHintOnly(t *testing.T) {
	cfg := core.DefaultConfig()

	// These should only print hints, not count as fixes
	results := []checkResult{
		{"docker binary", "FAIL", "not found", nil},
		{"docker-compose.yml", "FAIL", "not found", nil},
		{".env file", "WARN", "not found", nil},
		{"HTTP health", "WARN", "not reachable", nil},
		{"disk space", "SKIP", "n/a", nil},
	}

	old := os.Stdout
	devNull, _ := os.Open(os.DevNull)
	os.Stdout = devNull

	fixCount := runAutoFix(context.Background(), results, cfg)

	os.Stdout = old
	devNull.Close()

	if fixCount != 0 {
		t.Errorf("hints should not count as fixes, got %d", fixCount)
	}
}
