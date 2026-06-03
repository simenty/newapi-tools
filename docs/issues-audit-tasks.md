# Issue 审计报告 & TRAE 修复任务

> **审计基准**：v3.5.0（commit `d8e189f8`）  
> **审计来源**：GitHub Issues #1, #2, #3 + 代码直接检查  
> **审计日期**：2026-06-04

---

## 一、Issue 清单与状态

### Issue #1 — [SEC][P0] selfupdate 四个核心函数无测试覆盖

**问题**：`Run()`、`downloadAsset()`、`verifySHA256()`、`backupAndReplace()` 覆盖率 0%。

**当前状态**：selfupdate 覆盖率 21.7%（TRAE 之前补了 parseSemver/CompareVersions/resolveAssetName/copyFile 的测试，但核心流程无测试）

**需要**：为 4 个核心函数写测试，至少覆盖正常路径和主要异常路径。

---

### Issue #2 — [SEC][P0] SHA256 校验默认 RequireSHA256=false

**问题**：`SelfUpdateOptions.RequireSHA256` 默认 false，导致 SHA256 校验失败时静默跳过。

**当前代码**：
```go
// internal/cli/update.go:401
opts := selfupdate.SelfUpdateOptions{
    CurrentBinary: currentBinary,
    // RequireSHA256 未设置 → 默认 false
}
```

**需要**：加一行 `RequireSHA256: true`。

---

### Issue #3 — [SEC][P1] backup 目录权限 + 归档校验

**问题 A**：`backup.go:70` — 输出目录 `os.MkdirAll(outputDir, 0755)`，应改为 0700。  
**问题 B**：`backup.go:302` — 恢复时解压目录 `os.MkdirAll(filepath.Dir(dst), 0755)`，应改为 0700。  
**问题 C**：备份归档缺少 checksum 文件，无法检测篡改。  
**问题 D**：`.env` 文件在归档中是明文，包含数据库密码等敏感信息（先评估风险，暂不改）。

---

## 二、TRAE 执行任务清单

按优先级排列，依次完成：

---

### T-01: 修复 RequireSHA256 默认值

**文件**：`internal/cli/update.go`  
**修改**：在 `SelfUpdateOptions` 初始化中增加 `RequireSHA256: true`

```go
// 第 401 行附近
opts := selfupdate.SelfUpdateOptions{
    CurrentBinary:  currentBinary,
    RequireSHA256:  true,  // 新增
}
```

**验证**：
- `go build ./...`
- `go test ./internal/selfupdate/...`

---

### T-02: 修复 backup 目录权限

**文件**：`internal/cli/backup.go`  
**修改**：

```go
// 第 70 行：0755 → 0700
if err := os.MkdirAll(outputDir, 0700); err != nil {

// 第 302 行：0755 → 0700
if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
```

**验证**：
- `go test ./internal/cli/...`

---

### T-03: 补 `verifySHA256` 测试

**文件**：`internal/selfupdate/selfupdate_test.go`  
**目标**：覆盖以下场景

```go
func TestVerifySHA256(t *testing.T) {
    // 1. 正常：文件 hash 匹配 → 无错误
    // 2. 正常：requireSHA256=false + .sha256 不可用 → 不报错
    // 3. 异常：requireSHA256=true + .sha256 不可用 → 返回 error
    // 4. 异常：文件 hash 不匹配 → 返回 error
    // 5. 边界：空文件
}
```

**提示**：需要创建临时文件和 mock HTTP server（`httptest.NewServer`）。

---

### T-04: 补 `backupAndReplace` 测试

**文件**：`internal/selfupdate/selfupdate_test.go`  
**目标**：覆盖正常替换和跨分区回退

```go
func TestBackupAndReplace(t *testing.T) {
    // 1. 正常：同分区替换成功
    // 2. 异常：新文件不存在 → 返回 error
    // 3. 边界：备份目录不存在 → 自动创建
}
```

**提示**：在临时目录中创建源文件和目标文件，用 `os.Rename` 模拟。

---

### T-05: 补 `downloadAsset` 测试

**文件**：`internal/selfupdate/selfupdate_test.go`  
**目标**：覆盖下载流程

```go
func TestDownloadAsset(t *testing.T) {
    // 1. 正常：HTTP 200 + 正确内容
    // 2. 异常：HTTP 404 → 返回 error
    // 3. 异常：网络超时 → 返回 error
}
```

**提示**：用 `httptest.NewServer` mock HTTP 端点，返回测试数据。

---

### T-06: 补 `Run` 核心流程测试

**文件**：`internal/selfupdate/selfupdate_test.go`  
**目标**：用 mock server 模拟完整更新流程

```go
func TestRun(t *testing.T) {
    // 1. 正常：完整流程（mock GitHub API + 下载 + SHA256 + 替换）
    // 2. 异常：GitHub API 返回错误 → 流程终止
    // 3. 异常：SHA256 不匹配 → 流程终止
}
```

**提示**：用 `httptest.NewServer` 模拟 GitHub Releases API、下载 URL 和 SHA256 文件。

---

## 三、验证清单

每项完成后确认：

- [ ] `go build ./...` 通过
- [ ] `go vet ./...` 通过
- [ ] `go test ./...` 全通过
- [ ] `internal/selfupdate` 覆盖率提升（目标：≥40%）
- [ ] 对应 Issue 可 close

---

## 四、执行顺序

```
T-01 (RequireSHA256)  ← 最简单，1 行代码
  ↓
T-02 (backup 权限)    ← 简单，2 行代码
  ↓
T-03 (verifySHA256 test)  ← 中等复杂度
  ↓
T-04 (backupAndReplace test)  ← 中等复杂度
  ↓
T-05 (downloadAsset test)  ← 较复杂
  ↓
T-06 (Run test)  ← 最复杂，需要完整 mock
```
