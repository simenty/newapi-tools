// NewAPI Tools - Security utilities tests
package security

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestMaskSecret(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"mysecretpassword", "my****rd"},
		{"abcdefghijklmnop", "ab****op"},
		{"123456", "12****56"},
		{"abcdefghij", "ab****ij"},
	}
	for _, tt := range tests {
		got := MaskSecret(tt.input)
		if got != tt.expected {
			t.Errorf("MaskSecret(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestMaskSecretShort(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"abc", "****"},
		{"hello", "****"},
		{"12345", "****"},
		{"a", "****"},
	}
	for _, tt := range tests {
		got := MaskSecret(tt.input)
		if got != tt.expected {
			t.Errorf("MaskSecret(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestMaskSecretEmpty(t *testing.T) {
	got := MaskSecret("")
	if got != "****" {
		t.Errorf("MaskSecret(\"\") = %q, want %q", got, "****")
	}
}

func TestCheckConfigPermNonExistent(t *testing.T) {
	err := CheckConfigPerm("/tmp/nonexistent-config-file-xyz.yml")
	if err != nil {
		t.Errorf("CheckConfigPerm on non-existent file should return nil, got: %v", err)
	}
}

func TestCheckConfigPermValid(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping permission check on Windows")
	}

	// Create a temp file with restricted permissions (0600)
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.yml")
	if err := os.WriteFile(path, []byte("test"), 0600); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	err := CheckConfigPerm(path)
	if err != nil {
		t.Errorf("CheckConfigPerm on 0600 file should return nil, got: %v", err)
	}
}

func TestCheckConfigPermTooOpen(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping permission check on Windows")
	}

	// Create a temp file with open permissions (0666)
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config-open.yml")
	if err := os.WriteFile(path, []byte("test"), 0666); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	err := CheckConfigPerm(path)
	if err == nil {
		t.Error("CheckConfigPerm on 0666 file should return an error")
	}
}

func TestCheckDockerGroup(t *testing.T) {
	// Just verify it doesn't error on the current system
	inGroup, err := CheckDockerGroup()
	if err != nil {
		t.Errorf("CheckDockerGroup() returned error: %v", err)
	}
	if runtime.GOOS == "windows" && !inGroup {
		t.Error("CheckDockerGroup() should return true on Windows")
	}
	// On Linux, we just check it runs without error; the result depends on the system
	t.Logf("Current user in docker group: %v", inGroup)
}
