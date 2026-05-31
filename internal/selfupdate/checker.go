// Package selfupdate provides automatic update functionality for newapi-tools
package selfupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ReleaseInfo represents GitHub Release version information
type ReleaseInfo struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	PublishedAt time.Time `json:"published_at"`
	HTMLURL     string    `json:"html_url"`
	Assets      []Asset   `json:"assets"`
}

// Asset represents a download asset in a GitHub Release
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// CheckLatest queries GitHub Releases API for the latest version
// repo format: "owner/repo" e.g. "Bonus520/newapi-tools"
func CheckLatest(ctx context.Context, repo string) (*ReleaseInfo, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "newapi-tools")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &release, nil
}

// CompareVersions compares current version with latest version using semver logic.
// Returns true if latest is strictly greater than current.
// Pre-release versions (e.g. "v3.3.0-rc1") are always treated as older than the release version.
func CompareVersions(current, latest string) (bool, error) {
	cur := parseSemver(current)
	lat := parseSemver(latest)

	if cur.major != lat.major {
		return lat.major > cur.major, nil
	}
	if cur.minor != lat.minor {
		return lat.minor > cur.minor, nil
	}
	if cur.patch != lat.patch {
		return lat.patch > cur.patch, nil
	}
	// Same base version: pre-release is older than release
	if cur.preRelease == "" && lat.preRelease != "" {
		return false, nil // latest is pre-release of same version → not an update
	}
	if cur.preRelease != "" && lat.preRelease == "" {
		return true, nil // current is pre-release, latest is release → update
	}
	// Both are pre-release: compare pre-release strings
	return lat.preRelease > cur.preRelease, nil
}

// semver represents a parsed semantic version.
type semver struct {
	major      int
	minor      int
	patch      int
	preRelease string // e.g. "rc1", "dev", "alpha"
}

// parseSemver parses a version string like "v3.2.0" or "v3.2.0-rc1".
// Returns zero semver on parse failure.
func parseSemver(v string) semver {
	s := strings.TrimPrefix(v, "v")

	// Split pre-release suffix
	parts := strings.SplitN(s, "-", 2)
	preRelease := ""
	if len(parts) == 2 {
		preRelease = parts[1]
	}

	// Parse major.minor.patch
	nums := strings.SplitN(parts[0], ".", 3)
	if len(nums) != 3 {
		return semver{}
	}

	major, _ := strconv.Atoi(nums[0])
	minor, _ := strconv.Atoi(nums[1])
	patch, _ := strconv.Atoi(nums[2])

	return semver{major: major, minor: minor, patch: patch, preRelease: preRelease}
}
