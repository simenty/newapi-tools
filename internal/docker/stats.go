// NewAPI Tools - Docker container stats via CLI
package docker

import (
	"fmt"
	"os/exec"
	"strings"
)

// ContainerStats holds resource usage statistics for a single container.
type ContainerStats struct {
	Name     string // Container name
	CPUPerc  string // CPU percentage, e.g. "0.50%"
	MemUsage string // Memory usage, e.g. "50MiB / 1GiB"
	MemPerc  string // Memory percentage, e.g. "5.00%"
	NetIO    string // Network I/O, e.g. "1.2kB / 680B"
	BlockIO  string // Block I/O, e.g. "5.5MB / 0B"
}

// GetContainerStats retrieves resource usage statistics for a container
// by running `docker stats --no-stream` with a Go template format string.
func GetContainerStats(containerName string) (*ContainerStats, error) {
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return nil, fmt.Errorf("docker not found in PATH: %w", err)
	}

	format := "{{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}\t{{.NetIO}}\t{{.BlockIO}}"
	cmd := exec.Command(dockerPath, "stats", "--no-stream", "--format", format, containerName)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats for container %s: %w", containerName, err)
	}

	line := strings.TrimSpace(string(output))
	if line == "" {
		return nil, fmt.Errorf("no stats output for container %s", containerName)
	}

	parts := strings.SplitN(line, "\t", 6)
	if len(parts) < 6 {
		return nil, fmt.Errorf("unexpected stats format for container %s: %q", containerName, line)
	}

	// Strip leading "/" from container name (docker stats prefixes it)
	name := strings.TrimPrefix(parts[0], "/")

	return &ContainerStats{
		Name:     name,
		CPUPerc:  parts[1],
		MemUsage: parts[2],
		MemPerc:  parts[3],
		NetIO:    parts[4],
		BlockIO:  parts[5],
	}, nil
}
