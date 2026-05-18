// NewAPI Tools - Docker management platform for newapi
package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Bonus520/newapi-tools/internal/core"
	"github.com/Bonus520/newapi-tools/internal/docker"
	"github.com/Bonus520/newapi-tools/internal/ui"
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

	rootCmd.AddCommand(doctorCmd)
}

// checkResult represents the outcome of a single diagnostic check.
type checkResult struct {
	Name    string
	Status  string // "OK", "WARN", "FAIL", "SKIP"
	Message string
}

func runDoctor(cmd *cobra.Command, args []string) error {
	cfg := core.GetConfig()
	if cfg == nil {
		return fmt.Errorf("configuration not loaded")
	}

	fix, _ := cmd.Flags().GetBool("fix")
	jsonOut, _ := cmd.Flags().GetBool("json")

	fmt.Println("Running diagnostics...")
	fmt.Println()

	ctx := cmd.Context()
	var results []checkResult

	// --- Check 1: Docker binary available ---
	results = append(results, checkDockerBinary())

	// --- Check 2: Docker daemon accessible ---
	results = append(results, checkDockerDaemon())

	// --- Check 3: new-api home directory exists ---
	results = append(results, checkHomeDir(cfg.NewAPI.Home))

	// --- Check 4: docker-compose.yml exists ---
	results = append(results, checkComposeFile(cfg.NewAPI.Home))

	// --- Check 5: .env file exists ---
	results = append(results, checkEnvFile(cfg.NewAPI.Home))

	// --- Check 6: new-api container running ---
	results = append(results, checkContainer(ctx, "new-api"))

	// --- Check 7: mysql container running ---
	results = append(results, checkContainer(ctx, "mysql"))

	// --- Check 8: redis container running ---
	results = append(results, checkContainer(ctx, "redis"))

	// --- Check 9: HTTP health check on configured port ---
	results = append(results, checkHTTPHealth(cfg.NewAPI.Port))

	// --- Check 10: Disk space ---
	results = append(results, checkDiskSpace(cfg.NewAPI.Home))

	// --- Output results ---
	if jsonOut {
		printDoctorJSON(results)
	} else {
		printDoctorTable(results)
	}

	// Count failures
	failCount, warnCount := 0, 0
	for _, r := range results {
		switch r.Status {
		case "FAIL":
			failCount++
		case "WARN":
			warnCount++
		}
	}

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
			results = nil
			results = append(results, checkDockerBinary())
			results = append(results, checkDockerDaemon())
			results = append(results, checkHomeDir(cfg.NewAPI.Home))
			results = append(results, checkComposeFile(cfg.NewAPI.Home))
			results = append(results, checkEnvFile(cfg.NewAPI.Home))
			results = append(results, checkContainer(ctx, "new-api"))
			results = append(results, checkContainer(ctx, "mysql"))
			results = append(results, checkContainer(ctx, "redis"))
			results = append(results, checkHTTPHealth(cfg.NewAPI.Port))
			results = append(results, checkDiskSpace(cfg.NewAPI.Home))

			if jsonOut {
				printDoctorJSON(results)
			} else {
				printDoctorTable(results)
			}

			// Recount
			failCount, warnCount = 0, 0
			for _, r := range results {
				switch r.Status {
				case "FAIL":
					failCount++
				case "WARN":
					warnCount++
				}
			}
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
		return fmt.Errorf("%d diagnostic check(s) failed", failCount)
	}
	return nil
}

func checkDockerBinary() checkResult {
	path, err := exec.LookPath("docker")
	if err != nil {
		return checkResult{"docker binary", "FAIL", "docker not found in PATH"}
	}
	return checkResult{"docker binary", "OK", path}
}

func checkDockerDaemon() checkResult {
	c, err := docker.NewClient()
	if err != nil {
		return checkResult{"docker daemon", "FAIL", err.Error()}
	}
	defer c.Close()

	if !c.IsAvailable() {
		return checkResult{"docker daemon", "FAIL", "docker daemon is not running"}
	}
	return checkResult{"docker daemon", "OK", "daemon is accessible"}
}

func checkHomeDir(home string) checkResult {
	if home == "" {
		return checkResult{"home directory", "WARN", "not configured"}
	}
	if _, err := os.Stat(home); os.IsNotExist(err) {
		return checkResult{"home directory", "FAIL", fmt.Sprintf("%s does not exist", home)}
	}
	return checkResult{"home directory", "OK", home}
}

func checkComposeFile(home string) checkResult {
	path := filepath.Join(home, "docker-compose.yml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return checkResult{"docker-compose.yml", "FAIL",
			fmt.Sprintf("%s not found — run 'newapi-tools install'", path)}
	}
	return checkResult{"docker-compose.yml", "OK", path}
}

func checkEnvFile(home string) checkResult {
	path := filepath.Join(home, ".env")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return checkResult{".env file", "WARN",
			fmt.Sprintf("%s not found — credentials may be missing", path)}
	}
	return checkResult{".env file", "OK", path}
}

func checkContainer(ctx context.Context, name string) checkResult {
	c, err := docker.NewClient()
	if err != nil {
		return checkResult{name + " container", "SKIP", "docker unavailable"}
	}
	defer c.Close()

	containers, listErr := c.ContainerList(ctx)
	if listErr != nil {
		return checkResult{name + " container", "WARN",
			fmt.Sprintf("failed to list containers: %v", listErr)}
	}

	for _, ctr := range containers {
		if strings.Contains(ctr.Name, name) {
			if ctr.State == "running" {
				return checkResult{name + " container", "OK",
					fmt.Sprintf("%s (%s)", ctr.Image, ctr.Status)}
			}
			return checkResult{name + " container", "FAIL",
				fmt.Sprintf("state=%s, status=%s", ctr.State, ctr.Status)}
		}
	}
	return checkResult{name + " container", "FAIL", "container not found"}
}

func checkHTTPHealth(port int) checkResult {
	if port <= 0 {
		port = 3000
	}
	url := fmt.Sprintf("http://localhost:%d/api/status", port)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return checkResult{"HTTP health", "WARN",
			fmt.Sprintf("port %d not reachable: %v", port, err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 500 {
		return checkResult{"HTTP health", "OK",
			fmt.Sprintf("port %d responded with HTTP %d", port, resp.StatusCode)}
	}
	return checkResult{"HTTP health", "WARN",
		fmt.Sprintf("port %d returned HTTP %d", port, resp.StatusCode)}
}

func checkDiskSpace(home string) checkResult {
	if home == "" {
		return checkResult{"disk space", "SKIP", "home not configured"}
	}

	// Try df first (Linux/macOS)
	out, err := exec.Command("df", "-h", home).Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		if len(lines) >= 2 {
			return checkResult{"disk space", "OK", strings.TrimSpace(lines[1])}
		}
	}

	// Fallback for Windows: use dir command
	out2, err2 := exec.Command("cmd", "/c", "dir", home).Output()
	if err2 == nil {
		lines := strings.Split(string(out2), "\n")
		for _, l := range lines {
			if strings.Contains(l, "bytes free") || strings.Contains(l, "字节可用") {
				return checkResult{"disk space", "OK", strings.TrimSpace(l)}
			}
		}
	}

	return checkResult{"disk space", "SKIP", "disk info unavailable on this platform"}
}

func printDoctorTable(results []checkResult) {
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
	}
}

func printDoctorJSON(results []checkResult) {
	fmt.Println("[")
	for i, r := range results {
		comma := ","
		if i == len(results)-1 {
			comma = ""
		}
		fmt.Printf("  {\"check\": %q, \"status\": %q, \"message\": %q}%s\n",
			r.Name, r.Status, r.Message, comma)
	}
	fmt.Println("]")
}

// ---- Auto-fix logic ----

// runAutoFix attempts to automatically resolve detected issues.
// Returns the number of fixes applied.
func runAutoFix(ctx context.Context, results []checkResult, cfg *core.Config) int {
	fixCount := 0

	for _, r := range results {
		if r.Status != "FAIL" && r.Status != "WARN" {
			continue
		}

		switch r.Name {
		case "home directory":
			if cfg.NewAPI.Home != "" {
				if err := os.MkdirAll(cfg.NewAPI.Home, 0755); err == nil {
					fmt.Printf("  [FIXED] Created home directory: %s\n", cfg.NewAPI.Home)
					fixCount++
				} else {
					fmt.Printf("  [SKIP] Cannot create home directory: %v\n", err)
				}
			}

		case "docker-compose.yml":
			fmt.Printf("  [HINT] Run 'newapi-tools install' to generate docker-compose.yml\n")

		case ".env file":
			fmt.Printf("  [HINT] Run 'newapi-tools install' to generate .env file\n")

		case "new-api container", "mysql container", "redis container":
			// Attempt to start containers if compose file exists
			composePath := filepath.Join(cfg.NewAPI.Home, "docker-compose.yml")
			if _, err := os.Stat(composePath); err == nil {
				fmt.Printf("  [FIX] Starting containers with docker compose up -d...\n")
				if err := composeUpFix(ctx, cfg); err != nil {
					fmt.Printf("  [FAIL] Could not start containers: %v\n", err)
				} else {
					fmt.Printf("  [FIXED] Containers started\n")
					fixCount++
				}
			} else {
				fmt.Printf("  [HINT] Run 'newapi-tools install' first\n")
			}

		case "docker binary":
			fmt.Printf("  [HINT] Install Docker: https://docs.docker.com/get-docker/\n")

		case "docker daemon":
			fmt.Printf("  [HINT] Start the Docker daemon (e.g., 'sudo systemctl start docker')\n")

		case "HTTP health":
			fmt.Printf("  [HINT] Check that new-api is listening on port %d\n", cfg.NewAPI.Port)

		case "disk space":
			fmt.Printf("  [HINT] Free up disk space or move data to a larger volume\n")
		}
	}

	return fixCount
}

// composeUpFix runs docker compose up -d in the configured home directory.
func composeUpFix(ctx context.Context, cfg *core.Config) error {
	return docker.ComposeUp(ctx, cfg.NewAPI.Home, cfg.Docker.ComposeCmd)
}
