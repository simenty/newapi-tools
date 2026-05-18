// NewAPI Tools - Docker management platform for newapi
package cli

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path/filepath"

	"github.com/Bonus520/newapi-tools/internal/core"
	"github.com/Bonus520/newapi-tools/internal/docker"
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

	rootCmd.AddCommand(installCmd)
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

	// Check if Docker is available
	client, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("docker not available: %w", err)
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
			return fmt.Errorf("failed to remove existing container: %w", err)
		}
	}

	// Create install directory
	if err := os.MkdirAll(cfg.NewAPI.Home, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", cfg.NewAPI.Home, err)
	}

	// Generate docker-compose.yml
	composeContent := generateComposeYAML(cfg)
	composePath := filepath.Join(cfg.NewAPI.Home, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		return fmt.Errorf("failed to write docker-compose.yml: %w", err)
	}

	// Generate .env file
	envContent := generateEnvFile()
	envPath := filepath.Join(cfg.NewAPI.Home, ".env")
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		return fmt.Errorf("failed to write .env: %w", err)
	}

	// Pull image
	ui.L().Info("pulling new-api image", "image", cfg.NewAPI.DockerImage)
	fmt.Printf("Pulling image %s...\n", cfg.NewAPI.DockerImage)
	if err := client.ImagePull(cmd.Context(), cfg.NewAPI.DockerImage); err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}

	// Deploy with compose
	ui.L().Info("starting new-api with docker compose", "home", cfg.NewAPI.Home)
	fmt.Println("Starting new-api with Docker Compose...")
	if err := docker.ComposeUp(cmd.Context(), cfg.NewAPI.Home, cfg.Docker.ComposeCmd); err != nil {
		return fmt.Errorf("compose up failed: %w", err)
	}

	fmt.Println()
	fmt.Println("new-api installed successfully!")
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

// generateComposeYAML generates the docker-compose.yml content.
func generateComposeYAML(cfg *core.Config) string {
	return fmt.Sprintf(`version: '3.8'
services:
  new-api:
    image: %s
    container_name: new-api
    restart: always
    ports:
      - "%d:3000"
    env_file: .env
    volumes:
      - ./data:/data
    depends_on:
      - mysql
      - redis
  mysql:
    image: mysql:8.0
    container_name: mysql
    restart: always
    environment:
      MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD}
      MYSQL_DATABASE: newapi
    volumes:
      - ./mysql-data:/var/lib/mysql
  redis:
    image: redis:7
    container_name: redis
    restart: always
    volumes:
      - ./redis-data:/data
`, cfg.NewAPI.DockerImage, cfg.NewAPI.Port)
}

// generateEnvFile generates the .env file content with random passwords.
func generateEnvFile() string {
	mysqlPassword := randomPassword(16)
	sessionSecret := randomPassword(32)
	return fmt.Sprintf(`SQL_DSN=root:%s@tcp(mysql:3306)/newapi
REDIS_CONN_STRING=redis://redis:6379
SESSION_SECRET=%s
MYSQL_ROOT_PASSWORD=%s
`, mysqlPassword, sessionSecret, mysqlPassword)
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
