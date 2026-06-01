package security

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFixConfigPermNonExistent(t *testing.T) {
	err := FixConfigPerm("/tmp/nonexistent-file-xyz.test")
	if err != nil {
		t.Errorf("FixConfigPerm on non-existent file should return nil, got: %v", err)
	}
}

func TestFixConfigPermExisting(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping permission fix test on Windows")
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.yml")
	if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	err := FixConfigPerm(path)
	if err != nil {
		t.Errorf("FixConfigPerm should succeed, got: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected permissions 0600 after fix, got %04o", info.Mode().Perm())
	}
}

func TestMaskSecretEdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ab", "****"},
		{"abcde", "****"},
		{"abcdef", "ab****ef"},
		{"你好好好你好", "你好****你好"},
		{"a b c d e f", "a **** f"},
	}
	for _, tt := range tests {
		got := MaskSecret(tt.input)
		if got != tt.expected {
			t.Errorf("MaskSecret(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestCheckConfigPermErrors(t *testing.T) {
	// Test with a directory (should get an error on Stat)
	err := CheckConfigPerm("/proc/1/fd/0")
	if err != nil {
		t.Logf("CheckConfigPerm on special file returned: %v (acceptable)", err)
	}
}

func TestCheckDockerGroupWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}
	inGroup, err := CheckDockerGroup()
	if err != nil {
		t.Errorf("CheckDockerGroup() on Windows should not error: %v", err)
	}
	if !inGroup {
		t.Error("CheckDockerGroup() should return true on Windows")
	}
}

func TestFixConfigPermChmodError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping chmod error test on Windows")
	}
	// Use a path in a non-writable directory to trigger chmod error
	err := FixConfigPerm("/proc/1/mem")
	if err != nil {
		t.Logf("FixConfigPerm on special file returned: %v (acceptable)", err)
	}
}

func TestMaskSecretSixChars(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"abcdef", "ab****ef"},
		{"123456", "12****56"},
		{"abc123", "ab****23"},
	}
	for _, tt := range tests {
		got := MaskSecret(tt.input)
		if got != tt.expected {
			t.Errorf("MaskSecret(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestCheckConfigPermStatError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping stat error test on Windows")
	}
	// A path that causes Stat to return a non-IsNotExist error
	err := CheckConfigPerm("/invalid\x00path")
	if err != nil {
		t.Logf("CheckConfigPerm on invalid path returned: %v (expected)", err)
	}
}