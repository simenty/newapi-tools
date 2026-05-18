// NewAPI Tools - Docker management platform for newapi
package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Bonus520/newapi-tools/internal/core"
	"github.com/Bonus520/newapi-tools/internal/docker"
	"github.com/Bonus520/newapi-tools/internal/ui"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update new-api to latest version",
	Long:  `Update new-api to the latest Docker image. Pulls the new image, recreates the container, and restarts with updated configuration.`,
	RunE:  runUpdate,
}

func init() {
	updateCmd.Flags().String("image", "", "specific image tag to update to (default: latest)")
	updateCmd.Flags().Bool("backup", true, "create backup before updating")
	updateCmd.Flags().Bool("force", false, "force update without backup")

	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	cfg := core.GetConfig()
	if cfg == nil {
		return fmt.Errorf("configuration not loaded")
	}

	imageTag, _ := cmd.Flags().GetString("image")
	doBackup, _ := cmd.Flags().GetBool("backup")
	force, _ := cmd.Flags().GetBool("force")

	// Override image if specified
	if imageTag != "" {
		cfg.NewAPI.DockerImage = imageTag
	}

	// Check Docker is available
	client, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}
	defer client.Close()

	if !client.IsAvailable() {
		return fmt.Errorf("docker daemon is not accessible")
	}

	// Verify new-api is installed
	container, err := client.FindContainerByName(cmd.Context(), "new-api")
	if err != nil {
		return fmt.Errorf("failed to check container: %w", err)
	}
	if container == nil {
		return fmt.Errorf("new-api is not installed. Run 'newapi-tools install' first")
	}

	currentImage := container.Image
	ui.L().Info("starting update",
		"current_image", currentImage,
		"target_image", cfg.NewAPI.DockerImage,
	)
	fmt.Printf("Current image: %s\n", currentImage)
	fmt.Printf("Target image:  %s\n", cfg.NewAPI.DockerImage)

	// --- Step 1: Backup (unless --force or --backup=false) ---
	if doBackup && !force {
		fmt.Println()
		fmt.Println("Step 1/3: Creating pre-update backup...")
		if err := performBackup(cmd.Context(), cfg); err != nil {
			ui.L().Warn("pre-update backup failed", "error", err)
			fmt.Printf("  Warning: backup failed: %v\n", err)
			fmt.Println("  Continuing with update (use --force to skip backup entirely)...")
		} else {
			fmt.Println("  Backup completed.")
		}
	} else {
		fmt.Println("Step 1/3: Backup skipped.")
	}

	// --- Step 2: Pull new image ---
	fmt.Println()
	fmt.Println("Step 2/3: Pulling new image...")
	if err := client.ImagePull(cmd.Context(), cfg.NewAPI.DockerImage); err != nil {
		return fmt.Errorf("failed to pull image %s: %w", cfg.NewAPI.DockerImage, err)
	}
	fmt.Println("  Image pulled.")

	// Check if we actually got a newer image
	newDigest, digestErr := getImageDigest(cmd.Context(), cfg.NewAPI.DockerImage)
	if digestErr == nil {
		fmt.Printf("  Digest: %s\n", shortDigest(newDigest))
	}

	// --- Step 3: Recreate container via compose up --force-recreate ---
	fmt.Println()
	fmt.Println("Step 3/3: Recreating containers...")
	if err := composeUpForceRecreate(cmd.Context(), cfg.NewAPI.Home, cfg.Docker.ComposeCmd); err != nil {
		return fmt.Errorf("failed to recreate containers: %w", err)
	}
	fmt.Println("  Containers recreated.")

	// Verify the container is running
	time.Sleep(2 * time.Second)
	updated, _ := client.FindContainerByName(cmd.Context(), "new-api")
	if updated != nil {
		fmt.Printf("  Container state: %s\n", updated.State)
	}

	fmt.Println()
	fmt.Println("Update complete!")
	fmt.Printf("  Image:  %s\n", cfg.NewAPI.DockerImage)
	fmt.Printf("  Port:   %d\n", cfg.NewAPI.Port)

	ui.L().Info("update completed",
		"image", cfg.NewAPI.DockerImage,
		"home", cfg.NewAPI.Home,
	)
	return nil
}

// performBackup creates a backup before update using the same logic as runBackup.
func performBackup(ctx context.Context, cfg *core.Config) error {
	// Reuse the same backup directory logic
	backupDir := cfg.NewAPI.BackupDir
	if backupDir == "" {
		backupDir = cfg.NewAPI.Home + "/backups"
	}

	stageDir, err := os.MkdirTemp("", "newapi-update-backup-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stageDir)

	for _, f := range []string{"docker-compose.yml", ".env"} {
		_ = copyFileIfExists(cfg.NewAPI.Home+"/"+f, stageDir+"/"+f)
	}

	timestamp := time.Now().Format("20060102-150405")
	archivePath := backupDir + "/newapi-backup-" + timestamp + "-preupdate.tar.gz"

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return err
	}

	return createTarArchive(archivePath, stageDir, true)
}

// composeUpForceRecreate runs "docker compose up -d --force-recreate".
func composeUpForceRecreate(ctx context.Context, projectDir, composeCmd string) error {
	if composeCmd == "" {
		composeCmd = "docker compose"
	}
	parts := strings.Split(composeCmd, " ")
	args := append(parts, "-f", projectDir+"/docker-compose.yml", "up", "-d", "--force-recreate")
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = projectDir
	cmd.Stdout = docker.ComposeStdout
	cmd.Stderr = docker.ComposeStderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("compose up --force-recreate failed: %w", err)
	}
	return nil
}

// getImageDigest returns the RepoDigests of a local image.
func getImageDigest(ctx context.Context, image string) (string, error) {
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return "", err
	}
	out, err := exec.CommandContext(ctx, dockerPath,
		"inspect", "--format", "{{index .RepoDigests 0}}", image,
	).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// shortDigest returns the last 12 chars of a digest string.
func shortDigest(digest string) string {
	if len(digest) <= 12 {
		return digest
	}
	// e.g. "sha256:abc123..." → "...abc123..."
	parts := strings.SplitN(digest, ":", 2)
	if len(parts) == 2 && len(parts[1]) > 12 {
		return parts[1][:12] + "..."
	}
	return digest[len(digest)-12:]
}
