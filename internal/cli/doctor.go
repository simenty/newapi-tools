// NewAPI Tools - Docker management platform for newapi
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/simenty/newapi-tools/internal/apperr"
	"github.com/simenty/newapi-tools/internal/core"
	"github.com/simenty/newapi-tools/internal/docker"
	"github.com/simenty/newapi-tools/internal/security"
	"github.com/simenty/newapi-tools/internal/ui"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose new-api issues",
	Long:  `Run diagnostic checks on the new-api deployment including Docker connectivity, container health, configuration validity, and common issues.`,
	RunE:  runDoctor,
}

func init() {
	doctorCmd.Flags().Bool("fix", false, "attempt to auto-fix detected issues")
	doctorCmd.Flags().Bool("json", false, "output results in JSON format")
	doctorCmd.Flags().BoolP("verbose", "v", false, "show detailed diagnostic info for each check")

	rootCmd.AddCommand(doctorCmd)
}

// checkResult represents the outcome of a single diagnostic check.
type checkResult struct {
	Name    string         `json:"check"`
	Status  string         `json:"status"` // "OK", "WARN", "FAIL", "SKIP"
	Message string         `json:"message"`
	Detail  *VerboseCheck  `json:"detail,omitempty"` // Detailed diagnostic info (--verbose)
}

// VerboseCheck contains detailed info for a check (--verbose mode).
type VerboseCheck struct {
	FilePath  string `json:"file,omitempty"`  // File path involved in the check (if any)
	Command   string `json:"command,omitempty"`   // Command executed (if any)
	Expected  string `json:"expected,omitempty"`  // Expected value
	Actual    string `json:"actual,omitempty"`    // Actual value
	RawOutput string `json:"raw,omitempty"` // Raw command output (if any)
}

// Check represents a single diagnostic check with its name, runner, fixer, and hint.
type Check struct {
	Name    string
	Run     func(ctx context.Context, cfg *core.Config) checkResult
	Fix     func(ctx context.Context, cfg *core.Config) error // nil = not auto-fixable
	FixHint string                                            // Fix is nil: what user should do
}

// checks defines all diagnostic checks.
var checks = []Check{
	{
		Name:    "docker-binary",
		Run:     func(ctx context.Context, cfg *core.Config) checkResult { return checkDockerBinary() },
		Fix:     nil,
		FixHint: "请先安装 Docker: https://docs.docker.com/get-docker/",
	},
	{
		Name: "docker-daemon",
		Run: func(ctx context.Context, cfg *core.Config) checkResult {
			return checkDockerDaemon(cfg.Docker.ComposeCmd)
		},
		Fix:     nil,
		FixHint: "请启动 Docker daemon (例如: sudo systemctl start docker)",
	},
	{
		Name:    "home-dir",
		Run:     func(ctx context.Context, cfg *core.Config) checkResult { return checkHomeDir(cfg.NewAPI.Home) },
		Fix:     fixCreateHomeDir,
		FixHint: "",
	},
	{
		Name:    "compose-file",
		Run:     func(ctx context.Context, cfg *core.Config) checkResult { return checkComposeFile(cfg.NewAPI.Home) },
		Fix:     nil,
		FixHint: "运行 'newapi-tools install' 生成 docker-compose.yml",
	},
	{
		Name:    "env-file",
		Run:     func(ctx context.Context, cfg *core.Config) checkResult { return checkEnvFile(cfg.NewAPI.Home) },
		Fix:     nil,
		FixHint: "运行 'newapi-tools install' 生成 .env 文件",
	},
	{
		Name: "new-api-container",
		Run: func(ctx context.Context, cfg *core.Config) checkResult {
			return checkContainer(ctx, "new-api", cfg.Docker.ComposeCmd)
		},
		Fix:     composeUpFix,
		FixHint: "",
	},
	{
		Name: "mysql-container",
		Run: func(ctx context.Context, cfg *core.Config) checkResult {
			return checkContainer(ctx, "mysql", cfg.Docker.ComposeCmd)
		},
		Fix:     composeUpFix,
		FixHint: "",
	},
	{
		Name: "redis-container",
		Run: func(ctx context.Context, cfg *core.Config) checkResult {
			return checkContainer(ctx, "redis", cfg.Docker.ComposeCmd)
		},
		Fix:     composeUpFix,
		FixHint: "",
	},
	{
		Name:    "http-health",
		Run:     func(ctx context.Context, cfg *core.Config) checkResult { return checkHTTPHealth(cfg.NewAPI.Port) },
		Fix:     nil,
		FixHint: "检查 new-api 是否在监听端口",
	},
	{
		Name:    "disk-space",
		Run:     func(ctx context.Context, cfg *core.Config) checkResult { return checkDiskSpace(cfg.NewAPI.Home) },
		Fix:     nil,
		FixHint: "释放磁盘空间或移动数据到更大的卷",
	},
	{
		Name:    "config-permissions",
		Run:     func(ctx context.Context, cfg *core.Config) checkResult { return checkConfigPermissions(cfg) },
		Fix:     fixConfigPermissions,
		FixHint: "",
	},
	{
		Name:    "docker-group",
		Run:     func(ctx context.Context, cfg *core.Config) checkResult { return checkDockerGroupMembership() },
		Fix:     nil,
		FixHint: "运行: sudo usermod -aG docker $USER && newgrp docker",
	},
}

func runDoctor(cmd *cobra.Command, args []string) error {
	cfg := core.GetConfig()
	if cfg == nil {
		return fmt.Errorf("configuration not loaded")
	}

	fix, _ := cmd.Flags().GetBool("fix")
	jsonOut, _ := cmd.Flags().GetBool("json")
	verbose, _ := cmd.Flags().GetBool("verbose")

	fmt.Println("Running diagnostics...")
	fmt.Println()

	ctx := cmd.Context()

	// Run all checks
	results := runAllChecks(ctx, cfg)

	// ---- Output results ----
	if jsonOut {
		printDoctorJSON(results, verbose)
	} else {
		printDoctorTable(results, verbose)
	}

	// Count failures
	failCount, warnCount := countResults(results)

	fmt.Println()
	fmt.Printf("Checks: %d total, %d warnings, %d failures\n",
		len(results), warnCount, failCount)

	if fix && failCount > 0 {
		fmt.Println()
		fmt.Println("Attempting auto-fix...")
		fmt.Println()
		fixCount := runAutoFix(ctx, results, cfg)
		fmt.Printf("Auto-fix: %d issue(s) addressed\n", fixCount)

		// Re-run checks after fix to show updated status
		if fixCount > 0 {
			fmt.Println()
			fmt.Println("Re-checking after fix...")
			fmt.Println()
			results = runAllChecks(ctx, cfg)

			if jsonOut {
				printDoctorJSON(results, verbose)
			} else {
				printDoctorTable(results, verbose)
			}

			// Recount
			failCount, warnCount = countResults(results)
			fmt.Println()
			fmt.Printf("After fix: %d checks, %d warnings, %d failures\n",
				len(results), warnCount, failCount)
		}
	}

	ui.L().Info("doctor completed",
		"checks", len(results),
		"warnings", warnCount,
		"failures", failCount,
	)

	if failCount > 0 {
		return apperr.New(apperr.CodeDoctorFailed, fmt.Sprintf("%d 项诊断检查失败", failCount), "", nil)
	}
	return nil
}

func runAllChecks(ctx context.Context, cfg *core.Config) []checkResult {
	var results []checkResult
	for _, check := range checks {
		results = append(results, check.Run(ctx, cfg))
	}
	return results
}

func countResults(results []checkResult) (int, int) {
	failCount, warnCount := 0, 0
	for _, r := range results {
		switch r.Status {
		case "FAIL":
			failCount++
		case "WARN":
			warnCount++
		}
	}
	return failCount, warnCount
}

func checkDockerBinary() checkResult {
	path, err := exec.LookPath("docker")
	if err != nil {
		return checkResult{
			Name:    "docker-binary",
			Status:  "FAIL",
			Message: "docker not found in PATH",
			Detail: &VerboseCheck{
				Command:  "docker (PATH lookup)",
				Expected: "docker found in PATH",
				Actual:   fmt.Sprintf("err: %v", err),
			},
		}
	}
	return checkResult{
		Name:    "docker-binary",
		Status:  "OK",
		Message: path,
		Detail: &VerboseCheck{
			FilePath: path,
			Expected: "docker binary exists",
			Actual:   fmt.Sprintf("found at: %s", path),
		},
	}
}

func checkDockerDaemon(composeCmd string) checkResult {
	c, err := docker.NewClient(composeCmd)
	if err != nil {
		return checkResult{
			Name:    "docker-daemon",
			Status:  "FAIL",
			Message: err.Error(),
			Detail: &VerboseCheck{
				Command:  "docker.NewClient()",
				Expected: "successfully connected to Docker daemon",
				Actual:   fmt.Sprintf("err: %v", err),
			},
		}
	}
	defer c.Close()

	if !c.IsAvailable() {
		return checkResult{
			Name:    "docker-daemon",
			Status:  "FAIL",
			Message: "docker daemon is not running",
			Detail: &VerboseCheck{
				Command:  "docker.IsAvailable()",
				Expected: "daemon is running",
				Actual:   "daemon not responding",
			},
		}
	}
	return checkResult{
		Name:    "docker-daemon",
		Status:  "OK",
		Message: "daemon is accessible",
		Detail: &VerboseCheck{
			Expected: "daemon accessible",
			Actual:   "daemon connected",
		},
	}
}

func checkHomeDir(home string) checkResult {
	if home == "" {
		return checkResult{
			Name:    "home-dir",
			Status:  "WARN",
			Message: "not configured",
			Detail: &VerboseCheck{
				Expected: "home directory configured",
				Actual:   "home is empty",
			},
		}
	}
	if _, err := os.Stat(home); os.IsNotExist(err) {
		return checkResult{
			Name:    "home-dir",
			Status:  "FAIL",
			Message: fmt.Sprintf("%s does not exist", home),
			Detail: &VerboseCheck{
				FilePath: home,
				Expected: "directory exists",
				Actual:   fmt.Sprintf("err: %v", err),
			},
		}
	}
	return checkResult{
		Name:    "home-dir",
		Status:  "OK",
		Message: home,
		Detail: &VerboseCheck{
			FilePath: home,
			Expected: "directory exists",
			Actual:   fmt.Sprintf("directory exists at: %s", home),
		},
	}
}

func checkComposeFile(home string) checkResult {
	path := filepath.Join(home, "docker-compose.yml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return checkResult{
			Name:    "compose-file",
			Status:  "FAIL",
			Message: fmt.Sprintf("%s not found — run 'newapi-tools install'", path),
			Detail: &VerboseCheck{
				FilePath: path,
				Expected: "docker-compose.yml exists",
				Actual:   fmt.Sprintf("err: %v", err),
			},
		}
	}
	return checkResult{
		Name:    "compose-file",
		Status:  "OK",
		Message: path,
		Detail: &VerboseCheck{
			FilePath: path,
			Expected: "file exists",
			Actual:   fmt.Sprintf("found at: %s", path),
		},
	}
}

func checkEnvFile(home string) checkResult {
	path := filepath.Join(home, ".env")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return checkResult{
			Name:    "env-file",
			Status:  "WARN",
			Message: fmt.Sprintf("%s not found — credentials may be missing", path),
			Detail: &VerboseCheck{
				FilePath: path,
				Expected: ".env exists",
				Actual:   fmt.Sprintf("err: %v", err),
			},
		}
	}
	return checkResult{
		Name:    "env-file",
		Status:  "OK",
		Message: path,
		Detail: &VerboseCheck{
			FilePath: path,
			Expected: "file exists",
			Actual:   fmt.Sprintf("found at: %s", path),
		},
	}
}

func checkContainer(ctx context.Context, name string, composeCmd string) checkResult {
	checkName := name + "-container"
	c, err := docker.NewClient(composeCmd)
	if err != nil {
		return checkResult{
			Name:    checkName,
			Status:  "SKIP",
			Message: "docker unavailable",
			Detail: &VerboseCheck{
				Command:  "docker.NewClient()",
				Expected: "docker available",
				Actual:   fmt.Sprintf("err: %v", err),
			},
		}
	}
	defer c.Close()

	containers, listErr := c.ContainerList(ctx)
	if listErr != nil {
		return checkResult{
			Name:    checkName,
			Status:  "WARN",
			Message: fmt.Sprintf("failed to list containers: %v", listErr),
			Detail: &VerboseCheck{
				Command:  "docker.ContainerList()",
				Expected: "container list retrieved",
				Actual:   fmt.Sprintf("err: %v", listErr),
			},
		}
	}

	for _, ctr := range containers {
		if strings.Contains(ctr.Name, name) {
			if ctr.State == "running" {
				return checkResult{
					Name:    checkName,
					Status:  "OK",
					Message: fmt.Sprintf("%s (%s)", ctr.Image, ctr.Status),
					Detail: &VerboseCheck{
						Expected: "container is running",
						Actual:   fmt.Sprintf("state=%s, status=%s, image=%s", ctr.State, ctr.Status, ctr.Image),
					},
				}
			}
			return checkResult{
				Name:    checkName,
				Status:  "FAIL",
				Message: fmt.Sprintf("state=%s, status=%s", ctr.State, ctr.Status),
				Detail: &VerboseCheck{
					Expected: "container is running",
					Actual:   fmt.Sprintf("state=%s, status=%s", ctr.State, ctr.Status),
				},
			}
		}
	}
	return checkResult{
		Name:    checkName,
		Status:  "FAIL",
		Message: "container not found",
		Detail: &VerboseCheck{
			Expected: fmt.Sprintf("container '%s' container exists", name),
			Actual:   "container not found in list",
		},
	}
}

func checkHTTPHealth(port int) checkResult {
	if port <= 0 {
		port = 3000
	}
	url := fmt.Sprintf("http://localhost:%d/api/status", port)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return checkResult{
			Name:    "http-health",
			Status:  "WARN",
			Message: fmt.Sprintf("port %d not reachable: %v", port, err),
			Detail: &VerboseCheck{
				Command:  fmt.Sprintf("GET %s", url),
				Expected: "HTTP 200-499",
				Actual:   fmt.Sprintf("err: %v", err),
			},
		}
	}
	defer resp.Body.Close()
	resp.Body = http.MaxBytesReader(nil, resp.Body, 4096)

	if resp.StatusCode < 500 {
		return checkResult{
			Name:    "http-health",
			Status:  "OK",
			Message: fmt.Sprintf("port %d responded with HTTP %d", port, resp.StatusCode),
			Detail: &VerboseCheck{
				FilePath: url,
				Command:  fmt.Sprintf("GET %s", url),
				Expected: "HTTP 200-499",
				Actual:   fmt.Sprintf("HTTP %d", resp.StatusCode),
			},
		}
	}
	return checkResult{
		Name:    "http-health",
		Status:  "WARN",
		Message: fmt.Sprintf("port %d returned HTTP %d", port, resp.StatusCode),
		Detail: &VerboseCheck{
			FilePath: url,
			Command:  fmt.Sprintf("GET %s", url),
			Expected: "HTTP 200-499",
			Actual:   fmt.Sprintf("HTTP %d", resp.StatusCode),
		},
	}
}

func checkDiskSpace(home string) checkResult {
	if home == "" {
		return checkResult{
			Name:    "disk-space",
			Status:  "SKIP",
			Message: "home not configured",
			Detail: &VerboseCheck{
				Expected: "home directory configured",
				Actual:   "home is empty",
			},
		}
	}

	// Try df first (Linux/macOS)
	out, err := exec.Command("df", "-h", home).Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		if len(lines) >= 2 {
			return checkResult{
				Name:    "disk-space",
				Status:  "OK",
				Message: strings.TrimSpace(lines[1]),
				Detail: &VerboseCheck{
					FilePath:  home,
					Command:   "df -h " + home,
					RawOutput: strings.TrimSpace(string(out)),
				},
			}
		}
	}

	// Fallback for Windows: use dir command
	out2, err2 := exec.Command("cmd", "/c", "dir", home).Output()
	if err2 == nil {
		lines := strings.Split(string(out2), "\n")
		for _, l := range lines {
			if strings.Contains(l, "bytes free") || strings.Contains(l, "字节可用") {
				return checkResult{
					Name:    "disk-space",
					Status:  "OK",
					Message: strings.TrimSpace(l),
					Detail: &VerboseCheck{
						FilePath:  home,
						Command:   "cmd /c dir " + home,
						RawOutput: strings.TrimSpace(string(out2)),
					},
				}
			}
		}
	}

	return checkResult{
		Name:    "disk-space",
		Status:  "SKIP",
		Message: "disk info unavailable on this platform",
		Detail: &VerboseCheck{
			Expected: "df/dir command available",
			Actual:   "platform not supported",
		},
	}
}

func checkConfigPermissions(cfg *core.Config) checkResult {
	configPaths := []string{}
	if cfg.NewAPI.Home != "" {
		configPaths = append(configPaths,
			filepath.Join(cfg.NewAPI.Home, ".env"),
			filepath.Join(cfg.NewAPI.Home, "docker-compose.yml"),
		)
	}
	configFile := core.ConfigFileUsed()
	if configFile != "" {
		configPaths = append(configPaths, configFile)
	}

	if len(configPaths) == 0 {
		return checkResult{
			Name:    "config-permissions",
			Status:  "SKIP",
			Message: "no config files to check",
			Detail: &VerboseCheck{
				Expected: "config files to check",
				Actual:   "no config files found",
			},
		}
	}

	var issues []string
	for _, path := range configPaths {
		if err := security.CheckConfigPerm(path); err != nil {
			issues = append(issues, err.Error())
		}
	}

	if len(issues) > 0 {
		return checkResult{
			Name:    "config-permissions",
			Status:  "WARN",
			Message: strings.Join(issues, "; "),
			Detail: &VerboseCheck{
				FilePath: strings.Join(configPaths, ", "),
				Expected: "all configs have secure permissions (0600",
				Actual:   strings.Join(issues, "; "),
			},
		}
	}
	return checkResult{
		Name:    "config-permissions",
		Status:  "OK",
		Message: "all config files have secure permissions",
		Detail: &VerboseCheck{
			FilePath: strings.Join(configPaths, ", "),
			Expected: "all configs have secure permissions",
			Actual:   "all permissions OK",
		},
	}
}

func checkDockerGroupMembership() checkResult {
	inGroup, err := security.CheckDockerGroup()
	if err != nil {
		return checkResult{
			Name:    "docker-group",
			Status:  "WARN",
			Message: fmt.Sprintf("cannot check: %v", err),
			Detail: &VerboseCheck{
				Command:  "security.CheckDockerGroup()",
				Expected: "check successful",
				Actual:   fmt.Sprintf("err: %v", err),
			},
		}
	}
	if !inGroup {
		return checkResult{
			Name:    "docker-group",
			Status:  "WARN",
			Message: "current user is not in 'docker' group — may need sudo",
			Detail: &VerboseCheck{
				Command:  "security.CheckDockerGroup()",
				Expected: "user in docker group",
				Actual:   "user not in docker group",
			},
		}
	}
	return checkResult{
		Name:    "docker-group",
		Status:  "OK",
		Message: "current user is in 'docker' group",
		Detail: &VerboseCheck{
			Command:  "security.CheckDockerGroup()",
			Expected: "user in docker group",
			Actual:   "user in docker group",
		},
	}
}

func printDoctorTable(results []checkResult, verbose bool) {
	maxName := 20
	for _, r := range results {
		if len(r.Name) > maxName {
			maxName = len(r.Name)
		}
	}

	fmt.Printf("%-*s  %-4s  %s\n", maxName, "CHECK", "STATUS", "DETAILS")
	fmt.Println(strings.Repeat("-", maxName+2+4+2+50))

	for _, r := range results {
		fmt.Printf("%-*s  %-4s  %s\n", maxName, r.Name, r.Status, r.Message)
		if verbose && r.Detail != nil {
			printVerboseDetail(r.Detail, maxName)
		}
	}
}

func printVerboseDetail(d *VerboseCheck, indent int) {
	prefix := strings.Repeat(" ", indent+2+4+2)
	if d.FilePath != "" {
		fmt.Printf("%s[FILE]: %s\n", prefix, d.FilePath)
	}
	if d.Command != "" {
		fmt.Printf("%s[CMD]: %s\n", prefix, d.Command)
	}
	if d.Expected != "" {
		fmt.Printf("%s[EXPECTED]: %s\n", prefix, d.Expected)
	}
	if d.Actual != "" {
		fmt.Printf("%s[ACTUAL]: %s\n", prefix, d.Actual)
	}
	if d.RawOutput != "" {
		fmt.Printf("%s[RAW]:\n%s\n", prefix, d.RawOutput)
	}
}

func printDoctorJSON(results []checkResult, verbose bool) {
	items := make([]checkResult, 0, len(results))
	for _, r := range results {
		if verbose && r.Detail != nil {
			items = append(items, r)
		} else {
			items = append(items, checkResult{
				Name:    r.Name,
				Status:  r.Status,
				Message: r.Message,
			})
		}
	}
	data, _ := json.MarshalIndent(items, "", "  ")
	fmt.Println(string(data))
}

// ---- Auto-fix logic ----

// runAutoFix attempts to automatically resolve detected issues using Check definitions.
// Returns the number of fixes applied.
func runAutoFix(ctx context.Context, results []checkResult, cfg *core.Config) int {
	fixCount := 0
	composeStarted := false // guard: run docker compose up only once
	resultByName := make(map[string]checkResult)
	for _, r := range results {
		resultByName[r.Name] = r
	}

	for _, check := range checks {
		result, ok := resultByName[check.Name]
		if !ok || (result.Status != "FAIL" && result.Status != "WARN") {
			continue
		}

		if check.Fix != nil {
			if check.Name == "new-api-container" || check.Name == "mysql-container" || check.Name == "redis-container" {
				if composeStarted {
					continue
				}
				composePath := filepath.Join(cfg.NewAPI.Home, "docker-compose.yml")
				if _, err := os.Stat(composePath); err != nil {
					fmt.Printf("  [HINT] 运行 'newapi-tools install' 先安装\n")
					continue
				}
				fmt.Printf("  [FIX] Starting containers with docker compose up -d...\n")
				if err := check.Fix(ctx, cfg); err != nil {
					fmt.Printf("  [FAIL] Could not start containers: %v\n", err)
				} else {
					fmt.Printf("  [FIXED] Containers started\n")
					fixCount++
					composeStarted = true
				}
			} else {
				fmt.Printf("  [FIX] Applying fix for %s...\n", check.Name)
				if err := check.Fix(ctx, cfg); err == nil {
					fmt.Printf("  [FIXED] %s\n", check.Name)
					fixCount++
				} else {
					fmt.Printf("  [FAIL] Could not fix %s: %v\n", check.Name, err)
				}
			}
		} else if check.FixHint != "" {
			fmt.Printf("  [HINT] %s\n", check.FixHint)
		}
	}

	return fixCount
}

// fixCreateHomeDir creates the home directory if it doesn't exist.
func fixCreateHomeDir(ctx context.Context, cfg *core.Config) error {
	if cfg.NewAPI.Home == "" {
		return fmt.Errorf("home not configured")
	}
	return os.MkdirAll(cfg.NewAPI.Home, 0755)
}

// fixConfigPermissions fixes config file permissions.
func fixConfigPermissions(ctx context.Context, cfg *core.Config) error {
	configPaths := []string{}
	if cfg.NewAPI.Home != "" {
		configPaths = append(configPaths,
			filepath.Join(cfg.NewAPI.Home, ".env"),
			filepath.Join(cfg.NewAPI.Home, "docker-compose.yml"),
		)
	}
	configFile := core.ConfigFileUsed()
	if configFile != "" {
		configPaths = append(configPaths, configFile)
	}

	for _, path := range configPaths {
		if err := os.Chmod(path, 0600); err != nil {
			ui.L().Warn("could not fix permissions", "path", path, "err", err)
		} else {
			ui.L().Info("set secure permissions", "path", path)
		}
	}
	return nil
}

// composeUpFix runs docker compose up -d in the configured home directory.
func composeUpFix(ctx context.Context, cfg *core.Config) error {
	return docker.ComposeUp(ctx, cfg.NewAPI.Home, cfg.Docker.ComposeCmd)
}
