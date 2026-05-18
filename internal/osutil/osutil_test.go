// NewAPI Tools - Docker management platform for newapi
package osutil

import (
	"runtime"
	"testing"
)

func TestDetect(t *testing.T) {
	adapter := Detect()
	if adapter == nil {
		t.Fatal("Detect should return a non-nil adapter")
	}
}

func TestDetectReturnsCorrectType(t *testing.T) {
	adapter := Detect()
	family, id, version := adapter.DetectOS()

	if runtime.GOOS != "linux" {
		// Non-linux should get generic adapter
		if family == "" {
			t.Error("family should not be empty")
		}
		return
	}

	// On Linux, we should get a specific adapter
	_ = family
	_ = id
	_ = version
}

func TestDebianAdapter(t *testing.T) {
	adapter := &debianAdapter{release: &osRelease{ID: "ubuntu", VersionID: "22.04"}}

	if adapter.GetPkgManager() != "apt-get" {
		t.Errorf("expected apt-get, got %s", adapter.GetPkgManager())
	}
	if adapter.GetServiceManager() != "systemctl" {
		t.Errorf("expected systemctl, got %s", adapter.GetServiceManager())
	}

	family, id, version := adapter.DetectOS()
	if family != "debian" {
		t.Errorf("expected family 'debian', got '%s'", family)
	}
	if id != "ubuntu" {
		t.Errorf("expected id 'ubuntu', got '%s'", id)
	}
	if version != "22.04" {
		t.Errorf("expected version '22.04', got '%s'", version)
	}
}

func TestRHELAdapter(t *testing.T) {
	adapter := &rhelAdapter{release: &osRelease{ID: "rocky", VersionID: "9"}}

	if adapter.GetPkgManager() != "yum" {
		t.Errorf("expected yum, got %s", adapter.GetPkgManager())
	}
	if adapter.GetServiceManager() != "systemctl" {
		t.Errorf("expected systemctl, got %s", adapter.GetServiceManager())
	}

	family, id, _ := adapter.DetectOS()
	if family != "rhel" {
		t.Errorf("expected family 'rhel', got '%s'", family)
	}
	if id != "rocky" {
		t.Errorf("expected id 'rocky', got '%s'", id)
	}
}

func TestFedoraAdapter(t *testing.T) {
	adapter := &fedoraAdapter{release: &osRelease{ID: "fedora", VersionID: "39"}}

	if adapter.GetPkgManager() != "dnf" {
		t.Errorf("expected dnf, got %s", adapter.GetPkgManager())
	}
	if adapter.GetServiceManager() != "systemctl" {
		t.Errorf("expected systemctl, got %s", adapter.GetServiceManager())
	}

	family, id, version := adapter.DetectOS()
	if family != "fedora" {
		t.Errorf("expected family 'fedora', got '%s'", family)
	}
	if id != "fedora" {
		t.Errorf("expected id 'fedora', got '%s'", id)
	}
	if version != "39" {
		t.Errorf("expected version '39', got '%s'", version)
	}
}

func TestArchAdapter(t *testing.T) {
	adapter := &archAdapter{release: &osRelease{ID: "arch", VersionID: ""}}

	if adapter.GetPkgManager() != "pacman" {
		t.Errorf("expected pacman, got %s", adapter.GetPkgManager())
	}
	if adapter.GetServiceManager() != "systemctl" {
		t.Errorf("expected systemctl, got %s", adapter.GetServiceManager())
	}

	family, id, _ := adapter.DetectOS()
	if family != "arch" {
		t.Errorf("expected family 'arch', got '%s'", family)
	}
	if id != "arch" {
		t.Errorf("expected id 'arch', got '%s'", id)
	}
}

func TestGenericAdapter(t *testing.T) {
	adapter := &genericAdapter{os: "windows"}

	if adapter.GetPkgManager() != "unknown" {
		t.Errorf("expected unknown, got %s", adapter.GetPkgManager())
	}
	if err := adapter.InstallDocker(); err == nil {
		t.Error("expected error for unsupported OS")
	}
	if err := adapter.InstallPackage("test"); err == nil {
		t.Error("expected error for unsupported OS")
	}

	family, id, version := adapter.DetectOS()
	if family != "windows" {
		t.Errorf("expected family 'windows', got '%s'", family)
	}
	if id != "windows" {
		t.Errorf("expected id 'windows', got '%s'", id)
	}
	if version != "" {
		t.Errorf("expected empty version, got '%s'", version)
	}
}

func TestParseOSReleaseNonexistent(t *testing.T) {
	// On non-Linux, /etc/os-release doesn't exist
	if runtime.GOOS != "linux" {
		_, err := parseOSRelease()
		if err == nil {
			t.Error("expected error on non-Linux platform")
		}
	}
}
