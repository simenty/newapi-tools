package docker

import (
	"context"
	"strings"
	"testing"
)

func TestSafeProjectDir(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"/safe/path", true},
		{"/safe/./path", true},
		{"/safe/../path", true}, // resolves to /path
		{"relative/path", true},
		{".", true},
	}

	for _, tc := range tests {
		result, err := SafeProjectDir(tc.input)
		if tc.valid && err != nil {
			t.Errorf("SafeProjectDir(%q) unexpected error: %v", tc.input, err)
		}
		if tc.valid && result == "" {
			t.Errorf("SafeProjectDir(%q) returned empty", tc.input)
		}
		if tc.valid {
			if !strings.HasPrefix(result, "/") && !strings.Contains(result, ":") {
				t.Logf("SafeProjectDir(%q) = %q (not absolute, relative input)", tc.input, result)
			}
		}
	}
}

func TestSplitComposeCommand(t *testing.T) {
	tests := []struct {
		cmd  string
		args []string
		want []string
	}{
		{
			cmd:  "docker compose",
			args: []string{"-f", "/path/docker-compose.yml", "up", "-d"},
			want: []string{"docker", "compose", "-f", "/path/docker-compose.yml", "up", "-d"},
		},
		{
			cmd:  "docker-compose",
			args: []string{"pull"},
			want: []string{"docker-compose", "pull"},
		},
		{
			cmd:  "/usr/local/bin/docker compose",
			args: []string{"ps", "--format", "json"},
			want: []string{"/usr/local/bin/docker", "compose", "ps", "--format", "json"},
		},
	}

	for _, tc := range tests {
		got := splitComposeCommand(tc.cmd, tc.args...)
		if len(got) != len(tc.want) {
			t.Errorf("splitComposeCommand(%q) len=%d, want %d\ngot:  %v\nwant: %v", tc.cmd, len(got), len(tc.want), got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("splitComposeCommand(%q)[%d] = %q, want %q", tc.cmd, i, got[i], tc.want[i])
			}
		}
	}
}

func TestSplitComposeCommandEmpty(t *testing.T) {
	got := splitComposeCommand("")
	if len(got) != 0 {
		t.Errorf("expected empty result for empty cmd, got %v", got)
	}
}

func TestServiceStatusStruct(t *testing.T) {
	s := ServiceStatus{
		Name:    "web",
		State:   "running",
		Ports:   "0.0.0.0:80->80/tcp",
		Running: true,
	}
	if s.Name != "web" || !s.Running {
		t.Error("ServiceStatus struct fields not set correctly")
	}
}

func TestComposePsParse(t *testing.T) {
	if !IsDockerAvailable() {
		t.Skip("docker not available, skipping")
	}
	c, err := NewClient("")
	if err != nil {
		t.Fatal(err)
	}
	// Test with non-existent dir — expect error, not panic
	_, err = c.ComposePs(context.Background(), "/nonexistent-dir-xyz")
	if err == nil {
		t.Log("ComposePs with nonexistent dir returned nil (may have docker fallback)")
	}
}
