// NewAPI Tools - Docker management platform for newapi
// Package osutil provides OS-level adapters for package management and system detection.
package osutil

import (
	"fmt"
	"os/exec"
	"runtime"
)

// OSAdapter defines the interface for OS-specific operations.
type OSAdapter interface {
	// GetPkgManager returns the primary package manager command (e.g., "apt-get", "yum").
	GetPkgManager() string
	// InstallDocker installs Docker using the OS package manager.
	InstallDocker() error
	// InstallPackage installs a package by name using the OS package manager.
	InstallPackage(name string) error
	// GetServiceManager returns the service manager command (e.g., "systemctl").
	GetServiceManager() string
	// DetectOS returns the OS family, distribution ID, and version.
	DetectOS() (family string, id string, version string)
}

// Detect auto-detects the current OS and returns the appropriate adapter.
// Falls back to a generic adapter if the OS is not recognized.
func Detect() OSAdapter {
	if runtime.GOOS != "linux" {
		return &genericAdapter{os: runtime.GOOS}
	}

	release, err := parseOSRelease()
	if err != nil {
		return &genericAdapter{os: "linux-unknown"}
	}

	switch release.ID {
	case "debian", "ubuntu":
		return &debianAdapter{release: release}
	case "centos", "rocky", "almalinux", "rhel":
		return &rhelAdapter{release: release}
	case "fedora":
		return &fedoraAdapter{release: release}
	case "arch", "manjaro", "endeavouros":
		return &archAdapter{release: release}
	default:
		return &genericAdapter{os: release.ID}
	}
}

// osRelease holds parsed /etc/os-release data.
type osRelease struct {
	ID        string
	IDLike    string
	Name      string
	Version   string
	VersionID string
}

// runCmd executes a command and returns an error if it fails.
func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %v failed: %w", name, args, err)
	}
	return nil
}

// genericAdapter is a fallback adapter for unrecognized operating systems.
type genericAdapter struct {
	os string
}

func (g *genericAdapter) GetPkgManager() string    { return "unknown" }
func (g *genericAdapter) InstallDocker() error      { return fmt.Errorf("unsupported OS: %s", g.os) }
func (g *genericAdapter) InstallPackage(name string) error { return fmt.Errorf("unsupported OS: %s", g.os) }
func (g *genericAdapter) GetServiceManager() string { return "systemctl" }
func (g *genericAdapter) DetectOS() (string, string, string) {
	return g.os, g.os, ""
}
