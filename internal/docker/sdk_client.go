package docker

import (
	"context"
	"fmt"
)

// SDKClient provides Docker operations via the Docker REST API (HTTP).
// Uses net/http standard library — no external SDK dependencies.
// For compose operations, it delegates to the CLI-based Client.
type SDKClient struct {
	cliClient *Client
}

// NewSDKClient creates an SDKClient.
// The underlying implementation uses HTTP for container operations
// and CLI for compose operations.
func NewSDKClient(composeCmd string) (*SDKClient, error) {
	cli, err := NewClient(composeCmd)
	if err != nil {
		return nil, err
	}
	return &SDKClient{cliClient: cli}, nil
}

// Close is a no-op.
func (c *SDKClient) Close() error { return nil }

// IsAvailable checks Docker daemon availability via HTTP API ping.
func (c *SDKClient) IsAvailable() bool {
	return c.cliClient.IsAvailableHTTP()
}

// ContainerList returns all containers via the Docker REST API.
func (c *SDKClient) ContainerList(ctx context.Context) ([]ContainerInfo, error) {
	return c.cliClient.ContainerListHTTP(ctx)
}

// ContainerInspect returns container state via CLI.
func (c *SDKClient) ContainerInspect(ctx context.Context, name string) (string, error) {
	return c.cliClient.ContainerInspect(ctx, name)
}

// ContainerStart starts a container via CLI.
func (c *SDKClient) ContainerStart(ctx context.Context, name string) error {
	return c.cliClient.ContainerStart(ctx, name)
}

// ContainerStop stops a container via CLI.
func (c *SDKClient) ContainerStop(ctx context.Context, name string) error {
	return c.cliClient.ContainerStop(ctx, name)
}

// ContainerRemove removes a container via CLI.
func (c *SDKClient) ContainerRemove(ctx context.Context, name string) error {
	return c.cliClient.ContainerRemove(ctx, name)
}

// ImagePull pulls a Docker image via CLI.
func (c *SDKClient) ImagePull(ctx context.Context, ref string) error {
	return c.cliClient.ImagePull(ctx, ref)
}

// FindContainerByName finds a container by name using HTTP API.
func (c *SDKClient) FindContainerByName(ctx context.Context, name string) (*ContainerInfo, error) {
	return c.cliClient.FindContainerByName(ctx, name)
}

// ComposeUp starts compose services via CLI.
func (c *SDKClient) ComposeUp(ctx context.Context, projectDir string) error {
	return c.cliClient.ComposeUp(ctx, projectDir)
}

// ComposeDown stops compose services via CLI.
func (c *SDKClient) ComposeDown(ctx context.Context, projectDir string) error {
	return c.cliClient.ComposeDown(ctx, projectDir)
}

// ComposePull pulls compose images via CLI.
func (c *SDKClient) ComposePull(ctx context.Context, projectDir string) error {
	return c.cliClient.ComposePull(ctx, projectDir)
}

// ComposePs lists compose services via CLI.
func (c *SDKClient) ComposePs(ctx context.Context, projectDir string) ([]ServiceStatus, error) {
	return c.cliClient.ComposePs(ctx, projectDir)
}

// GetContainerStats retrieves container stats via the Docker REST API.
func GetContainerStatsSDK(containerName string) (*ContainerStats, error) {
	return GetContainerStatsHTTP(containerName)
}

// Ensure SDKClient implements DockerClient.
var _ DockerClient = (*SDKClient)(nil)

// Compile-time check: SDKClient is an alias for the HTTP + CLI hybrid.
// The original Docker SDK has been replaced with net/http standard library
// calls to avoid heavy dependencies and Windows compatibility issues.
//
// SDK v27.5.1:    Windows sockets.DialPipe undefined
// SDK v28.0.0:    requires Go 1.25+
// go-dockerclient: requires Go 1.25.5+
// net/http:       works everywhere, zero dependencies
func init() {
	// Force import of fmt to avoid unused import error
	_ = fmt.Sprintf
}
