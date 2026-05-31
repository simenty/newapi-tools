// NewAPI Tools - Docker daemon registry mirror management
package docker

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/simenty/newapi-tools/internal/core"
)

const daemonJSONPath = "/etc/docker/daemon.json"

// DaemonJSONPath returns the path of the Docker daemon configuration file.
func DaemonJSONPath() string { return daemonJSONPath }

// BuiltinMirrors is the list of well-known registry mirrors for Chinese users.
// Keys are short names, values are the mirror URLs.
var BuiltinMirrors = map[string]string{
	"tuna":     "https://docker.mirrors.tuna.tsinghua.edu.cn",
	"aliyun":   "https://registry.cn-hangzhou.aliyuncs.com",
	"ustc":     "https://docker.mirrors.ustc.edu.cn",
	"163":      "https://hub-mirror.c.163.com",
	"azure":    "https://dockerhub.azk8s.cn",
	"daocloud": "https://f1361db2.m.daocloud.io",
}

// MirrorConfig represents the registry-mirrors section in daemon.json.
type MirrorConfig struct {
	Mirrors []string `json:"registry-mirrors"`
}

// ReadDaemonJSON reads /etc/docker/daemon.json and returns parsed content.
// Returns empty map if file doesn't exist.
func ReadDaemonJSON() (map[string]interface{}, error) {
	data, err := os.ReadFile(daemonJSONPath)
	if os.IsNotExist(err) {
		return map[string]interface{}{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", daemonJSONPath, err)
	}
	if len(data) == 0 {
		return map[string]interface{}{}, nil
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", daemonJSONPath, err)
	}
	return cfg, nil
}

// WriteDaemonJSON writes a map to /etc/docker/daemon.json.
// Creates a backup of the original file first.
func WriteDaemonJSON(cfg map[string]interface{}) error {
	// Backup existing file
	if _, err := os.Stat(daemonJSONPath); err == nil {
		backupPath := daemonJSONPath + ".bak." + time.Now().Format("20060102150405")
		data, _ := os.ReadFile(daemonJSONPath)
		_ = os.WriteFile(backupPath, data, 0600)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal daemon.json: %w", err)
	}

	if err := os.MkdirAll("/etc/docker", 0755); err != nil {
		return fmt.Errorf("failed to create /etc/docker: %w", err)
	}

	if err := os.WriteFile(daemonJSONPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write %s: %w", daemonJSONPath, err)
	}
	return nil
}

// GetCurrentMirrors reads daemon.json and returns the current registry-mirrors list.
func GetCurrentMirrors() ([]string, error) {
	cfg, err := ReadDaemonJSON()
	if err != nil {
		return nil, err
	}

	raw, ok := cfg["registry-mirrors"]
	if !ok {
		return []string{}, nil
	}

	// Handle both []interface{} and []string
	switch v := raw.(type) {
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result, nil
	case []string:
		return v, nil
	default:
		return []string{}, nil
	}
}

// SetMirrors writes the given mirrors to daemon.json, preserving other fields.
// Deduplicates the list before writing.
func SetMirrors(mirrors []string) error {
	// Deduplicate
	seen := make(map[string]bool)
	deduped := make([]string, 0, len(mirrors))
	for _, m := range mirrors {
		m = strings.TrimRight(m, "/")
		if m != "" && !seen[m] {
			seen[m] = true
			deduped = append(deduped, m)
		}
	}

	cfg, err := ReadDaemonJSON()
	if err != nil {
		return err
	}

	if len(deduped) == 0 {
		delete(cfg, "registry-mirrors")
	} else {
		cfg["registry-mirrors"] = deduped
	}

	return WriteDaemonJSON(cfg)
}

// AddMirror appends a mirror URL to the current list (skips duplicates).
func AddMirror(mirror string) error {
	mirror = strings.TrimRight(mirror, "/")
	current, err := GetCurrentMirrors()
	if err != nil {
		return err
	}
	for _, m := range current {
		if m == mirror {
			return nil // already present
		}
	}
	return SetMirrors(append(current, mirror))
}

// RemoveMirror removes a specific mirror from daemon.json.
func RemoveMirror(mirror string) error {
	mirror = strings.TrimRight(mirror, "/")
	current, err := GetCurrentMirrors()
	if err != nil {
		return err
	}
	filtered := make([]string, 0, len(current))
	for _, m := range current {
		if m != mirror {
			filtered = append(filtered, m)
		}
	}
	return SetMirrors(filtered)
}

// ReloadDocker sends SIGHUP to Docker daemon (systemctl reload docker).
// Falls back to "systemctl restart docker" if reload fails.
func ReloadDocker() error {
	// Try systemctl reload (graceful, no downtime)
	if err := exec.Command("systemctl", "reload", "docker").Run(); err == nil {
		return nil
	}
	// Try kill -HUP for non-systemd setups
	if err := exec.Command("killall", "-HUP", "dockerd").Run(); err == nil {
		return nil
	}
	// Last resort: restart
	return exec.Command("systemctl", "restart", "docker").Run()
}

// TestMirror checks if a mirror URL is reachable by doing a quick HTTP HEAD.
func TestMirror(mirror string) error {
	// Resolve short name to URL if needed
	if url, ok := BuiltinMirrors[mirror]; ok {
		mirror = url
	}

	// Use curl with short timeout; don't fail on HTTP error codes (just network)
	cmd := exec.Command("curl", "-sf", "--max-time", "5", "--head", mirror+"/v2/")
	out, err := cmd.CombinedOutput()
	if err != nil {
		// curl exit 7 = connection refused, exit 28 = timeout
		return fmt.Errorf("unreachable: %w (output: %s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// ResolveShortName expands a short mirror name (e.g. "tuna") to full URL.
// Returns the input unchanged if it's already a full URL.
func ResolveShortName(nameOrURL string) (string, bool) {
	if url, ok := BuiltinMirrors[nameOrURL]; ok {
		return url, true
	}
	if strings.HasPrefix(nameOrURL, "http://") || strings.HasPrefix(nameOrURL, "https://") {
		return strings.TrimRight(nameOrURL, "/"), true
	}
	return nameOrURL, false
}

// MirrorTestTarget holds the name and URL of a mirror to test.
type MirrorTestTarget struct {
	Name string
	URL  string
}

// MirrorTestResult holds the result of testing a single mirror.
type MirrorTestResult struct {
	Name      string
	URL       string
	Reachable bool
	Latency   time.Duration
	Error     string // failure reason; empty on success
}

// ConcurrentMirrorTest tests multiple mirrors concurrently using HTTP HEAD requests.
// concurrency controls the maximum number of parallel requests (0 or negative uses 6).
// timeout is the per-request deadline (0 uses 3s).
// Results are returned sorted by Latency ascending; unreachable mirrors are placed last
// (their Latency is set to math.MaxInt64 nanoseconds).
func ConcurrentMirrorTest(mirrors []MirrorTestTarget, concurrency int, timeout time.Duration) []MirrorTestResult {
	if concurrency <= 0 {
		concurrency = 6
	}
	if timeout <= 0 {
		timeout = 3 * time.Second
	}

	results := make([]MirrorTestResult, len(mirrors))
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	client := &http.Client{
		Timeout: timeout,
		// Don't follow redirects for latency accuracy.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	for i, m := range mirrors {
		wg.Add(1)
		go func(idx int, target MirrorTestTarget) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			url := strings.TrimRight(target.URL, "/") + "/v2/"
			req, err := http.NewRequest(http.MethodHead, url, nil)
			if err != nil {
				results[idx] = MirrorTestResult{
					Name:      target.Name,
					URL:       target.URL,
					Reachable: false,
					Latency:   time.Duration(math.MaxInt64),
					Error:     err.Error(),
				}
				return
			}
			req.Header.Set("User-Agent", "newapi-tools/"+core.Version)

			start := time.Now()
			resp, err := client.Do(req)
			latency := time.Since(start)
			if err != nil {
				results[idx] = MirrorTestResult{
					Name:      target.Name,
					URL:       target.URL,
					Reachable: false,
					Latency:   time.Duration(math.MaxInt64),
					Error:     err.Error(),
				}
				return
			}
			_ = resp.Body.Close()

			results[idx] = MirrorTestResult{
				Name:      target.Name,
				URL:       target.URL,
				Reachable: true,
				Latency:   latency,
			}
		}(i, m)
	}

	wg.Wait()

	// Sort: reachable mirrors by ascending latency; unreachable mirrors at the end.
	sort.Slice(results, func(i, j int) bool {
		return results[i].Latency < results[j].Latency
	})

	return results
}

// AutoSelectMirror tests all built-in mirrors concurrently and returns the
// fastest reachable one. Returns nil if none are reachable.
// The timeout per mirror is 5 seconds.
func AutoSelectMirror() *MirrorTestResult {
	type result struct {
		Name      string
		URL       string
		Reachable bool
		Latency   time.Duration
	}

	ch := make(chan result, len(BuiltinMirrors))

	for name, url := range BuiltinMirrors {
		go func(name, url string) {
			start := time.Now()
			err := TestMirror(url)
			latency := time.Since(start)
			ch <- result{
				Name:      name,
				URL:       url,
				Reachable: err == nil,
				Latency:   latency,
			}
		}(name, url)
	}

	var best *MirrorTestResult
	for range BuiltinMirrors {
		r := <-ch
		if !r.Reachable {
			continue
		}
		if best == nil || r.Latency < best.Latency {
			best = &MirrorTestResult{
				Name:      r.Name,
				URL:       r.URL,
				Reachable: r.Reachable,
				Latency:   r.Latency,
			}
		}
	}
	return best
}
