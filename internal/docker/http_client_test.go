package docker

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestDockerHostDefault(t *testing.T) {
	// Save and restore env
	oldHost := os.Getenv("DOCKER_HOST")
	defer os.Setenv("DOCKER_HOST", oldHost)

	os.Unsetenv("DOCKER_HOST")
	host := dockerHost()
	if host == "" {
		t.Error("dockerHost() should return a non-empty default")
	}
	if !strings.HasPrefix(host, "tcp://") && !strings.HasPrefix(host, "npipe://") {
		t.Errorf("unexpected host scheme: %q", host)
	}
}

func TestDockerHostCustom(t *testing.T) {
	oldHost := os.Getenv("DOCKER_HOST")
	defer os.Setenv("DOCKER_HOST", oldHost)

	os.Setenv("DOCKER_HOST", "tcp://192.168.1.100:2376")
	host := dockerHost()
	if host != "tcp://192.168.1.100:2376" {
		t.Errorf("expected custom DOCKER_HOST, got %q", host)
	}
}

func TestDockerHostCustomUnix(t *testing.T) {
	oldHost := os.Getenv("DOCKER_HOST")
	defer os.Setenv("DOCKER_HOST", oldHost)

	os.Setenv("DOCKER_HOST", "unix:///var/run/docker.sock")
	host := dockerHost()
	if host != "unix:///var/run/docker.sock" {
		t.Errorf("expected unix DOCKER_HOST, got %q", host)
	}
}

func TestHTTPClientErrors(t *testing.T) {
	// npipe scheme should error on httpClient
	oldHost := os.Getenv("DOCKER_HOST")
	defer os.Setenv("DOCKER_HOST", oldHost)

	os.Setenv("DOCKER_HOST", "npipe:////./pipe/docker_engine")
	cli, baseURL, err := httpClient()
	if err == nil {
		t.Error("expected error for npipe scheme")
	}
	if cli != nil {
		t.Error("expected nil client for npipe scheme")
	}
	if baseURL != "" {
		t.Errorf("expected empty baseURL, got %q", baseURL)
	}
}

func TestDockerContainerStruct(t *testing.T) {
	// Verify the struct exists and fields are accessible
	ctr := dockerContainer{
		ID:     "abc123",
		Names:  []string{"/test"},
		Image:  "nginx:latest",
		State:  "running",
		Status: "Up 2 hours",
		Labels: map[string]string{"com.docker.compose.project": "proj"},
	}
	if ctr.ID != "abc123" {
		t.Errorf("ID = %q, want %q", ctr.ID, "abc123")
	}
	if ctr.Image != "nginx:latest" {
		t.Errorf("Image = %q, want %q", ctr.Image, "nginx:latest")
	}
	if len(ctr.Names) == 0 || ctr.Names[0] != "/test" {
		t.Errorf("Names = %v, want [\"/test\"]", ctr.Names)
	}
}

func TestHTTPClientTCP(t *testing.T) {
	oldHost := os.Getenv("DOCKER_HOST")
	defer os.Setenv("DOCKER_HOST", oldHost)

	os.Setenv("DOCKER_HOST", "tcp://127.0.0.1:65535")
	cli, baseURL, err := httpClient()
	if err != nil {
		t.Fatalf("httpClient should succeed for TCP even if unreachable: %v", err)
	}
	if cli == nil {
		t.Fatal("expected non-nil client")
	}
	if baseURL != "http://127.0.0.1:65535/v1.47" {
		t.Errorf("unexpected baseURL: %q", baseURL)
	}
}

func TestHTTPClientUnix(t *testing.T) {
	oldHost := os.Getenv("DOCKER_HOST")
	defer os.Setenv("DOCKER_HOST", oldHost)

	os.Setenv("DOCKER_HOST", "unix:///var/run/docker.sock")
	cli, baseURL, err := httpClient()
	if err != nil {
		t.Fatalf("httpClient should succeed for unix socket: %v", err)
	}
	if cli == nil {
		t.Fatal("expected non-nil client")
	}
	if baseURL != "http://localhost/v1.47" {
		t.Errorf("unexpected baseURL: %q", baseURL)
	}
}

func TestIsAvailableHTTPWithMock(t *testing.T) {
	oldHost := os.Getenv("DOCKER_HOST")
	defer os.Setenv("DOCKER_HOST", oldHost)

	os.Setenv("DOCKER_HOST", "tcp://127.0.0.1:1")
	c := &Client{}
	result := c.IsAvailableHTTP()
	t.Logf("IsAvailableHTTP (unreachable) = %v", result)
}

func TestContainerListHTTPFallback(t *testing.T) {
	oldHost := os.Getenv("DOCKER_HOST")
	defer os.Setenv("DOCKER_HOST", oldHost)

	os.Setenv("DOCKER_HOST", "tcp://127.0.0.1:1")
	c := &Client{}
	containers, err := c.ContainerListHTTP(context.Background())
	if err != nil {
		t.Logf("ContainerListHTTP error (expected if no docker): %v", err)
	}
	_ = containers
}
