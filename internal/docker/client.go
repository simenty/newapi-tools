// NewAPI Tools - Docker management platform for newapi
// Package docker provides Docker interaction through the CLI (exec.Command).
// Docker SDK integration will be added in a future iteration for type-safe
// container operations. For now, all operations go through docker CLI
// which is more portable and works with compose out of the box.
package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Client provides Docker operations via the CLI.
type Client struct {
	dockerPath string
	composeCmd string
}

// DockerClient defines the interface for Docker operations, enabling mock implementations for testing.
type DockerClient interface {
	Close() error
	IsAvailable() bool
	ContainerList(ctx context.Context) ([]ContainerInfo, error)
	ContainerInspect(ctx context.Context, name string) (string, error)
	ContainerStart(ctx context.Context, name string) error
	ContainerStop(ctx context.Context, name string) error
	ContainerRemove(ctx context.Context, name string) error
	ImagePull(ctx context.Context, ref string) error
	FindContainerByName(ctx context.Context, name string) (*ContainerInfo, error)
	ComposeUp(ctx context.Context, projectDir string) error
	ComposeDown(ctx context.Context, projectDir string) error
	ComposePull(ctx context.Context, projectDir string) error
	ComposePs(ctx context.Context, projectDir string) ([]ServiceStatus, error)
}

// NewClient creates a new Docker client by locating the docker binary.
func NewClient(composeCmd string) (*Client, error) {
	path, err := exec.LookPath("docker")
	if err != nil {
		return nil, fmt.Errorf("docker not found in PATH: %w", err)
	}
	if composeCmd == "" {
		composeCmd = "docker compose"
	}
	return &Client{dockerPath: path, composeCmd: composeCmd}, nil
}

// Close is a no-op for CLI-based client.
func (c *Client) Close() error { return nil }

// IsAvailable checks whether Docker is accessible.
func (c *Client) IsAvailable() bool {
	cmd := exec.Command(c.dockerPath, "info")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

// ContainerInfo holds summary information about a container.
type ContainerInfo struct {
	ID             string
	Name           string
	Image          string
	State          string
	Status         string
	ComposeProject string // compose project name (com.docker.compose.project label)
}

// ContainerList returns all containers, including stopped ones.
func (c *Client) ContainerList(ctx context.Context) ([]ContainerInfo, error) {
	cmd := exec.CommandContext(ctx, c.dockerPath, "ps", "-a", "--format", "{{.ID}}|{{.Names}}|{{.Image}}|{{.State}}|{{.Status}}|{{.Label \"com.docker.compose.project\"}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var containers []ContainerInfo
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 6) // now 6 parts including compose project
		if len(parts) < 5 {
			continue
		}
		containers = append(containers, ContainerInfo{
			ID:             parts[0],
			Name:           parts[1],
			Image:          parts[2],
			State:          parts[3],
			Status:         parts[4],
			ComposeProject: parts[5],
		})
	}

	return containers, nil
}

// ContainerInspect returns detailed information about a container.
func (c *Client) ContainerInspect(ctx context.Context, name string) (string, error) {
	cmd := exec.CommandContext(ctx, c.dockerPath, "inspect", "--format", "{{.State.Status}}", name)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to inspect container %s: %w", name, err)
	}
	return strings.TrimSpace(string(output)), nil
}

// ContainerStart starts a container by name or ID.
func (c *Client) ContainerStart(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, c.dockerPath, "start", name)
	cmd.Stdout = ComposeStdout
	cmd.Stderr = ComposeStderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start container %s: %w", name, err)
	}
	return nil
}

// ContainerStop stops a container by name or ID.
func (c *Client) ContainerStop(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, c.dockerPath, "stop", name)
	cmd.Stdout = ComposeStdout
	cmd.Stderr = ComposeStderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop container %s: %w", name, err)
	}
	return nil
}

// ContainerRemove removes a container by name or ID.
func (c *Client) ContainerRemove(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, c.dockerPath, "rm", "-f", name)
	cmd.Stdout = ComposeStdout
	cmd.Stderr = ComposeStderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove container %s: %w", name, err)
	}
	return nil
}

// ImagePull pulls a Docker image.
func (c *Client) ImagePull(ctx context.Context, ref string) error {
	cmd := exec.CommandContext(ctx, c.dockerPath, "pull", ref)
	cmd.Stdout = ComposeStdout
	cmd.Stderr = ComposeStderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pull image %s: %w", ref, err)
	}
	return nil
}

// FindContainerByName finds a container by name.
func (c *Client) FindContainerByName(ctx context.Context, name string) (*ContainerInfo, error) {
	containers, err := c.ContainerList(ctx)
	if err != nil {
		return nil, err
	}

	for _, ctr := range containers {
		if strings.Contains(ctr.Name, name) {
			return &ctr, nil
		}
	}
	return nil, nil
}

// IsDockerAvailable checks if Docker is available without creating a Client.
func IsDockerAvailable() bool {
	_, err := exec.LookPath("docker")
	return err == nil
}

// Stdout and stderr for Docker/compose operations, overridable in tests.
var (
	ComposeStdout io.Writer = os.Stdout
	ComposeStderr io.Writer = os.Stderr
)
