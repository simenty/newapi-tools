package docker

import (
	"testing"
)

func TestNewSDKClient(t *testing.T) {
	if !IsDockerAvailable() {
		t.Skip("docker not available, skipping")
	}
	client, err := NewSDKClient("")
	if err != nil {
		t.Fatalf("NewSDKClient failed: %v", err)
	}
	if client.cliClient == nil {
		t.Error("expected non-nil cliClient")
	}
	if err := client.Close(); err != nil {
		t.Errorf("Close() = %v, want nil", err)
	}
}

func TestSDKClientInterface(t *testing.T) {
	var _ DockerClient = (*SDKClient)(nil)
}