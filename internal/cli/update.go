// NewAPI Tools - Docker management platform for newapi
package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Bonus520/newapi-tools/internal/apperr"
	"github.com/Bonus520/newapi-tools/internal/core"
	"github.com/Bonus520/newapi-tools/internal/docker"
	"github.com/Bonus520/newapi-tools/internal/i18n"
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
	updateCmd.Flags().String("mirror", "", "registry mirror to use for this pull (e.g. tuna, aliyun, or a full URL)")
	updateCmd.Flags().Bool("no-auto-mirror", false, "skip auto-detecting and applying the fastest registry mirror")

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
	mirrorFlag, _ := cmd.Flags().GetString("mirror")
	noAutoMirror, _ := cmd.Flags().GetBool("no-auto-mirror")

	// Override image if specified
	if imageTag != "" {
		cfg.NewAPI.DockerImage = imageTag
	}

	// Apply one-time mirror if specified (writes daemon.json + reloads)
	if mirrorFlag != "" {
		if err := applyTempMirror(mirrorFlag); err != nil {
			fmt.Printf("  Warning: could not apply mirror %q: %v\n", mirrorFlag, err)
			fmt.Println("  Continuing without mirror...")
		}
	} else {
		// Check if any mirrors already configured in daemon.json
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

	// Check Docker is available
	client, err := docker.NewClient()
	if err != nil {
		return apperr.Wrap(apperr.CodeDockerNotFound, "", err)
	}
	defer client.Close()

	if !client.IsAvailable() {
		return apperr.New(apperr.CodeDockerDaemonDown, "Docker daemon 不可访问", "", nil)
	}

	// Verify new-api is installed
	container, err := client.FindContainerByName(cmd.Context(), "new-api")
	if err != nil {
		return fmt.Errorf("failed to check container: %w", err)
	}
	if container == nil {
		return apperr.New(apperr.CodeInstallFailed, "new-api 未安装，请先运行 'newapi-tools install'", "", nil)
	}

	currentImage := container.Image
	ui.L().Info("starting update",
		"current_image", currentImage,
		"target_image", cfg.NewAPI.DockerImage,
	)
	fmt.Printf("Current image: %s\n", currentImage)
	fmt.Printf("Target image:  %s\n", cfg.NewAPI.DockerImage)

	// --- Pre-step: Backup (unless --force or --backup=false) ---
	if doBackup && !force {
		if err := performBackup(cmd.Context(), cfg); err != nil {
			ui.L().Warn("pre-update backup failed", "error", err)
			fmt.Printf("  Warning: backup failed: %v\n", err)
			fmt.Println("  Continuing with update (use --force to skip backup entirely)...")
		}
	}

	// --- Step [1/3]: Pull new image ---
	ui.PrintStep(1, 3, i18n.T("update.step_pull"))
	if err := client.ImagePull(cmd.Context(), cfg.NewAPI.DockerImage); err != nil {
		return apperr.Wrap(apperr.CodeInstallFailed, "", err)
	}
	fmt.Println("  Image pulled.")

	// Check if we actually got a newer image
	newDigest, digestErr := getImageDigest(cmd.Context(), cfg.NewAPI.DockerImage)
	if digestErr == nil {
		fmt.Printf("  Digest: %s\n", shortDigest(newDigest))
	}

	// --- Step [2/3]: Recreate container ---
	ui.PrintStep(2, 3, i18n.T("update.step_recreate"))
	if err := composeUpForceRecreate(cmd.Context(), cfg.NewAPI.Home, cfg.Docker.ComposeCmd); err != nil {
		return apperr.Wrap(apperr.CodeMirrorApply, "", err)
	}
	fmt.Println("  Containers recreated.")

	// --- Step [3/3]: Verify service ---
	ui.PrintStep(3, 3, "update.complete")

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

// applyTempMirror adds a mirror to daemon.json and reloads Docker for this pull.
// It resolves short names (e.g. "tuna") to full URLs.
func applyTempMirror(nameOrURL string) error {
	url, ok := docker.ResolveShortName(nameOrURL)
	if !ok {
		return apperr.New(apperr.CodeMirrorApply, fmt.Sprintf("未知镜像源 %q", nameOrURL), "", nil)
	}
	fmt.Printf("  Applying mirror: %s\n", url)
	if err := docker.AddMirror(url); err != nil {
		return apperr.Wrap(apperr.CodeMirrorApply, "", err)
	}
	fmt.Println("  Reloading Docker daemon...")
	if err := docker.ReloadDocker(); err != nil {
		return apperr.Wrap(apperr.CodeDockerDaemonDown, "", err)
	}
	fmt.Println("  Mirror applied.")
	return nil
}
