package selfupdate

import (
	"os"
	"strings"
	"testing"
)

func TestParseSemver(t *testing.T) {
	tests := []struct {
		input        string
		wantOK       bool
		wantMajor    int
		wantMinor    int
		wantPatch    int
		wantPreRelease string
	}{
		{"v3.2.0", true, 3, 2, 0, ""},
		{"v3.2.0-rc1", true, 3, 2, 0, "rc1"},
		{"v3.2.0-dev", true, 3, 2, 0, "dev"},
		{"v10.20.30", true, 10, 20, 30, ""},
		{"3.2.0", true, 3, 2, 0, ""},           // no v prefix
		{"invalid", false, 0, 0, 0, ""},
		{"v3.2", false, 0, 0, 0, ""},
		{"v3.2.a", false, 0, 0, 0, ""},
		{"", false, 0, 0, 0, ""},
		{"v3.2.0-alpha.1", true, 3, 2, 0, "alpha.1"},
	}

	for _, tc := range tests {
		got, ok := parseSemver(tc.input)
		if ok != tc.wantOK {
			t.Errorf("parseSemver(%q) ok=%v, want %v", tc.input, ok, tc.wantOK)
		}
		if ok {
			if got.major != tc.wantMajor {
				t.Errorf("parseSemver(%q) major=%d, want %d", tc.input, got.major, tc.wantMajor)
			}
			if got.minor != tc.wantMinor {
				t.Errorf("parseSemver(%q) minor=%d, want %d", tc.input, got.minor, tc.wantMinor)
			}
			if got.patch != tc.wantPatch {
				t.Errorf("parseSemver(%q) patch=%d, want %d", tc.input, got.patch, tc.wantPatch)
			}
			if got.preRelease != tc.wantPreRelease {
				t.Errorf("parseSemver(%q) preRelease=%q, want %q", tc.input, got.preRelease, tc.wantPreRelease)
			}
		}
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		current string
		latest  string
		want    bool
		wantErr bool
	}{
		{"v3.2.0", "v3.3.0", true, false},
		{"v3.3.0", "v3.2.0", false, false},
		{"v3.2.0", "v3.2.1", true, false},
		{"v3.2.1", "v3.2.0", false, false},
		{"v3.2.0", "v4.0.0", true, false},
		{"v4.0.0", "v3.2.0", false, false},
		{"v3.2.0", "v3.2.0", false, false},           // same version
		{"v3.2.0-rc1", "v3.2.0", true, false},        // pre-release → release
		{"v3.2.0", "v3.2.0-rc1", false, false},       // release → pre-release (not an update)
		{"v3.2.0-rc1", "v3.2.0-rc2", true, false},    // newer pre-release
		{"v3.2.0-rc2", "v3.2.0-rc1", false, false},   // older pre-release
		{"invalid", "v3.2.0", false, true},            // invalid current
		{"v3.2.0", "invalid", false, true},            // invalid latest
		{"v3.2.0-alpha", "v3.2.0-beta", true, false}, // alpha < beta (string comparison)
	}

	for _, tc := range tests {
		got, err := CompareVersions(tc.current, tc.latest)
		if tc.wantErr {
			if err == nil {
				t.Errorf("CompareVersions(%q, %q) expected error", tc.current, tc.latest)
			}
			continue
		}
		if err != nil {
			t.Errorf("CompareVersions(%q, %q) unexpected error: %v", tc.current, tc.latest, err)
			continue
		}
		if got != tc.want {
			t.Errorf("CompareVersions(%q, %q) = %v, want %v", tc.current, tc.latest, got, tc.want)
		}
	}
}

func TestResolveAssetName(t *testing.T) {
	name := resolveAssetName("v3.4.0")
	if name == "" {
		t.Fatal("resolveAssetName returned empty")
	}
	if len(name) < 10 {
		t.Errorf("resolveAssetName result too short: %q", name)
	}
}

func TestReleaseInfoStruct(t *testing.T) {
	r := ReleaseInfo{
		TagName: "v3.4.0",
		Name:    "v3.4.0 Release",
	}
	if r.TagName != "v3.4.0" {
		t.Errorf("TagName = %q, want %q", r.TagName, "v3.4.0")
	}
}

func TestAssetStruct(t *testing.T) {
	a := Asset{
		Name:               "newapi-tools_v3.4.0_Linux_x86_64.tar.gz",
		BrowserDownloadURL: "https://github.com/simenty/newapi-tools/releases/download/v3.4.0/asset.tar.gz",
		Size:               1024,
	}
	if a.Name == "" || a.BrowserDownloadURL == "" {
		t.Error("Asset fields not set")
	}
}

func TestCopyFile(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	srcPath := src + "/source.txt"
	dstPath := dst + "/dest.txt"
	content := "test content for copy"
	if err := os.WriteFile(srcPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(srcPath, dstPath); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}
	data, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != content {
		t.Errorf("content mismatch: got %q, want %q", string(data), content)
	}
}

func TestCopyFileSrcNotFound(t *testing.T) {
	err := copyFile("/nonexistent/path/file.txt", "/tmp/dest.txt")
	if err == nil {
		t.Error("expected error for non-existent source")
	}
}

func TestResolveAssetNameAllPlatforms(t *testing.T) {
	name := resolveAssetName("v3.4.0")
	if name == "" {
		t.Fatal("resolveAssetName returned empty")
	}
	if !strings.Contains(name, "v3.4.0") {
		t.Errorf("expected version in asset name, got %q", name)
	}
	if !strings.Contains(name, "newapi-tools_") {
		t.Errorf("expected project prefix in asset name, got %q", name)
	}
}