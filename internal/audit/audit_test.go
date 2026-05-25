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
