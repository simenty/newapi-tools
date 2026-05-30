// NewAPI Tools - Docker management platform for newapi
package docker

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// ServiceStatus represents the status of a compose service.
type ServiceStatus struct {
	Name    string
	State   string
	Ports   string
	Running bool
}

// ComposeUp starts the compose services in detached mode.
// Uses exec.Command to call docker compose CLI.
func (c *Client) ComposeUp(ctx context.Context, projectDir string) error {
	composeCmd := c.composeCmd
	parts := strings.Split(composeCmd, " ")
	args := append(parts, "-f", projectDir+"/docker-compose.yml", "up", "-d")

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = projectDir
	cmd.Stdout = ComposeStdout
	cmd.Stderr = ComposeStderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("compose up failed: %w", err)
	}
	return nil
}

// ComposeDown stops and removes the compose services.
func (c *Client) ComposeDown(ctx context.Context, projectDir string) error {
	composeCmd := c.composeCmd
	parts := strings.Split(composeCmd, " ")
	args := append(parts, "-f", projectDir+"/docker-compose.yml", "down")

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = projectDir
	cmd.Stdout = ComposeStdout
	cmd.Stderr = ComposeStderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("compose down failed: %w", err)
	}
	return nil
}

// ComposePull pulls the latest images for the compose services.
func (c *Client) ComposePull(ctx context.Context, projectDir string) error {
	composeCmd := c.composeCmd
	parts := strings.Split(composeCmd, " ")
	args := append(parts, "-f", projectDir+"/docker-compose.yml", "pull")

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = projectDir
	cmd.Stdout = ComposeStdout
	cmd.Stderr = ComposeStderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("compose pull failed: %w", err)
	}
	return nil
}

// ComposePs lists the status of compose services.
func (c *Client) ComposePs(ctx context.Context, projectDir string) ([]ServiceStatus, error) {
	composeCmd := c.composeCmd
	parts := strings.Split(composeCmd, " ")
	args := append(parts, "-f", projectDir+"/docker-compose.yml", "ps", "--format", "json")

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = projectDir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("compose ps failed: %w", err)
	}

	// Parse the output (simplified - just return raw output for now)
	var services []ServiceStatus
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		services = append(services, ServiceStatus{
			Name:    line,
			Running: strings.Contains(line, "running") || strings.Contains(line, "Up"),
		})
	}

	return services, nil
}

// --- Legacy functions for backwards compatibility (will be deprecated later)
func ComposeUp(ctx context.Context, projectDir string, composeCmd string) error {
	c, err := NewClient(composeCmd)
	if err != nil {
		return err
	}
	return c.ComposeUp(ctx, projectDir)
}

func ComposeDown(ctx context.Context, projectDir string, composeCmd string) error {
	c, err := NewClient(composeCmd)
	if err != nil {
		return err
	}
	return c.ComposeDown(ctx, projectDir)
}

func ComposePull(ctx context.Context, projectDir string, composeCmd string) error {
	c, err := NewClient(composeCmd)
	if err != nil {
		return err
	}
	return c.ComposePull(ctx, projectDir)
}

func ComposePs(ctx context.Context, projectDir string, composeCmd string) ([]ServiceStatus, error) {
	c, err := NewClient(composeCmd)
	if err != nil {
		return nil, err
	}
	return c.ComposePs(ctx, projectDir)
}
