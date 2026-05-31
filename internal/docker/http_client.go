package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// windowsPipePath is the default Docker named pipe path on Windows.
const windowsPipePath = `\\.\pipe\docker_engine`

// defaultDockerHost is the default TCP address for Docker daemon.
const defaultDockerHost = "tcp://localhost:2375"

// dockerHost returns the Docker daemon address from DOCKER_HOST or the default.
func dockerHost() string {
	if h := os.Getenv("DOCKER_HOST"); h != "" {
		return h
	}
	// On Windows, try named pipe first, fall back to TCP
	if _, err := os.Stat(windowsPipePath); err == nil {
		return "npipe://" + windowsPipePath
	}
	return defaultDockerHost
}

// httpClient returns an HTTP client configured to talk to the Docker daemon.
// Supports unix://, tcp://, and npipe:// schemes.
func httpClient() (*http.Client, string, error) {
	host := dockerHost()

	switch {
	case strings.HasPrefix(host, "unix://"):
		socketPath := strings.TrimPrefix(host, "unix://")
		// Also trim /var/run/docker.sock from paths like unix:///var/run/docker.sock
		if !filepath.IsAbs(socketPath) {
			socketPath = strings.TrimPrefix(socketPath, "/")
			socketPath = "/" + socketPath
		}
		baseURL := "http://localhost/v1.47"
		transport := &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		}
		return &http.Client{Transport: transport, Timeout: 30 * time.Second}, baseURL, nil

	case strings.HasPrefix(host, "npipe://"):
		// Named pipes on Windows — fall back to CLI for now
		return nil, "", fmt.Errorf("named pipes not supported via HTTP; use CLI fallback")

	default:
		// TCP: tcp://host:port or host:port
		addr := strings.TrimPrefix(host, "tcp://")
		baseURL := fmt.Sprintf("http://%s/v1.47", addr)
		return &http.Client{Timeout: 30 * time.Second}, baseURL, nil
	}
}

// dockerContainer holds the JSON structure from Docker API /containers/json.
type dockerContainer struct {
	ID     string            `json:"Id"`
	Names  []string          `json:"Names"`
	Image  string            `json:"Image"`
	State  string            `json:"State"`
	Status string            `json:"Status"`
	Labels map[string]string `json:"Labels"`
}

// ContainerListHTTP returns all containers via the Docker REST API.
// Falls back to CLI-based ContainerList if the API is unreachable.
func (c *Client) ContainerListHTTP(ctx context.Context) ([]ContainerInfo, error) {
	cli, baseURL, err := httpClient()
	if err != nil {
		return c.ContainerList(ctx)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/containers/json?all=true", nil)
	if err != nil {
		return c.ContainerList(ctx)
	}

	resp, err := cli.Do(req)
	if err != nil {
		return c.ContainerList(ctx)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.ContainerList(ctx)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.ContainerList(ctx)
	}

	var containers []dockerContainer
	if err := json.Unmarshal(body, &containers); err != nil {
		return c.ContainerList(ctx)
	}

	result := make([]ContainerInfo, 0, len(containers))
	for _, ctr := range containers {
		info := ContainerInfo{
			ID:     ctr.ID[:12],
			Image:  ctr.Image,
			State:  ctr.State,
			Status: ctr.Status,
		}
		if len(ctr.Names) > 0 {
			info.Name = strings.TrimPrefix(ctr.Names[0], "/")
		}
		if project, ok := ctr.Labels["com.docker.compose.project"]; ok {
			info.ComposeProject = project
		}
		result = append(result, info)
	}
	return result, nil
}

// IsAvailableHTTP checks Docker daemon availability via REST API ping.
// Falls back to CLI-based IsAvailable if the API is unreachable.
func (c *Client) IsAvailableHTTP() bool {
	cli, baseURL, err := httpClient()
	if err != nil {
		return c.IsAvailable()
	}

	req, err := http.NewRequest(http.MethodGet, baseURL+"/_ping", nil)
	if err != nil {
		return c.IsAvailable()
	}

	resp, err := cli.Do(req)
	if err != nil {
		return c.IsAvailable()
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// GetContainerStatsHTTP retrieves container stats via the Docker REST API.
// Falls back to CLI-based GetContainerStats if the API is unreachable.
func GetContainerStatsHTTP(containerName string) (*ContainerStats, error) {
	cli, baseURL, err := httpClient()
	if err != nil {
		return GetContainerStats(containerName)
	}

	req, err := http.NewRequest(http.MethodGet, baseURL+"/containers/"+containerName+"/stats?stream=false", nil)
	if err != nil {
		return GetContainerStats(containerName)
	}

	resp, err := cli.Do(req)
	if err != nil {
		return GetContainerStats(containerName)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return GetContainerStats(containerName)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return GetContainerStats(containerName)
	}

	// Parse minimal stats from the Docker API response
	var statsRaw map[string]interface{}
	if err := json.Unmarshal(body, &statsRaw); err != nil {
		return GetContainerStats(containerName)
	}

	stats := &ContainerStats{Name: containerName}

	// Calculate CPU percentage
	cpuStats, ok := statsRaw["cpu_stats"].(map[string]interface{})
	if !ok {
		return GetContainerStats(containerName)
	}
	preCPUStats, ok := statsRaw["precpu_stats"].(map[string]interface{})
	if !ok {
		return GetContainerStats(containerName)
	}

	cpuUsage, ok := cpuStats["cpu_usage"].(map[string]interface{})
	if !ok {
		return GetContainerStats(containerName)
	}
	preCPUUsage, ok := preCPUStats["cpu_usage"].(map[string]interface{})
	if !ok {
		return GetContainerStats(containerName)
	}

	totalUsage, _ := cpuUsage["total_usage"].(float64)
	preTotalUsage, _ := preCPUUsage["total_usage"].(float64)
	systemUsage, _ := cpuStats["system_cpu_usage"].(float64)
	preSystemUsage, _ := preCPUStats["system_cpu_usage"].(float64)

	cpuDelta := totalUsage - preTotalUsage
	sysDelta := systemUsage - preSystemUsage

	if sysDelta > 0 && cpuDelta > 0 {
		numCPU, _ := cpuUsage["percpu_usage"].([]interface{})
		cpuPerc := (cpuDelta / sysDelta) * 100.0 * float64(len(numCPU))
		stats.CPUPerc = fmt.Sprintf("%.2f%%", cpuPerc)
	} else {
		stats.CPUPerc = "0.00%"
	}

	// Parse memory
	memoryStats, ok := statsRaw["memory_stats"].(map[string]interface{})
	if ok {
		usage, _ := memoryStats["usage"].(float64)
		limit, _ := memoryStats["limit"].(float64)
		if limit > 0 {
			stats.MemUsage = fmt.Sprintf("%.0fMiB / %.0fMiB", usage/1024/1024, limit/1024/1024)
			stats.MemPerc = fmt.Sprintf("%.2f%%", (usage/limit)*100.0)
		}
	}

	return stats, nil
}

// Ensure DockerClient interface is satisfied
var _ DockerClient = (*Client)(nil)
var _ DockerClient = (*SDKClient)(nil)
