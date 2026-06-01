package docker

import (
	"context"
	"testing"
)

func TestParseContainerLine(t *testing.T) {
	tests := []struct {
		line     string
		wantOK   bool
		wantName string
		wantID   string
		wantImg  string
		wantSt   string
	}{
		{
			line:     "abc123|my-container|nginx:latest|running|Up 2 hours|",
			wantOK:   true,
			wantID:   "abc123",
			wantName: "my-container",
			wantImg:  "nginx:latest",
			wantSt:   "running",
		},
		{
			line:     "def456|web|alpine:3.18|exited|Exited (0) 1 hour ago|myproject",
			wantOK:   true,
			wantID:   "def456",
			wantName: "web",
			wantImg:  "alpine:3.18",
			wantSt:   "exited",
		},
		{
			line:     "a1b2c3|db|mysql:8|running|Up 5 hours|",
			wantOK:   true,
			wantID:   "a1b2c3",
			wantName: "db",
			wantImg:  "mysql:8",
			wantSt:   "running",
		},
	}

	for _, tc := range tests {
		got, ok := parseContainerLine(tc.line)
		if ok != tc.wantOK {
			t.Errorf("parseContainerLine(%q) ok=%v, want %v", tc.line, ok, tc.wantOK)
		}
		if ok {
			if got.ID != tc.wantID {
				t.Errorf("ID = %q, want %q", got.ID, tc.wantID)
			}
			if got.Name != tc.wantName {
				t.Errorf("Name = %q, want %q", got.Name, tc.wantName)
			}
			if got.Image != tc.wantImg {
				t.Errorf("Image = %q, want %q", got.Image, tc.wantImg)
			}
			if got.State != tc.wantSt {
				t.Errorf("State = %q, want %q", got.State, tc.wantSt)
			}
		}
	}
}

func TestParseContainerLineInvalid(t *testing.T) {
	lines := []string{
		"",
		"only-one-field",
		"a|b|c|d",         // only 4 parts
		"a|b|c|d|e|f|g|h", // too many — still valid with 6 max
	}
	for _, line := range lines {
		got, ok := parseContainerLine(line)
		if line == "a|b|c|d|e|f|g|h" {
			if !ok {
				t.Errorf("expected 6-part line to be valid: %q", line)
			}
			continue
		}
		if ok {
			t.Errorf("expected invalid for line %q, got %+v", line, got)
		}
	}
}

func TestParseContainerLineProject(t *testing.T) {
	line := "id123|svc|img:tag|running|Up 1m|myproject"
	got, ok := parseContainerLine(line)
	if !ok {
		t.Fatal("expected valid line")
	}
	if got.ComposeProject != "myproject" {
		t.Errorf("ComposeProject = %q, want %q", got.ComposeProject, "myproject")
	}

	lineNoProject := "id124|svc2|img2|paused|Paused|"
	got2, ok2 := parseContainerLine(lineNoProject)
	if !ok2 {
		t.Fatal("expected valid line")
	}
	if got2.ComposeProject != "" {
		t.Errorf("ComposeProject should be empty, got %q", got2.ComposeProject)
	}
}

func TestIsDockerAvailable(t *testing.T) {
	// Just verify it runs without panic; result depends on system
	result := IsDockerAvailable()
	// Should return false if docker not in PATH, or true if it is
	t.Logf("IsDockerAvailable() = %v", result)
}

func TestNewClient(t *testing.T) {
	// Test with default compose command
	c, err := NewClient("")
	if err != nil && !IsDockerAvailable() {
		t.Skip("docker not available, skipping")
	}
	if err == nil {
		if c.dockerPath == "" {
			t.Error("expected non-empty dockerPath")
		}
		if c.composeCmd != "docker compose" {
			t.Errorf("expected default composeCmd 'docker compose', got %q", c.composeCmd)
		}
		c.Close()
	}

	// Test with custom compose command
	c2, err := NewClient("docker-compose")
	if err == nil {
		if c2.composeCmd != "docker-compose" {
			t.Errorf("expected composeCmd 'docker-compose', got %q", c2.composeCmd)
		}
		c2.Close()
	}
}

func TestClose(t *testing.T) {
	c := &Client{}
	if err := c.Close(); err != nil {
		t.Errorf("Close should return nil, got %v", err)
	}
}

func TestFindContainerByNameEmpty(t *testing.T) {
	if !IsDockerAvailable() {
		t.Skip("docker not available, skipping")
	}
	c, err := NewClient("")
	if err != nil {
		t.Fatal(err)
	}
	result, err := c.FindContainerByName(context.Background(), "nonexistent-container-xyz-123")
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if result != nil {
		t.Logf("unexpectedly found container: %+v", result)
	}
}
