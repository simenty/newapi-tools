// NewAPI Tools - Docker management platform for newapi
package cli

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Bonus520/newapi-tools/internal/apperr"
	"github.com/Bonus520/newapi-tools/internal/core"
	"github.com/Bonus520/newapi-tools/internal/docker"
	"github.com/Bonus520/newapi-tools/internal/i18n"
	"github.com/Bonus520/newapi-tools/internal/ui"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install new-api",
	Long:  `Install new-api with Docker Compose. This command pulls the Docker image, generates configuration, and starts the container.`,
	RunE:  runInstall,
}

func init() {
	installCmd.Flags().Int("port", 0, "new-api listen port (default from config)")
	installCmd.Flags().String("image", "", "new-api Docker image (default from config)")
	installCmd.Flags().Bool("force", false, "force reinstall even if already installed")
	installCmd.Flags().String("mirror", "", "registry mirror to use for this pull (e.g. tuna, aliyun, or a full URL)")
	installCmd.Flags().Bool("no-auto-mirror", false, "skip auto-detecting and applying the fastest registry mirror")
	installCmd.Flags().Bool("interactive", false, "run interactive installation wizard")

	rootCmd.AddCommand(installCmd)
}

// installOptions holds the user's choices from the interactive wizard.
type installOptions struct {
	Port       int
	DBType     string // "mysql" or "sqlite"
	RedisAddr  string
}

func runInstall(cmd *cobra.Command, args []string) error {
	cfg := core.GetConfig()
	if cfg == nil {
		return fmt.Errorf("configuration not loaded")
	}

	// Apply CLI flag overrides
	if port, _ := cmd.Flags().GetInt("port"); port != 0 {
		cfg.NewAPI.Port = port
	}
	if image, _ := cmd.Flags().GetString("image"); image != "" {
		cfg.NewAPI.DockerImage = image
	}
	force, _ := cmd.Flags().GetBool("force")
	mirrorFlag, _ := cmd.Flags().GetString("mirror")
	noAutoMirror, _ := cmd.Flags().GetBool("no-auto-mirror")
	interactive, _ := cmd.Flags().GetBool("interactive")

	// Interactive wizard
	opts := installOptions{
		Port:      cfg.NewAPI.Port,
		DBType:    "mysql",
		RedisAddr: "redis:6379",
	}
	if interactive {
		var err error
		opts, err = runInstallWizard(cfg)
		if err != nil {
			return err
		}
		cfg.NewAPI.Port = opts.Port
	}

	// Apply mirror if specified
	if mirrorFlag != "" {
		if err := applyTempMirror(mirrorFlag); err != nil {
			fmt.Printf("  Warning: could not apply mirror %q: %v\n", mirrorFlag, err)
			fmt.Println("  Continuing without mirror...")
		}
	} else {
		mirrors, _ := docker.GetCurrentMirrors()
		if len(mirrors) == 0 && !noAutoMirror {
			// Auto-detect the fastest mirror
			fmt.Println("No registry mirror configured. Auto-detecting fastest mirror...")
			best := docker.AutoSelectMirror()
			if best != nil {
				fmt.Printf("  Fastest mirror: %s (%s, latency %s)\n", best.Name, best.URL, best.Latency.Round(time.Millisecond))
				fmt.Printf("  Applying mirror %s...\n", best.Name)
				if err := applyTempMirror(best.Name); err != nil {
					fmt.Printf("  Warning: could not apply mirror: %v\n", err)
					fmt.Println("  Continuing without mirror...")
				} else {
					fmt.Println("  Mirror applied. Image pull will use the mirror.")
				}
			} else {
				fmt.Println("  No reachable mirror found. Pull may be slow in mainland China.")
				fmt.Println("  You can manually add one later: newapi-tools mirror add tuna")
			}
		} else if len(mirrors) == 0 && noAutoMirror {
			fmt.Println("Tip: if pull is slow, run 'newapi-tools mirror add tuna' first.")
		}
	}

	// Check if Docker is available
	client, err := docker.NewClient()
	if err != nil {
		return apperr.Wrap(apperr.CodeDockerNotFound, "", err)
	}
	defer client.Close()

	// Check if already installed
	existing, _ := client.FindContainerByName(cmd.Context(), "new-api")
	if existing != nil && existing.State == "running" && !force {
		ui.L().Info("new-api is already running",
			"container", existing.Name,
			"status", existing.Status,
		)
		fmt.Println("new-api is already running. Use --force to reinstall.")
		return nil
	}

	if existing != nil && force {
		ui.L().Info("removing existing container for reinstall",
			"container", existing.Name,
		)
		if err := client.ContainerRemove(cmd.Context(), existing.Name); err != nil {
			return apperr.Wrap(apperr.CodeInstallFailed, "", err)
		}
	}

	// Create install directory
	if err := os.MkdirAll(cfg.NewAPI.Home, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", cfg.NewAPI.Home, err)
	}

	// Step [1/4]: Pull image
	ui.PrintStep(1, 4, i18n.T("install.pulling_image", cfg.NewAPI.DockerImage))
	ui.L().Info("pulling new-api image", "image", cfg.NewAPI.DockerImage)
	if err := client.ImagePull(cmd.Context(), cfg.NewAPI.DockerImage); err != nil {
		return apperr.Wrap(apperr.CodeInstallFailed, "", err)
	}

	// Step [2/4]: Generate configuration
	ui.PrintStep(2, 4, "install.generating_compose")
	composeContent := generateComposeYAML(cfg, opts)
	composePath := filepath.Join(cfg.NewAPI.Home, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		return fmt.Errorf("failed to write docker-compose.yml: %w", err)
	}

	ui.PrintStep(2, 4, "install.generating_env")
	envContent := generateEnvFile(opts)
	envPath := filepath.Join(cfg.NewAPI.Home, ".env")
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		return fmt.Errorf("failed to write .env: %w", err)
	}

	// Step [3/4]: Start container
	ui.PrintStep(3, 4, "install.starting_compose")
	ui.L().Info("starting new-api with docker compose", "home", cfg.NewAPI.Home)
	if err := docker.ComposeUp(cmd.Context(), cfg.NewAPI.Home, cfg.Docker.ComposeCmd); err != nil {
		return apperr.New(apperr.CodeInstallTimeout, "安装超时，请检查容器状态", "", err)
	}

	// Post-install healthcheck
	waitForHealth(cfg.NewAPI.Port)

	// Step [4/4]: Verify service
	ui.PrintStep(4, 4, "install.success")
	fmt.Println()
	fmt.Printf("  Home:    %s\n", cfg.NewAPI.Home)
	fmt.Printf("  Port:    %d\n", cfg.NewAPI.Port)
	fmt.Printf("  Image:   %s\n", cfg.NewAPI.DockerImage)

	ui.L().Info("new-api installed successfully",
		"home", cfg.NewAPI.Home,
		"port", cfg.NewAPI.Port,
		"image", cfg.NewAPI.DockerImage,
	)
	return nil
}

// runInstallWizard runs the interactive installation wizard.
func runInstallWizard(cfg *core.Config) (installOptions, error) {
	opts := installOptions{
		Port:      cfg.NewAPI.Port,
		DBType:    "mysql",
		RedisAddr: "redis:6379",
	}

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println()
	fmt.Println("🚀 NewAPI 安装向导")
	fmt.Println()

	// Step 1: Port
	fmt.Printf("[1/3] 请选择端口 (默认 %d): ", opts.Port)
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input != "" {
			var port int
			if _, err := fmt.Sscanf(input, "%d", &port); err == nil && port > 0 && port <= 65535 {
				opts.Port = port
			}
		}
	}

	// Step 2: Database type
	fmt.Printf("[2/3] 请选择数据库 (mysql/sqlite，默认 %s): ", opts.DBType)
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input != "" {
			lower := strings.ToLower(input)
			if lower == "mysql" || lower == "sqlite" {
				opts.DBType = lower
			}
		}
	}

	// Step 3: Redis address
	fmt.Printf("[3/3] Redis 地址 (默认 %s): ", opts.RedisAddr)
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input != "" {
			opts.RedisAddr = input
		}
	}

	fmt.Println()
	return opts, nil
}

// waitForHealth polls the new-api health endpoint until it responds or times out.
// It prints a success or warning message but never returns an error (non-blocking).
func waitForHealth(port int) {
	if port <= 0 {
		port = 3000
	}
	url := fmt.Sprintf("http://localhost:%d/api/status", port)
	client := &http.Client{Timeout: 3 * time.Second}

	fmt.Println()
	fmt.Println("Waiting for service to become healthy...")

	maxWait := 60 * time.Second
	interval := 3 * time.Second
	deadline := time.Now().Add(maxWait)

	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 500 {
				fmt.Println("✅ 服务启动成功！")
				return
			}
		}
		time.Sleep(interval)
	}

	fmt.Println("⚠️  服务健康检查超时（60秒），请手动检查容器状态: docker ps")
}

// generateComposeYAML generates the docker-compose.yml content.
// It supports multi-arch (arm64) and different database types based on opts.
func generateComposeYAML(cfg *core.Config, opts installOptions) string {
	// Determine platform directive for arm64
	platformLine := ""
	if runtime.GOARCH == "arm64" {
		platformLine = "    platform: linux/arm64\n"
	}

	// Build new-api service
	newAPIService := fmt.Sprintf(`  new-api:
    image: %s
    container_name: new-api
    restart: always
%s    ports:
      - "%d:3000"
    env_file: .env
    volumes:
      - ./data:/data`, cfg.NewAPI.DockerImage, platformLine, cfg.NewAPI.Port)

	// Build dependent services based on DB type
	var services []string
	services = append(services, newAPIService)

	if opts.DBType == "mysql" {
		services = append(services, `  mysql:
    image: mysql:8.0
    container_name: mysql
    restart: always
    environment:
      MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD}
      MYSQL_DATABASE: newapi
    volumes:
      - ./mysql-data:/var/lib/mysql`)
	}

	// Always include Redis (new-api requires it)
	services = append(services, `  redis:
    image: redis:7
    container_name: redis
    restart: always
    volumes:
      - ./redis-data:/data`)

	// Build depends_on list
	dependsOn := []string{}
	if opts.DBType == "mysql" {
		dependsOn = append(dependsOn, "      - mysql")
	}
	dependsOn = append(dependsOn, "      - redis")

	// Assemble the final YAML
	yaml := fmt.Sprintf(`version: '3.8'
services:
%s
    depends_on:
%s
`, strings.Join(services, "\n"), strings.Join(dependsOn, "\n"))

	return yaml
}

// generateEnvFile generates the .env file content with random passwords.
// The content varies based on the database type selected in opts.
func generateEnvFile(opts installOptions) string {
	sessionSecret := randomPassword(32)

	if opts.DBType == "sqlite" {
		return fmt.Sprintf(`SQL_DSN=
REDIS_CONN_STRING=redis://%s
SESSION_SECRET=%s
`, opts.RedisAddr, sessionSecret)
	}

	mysqlPassword := randomPassword(16)
	return fmt.Sprintf(`SQL_DSN=root:%s@tcp(mysql:3306)/newapi
REDIS_CONN_STRING=redis://%s
SESSION_SECRET=%s
MYSQL_ROOT_PASSWORD=%s
`, mysqlPassword, opts.RedisAddr, sessionSecret, mysqlPassword)
}

// randomPassword generates a cryptographically secure random password.
func randomPassword(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	buf := make([]byte, length)
	for i := range buf {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			// Fallback should never happen with crypto/rand
			buf[i] = chars[i%len(chars)]
			continue
		}
		buf[i] = chars[n.Int64()]
	}
	return string(buf)
}
