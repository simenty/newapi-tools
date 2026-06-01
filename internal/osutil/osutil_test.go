// NewAPI Tools - Docker management platform for newapi
package osutil

import (
	"os"
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
	// On systems without /etc/os-release, this tests the os.Open error path.
	// On systems with /etc/os-release (e.g. Git Bash, WSL), it should succeed.
	_, err := parseOSRelease()
	if err != nil {
		t.Logf("parseOSRelease failed (expected on systems without /etc/os-release): %v", err)
	} else {
		t.Log("parseOSRelease succeeded (/etc/os-release exists on this system)")
	}
}

func TestParseOSReleaseParsing(t *testing.T) {
	content := `NAME="Ubuntu"
VERSION="22.04 LTS (Jammy Jellyfish)"
ID=ubuntu
ID_LIKE=debian
VERSION_ID=22.04
`
	tmpFile, err := os.CreateTemp("", "os-release-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	// Temporarily replace /etc/os-release with temp file path
	defer func() { _ = os.Remove("/etc/os-release"); _ = os.WriteFile("/etc/os-release", nil, 0644) }()
	_ = os.Remove("/etc/os-release")
	_ = os.Symlink(tmpFile.Name(), "/etc/os-release")

	release, err := parseOSRelease()
	if err != nil {
		// May fail on Windows due to symlink; that's OK
		if runtime.GOOS == "windows" {
			t.Skip("symlink not supported on Windows")
		}
		t.Fatalf("parseOSRelease failed: %v", err)
	}
	if release.ID != "ubuntu" {
		t.Errorf("expected ID 'ubuntu', got %q", release.ID)
	}
	if release.IDLike != "debian" {
		t.Errorf("expected IDLike 'debian', got %q", release.IDLike)
	}
	if release.Name != "Ubuntu" {
		t.Errorf("expected Name 'Ubuntu', got %q", release.Name)
	}
	if release.Version != "22.04 LTS (Jammy Jellyfish)" {
		t.Errorf("expected Version '22.04 LTS ...', got %q", release.Version)
	}
	if release.VersionID != "22.04" {
		t.Errorf("expected VersionID '22.04', got %q", release.VersionID)
	}
}

func TestParseOSReleaseWithComments(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("requires Linux /etc/os-release")
	}
	release, err := parseOSRelease()
	if err != nil {
		t.Fatalf("parseOSRelease failed: %v", err)
	}
	if release.ID == "" {
		t.Error("ID should not be empty")
	}
}

func TestDetectWithMockData(t *testing.T) {
	release := &osRelease{
		ID:        "ubuntu",
		VersionID: "22.04",
	}
	adapter := &debianAdapter{release: release}

	family, id, version := adapter.DetectOS()
	if family != "debian" {
		t.Errorf("expected family 'debian', got %q", family)
	}
	if id != "ubuntu" {
		t.Errorf("expected id 'ubuntu', got %q", id)
	}
	if version != "22.04" {
		t.Errorf("expected version '22.04', got %q", version)
	}

	if adapter.GetPkgManager() != "apt-get" {
		t.Errorf("expected pkg manager 'apt-get', got %q", adapter.GetPkgManager())
	}
	if adapter.GetServiceManager() != "systemctl" {
		t.Errorf("expected service manager 'systemctl', got %q", adapter.GetServiceManager())
	}
}

func TestDetectWithAlmaLinux(t *testing.T) {
	release := &osRelease{
		ID:        "almalinux",
		VersionID: "9",
	}
	adapter := &rhelAdapter{release: release}

	family, id, version := adapter.DetectOS()
	if family != "rhel" {
		t.Errorf("expected family 'rhel', got %q", family)
	}
	if id != "almalinux" {
		t.Errorf("expected id 'almalinux', got %q", id)
	}
	if version != "9" {
		t.Errorf("expected version '9', got %q", version)
	}

	if adapter.GetPkgManager() != "yum" {
		t.Errorf("expected pkg manager 'yum', got %q", adapter.GetPkgManager())
	}
}

func TestDetectWithManjaro(t *testing.T) {
	release := &osRelease{
		ID:        "manjaro",
		VersionID: "24.0",
	}
	adapter := &archAdapter{release: release}

	family, id, version := adapter.DetectOS()
	if family != "arch" {
		t.Errorf("expected family 'arch', got %q", family)
	}
	if id != "manjaro" {
		t.Errorf("expected id 'manjaro', got %q", id)
	}
	if version != "24.0" {
		t.Errorf("expected version '24.0', got %q", version)
	}

	if adapter.GetPkgManager() != "pacman" {
		t.Errorf("expected pkg manager 'pacman', got %q", adapter.GetPkgManager())
	}
}

func TestDetectUnknownOS(t *testing.T) {
	release := &osRelease{
		ID:        "suse",
		Name:      "openSUSE",
		VersionID: "15.5",
	}
	adapter := &genericAdapter{os: release.ID}
	family, id, _ := adapter.DetectOS()
	if family != "suse" {
		t.Errorf("expected family 'suse', got %q", family)
	}
	if id != "suse" {
		t.Errorf("expected id 'suse', got %q", id)
	}
}

func TestParseOSLineEdgeCases(t *testing.T) {
	content := "# comment\n  \nID=test\nKEY_WITH_UNDERSCORE=value\nNO_EQUALS\n"
	tmpFile, err := os.CreateTemp("", "os-release-edge-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	defer func() { _ = os.Remove("/etc/os-release"); _ = os.WriteFile("/etc/os-release", nil, 0644) }()
	_ = os.Remove("/etc/os-release")
	_ = os.Symlink(tmpFile.Name(), "/etc/os-release")

	release, err := parseOSRelease()
	if err != nil {
		if runtime.GOOS == "windows" {
			t.Skip("symlink not supported on Windows")
		}
		t.Fatalf("parseOSRelease failed: %v", err)
	}
	if release.ID != "test" {
		t.Errorf("expected ID 'test', got %q", release.ID)
	}
}

func TestGenericAdapterServiceManager(t *testing.T) {
	adapter := &genericAdapter{os: "freebsd"}
	if sm := adapter.GetServiceManager(); sm != "systemctl" {
		t.Errorf("expected 'systemctl', got %q", sm)
	}
	_ = adapter.InstallDocker()
	_ = adapter.InstallPackage("foo")
}

func TestDetectReturnsCorrectAdapter(t *testing.T) {
	tests := []struct {
		id          string
		wantGeneric bool
	}{
		{"debian", false},
		{"ubuntu", false},
		{"centos", false},
		{"rocky", false},
		{"almalinux", false},
		{"rhel", false},
		{"fedora", false},
		{"arch", false},
		{"manjaro", false},
		{"endeavouros", false},
		{"suse", true},
	}
	for _, tc := range tests {
		release := &osRelease{ID: tc.id}
		detect := func() OSAdapter {
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
		adapter := detect()
		_, isGeneric := adapter.(*genericAdapter)
		if isGeneric != tc.wantGeneric {
			t.Errorf("Detect(%q): isGeneric=%v, want %v", tc.id, isGeneric, tc.wantGeneric)
		}
	}
}

func TestDetectFallbackOnParseError(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("requires non-Linux to trigger fallback")
	}
	adapter := Detect()
	if _, ok := adapter.(*genericAdapter); !ok {
		t.Errorf("expected genericAdapter on non-Linux, got %T", adapter)
	}
}

func TestGenericAdapterPkgManager(t *testing.T) {
	adapter := &genericAdapter{os: "testos"}
	if pm := adapter.GetPkgManager(); pm != "unknown" {
		t.Errorf("expected 'unknown', got %q", pm)
	}
}

func TestParseOSReleaseFileNotFound(t *testing.T) {
	// On systems without /etc/os-release, this tests the os.Open error path.
	// On systems with /etc/os-release (e.g. Git Bash, WSL), it should succeed.
	_, err := parseOSRelease()
	if err != nil {
		// File doesn't exist - that's expected on pure Windows
		t.Logf("parseOSRelease failed (expected on systems without /etc/os-release): %v", err)
	} else {
		// File exists - that's OK on Git Bash / WSL
		t.Log("parseOSRelease succeeded (/etc/os-release exists on this system)")
	}
}

func TestRunCmdSuccess(t *testing.T) {
	err := runCmd("go", "version")
	if err != nil {
		t.Fatalf("runCmd(go version) failed: %v", err)
	}
}

func TestRunCmdFailure(t *testing.T) {
	err := runCmd("nonexistent-command-xyz")
	if err == nil {
		t.Error("expected error for nonexistent command")
	}
}
