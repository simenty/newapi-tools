// Package selfupdate provides automatic update functionality for newapi-tools
package selfupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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

// CompareVersions compares current version with latest version
// Uses simple string comparison.
// For pre-release tags like "v3.3.0-rc1", compares the base version for updates
// and treats any pre-release suffix as a newer version than the base.
func CompareVersions(current, latest string) (bool, error) {
	// Strip leading 'v' prefix for comparison
	cur := strings.TrimPrefix(current, "v")
	lat := strings.TrimPrefix(latest, "v")
	return cur != lat, nil
}
