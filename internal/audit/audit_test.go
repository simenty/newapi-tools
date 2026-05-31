// NewAPI Tools - Audit logging tests
package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewAuditLogger(t *testing.T) {
	logger := NewAuditLogger("/tmp/test-audit.log")
	if logger.path != "/tmp/test-audit.log" {
		t.Errorf("expected path /tmp/test-audit.log, got %s", logger.path)
	}
	if logger.maxSize != 10*1024*1024 {
		t.Errorf("expected maxSize 10MB, got %d", logger.maxSize)
	}
	if logger.keep != 5 {
		t.Errorf("expected keep 5, got %d", logger.keep)
	}
}

func TestNewAuditLoggerDefaultPath(t *testing.T) {
	logger := NewAuditLogger("")
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".config", "newapi-tools", "audit.log")
	if logger.path != expected {
		t.Errorf("expected default path %s, got %s", expected, logger.path)
	}
}

func TestLog(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")
	logger := NewAuditLogger(logPath)

	entry := AuditEntry{
		Timestamp:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Command:    "install",
		User:       "testuser",
		Args:       []string{"--port", "8080"},
		Result:     "ok",
		DurationMs: 1500,
	}

	if err := logger.Log(entry); err != nil {
		t.Fatalf("Log() failed: %v", err)
	}

	// Read the log file and verify content
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read audit log: %v", err)
	}

	var got AuditEntry
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("failed to parse audit log entry: %v", err)
	}

	if got.Command != "install" {
		t.Errorf("expected command 'install', got %q", got.Command)
	}
	if got.User != "testuser" {
		t.Errorf("expected user 'testuser', got %q", got.User)
	}
	if got.Result != "ok" {
		t.Errorf("expected result 'ok', got %q", got.Result)
	}
	if got.DurationMs != 1500 {
		t.Errorf("expected duration 1500, got %d", got.DurationMs)
	}
	if len(got.Args) != 2 || got.Args[0] != "--port" || got.Args[1] != "8080" {
		t.Errorf("expected args [--port 8080], got %v", got.Args)
	}
}

func TestLogError(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")
	logger := NewAuditLogger(logPath)

	entry := AuditEntry{
		Timestamp:  time.Now(),
		Command:    "install",
		User:       "testuser",
		Args:       []string{},
		Result:     "error",
		Error:      "docker not found",
		DurationMs: 200,
	}

	if err := logger.Log(entry); err != nil {
		t.Fatalf("Log() failed: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read audit log: %v", err)
	}

	var got AuditEntry
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("failed to parse audit log entry: %v", err)
	}

	if got.Result != "error" {
		t.Errorf("expected result 'error', got %q", got.Result)
	}
	if got.Error != "docker not found" {
		t.Errorf("expected error 'docker not found', got %q", got.Error)
	}
}

func TestLogRotation(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")
	logger := NewAuditLogger(logPath)
	logger.maxSize = 100 // Small size to trigger rotation quickly
	logger.keep = 3

	// Write enough entries to exceed the max size
	entry := AuditEntry{
		Timestamp:  time.Now(),
		Command:    "test-command",
		User:       "testuser",
		Args:       []string{},
		Result:     "ok",
		DurationMs: 100,
	}

	// Write many entries to trigger rotation
	for i := 0; i < 50; i++ {
		if err := logger.Log(entry); err != nil {
			t.Fatalf("Log() iteration %d failed: %v", i, err)
		}
	}

	// Verify that rotation happened: at least audit.log and audit.log.1 should exist
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("audit.log should exist after logging")
	}

	rotatedPath := logPath + ".1"
	if _, err := os.Stat(rotatedPath); os.IsNotExist(err) {
		t.Error("audit.log.1 should exist after rotation")
	}
}

func TestLogRotationKeepLimit(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")
	logger := NewAuditLogger(logPath)
	logger.maxSize = 50 // Very small to trigger multiple rotations
	logger.keep = 2

	entry := AuditEntry{
		Timestamp:  time.Now(),
		Command:    "test-command",
		User:       "testuser",
		Args:       []string{},
		Result:     "ok",
		DurationMs: 100,
	}

	// Write many entries to trigger multiple rotations
	for i := 0; i < 100; i++ {
		if err := logger.Log(entry); err != nil {
			t.Fatalf("Log() iteration %d failed: %v", i, err)
		}
	}

	// Files beyond keep limit should not exist
	beyondPath := logPath + ".3"
	if _, err := os.Stat(beyondPath); !os.IsNotExist(err) {
		t.Error("audit.log.3 should not exist when keep=2")
	}
}

func TestLogMultipleEntries(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")
	logger := NewAuditLogger(logPath)

	for i := 0; i < 5; i++ {
		entry := AuditEntry{
			Timestamp:  time.Now(),
			Command:    "test",
			User:       "testuser",
			Args:       []string{},
			Result:     "ok",
			DurationMs: int64(i * 100),
		}
		if err := logger.Log(entry); err != nil {
			t.Fatalf("Log() iteration %d failed: %v", i, err)
		}
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read audit log: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 5 {
		t.Errorf("expected 5 log lines, got %d", len(lines))
	}
}

func TestLogEmptyOmitError(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")
	logger := NewAuditLogger(logPath)

	// Entry with empty Error and ok result — error field should be omitted in JSON
	entry := AuditEntry{
		Timestamp:  time.Now(),
		Command:    "install",
		User:       "testuser",
		Result:     "ok",
		DurationMs: 500,
	}

	if err := logger.Log(entry); err != nil {
		t.Fatalf("Log() failed: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read audit log: %v", err)
	}

	// Verify "error" key is not present in JSON (omitempty)
	if strings.Contains(string(data), `"error"`) {
		t.Error("expected 'error' field to be omitted when empty, but it was present")
	}
}

func TestListWithRotation(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	// Create rotated files with known entries (oldest first)
	// audit.log.3: entries 0,1 (oldest)
	// audit.log.2: entries 2,3
	// audit.log.1: entries 4,5
	// audit.log:   entries 6,7,8,9 (newest)
	writeTestEntry(t, logPath+".3", 0, "test", 0)
	writeTestEntry(t, logPath+".3", 1, "test", 100)
	writeTestEntry(t, logPath+".2", 2, "test", 200)
	writeTestEntry(t, logPath+".2", 3, "test", 300)
	writeTestEntry(t, logPath+".1", 4, "test", 400)
	writeTestEntry(t, logPath+".1", 5, "test", 500)
	writeTestEntry(t, logPath, 6, "test", 600)
	writeTestEntry(t, logPath, 7, "test", 700)
	writeTestEntry(t, logPath, 8, "test", 800)
	writeTestEntry(t, logPath, 9, "test", 900)

	reader := NewAuditReader(logPath)

	// List all entries
	entries, err := reader.List(ListOption{})
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	if len(entries) != 10 {
		t.Errorf("expected 10 entries, got %d", len(entries))
	}

	// List with Last=3 should return the 3 most recent entries
	entries, err = reader.List(ListOption{Last: 3})
	if err != nil {
		t.Fatalf("List(Last=3) failed: %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("expected 3 entries with Last=3, got %d", len(entries))
	}

	// Verify they are the most recent (entries 7, 8, 9, newest first)
	expected := []int64{900, 800, 700}
	for i, e := range entries {
		if e.DurationMs != expected[i] {
			t.Errorf("entry %d: expected DurationMs %d, got %d", i, expected[i], e.DurationMs)
		}
	}
}

func writeTestEntry(t *testing.T, path string, seq int, cmd string, durationMs int64) {
	t.Helper()
	entry := AuditEntry{
		Timestamp:  time.Date(2024, 1, 1, 0, 0, seq, 0, time.UTC),
		Command:    cmd,
		User:       "user1",
		Result:     "ok",
		DurationMs: durationMs,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("failed to marshal entry: %v", err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatalf("failed to open %s: %v", path, err)
	}
	defer f.Close()
	if _, err := f.Write(append(data, '\n')); err != nil {
		t.Fatalf("failed to write to %s: %v", path, err)
	}
}

func TestListWithRotationLastExceedsAvailable(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	// Create one rotated file with 3 entries, main with 3 entries
	writeTestEntry(t, logPath+".1", 0, "test", 0)
	writeTestEntry(t, logPath+".1", 1, "test", 100)
	writeTestEntry(t, logPath+".1", 2, "test", 200)
	writeTestEntry(t, logPath, 3, "test", 300)
	writeTestEntry(t, logPath, 4, "test", 400)
	writeTestEntry(t, logPath, 5, "test", 500)

	reader := NewAuditReader(logPath)

	// Request more entries than available
	entries, err := reader.List(ListOption{Last: 100})
	if err != nil {
		t.Fatalf("List(Last=100) failed: %v", err)
	}

	if len(entries) != 6 {
		t.Errorf("expected 6 entries (all available), got %d", len(entries))
	}
}

func TestListWithRotationFilterByCommand(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	// Mix of commands across rotated files
	writeTestEntry(t, logPath+".1", 0, "install", 0)
	writeTestEntry(t, logPath+".1", 1, "install", 100)
	writeTestEntry(t, logPath, 2, "backup", 200)
	writeTestEntry(t, logPath, 3, "install", 300)
	writeTestEntry(t, logPath, 4, "backup", 400)

	reader := NewAuditReader(logPath)

	// Filter by "install"
	entries, err := reader.List(ListOption{Command: "install"})
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("expected 3 'install' entries, got %d", len(entries))
	}

	for _, e := range entries {
		if e.Command != "install" {
			t.Errorf("expected command 'install', got %q", e.Command)
		}
	}
}

func TestListWithRotationLastWithFilter(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	writeTestEntry(t, logPath+".2", 0, "install", 0)
	writeTestEntry(t, logPath+".2", 1, "install", 100)
	writeTestEntry(t, logPath+".1", 2, "backup", 200)
	writeTestEntry(t, logPath+".1", 3, "install", 300)
	writeTestEntry(t, logPath, 4, "install", 400)
	writeTestEntry(t, logPath, 5, "backup", 500)
	writeTestEntry(t, logPath, 6, "install", 600)

	reader := NewAuditReader(logPath)

	// Last 2 install entries
	entries, err := reader.List(ListOption{Last: 2, Command: "install"})
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("expected 2 'install' entries, got %d", len(entries))
	}

	// Should be the most recent 2 install entries: seq 4 (400ms) and seq 6 (600ms)
	for _, e := range entries {
		if e.Command != "install" {
			t.Errorf("expected 'install', got %q", e.Command)
		}
	}
}

func TestListNoLogFile(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "nonexistent.log")
	reader := NewAuditReader(logPath)

	entries, err := reader.List(ListOption{})
	if err != nil {
		t.Fatalf("List() with nonexistent file should not error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}

	entries, err = reader.List(ListOption{Last: 10})
	if err != nil {
		t.Fatalf("List(Last=10) with nonexistent file should not error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for Last=10, got %d", len(entries))
	}
}

func TestRotatedFilesOrder(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	// Create rotated files out of order
	os.WriteFile(logPath+".3", []byte{}, 0644)
	os.WriteFile(logPath+".1", []byte{}, 0644)
	os.WriteFile(logPath+".2", []byte{}, 0644)

	reader := NewAuditReader(logPath)
	files := reader.rotatedFiles()

	if len(files) != 3 {
		t.Fatalf("expected 3 rotated files, got %d", len(files))
	}

	// Should be oldest first: .3, .2, .1
	if !strings.HasSuffix(files[0], ".3") {
		t.Errorf("expected oldest file .3 first, got %s", files[0])
	}
	if !strings.HasSuffix(files[2], ".1") {
		t.Errorf("expected newest rotated file .1 last, got %s", files[2])
	}
}
