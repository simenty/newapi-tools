<img src="https://r2cdn.perplexity.ai/pplx-full-logo-primary-dark%402x.png" style="height:64px;margin-right:32px"/>

# newapi-tools V3.2 全面源码审计报告

> 审计时间：2026-05-31 | 审计基准：commit `924d70f`（main 分支）[^1]
> 覆盖范围：全部 13 个内部包 + plugins + CI/CD 流水线

***

## 一、项目概况

| 项目 | 状态 |
| :-- | :-- |
| 模块路径 | `github.com/simenty/newapi-tools` ✅ |
| Go 版本 | `go 1.23` ✅（已从 1.22 升级）[^2] |
| 直接依赖 | cobra v1.8.1 / viper v1.19.0 / yaml.v3 v3.0.1 |
| CI 矩阵 | ubuntu / macos / windows × go 1.23[^3] |
| 覆盖率门禁 | ≥ 60%[^3] |
| 最新发布 | v3.1.0[^1] |


***

## 二、问题清单（按严重程度分级）


***

### 🔴 BUG 级（影响正确性/安全性）

#### BUG-01：`MaskSecret` 字节切片导致多字节字符 panic

**文件**：`internal/security/security.go`[^4]

```go
// 当前代码——按字节切片，含中文 token 会 panic
return s[:2] + "****" + s[len(s)-2:]
```

`s[:2]` 是字节索引，对含中文、emoji 的 API Token 会截断 UTF-8 编码序列，造成 panic 或乱码。

**修复**：

```go
func MaskSecret(s string) string {
    r := []rune(s)
    if len(r) < 6 {
        return "****"
    }
    return string(r[:2]) + "****" + string(r[len(r)-2:])
}
```


***

#### BUG-02：`verifySHA256` 在 SHA256 文件不存在时静默跳过验证

**文件**：`internal/selfupdate/updater.go`[^5]

```go
// resp 为 nil 时仍解引用 resp.StatusCode → nil pointer panic
if err != nil || resp.StatusCode != http.StatusOK {
    fmt.Printf("...status: %d\n", resp.StatusCode) // resp 可能为 nil!
```

当网络请求失败（`err != nil`）时，`resp` 为 `nil`，后续 `resp.StatusCode` 将 panic。

**修复**：

```go
if err != nil {
    fmt.Println("Warning: SHA256 checksum unavailable, skipping verification")
    return nil
}
if resp.StatusCode != http.StatusOK {
    resp.Body.Close()
    fmt.Printf("Warning: SHA256 file not available (HTTP %d)\n", resp.StatusCode)
    return nil
}
```


***

#### BUG-03：`resolveAssetName` 生成的文件名与 goreleaser 实际产物不匹配

**文件**：`internal/selfupdate/updater.go` vs `.goreleaser.yml`[^6][^5]

goreleaser 生成产物名称格式为 `newapi-tools_VERSION_OS_ARCH.tar.gz`，但 `resolveAssetName()` 返回 `newapi-linux-amd64`（无 `tools_` 无版本无扩展名），导致自更新功能找不到任何 asset，始终返回 `"no asset found for newapi-linux-amd64"` 错误。

**修复**：资产名匹配逻辑应改为前缀匹配，或根据实际发布格式调整模板。

***

#### BUG-04：`store.go` 中 `Update()` 造成双重锁死锁

**文件**：`internal/instance/store.go`[^7]

```go
func (s *Store) Update(name string, updateFn func(*Instance)) error {
    s.mu.Lock()             // 获取锁
    defer s.mu.Unlock()
    instances, err := s.LoadWithoutLock()  // OK
    ...
    return s.SaveWithoutLock(instances)    // OK
}
```

`Update()` 调用链正常，但 `Load()` 和 `Save()` 本身也各自调用 `s.mu.Lock()`；若将来有人误调用 `Load()`（带锁版）而非 `LoadWithoutLock()`，将导致死锁。`LoadWithoutLock` / `SaveWithoutLock` 是公开方法（首字母大写），外部代码可直接绕过锁调用——这是竞态隐患。

**修复**：将 `LoadWithoutLock` / `SaveWithoutLock` 改为小写私有方法 `loadLocked` / `saveLocked`。

***

### 🟠 设计缺陷级（影响可维护性、测试、一致性）

#### DESIGN-01：`plugin.Context` 与 `core.NewAPIConfig` 字段重复

**文件**：`internal/plugin/plugin.go`、`internal/core/config.go`[^8][^9]

`plugin.Context` 完全复制了 `NewAPIConfig` 的 5 个核心字段（Home/Port/DockerImage/BackupDir/ComposeCmd），注释称"self-contained to avoid circular imports"。但 `plugin` 包可以直接引用 `core`（目前 `root.go` 已经同时 import 了两者），字段重复制造了配置漂移风险。

***

#### DESIGN-02：`NewAPIPlugin.Version()` 硬编码为 `"3.1.0"` 而非 `core.Version`

**文件**：`plugins/newapi/newapi.go`[^10]

```go
func (p *NewAPIPlugin) Version() string { return "3.1.0" }
```

项目已到 V3.2，插件版本与 `core.Version` 脱节。每次发版需手动同步，极易遗忘。

**修复**：

```go
import "github.com/simenty/newapi-tools/internal/core"
func (p *NewAPIPlugin) Version() string { return core.Version }
```


***

#### DESIGN-03：`core.Config` 缺少 `Validate()` 方法

**文件**：`internal/core/config.go`[^11]

新增的 `HealthTimeout`、`MaxBackups`、`Port` 字段没有边界校验。`Port=0`、`HealthTimeout=-1`、`MaxBackups=-5` 均可静默传入 Docker 操作层引发运行时错误。

***

#### DESIGN-04：`install.go` 中配置应用逻辑与 `root.go` 的 `syncInstanceToConfig` 重复

**文件**：`internal/cli/install.go`、`internal/cli/root.go`[^12][^13]

`runInstall` 中手动赋值 `cfg.NewAPI.Home = inst.Home` 等 3 行，与 `syncInstanceToConfig()` 完全重复，且 `install.go` 的版本遗漏了 `Domain`、`HealthTimeout`、`MaxBackups` 字段同步（仅同步了 3 个字段，`syncInstanceToConfig` 同步 6 个）。

***

#### DESIGN-05：`doctor.go` 错误码使用不当

**文件**：`internal/cli/doctor.go`[^14]

```go
// doctor 检查失败时，返回的是 Docker 未找到的错误码
return apperr.New(apperr.CodeDockerNotFound, fmt.Sprintf("%d 项诊断检查失败", failCount), "", nil)
```

`failCount` 个检查失败不等于"Docker not found"（D001），应使用更通用的诊断失败码，或直接返回 `fmt.Errorf`。

***

#### DESIGN-06：`AuditLogger.Log()` 声明返回 `error` 但调用方注释说"不应影响命令执行"

**文件**：`internal/audit/audit.go`、`internal/cli/root.go`[^13][^15]

函数文档写道"writing fails, error is logged via slog.Warn but not returned"，但实际上函数会在多处返回非 nil error。`root.go` 调用处：

```go
if auditErr := globalAuditLogger.Log(entry); auditErr != nil {
    slog.Warn("audit log write failed", "error", auditErr)
}
```

行为一致，但函数签名与文档形成混淆。应统一为返回 `error` 或改为 `void`（内部 warn）。

***

#### DESIGN-07：`CompareVersions` 仅做字符串相等比较，无语义版本解析

**文件**：`internal/selfupdate/checker.go`[^16]

```go
// Simple comparison for now - just check if they are different
return current != latest, nil
```

当 `current = "v3.2.0"` 而 `latest = "v3.2.0"` 时正确，但若存在 pre-release（`v3.3.0-rc1`），会误报有更新。代码注释也承认"production should use semantic version comparison"，属于已知技术债。

***

### 🟡 代码质量问题

#### QUALITY-01：`backup.go` 中 `dumpMySQL` 通过命令行拼接密码（安全风险）

**文件**：`internal/cli/backup.go`[^17]

```go
args = append(args, "--password="+password)
```

密码以 `--password=xxx` 明文传入命令行参数，在 Linux 上通过 `/proc/PID/cmdline` 可被任何同权限进程读取。

**修复**：使用 `MYSQL_PWD` 环境变量，或通过 `--defaults-extra-file` 传入临时配置文件。

***

#### QUALITY-02：`security.CheckConfigPerm` / `FixConfigPerm` 返回原生 `fmt.Errorf`，绕过 apperr 体系

**文件**：`internal/security/security.go`[^4]

全项目统一使用 `apperr.New()` 构建结构化错误，但 security 包直接返回 `fmt.Errorf`，且已有对应错误码 `C002`（`CodeConfigPerm`）。

**修复**：

```go
return apperr.New(apperr.CodeConfigPerm,
    fmt.Sprintf("config file %s has overly permissive permissions (%04o)", path, perm), "", nil)
```


***

#### QUALITY-03：`update.go` 中 `performRestore` 步骤序号硬编码错误

**文件**：`internal/cli/update.go`[^18]

```go
ui.PrintStep(3, 2, "Restarting containers...")  // 第3步，但总步数为2 → [3/2]
```

`PrintStep(3, 2, ...)` 会输出 `[3/2]`，逻辑错误。

***

#### QUALITY-04：`docker/compose.go` compose 命令路径拼接存在空格注入风险

**文件**：`internal/docker/compose.go`[^19]

```go
parts := strings.Split(composeCmd, " ")
args := append(parts, "-f", projectDir+"/docker-compose.yml", "up", "-d")
cmd := exec.CommandContext(ctx, args[^0], args[1:]...)
```

若 `projectDir` 包含空格（如 `/opt/my project/`），`projectDir+"/docker-compose.yml"` 作为单个参数传入不会有问题，但 `composeCmd` 用空格分割（如 `docker compose`）若用户配置为 `docker  compose`（双空格），会产生空字符串参数。

***

#### QUALITY-05：`i18n.T()` 在 `Init()` 未调用时静默返回 key

**文件**：`internal/i18n/i18n.go`[^20]

```go
func T(key string, args ...any) string {
    if defaultBundle == nil {
        return key  // 静默降级，可能导致用户看到原始 key 如 "install.pulling_image"
    }
    ...
}
```

若 `initConfig()` 中 `i18n.Init()` 失败（网络/文件问题），所有界面文字将显示为 key 字符串，用户体验极差，但系统不报错。应至少在 slog 中记录 warn。

***

#### QUALITY-06：`selfupdate` 包无任何单元测试

**文件**：`internal/selfupdate/`[^21]

`checker.go` 和 `updater.go` 均无 `_test.go` 文件，且核心逻辑（HTTP 请求、文件替换、SHA256 验证）未被测试覆盖，是 CI 覆盖率的薄弱点。

***

#### QUALITY-07：`golangci.yml` 排除 G104（未处理错误）

**文件**：`.golangci.yml`[^22]

```yaml
gosec:
  excludes:
    - G104  # Errors unhandled.
```

G104 是 gosec 最重要的规则之一，全局排除意味着所有 `os.Remove`、`f.Close()` 等被忽略的错误不会被静态分析工具发现。

***

#### QUALITY-08：`gocyclo` 复杂度阈值 15 过松

**文件**：`.golangci.yml`[^22]

注释写道 `# doctor.go 当前约 30+，先设宽松阈值逐步收紧`，但将门禁设为 15 后 `doctor.go` 中的大量函数（实际 CC > 15）仍会被放行，等于无效门禁。

***

### 🔵 依赖与 CI 问题

#### CI-01：CI actions 未固定到 SHA digest（供应链安全）

**文件**：`.github/

<div align="center">⁂</div>

[^1]: https://github.com/simenty/newapi-tools

[^2]: https://raw.githubusercontent.com/simenty/newapi-tools/main/go.mod

[^3]: https://raw.githubusercontent.com/simenty/newapi-tools/main/.github/workflows/ci.yml

[^4]: https://raw.githubusercontent.com/simenty/newapi-tools/main/internal/security/security.go

[^5]: https://raw.githubusercontent.com/simenty/newapi-tools/main/internal/selfupdate/updater.go

[^6]: https://raw.githubusercontent.com/simenty/newapi-tools/main/.goreleaser.yml

[^7]: https://raw.githubusercontent.com/simenty/newapi-tools/main/internal/instance/store.go

[^8]: https://raw.githubusercontent.com/simenty/newapi-tools/main/internal/plugin/plugin.go

[^9]: https://raw.githubusercontent.com/simenty/newapi-tools/main/internal/core/config.go

[^10]: https://raw.githubusercontent.com/simenty/newapi-tools/main/plugins/newapi/newapi.go

[^11]: https://raw.githubusercontent.com/simenty/newapi-tools/main/internal/core/config.go

[^12]: https://raw.githubusercontent.com/simenty/newapi-tools/main/internal/cli/install.go

[^13]: https://raw.githubusercontent.com/simenty/newapi-tools/main/internal/cli/root.go

[^14]: https://raw.githubusercontent.com/simenty/newapi-tools/main/internal/cli/doctor.go

[^15]: https://raw.githubusercontent.com/simenty/newapi-tools/main/internal/audit/audit.go

[^16]: https://raw.githubusercontent.com/simenty/newapi-tools/main/internal/selfupdate/checker.go

[^17]: https://raw.githubusercontent.com/simenty/newapi-tools/main/internal/cli/backup.go

[^18]: https://raw.githubusercontent.com/simenty/newapi-tools/main/internal/cli/update.go

[^19]: https://raw.githubusercontent.com/simenty/newapi-tools/main/internal/docker/compose.go

[^20]: https://raw.githubusercontent.com/simenty/newapi-tools/main/internal/i18n/i18n.go

[^21]: https://github.com/simenty/newapi-tools/tree/main/internal/selfupdate

[^22]: https://raw.githubusercontent.com/simenty/newapi-tools/main/.golangci.yml

