package docker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func dockerHostFromURL(rawURL string) string {
	return "tcp://" + strings.TrimPrefix(rawURL, "http://")
}

func TestIsAvailableHTTPWithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1.47/_ping" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	oldHost := os.Getenv("DOCKER_HOST")
	defer os.Setenv("DOCKER_HOST", oldHost)

	os.Setenv("DOCKER_HOST", dockerHostFromURL(server.URL))
	c := &Client{}
	result := c.IsAvailableHTTP()
	if !result {
		t.Error("IsAvailableHTTP should return true for mock server")
	}
}

func TestContainerListHTTPWithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1.47/containers/json" && r.URL.RawQuery == "all=true" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[
				{"Id":"abc123def45678","Names":["/test-container"],"Image":"nginx:latest","State":"running","Status":"Up 2 hours","Labels":{}},
				{"Id":"def45678901234","Names":["/web"],"Image":"alpine","State":"exited","Status":"Exited (0)","Labels":{"com.docker.compose.project":"myapp"}}
			]`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	oldHost := os.Getenv("DOCKER_HOST")
	defer os.Setenv("DOCKER_HOST", oldHost)

	os.Setenv("DOCKER_HOST", dockerHostFromURL(server.URL))
	c := &Client{}
	containers, err := c.ContainerListHTTP(context.Background())
	if err != nil {
		t.Fatalf("ContainerListHTTP failed: %v", err)
	}
	if len(containers) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(containers))
	}
	if containers[0].ID != "abc123def456" || containers[0].Name != "test-container" {
		t.Errorf("first container mismatch: %+v", containers[0])
	}
	if containers[1].ID != "def456789012" || containers[1].ComposeProject != "myapp" {
		t.Errorf("second container mismatch: %+v", containers[1])
	}
}

func TestContainerListHTTPWithMockServerEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	oldHost := os.Getenv("DOCKER_HOST")
	defer os.Setenv("DOCKER_HOST", oldHost)

	os.Setenv("DOCKER_HOST", dockerHostFromURL(server.URL))
	c := &Client{}
	containers, err := c.ContainerListHTTP(context.Background())
	if err != nil {
		t.Fatalf("ContainerListHTTP failed: %v", err)
	}
	if len(containers) != 0 {
		t.Errorf("expected 0 containers, got %d", len(containers))
	}
}

func TestGetContainerStatsHTTPWithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1.47/containers/test-container/stats" && r.URL.RawQuery == "stream=false" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"cpu_stats": {
					"cpu_usage": {"total_usage": 1000, "percpu_usage": [100, 200]},
					"system_cpu_usage": 50000
				},
				"precpu_stats": {
					"cpu_usage": {"total_usage": 500, "percpu_usage": [50, 100]},
					"system_cpu_usage": 40000
				},
				"memory_stats": {"usage": 104857600, "limit": 1073741824}
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	oldHost := os.Getenv("DOCKER_HOST")
	defer os.Setenv("DOCKER_HOST", oldHost)

	os.Setenv("DOCKER_HOST", dockerHostFromURL(server.URL))
	stats, err := GetContainerStatsHTTP("test-container")
	if err != nil {
		t.Fatalf("GetContainerStatsHTTP failed: %v", err)
	}
	if stats.Name != "test-container" {
		t.Errorf("Name = %q, want %q", stats.Name, "test-container")
	}
	if stats.MemUsage == "" {
		t.Error("MemUsage should not be empty")
	}
	if stats.MemPerc == "" {
		t.Error("MemPerc should not be empty")
	}
	t.Logf("Stats: CPU=%s Mem=%s MemPct=%s", stats.CPUPerc, stats.MemUsage, stats.MemPerc)
}