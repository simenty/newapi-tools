package selfupdate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// SelfUpdateOptions configures the self-update process
type SelfUpdateOptions struct {
	CurrentBinary string                          // Path to current binary (os.Executable())
	Repo          string                          // GitHub repo name "owner/repo"
	BackupDir     string                          // Backup directory (default ~/.config/newapi-tools/backups/)
	OnProgress    func(stage string, pct float64) // Progress callback
}

// SelfUpdateResult contains the result of a self-update
type SelfUpdateResult struct {
	PreviousVersion string // Version before update
	NewVersion      string // Version after update
	BinaryPath      string // Path to the binary
	BackupPath      string // Path to the backup of old version
}

// Run executes the self-update process: check -> download -> verify -> backup -> replace
func Run(ctx context.Context, opts SelfUpdateOptions) (*SelfUpdateResult, error) {
	// Check latest release
	release, err := CheckLatest(ctx, opts.Repo)
	if err != nil {
		return nil, fmt.Errorf("check latest: %w", err)
	}

	// Resolve asset name for current OS/ARCH
	assetName := resolveAssetName()

	// Find matching asset (prefix match for versioned names)
	var asset *Asset
	for _, a := range release.Assets {
		if assetMatches(a.Name, assetName) {
			asset = &a
			break
		}
	}
	if asset == nil {
		return nil, fmt.Errorf("no asset found for %s", assetName)
	}

	// Create temp file for download - with secure permissions
	tmpFile, err := os.CreateTemp("", "newapi-update-*.tmp")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	// Set secure permissions first
	if err := os.Chmod(tmpPath, 0600); err != nil {
		os.Remove(tmpPath)
		return nil, fmt.Errorf("set temp file permissions: %w", err)
	}
	defer os.Remove(tmpPath)

	// Download asset
	if err := downloadAsset(ctx, asset, tmpPath, opts.OnProgress); err != nil {
		return nil, fmt.Errorf("download asset: %w", err)
	}

	// Verify SHA256 (optional but recommended)
	if err := verifySHA256(ctx, tmpPath, asset); err != nil {
		return nil, fmt.Errorf("verify SHA256: %w", err)
	}

	// Now set executable permission after successful verification
	if err := os.Chmod(tmpPath, 0755); err != nil {
		return nil, fmt.Errorf("set executable permission: %w", err)
	}

	// Backup and replace
	backupPath, err := backupAndReplace(opts.CurrentBinary, tmpPath, opts.BackupDir)
	if err != nil {
		return nil, fmt.Errorf("backup and replace: %w", err)
	}

	return &SelfUpdateResult{
		PreviousVersion: "", // Would need to get this from binary
		NewVersion:      release.TagName,
		BinaryPath:      opts.CurrentBinary,
		BackupPath:      backupPath,
	}, nil
}

// downloadAsset downloads the specified asset to dest file
func downloadAsset(ctx context.Context, asset *Asset, dest string, onProgress func(stage string, pct float64)) error {
	req, err := http.NewRequestWithContext(ctx, "GET", asset.BrowserDownloadURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "newapi-tools")

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("open dest file: %w", err)
	}
	defer out.Close()

	var written int64
	buf := make([]byte, 32*1024) // 32KB buffer
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, err := out.Write(buf[:n]); err != nil {
				return fmt.Errorf("write file: %w", err)
			}
			written += int64(n)
			if onProgress != nil && asset.Size > 0 {
				pct := float64(written) / float64(asset.Size) * 100
				onProgress("download", pct)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read response: %w", err)
		}
	}

	return nil
}

// verifySHA256 verifies the SHA256 hash of the downloaded file
// Tries to read from <asset>.sha256 file from GitHub Release
func verifySHA256(ctx context.Context, filePath string, asset *Asset) error {
	// Calculate hash of downloaded file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("calculate hash: %w", err)
	}
	calculatedHash := hex.EncodeToString(hash.Sum(nil))

	// Try to download SHA256 checksum file
	sha256URL := asset.BrowserDownloadURL + ".sha256"
	req, err := http.NewRequestWithContext(ctx, "GET", sha256URL, nil)
	if err != nil {
		fmt.Printf("Warning: Could not create SHA256 request: %v\n", err)
		fmt.Printf("SHA256: %s\n", calculatedHash)
		fmt.Println("Continuing without verification...")
		return nil
	}
	req.Header.Set("User-Agent", "newapi-tools")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Warning: SHA256 checksum unavailable, skipping verification")
		fmt.Printf("SHA256: %s\n", calculatedHash)
		fmt.Println("Continuing without verification...")
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		fmt.Printf("Warning: SHA256 file not available (HTTP %d)\n", resp.StatusCode)
		fmt.Printf("SHA256: %s\n", calculatedHash)
		fmt.Println("Continuing without verification...")
		return nil
	}
	defer resp.Body.Close()

	// Read and parse SHA256
	checksumData, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Warning: Could not read SHA256 file: %v\n", err)
		fmt.Printf("SHA256: %s\n", calculatedHash)
		fmt.Println("Continuing without verification...")
		return nil
	}

	// Extract the hash from the content (format: "<hash>  <filename>")
	parts := strings.Fields(string(checksumData))
	if len(parts) < 1 {
		fmt.Printf("Warning: Invalid SHA256 file format\n")
		fmt.Printf("SHA256: %s\n", calculatedHash)
		fmt.Println("Continuing without verification...")
		return nil
	}

	expectedHash := strings.ToLower(parts[0])
	calculatedHashLower := strings.ToLower(calculatedHash)

	if expectedHash != calculatedHashLower {
		return fmt.Errorf("SHA256 verification failed:\nExpected: %s\nGot:      %s", expectedHash, calculatedHashLower)
	}

	fmt.Printf("SHA256 verified: %s\n", calculatedHashLower)
	return nil
}

// backupAndReplace backs up current binary and replaces with new one
func backupAndReplace(currentBin, newBin, backupDir string) (string, error) {
	// Create backup directory if needed
	if backupDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("get user home: %w", err)
		}
		backupDir = filepath.Join(home, ".config", "newapi-tools", "backups")
	}

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("create backup dir: %w", err)
	}

	// Create backup filename
	backupPath := filepath.Join(backupDir, filepath.Base(currentBin)+".bak")

	// Copy current binary to backup
	srcFile, err := os.Open(currentBin)
	if err != nil {
		return "", fmt.Errorf("open current bin: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(backupPath)
	if err != nil {
		return "", fmt.Errorf("create backup file: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return "", fmt.Errorf("copy to backup: %w", err)
	}

	// Make sure new binary is executable
	if err := os.Chmod(newBin, 0755); err != nil {
		return "", fmt.Errorf("chmod new bin: %w", err)
	}

	// Atomic replace: rename new file to current
	if err := os.Rename(newBin, currentBin); err != nil {
		// Try to restore backup if replace fails
		if restoreErr := os.Rename(backupPath, currentBin); restoreErr != nil {
			fmt.Printf("Warning: Failed to restore backup: %v\n", restoreErr)
		}
		return "", fmt.Errorf("replace binary: %w", err)
	}

	return backupPath, nil
}

// resolveAssetName resolves the download filename based on current GOOS/GOARCH.
// Matches goreleaser's name_template: {{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}
// Since version is unknown at asset resolution time, we do a prefix match in Run().
func resolveAssetName() string {
	osName := runtime.GOOS
	archName := runtime.GOARCH

	// Normalize OS names
	if osName == "windows" {
		osName = "windows"
	} else if osName == "darwin" {
		osName = "darwin"
	} else {
		osName = "linux"
	}

	// Normalize arch names
	if archName == "amd64" {
		archName = "amd64"
	} else if archName == "arm64" {
		archName = "arm64"
	} else {
		archName = "amd64" // Default to amd64
	}

	return fmt.Sprintf("newapi-tools_%s_%s", osName, archName)
}

// assetMatches checks if an asset name matches the desired platform.
// Uses prefix matching to handle version strings in the name.
func assetMatches(assetName, prefix string) bool {
	return strings.HasPrefix(assetName, prefix)
}
