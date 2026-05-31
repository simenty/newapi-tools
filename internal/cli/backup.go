// NewAPI Tools - Docker management platform for newapi
package cli

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/simenty/newapi-tools/internal/apperr"
	"github.com/simenty/newapi-tools/internal/core"
	"github.com/simenty/newapi-tools/internal/docker"
	"github.com/simenty/newapi-tools/internal/i18n"
	"github.com/simenty/newapi-tools/internal/ui"
	"github.com/spf13/cobra"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup new-api data",
	Long:  `Create a backup of new-api data including database, configuration files, and Docker Compose settings.`,
	RunE:  runBackup,
}

func init() {
	backupCmd.Flags().String("output", "", "backup output directory (default from config)")
	backupCmd.Flags().Bool("compress", true, "compress the backup archive")

	rootCmd.AddCommand(backupCmd)
}

func runBackup(cmd *cobra.Command, args []string) error {
	cfg := core.GetConfig()
	if cfg == nil {
		return fmt.Errorf("configuration not loaded")
	}

	// Resolve output directory
	outputDir, _ := cmd.Flags().GetString("output")
	if outputDir == "" {
		outputDir = cfg.NewAPI.BackupDir
	}
	if outputDir == "" {
		outputDir = filepath.Join(cfg.NewAPI.Home, "backups")
	}

	compress, _ := cmd.Flags().GetBool("compress")

	// Ensure new-api home exists
	if _, err := os.Stat(cfg.NewAPI.Home); os.IsNotExist(err) {
		return apperr.New(apperr.CodeBackupFailed, fmt.Sprintf("new-api 安装目录不存在: %s", cfg.NewAPI.Home), "", err)
	}

	// Check Docker availability (needed to dump MySQL)
	client, dockerErr := docker.NewClient(cfg.Docker.ComposeCmd)
	if dockerErr != nil {
		ui.L().Warn("docker not available, skipping database dump", "error", dockerErr)
		client = nil
	} else {
		defer client.Close()
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Build backup filename: newapi-backup-YYYYMMDD-HHMMSS.tar.gz
	timestamp := time.Now().Format("20060102-150405")
	var archiveName string
	if compress {
		archiveName = fmt.Sprintf("newapi-backup-%s.tar.gz", timestamp)
	} else {
		archiveName = fmt.Sprintf("newapi-backup-%s.tar", timestamp)
	}
	archivePath := filepath.Join(outputDir, archiveName)

	ui.L().Info("creating backup", "path", archivePath)
	ui.PrintStep(1, 3, i18n.T("backup.creating", archivePath))

	// Staging directory for backup contents
	stageDir, err := os.MkdirTemp("", "newapi-backup-*")
	if err != nil {
		return fmt.Errorf("failed to create staging directory: %w", err)
	}
	defer os.RemoveAll(stageDir)

	// Step [2/3]: Package data
	ui.PrintStep(2, 3, "backup.copying_data")

	// Copy config files from home directory
	for _, f := range []string{"docker-compose.yml", ".env"} {
		src := filepath.Join(cfg.NewAPI.Home, f)
		dst := filepath.Join(stageDir, f)
		if err := copyFileIfExists(src, dst); err != nil {
			ui.L().Warn("failed to copy config file", "file", f, "error", err)
		}
	}

	// Copy data directory if it exists
	dataDir := filepath.Join(cfg.NewAPI.Home, "data")
	if _, statErr := os.Stat(dataDir); statErr == nil {
		if err := copyDir(dataDir, filepath.Join(stageDir, "data")); err != nil {
			ui.L().Warn("failed to copy data directory", "error", err)
		} else {
			fmt.Println("  Copied data directory")
		}
	}

	// Dump MySQL database if Docker is accessible
	if client != nil {
		dbDumpPath := filepath.Join(stageDir, "newapi.sql")
		if err := dumpMySQL(cmd.Context(), dbDumpPath); err != nil {
			ui.L().Warn("mysql dump skipped", "error", err)
			fmt.Printf("  Warning: MySQL dump skipped: %v\n", err)
		} else {
			fmt.Println("  MySQL database dumped")
		}
	}

	// Step [3/3]: Create archive and report
	ui.PrintStep(3, 3, "backup.complete")

	// Create tar archive
	if err := createTarArchive(archivePath, stageDir, compress); err != nil {
		return apperr.Wrap(apperr.CodeBackupFailed, "", err)
	}

	// Report size
	var sizeStr string
	if info, err := os.Stat(archivePath); err == nil {
		sizeMB := float64(info.Size()) / 1024 / 1024
		sizeStr = fmt.Sprintf("%.2f MB", sizeMB)
		ui.L().Info("backup completed", "path", archivePath, "size_bytes", info.Size())
	} else {
		sizeStr = "unknown"
	}

	fmt.Println()
	fmt.Println("Backup complete!")
	fmt.Printf("  File:  %s\n", archivePath)
	fmt.Printf("  Size:  %s\n", sizeStr)
	fmt.Printf("  Time:  %s\n", timestamp)

	return nil
}

// dumpMySQL runs mysqldump inside the running mysql container and writes to dstPath.
// It reads MYSQL_ROOT_PASSWORD from the .env file in the newapi home directory.
func dumpMySQL(ctx context.Context, dstPath string) error {
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("docker not found: %w", err)
	}

	// Read MYSQL_ROOT_PASSWORD from .env
	cfg := core.GetConfig()
	password := ""
	if cfg != nil {
		password = readEnvValue(filepath.Join(cfg.NewAPI.Home, ".env"), "MYSQL_ROOT_PASSWORD")
	}
	if password == "" {
		// Fallback: read from container environment
		password = readContainerEnv(ctx, dockerPath, "mysql", "MYSQL_ROOT_PASSWORD")
	}

	f, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create dump file: %w", err)
	}
	defer f.Close()

	args := []string{
		"exec", "mysql",
		"mysqldump", "--no-tablespaces", "-u", "root",
	}
	if password != "" {
		// Use MYSQL_PWD env variable instead of --password= flag
		// to avoid exposing password in /proc/PID/cmdline
		args = append(args, "newapi")
		cmd := exec.CommandContext(ctx, dockerPath, args...)
		cmd.Env = append(os.Environ(), "MYSQL_PWD="+password)
		cmd.Stdout = f
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			os.Remove(dstPath)
			return fmt.Errorf("mysqldump failed: %w", err)
		}
		return nil
	}
	args = append(args, "newapi")

	cmd := exec.CommandContext(ctx, dockerPath, args...)
	cmd.Stdout = f
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		os.Remove(dstPath)
		return fmt.Errorf("mysqldump failed: %w", err)
	}
	return nil
}

// readEnvValue reads a KEY=VALUE pair from an env file.
// Returns empty string if the file or key is not found.
func readEnvValue(envFile, key string) string {
	data, err := os.ReadFile(envFile)
	if err != nil {
		return ""
	}
	prefix := key + "="
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			return strings.TrimPrefix(line, prefix)
		}
	}
	return ""
}

// readContainerEnv reads an environment variable from a running Docker container.
func readContainerEnv(ctx context.Context, dockerPath, container, envKey string) string {
	out, err := exec.CommandContext(ctx, dockerPath,
		"exec", container, "printenv", envKey,
	).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// createTarArchive creates a tar (optionally gzip-compressed) archive of srcDir.
func createTarArchive(archivePath, srcDir string, compress bool) error {
	f, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	var tw *tar.Writer
	if compress {
		gw := gzip.NewWriter(f)
		defer gw.Close()
		tw = tar.NewWriter(gw)
	} else {
		tw = tar.NewWriter(f)
	}
	defer tw.Close()

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = filepath.ToSlash(rel)

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		fh, err := os.Open(path)
		if err != nil {
			return err
		}

		_, err = io.Copy(tw, fh)
		fh.Close()
		return err
	})
}

// copyFileIfExists copies src to dst only if src exists. Skips silently if missing.
func copyFileIfExists(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// copyDir recursively copies a directory tree from src to dst.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		return copyFileIfExists(path, target)
	})
}
