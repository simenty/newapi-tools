// NewAPI Tools - Docker management platform for newapi
package cli

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/simenty/newapi-tools/internal/apperr"
	"github.com/simenty/newapi-tools/internal/core"
	"github.com/simenty/newapi-tools/internal/docker"
	"github.com/simenty/newapi-tools/internal/i18n"
	"github.com/simenty/newapi-tools/internal/ui"
	"github.com/spf13/cobra"
)

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore new-api from backup",
	Long:  `Restore new-api data from a previously created backup archive. This stops the container, restores files, and restarts.`,
	RunE:  runRestore,
}

func init() {
	restoreCmd.Flags().String("file", "", "backup file to restore from (required, or 'latest' to pick most recent)")
	restoreCmd.Flags().Bool("force", false, "skip confirmation prompt")

	rootCmd.AddCommand(restoreCmd)
}

func runRestore(cmd *cobra.Command, args []string) error {
	cfg := core.GetConfig()
	if cfg == nil {
		return fmt.Errorf("configuration not loaded")
	}

	backupFile, _ := cmd.Flags().GetString("file")
	force, _ := cmd.Flags().GetBool("force")

	// Determine backup directory
	backupDir := cfg.NewAPI.BackupDir
	if backupDir == "" {
		backupDir = filepath.Join(cfg.NewAPI.Home, "backups")
	}

	// Resolve "latest" keyword or find backup automatically
	if backupFile == "" || backupFile == "latest" {
		latest, err := findLatestBackup(backupDir)
		if err != nil {
			return fmt.Errorf("no backup files found in %s: %w", backupDir, err)
		}
		backupFile = latest
		fmt.Printf("Using latest backup: %s\n", filepath.Base(backupFile))
	}

	// Ensure the file exists
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		// Try relative to backup dir
		candidate := filepath.Join(backupDir, backupFile)
		if _, err2 := os.Stat(candidate); os.IsNotExist(err2) {
			return fmt.Errorf("backup file not found: %s", backupFile)
		}
		backupFile = candidate
	}

	// Confirm restore (destructive operation)
	if !force {
		fmt.Printf("Restore from: %s\n", backupFile)
		fmt.Printf("Target home:  %s\n", cfg.NewAPI.Home)
		fmt.Print("This will OVERWRITE existing data. Continue? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("Restore cancelled.")
			return nil
		}
	}

	// Connect to Docker to stop/start containers
	client, err := docker.NewClient()
	if err != nil {
		return apperr.Wrap(apperr.CodeDockerNotFound, "", err)
	}
	defer client.Close()

	// Step [1/3]: Stop running containers
	ui.PrintStep(1, 3, "restore.stopping")
	ui.L().Info("stopping new-api containers before restore")
	if err := docker.ComposeDown(cmd.Context(), cfg.NewAPI.Home, cfg.Docker.ComposeCmd); err != nil {
		ui.L().Warn("compose down failed, continuing anyway", "error", err)
	}

	// Step [2/3]: Extract backup archive
	ui.PrintStep(2, 3, i18n.T("restore.extracting", cfg.NewAPI.Home))
	ui.L().Info("extracting backup", "file", backupFile, "target", cfg.NewAPI.Home)
	if err := extractTarArchive(backupFile, cfg.NewAPI.Home); err != nil {
		return apperr.Wrap(apperr.CodeRestoreFailed, "", err)
	}

	// Step [3/3]: Restart containers
	ui.PrintStep(3, 3, "restore.restarting")
	ui.L().Info("restarting new-api after restore")
	if err := docker.ComposeUp(cmd.Context(), cfg.NewAPI.Home, cfg.Docker.ComposeCmd); err != nil {
		return apperr.Wrap(apperr.CodeRestoreFailed, "", err)
	}

	fmt.Println()
	ui.PrintStep(3, 3, "restore.complete")
	fmt.Printf("  Source: %s\n", backupFile)
	fmt.Printf("  Target: %s\n", cfg.NewAPI.Home)

	ui.L().Info("restore completed", "file", backupFile, "home", cfg.NewAPI.Home)
	return nil
}

// findLatestBackup returns the most recently modified backup archive in dir.
func findLatestBackup(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	var archives []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, "newapi-backup-") &&
			(strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".tar")) {
			archives = append(archives, filepath.Join(dir, name))
		}
	}

	if len(archives) == 0 {
		return "", fmt.Errorf("no backup files found")
	}

	// Sort lexicographically — timestamp format YYYYMMDD-HHMMSS guarantees latest is last
	sort.Strings(archives)
	return archives[len(archives)-1], nil
}

// extractTarArchive extracts a .tar or .tar.gz archive into dstDir.
func extractTarArchive(archivePath, dstDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	var tr *tar.Reader
	if strings.HasSuffix(archivePath, ".tar.gz") || strings.HasSuffix(archivePath, ".tgz") {
		gr, err := gzip.NewReader(f)
		if err != nil {
			return fmt.Errorf("failed to open gzip: %w", err)
		}
		defer gr.Close()
		tr = tar.NewReader(gr)
	} else {
		tr = tar.NewReader(f)
	}

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		target := filepath.Join(dstDir, filepath.FromSlash(hdr.Name))

		// Security: prevent path traversal
		if !strings.HasPrefix(target, filepath.Clean(dstDir)+string(os.PathSeparator)) &&
			target != filepath.Clean(dstDir) {
			return fmt.Errorf("invalid tar path (possible traversal): %s", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(hdr.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}
	return nil
}
