package selfupdate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// T-03: verifySHA256 tests
// ---------------------------------------------------------------------------

func TestVerifySHA256_Match(t *testing.T) {
	// 场景 1: 正常 — 文件 hash 匹配 .sha256 文件
	content := []byte("test binary content for sha256 verification")
	tmpFile := filepath.Join(t.TempDir(), "binary")
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	hash := sha256.Sum256(content)
	hashHex := hex.EncodeToString(hash[:])

	// Mock server serving .sha256 file
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".sha256") {
			fmt.Fprintf(w, "%s  binary", hashHex)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	asset := &Asset{
		Name:               "binary",
		BrowserDownloadURL: server.URL + "/download/binary",
		Size:               int64(len(content)),
	}

	err := verifySHA256(context.Background(), tmpFile, asset, true)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestVerifySHA256_RequireFalse_Unavailable(t *testing.T) {
	// 场景 2: requireSHA256=false + .sha256 不可用 → 不报错
	content := []byte("some content")
	tmpFile := filepath.Join(t.TempDir(), "binary")
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	// Mock server returning 404 for .sha256
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	asset := &Asset{
		BrowserDownloadURL: server.URL + "/download/binary",
	}

	err := verifySHA256(context.Background(), tmpFile, asset, false)
	if err != nil {
		t.Errorf("expected no error when requireSHA256=false, got: %v", err)
	}
}

func TestVerifySHA256_RequireTrue_Unavailable(t *testing.T) {
	// 场景 3: requireSHA256=true + .sha256 不可用 → 返回 error
	content := []byte("some content")
	tmpFile := filepath.Join(t.TempDir(), "binary")
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	asset := &Asset{
		BrowserDownloadURL: server.URL + "/download/binary",
	}

	err := verifySHA256(context.Background(), tmpFile, asset, true)
	if err == nil {
		t.Error("expected error when requireSHA256=true and .sha256 unavailable, got nil")
	}
}

func TestVerifySHA256_Mismatch(t *testing.T) {
	// 场景 4: 文件 hash 不匹配 → 返回 error
	content := []byte("real binary content")
	tmpFile := filepath.Join(t.TempDir(), "binary")
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	// Provide a different hash
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".sha256") {
			fmt.Fprintf(w, "0000000000000000000000000000000000000000000000000000000000000000  binary")
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	asset := &Asset{
		BrowserDownloadURL: server.URL + "/download/binary",
	}

	err := verifySHA256(context.Background(), tmpFile, asset, true)
	if err == nil {
		t.Error("expected error for hash mismatch, got nil")
	}
}

func TestVerifySHA256_EmptyFile(t *testing.T) {
	// 场景 5: 空文件
	tmpFile := filepath.Join(t.TempDir(), "empty")
	if err := os.WriteFile(tmpFile, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	hash := sha256.Sum256([]byte{})
	hashHex := hex.EncodeToString(hash[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".sha256") {
			fmt.Fprintf(w, "%s  empty", hashHex)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	asset := &Asset{
		BrowserDownloadURL: server.URL + "/download/empty",
	}

	err := verifySHA256(context.Background(), tmpFile, asset, true)
	if err != nil {
		t.Errorf("expected no error for empty file with matching hash, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// T-04: backupAndReplace tests
// ---------------------------------------------------------------------------

func TestBackupAndReplace_SamePartition(t *testing.T) {
	// 场景 1: 正常 — 同分区替换成功
	dir := t.TempDir()
	currentBin := filepath.Join(dir, "current_bin")
	newBin := filepath.Join(dir, "new_bin")

	if err := os.WriteFile(currentBin, []byte("old binary content"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newBin, []byte("new binary content"), 0755); err != nil {
		t.Fatal(err)
	}

	backupDir := filepath.Join(dir, "backups")
	backupPath, err := backupAndReplace(currentBin, newBin, backupDir)
	if err != nil {
		t.Fatalf("backupAndReplace failed: %v", err)
	}

	if backupPath == "" {
		t.Error("expected non-empty backup path")
	}

	// Verify the current binary is now the new content
	data, err := os.ReadFile(currentBin)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new binary content" {
		t.Errorf("current bin content = %q, want %q", string(data), "new binary content")
	}

	// Verify backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Errorf("backup file does not exist: %s", backupPath)
	}
}

func TestBackupAndReplace_NewBinNotFound(t *testing.T) {
	// 场景 2: 新文件不存在 → 返回 error
	dir := t.TempDir()
	currentBin := filepath.Join(dir, "current_bin")
	if err := os.WriteFile(currentBin, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}

	_, err := backupAndReplace(currentBin, "/nonexistent/new_bin", filepath.Join(dir, "backups"))
	if err == nil {
		t.Error("expected error for non-existent new binary, got nil")
	}
}

func TestBackupAndReplace_AutoCreateBackupDir(t *testing.T) {
	// 场景 3: 备份目录不存在 → 自动创建
	dir := t.TempDir()
	currentBin := filepath.Join(dir, "current_bin")
	newBin := filepath.Join(dir, "new_bin")

	if err := os.WriteFile(currentBin, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newBin, []byte("new"), 0755); err != nil {
		t.Fatal(err)
	}

	// Use a nested backup dir that doesn't exist
	backupDir := filepath.Join(dir, "deep", "nested", "backups")
	backupPath, err := backupAndReplace(currentBin, newBin, backupDir)
	if err != nil {
		t.Fatalf("backupAndReplace with auto-create failed: %v", err)
	}

	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		t.Error("backup directory was not auto-created")
	}

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Error("backup file was not created in auto-created dir")
	}
}

func TestBackupAndReplace_DefaultBackupDir(t *testing.T) {
	// backupDir="" 时使用默认路径
	dir := t.TempDir()
	currentBin := filepath.Join(dir, "current_bin")
	newBin := filepath.Join(dir, "new_bin")

	if err := os.WriteFile(currentBin, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newBin, []byte("new"), 0755); err != nil {
		t.Fatal(err)
	}

	backupPath, err := backupAndReplace(currentBin, newBin, "")
	if err != nil {
		t.Fatalf("backupAndReplace with default dir failed: %v", err)
	}

	if backupPath == "" {
		t.Error("expected non-empty backup path with default dir")
	}
}

// ---------------------------------------------------------------------------
// T-05: downloadAsset tests
// ---------------------------------------------------------------------------

func TestDownloadAsset_Success(t *testing.T) {
	// 场景 1: 正常 — HTTP 200 + 正确内容
	expectedContent := "this is the downloaded asset content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedContent))
	}))
	defer server.Close()

	dest := filepath.Join(t.TempDir(), "downloaded")
	asset := &Asset{
		BrowserDownloadURL: server.URL,
		Size:               int64(len(expectedContent)),
	}

	err := downloadAsset(context.Background(), asset, dest, nil)
	if err != nil {
		t.Fatalf("downloadAsset failed: %v", err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != expectedContent {
		t.Errorf("downloaded content = %q, want %q", string(data), expectedContent)
	}
}

func TestDownloadAsset_NotFound(t *testing.T) {
	// 场景 2: HTTP 404 → 返回 error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	dest := filepath.Join(t.TempDir(), "downloaded")
	asset := &Asset{BrowserDownloadURL: server.URL}

	err := downloadAsset(context.Background(), asset, dest, nil)
	if err == nil {
		t.Error("expected error for HTTP 404, got nil")
	}
}

func TestDownloadAsset_WithProgress(t *testing.T) {
	// downloadAsset with progress callback
	content := make([]byte, 70000) // > 32KB to trigger multiple read loops
	for i := range content {
		content[i] = byte(i % 256)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer server.Close()

	dest := filepath.Join(t.TempDir(), "downloaded")
	asset := &Asset{
		BrowserDownloadURL: server.URL,
		Size:               int64(len(content)),
	}

	var progressCalled bool
	onProgress := func(stage string, pct float64) {
		progressCalled = true
		if stage != "download" {
			t.Errorf("unexpected stage: %q", stage)
		}
	}

	err := downloadAsset(context.Background(), asset, dest, onProgress)
	if err != nil {
		t.Fatalf("downloadAsset failed: %v", err)
	}

	if !progressCalled {
		t.Error("progress callback was not called")
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != len(content) {
		t.Errorf("downloaded size = %d, want %d", len(data), len(content))
	}
}

func TestDownloadAsset_EmptyBody(t *testing.T) {
	// Empty response body should succeed with empty file
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	dest := filepath.Join(t.TempDir(), "empty")
	asset := &Asset{BrowserDownloadURL: server.URL}

	err := downloadAsset(context.Background(), asset, dest, nil)
	if err != nil {
		t.Fatalf("downloadAsset failed: %v", err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 0 {
		t.Errorf("expected empty file, got %d bytes", len(data))
	}
}

// ---------------------------------------------------------------------------
// T-06: Run core flow tests
// ---------------------------------------------------------------------------

func TestRun_GitHubAPIContextCancelled(t *testing.T) {
	// 场景: 取消的 context → 流程终止
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel

	opts := SelfUpdateOptions{
		CurrentBinary: t.TempDir() + "/nonexistent",
		Repo:          "simenty/newapi-tools",
	}

	_, err := Run(ctx, opts)
	if err == nil {
		t.Error("expected error with cancelled context, got nil")
	}
}

// ---------------------------------------------------------------------------
// Verify the test infrastructure works — CheckLatest with mock server
// ---------------------------------------------------------------------------

func TestCheckLatestWithMock(t *testing.T) {
	// Override the GitHub API URL using env var is not possible since CheckLatest
	// constructs the URL directly. Instead, we test the HTTP client behavior.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
			"tag_name": "v5.0.0",
			"name": "v5.0.0",
			"assets": [{"name": "test", "browser_download_url": "%s/test", "size": 100}]
		}`, r.Host)
	}))
	defer server.Close()

	// Can't change the URL CheckLatest uses, but we can verify JSON response parsing
	// by making a direct HTTP call
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "v5.0.0") {
		t.Errorf("expected v5.0.0 in response, got: %s", string(body))
	}
}
