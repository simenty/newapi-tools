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

	"github.com/simenty/newapi-tools/internal/apperr"
	"github.com/simenty/newapi-tools/internal/core"
	"github.com/simenty/newapi-tools/internal/docker"
	"github.com/simenty/newapi-tools/internal/i18n"
	"github.com/simenty/newapi-tools/internal/instance"
	"github.com/simenty/newapi-tools/internal/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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
	installCmd.Flags().String("instance", "", "instance name to install (for multi-instance management)")
	installCmd.Flags().Int("health-timeout", 120, "健康检查超时时间（秒）")

	rootCmd.AddCommand(installCmd)
}

// installOptions holds the user's choices from the interactive wizard.
type installOptions struct {
	Port        int
	Domain      string // optional domain name
	DBType      string // "mysql" or "sqlite"
	RedisAddr   string
	DockerImage string // Docker image to use
	AutoMirror  bool   // whether to auto-select the fastest registry mirror
}

func runInstall(cmd *cobra.Command, args []string) error {
	cfg := core.GetConfig()
	if cfg == nil {
		return fmt.Errorf("configuration not loaded")
	}

	instanceName, _ := cmd.Flags().GetString("instance")
	var inst *instance.Instance
	if instanceName != "" {
		store := instance.NewStore("")
		var err error
		inst, err = store.Get(instanceName)
		if err != nil {
			return fmt.Errorf("instance not found, please create it first with 'newapi-tools instance add %s", instanceName)
		}
		// Apply instance settings from the instance's config
		cfg.NewAPI.Home = inst.Home
		cfg.NewAPI.Port = inst.Port
		cfg.NewAPI.DockerImage = inst.DockerImage
	}

	// Apply CLI flag overrides
	if port, _ := cmd.Flags().GetInt("port"); port != 0 {
		cfg.NewAPI.Port = port
	}
	if image, _ := cmd.Flags().GetString("image"); image != "" {
		cfg.NewAPI.DockerImage = image
	}
	if healthTimeout, _ := cmd.Flags().GetInt("health-timeout"); healthTimeout != 0 {
		cfg.NewAPI.HealthTimeout = healthTimeout
	}
	force, _ := cmd.Flags().GetBool("force")
	mirrorFlag, _ := cmd.Flags().GetString("mirror")
	noAutoMirror, _ := cmd.Flags().GetBool("no-auto-mirror")
	interactive, _ := cmd.Flags().GetBool("interactive")

	// Interactive wizard
	opts := installOptions{
		Port:        cfg.NewAPI.Port,
		DBType:      "mysql",
		RedisAddr:   "redis:6379",
		DockerImage: cfg.NewAPI.DockerImage,
		AutoMirror:  true,
	}
	if interactive {
		var err error
		opts, err = runInstallWizard(cfg)
		if err != nil {
			return err
		}
		cfg.NewAPI.Port = opts.Port
		if opts.DockerImage != "" {
			cfg.NewAPI.DockerImage = opts.DockerImage
		}
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
	var composeProjectName string
	if instanceName != "" {
		if inst != nil {
			composeProjectName = inst.ComposeProject
		} else {
			composeProjectName = fmt.Sprintf("newapi-%s", instanceName)
		}
	}
	composeContent := generateComposeYAML(cfg, opts, composeProjectName)
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
	waitForHealth(cfg.NewAPI.Port, cfg.NewAPI.HealthTimeout)

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

	// Register to instance store if not already registered
	if instanceName == "" {
		// Default instance name
		instanceName = "default"
	}
	if inst == nil {
		store := instance.NewStore("")
		newInst := instance.NewInstance(instanceName, cfg.NewAPI.Home, cfg.NewAPI.Port, cfg.NewAPI.DockerImage)
		if err := store.Add(*newInst); err != nil {
			fmt.Printf("Warning: could not register instance: %v\n", err)
		} else {
			fmt.Printf("Instance '%s' registered successfully\n", instanceName)
		}
	}
	return nil
}

// runInstallWizard runs the interactive 7-step installation wizard.
// It loops back to step 1 if the user enters 'e' at the confirmation step.
func runInstallWizard(cfg *core.Config) (installOptions, error) {
	defaultImage := cfg.NewAPI.DockerImage
	if defaultImage == "" {
		defaultImage = "calciumion/new-api:latest"
	}

	opts := installOptions{
		Port:        cfg.NewAPI.Port,
		Domain:      "",
		DBType:      "mysql",
		RedisAddr:   "redis:6379",
		DockerImage: defaultImage,
		AutoMirror:  true,
	}

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Println()
		fmt.Println("🚀 NewAPI 安装向导")
		fmt.Println()

		// Step 1/7: Port
		fmt.Printf("[1/7] 请选择端口 (默认 %d): ", opts.Port)
		if scanner.Scan() {
			if input := strings.TrimSpace(scanner.Text()); input != "" {
				var port int
				if _, err := fmt.Sscanf(input, "%d", &port); err == nil && port > 0 && port <= 65535 {
					opts.Port = port
				} else {
					fmt.Printf("  无效端口，保持默认值 %d\n", opts.Port)
				}
			}
		}

		// Step 2/7: Domain (optional)
		fmt.Printf("[2/7] 请输入域名 (可选，直接回车跳过): ")
		if scanner.Scan() {
			opts.Domain = strings.TrimSpace(scanner.Text())
		}

		// Step 3/7: Database type
		fmt.Printf("[3/7] 请选择数据库 (mysql/sqlite，默认 %s): ", opts.DBType)
		if scanner.Scan() {
			if input := strings.TrimSpace(scanner.Text()); input != "" {
				lower := strings.ToLower(input)
				if lower == "mysql" || lower == "sqlite" {
					opts.DBType = lower
				} else {
					fmt.Printf("  无效数据库类型，保持默认值 %s\n", opts.DBType)
				}
			}
		}

		// Step 4/7: Redis address
		fmt.Printf("[4/7] Redis 地址 (默认 %s): ", opts.RedisAddr)
		if scanner.Scan() {
			if input := strings.TrimSpace(scanner.Text()); input != "" {
				opts.RedisAddr = input
			}
		}

		// Step 5/7: Docker image
		fmt.Printf("[5/7] Docker 镜像 (默认 %s): ", opts.DockerImage)
		if scanner.Scan() {
			if input := strings.TrimSpace(scanner.Text()); input != "" {
				opts.DockerImage = input
			}
		}

		// Step 6/7: Auto mirror
		autoMirrorDefault := "Y/n"
		if !opts.AutoMirror {
			autoMirrorDefault = "y/N"
		}
		fmt.Printf("[6/7] 自动选择最快镜像加速？ (%s): ", autoMirrorDefault)
		if scanner.Scan() {
			input := strings.TrimSpace(strings.ToLower(scanner.Text()))
			switch input {
			case "y", "yes", "":
				opts.AutoMirror = true
			case "n", "no":
				opts.AutoMirror = false
			}
		}

		// Step 7/7: Summary + confirm
		fmt.Println()
		fmt.Println("[7/7] 安装摘要")
		fmt.Println("  ─────────────────────────────────────")
		fmt.Printf("  端口:       %d\n", opts.Port)
		if opts.Domain != "" {
			fmt.Printf("  域名:       %s\n", opts.Domain)
		}
		fmt.Printf("  数据库:     %s\n", opts.DBType)
		fmt.Printf("  Redis:      %s\n", opts.RedisAddr)
		fmt.Printf("  镜像:       %s\n", opts.DockerImage)
		autoMirrorStr := "否"
		if opts.AutoMirror {
			autoMirrorStr = "是"
		}
		fmt.Printf("  自动镜像:   %s\n", autoMirrorStr)
		fmt.Println("  ─────────────────────────────────────")
		fmt.Println()
		fmt.Print("确认安装? (回车确认 / 输入 'e' 重新配置 / 'q' 取消): ")

		if scanner.Scan() {
			input := strings.TrimSpace(strings.ToLower(scanner.Text()))
			switch input {
			case "e":
				// Restart wizard loop
				continue
			case "q", "quit", "exit":
				return opts, fmt.Errorf("安装已取消")
			default:
				// Empty or any other key → confirm
			}
		}

		break
	}

	fmt.Println()
	return opts, nil
}

// waitForHealth polls the new-api health endpoint until it responds or times out.
// It prints a success or warning message but never returns an error (non-blocking).
func waitForHealth(port int, timeoutSeconds int) {
	if port <= 0 {
		port = 3000
	}
	if timeoutSeconds <= 0 {
		timeoutSeconds = 120
	}
	url := fmt.Sprintf("http://localhost:%d/api/status", port)
	client := &http.Client{Timeout: 3 * time.Second}

	fmt.Println()
	fmt.Printf("Waiting for service to become healthy (timeout: %d seconds)...\n", timeoutSeconds)

	maxWait := time.Duration(timeoutSeconds) * time.Second
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

	fmt.Printf("⚠️  服务健康检查超时（%d秒），请手动检查容器状态: docker ps\n", timeoutSeconds)
}

type composeService struct {
	Image         string            `yaml:"image"`
	ContainerName string            `yaml:"container_name"`
	Restart       string            `yaml:"restart"`
	Ports         []string          `yaml:"ports,omitempty"`
	EnvFile       []string          `yaml:"env_file,omitempty"`
	Volumes       []string          `yaml:"volumes,omitempty"`
	DependsOn     []string          `yaml:"depends_on,omitempty"`
	Environment   map[string]string `yaml:"environment,omitempty"`
	Platform      string            `yaml:"platform,omitempty"`
}

type composeFile struct {
	Name     string                    `yaml:"name,omitempty"`
	Version  string                    `yaml:"version"`
	Services map[string]composeService `yaml:"services"`
}

// generateComposeYAML generates the docker-compose.yml content using structs.
func generateComposeYAML(cfg *core.Config, opts installOptions, composeProjectName string) string {
	cf := composeFile{
		Version:  "3.8",
		Services: make(map[string]composeService),
	}

	if composeProjectName != "" {
		cf.Name = composeProjectName
	}

	// Build new-api service
	newAPIContainerName := "new-api"
	if composeProjectName != "" {
		newAPIContainerName = fmt.Sprintf("%s-new-api", composeProjectName)
	}
	newAPISvc := composeService{
		Image:         cfg.NewAPI.DockerImage,
		ContainerName: newAPIContainerName,
		Restart:       "always",
		Ports:         []string{fmt.Sprintf("%d:3000", cfg.NewAPI.Port)},
		EnvFile:       []string{".env"},
		Volumes:       []string{"./data:/data"},
		DependsOn:     []string{"redis"},
	}
	if runtime.GOARCH == "arm64" {
		newAPISvc.Platform = "linux/arm64"
	}
	if opts.DBType == "mysql" {
		newAPISvc.DependsOn = append(newAPISvc.DependsOn, "mysql")
	}
	cf.Services["new-api"] = newAPISvc

	// Build MySQL service if needed
	if opts.DBType == "mysql" {
		mysqlContainerName := "mysql"
		if composeProjectName != "" {
			mysqlContainerName = fmt.Sprintf("%s-mysql", composeProjectName)
		}
		cf.Services["mysql"] = composeService{
			Image:         "mysql:8.0",
			ContainerName: mysqlContainerName,
			Restart:       "always",
			Environment: map[string]string{
				"MYSQL_ROOT_PASSWORD": "${MYSQL_ROOT_PASSWORD}",
				"MYSQL_DATABASE":      "newapi",
			},
			Volumes: []string{"./mysql-data:/var/lib/mysql"},
		}
	}

	// Build Redis service (always included)
	redisContainerName := "redis"
	if composeProjectName != "" {
		redisContainerName = fmt.Sprintf("%s-redis", composeProjectName)
	}
	cf.Services["redis"] = composeService{
		Image:         "redis:7",
		ContainerName: redisContainerName,
		Restart:       "always",
		Volumes:       []string{"./redis-data:/data"},
	}

	// Marshal to YAML
	out, _ := yaml.Marshal(cf)
	return string(out)
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
