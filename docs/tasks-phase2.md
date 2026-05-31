# Phase 2 — 测试补全 & 功能完善

> 基于 main 审计报告（docs/audit-and-tasks.md）的 P1/P2 任务  
> 接手人：TRAE | 分支：handoff | 基准：v3.4.0

---

## 工作方式

```
每步：改一处 → go build → go test → git commit → git push
完成后：通知 WorkBuddy 复盘
```

---

## P1 — 测试补全（4 项）

---

### T5: 补 `internal/docker` 测试

**目标**：覆盖率从 9.6% 提升至 ≥30%  
**文件**：`internal/docker/*_test.go`  
**工作量**：⭐⭐⭐

#### 现有测试文件
```bash
internal/docker/mirror_test.go  # 已有部分测试
```

#### 需要覆盖的函数

| 函数 | 文件 | 测试策略 |
|------|------|---------|
| `NewClient` | `client.go` | 空字符串参数走默认路径 |
| `IsAvailable` | `client.go` | mock docker ps 调用 |
| `ContainerList` / `FindContainerByName` | `client.go` | mock 命令输出 |
| `ComposeUp` / `ComposeDown` | `compose.go` | SafeProjectDir 校验 + 命令构造 |
| `TestMirror` | `mirror.go` | 用 test server 模拟 HTTP HEAD |
| `ResolveShortName` | `mirror.go` | 6 个内置源 + 自定义 URL |
| `GetCurrentMirrors` | `mirror.go` | 读 daemon.json 场景 |

#### 关键提示

- `ResolveShortName` 很简单，先写它的测试（覆盖率高、容易）
- `ComposeUp` 调用了 `SafeProjectDir`，可直接测目录校验逻辑
- `TestMirror` 需要启动一个本地 test HTTP server（`httptest.NewServer`）
- 不要 mock Docker daemon——测试命令构造和路径校验就够了

---

### T6: 补 `internal/selfupdate` 测试

**目标**：覆盖率从 0% 提升至 ≥50%  
**文件**：`internal/selfupdate/*_test.go`  
**工作量**：⭐⭐

#### 需要覆盖的函数

| 函数 | 测试策略 |
|------|---------|
| `CheckLatest` | 用 `httptest.NewServer` mock GitHub API 返回 |
| `CompareVersions` | 正常 semver + pre-release + 异常版本 |
| `parseSemver` | 合法格式 + 无效格式 + pre-release |
| `resolveAssetName` | 每种 OS/ARCH 组合 |
| `copyFile` | 正常复制 + 源文件不存在 |

#### 关键提示

- `resolveAssetName` 最简单，先写
- `parseSemver` / `CompareVersions` 也没网络依赖，直接测
- `CheckLatest` 比较复杂，需要 mock HTTP server，可以最后写
- 不要测 `Run` 和 `downloadAsset`（涉及真实网络和文件操作）

---

### T7: 补 `internal/security` 测试

**目标**：覆盖率从 21.6% 提升至 ≥50%  
**文件**：`internal/security/security_test.go`  
**工作量**：⭐

#### 现有测试
```bash
# 已有 MaskSecret 测试（100%覆盖）
```

#### 需要覆盖

| 函数 | 测试策略 |
|------|---------|
| `FixConfigPerm` | 正常文件 chmod、不存在的文件（返回 nil）、Windows 上 no-op |
| `CheckConfigPerm` | 权限过宽检测 |
| `CheckDockerGroup` | 只测函数调用不崩溃（需要 root 的部分跳过） |

#### 关键提示

- `FixConfigPerm` 很简单：创建临时文件 → 改权限 → 验证
- `CheckDockerGroup` 在非 Linux 环境可能返回 error，测试中标记为 `t.Skip` 即可

---

### T8: 补 `internal/cli` 测试

**目标**：覆盖率从 13.2% 提升至 ≥25%  
**文件**：`internal/cli/*_test.go`  
**工作量**：⭐⭐⭐

#### 现有测试
- 已有命令注册测试（VerifyCommandRegistration）
- 需增加具体命令的业务逻辑测试

#### 需要覆盖

| 命令 | 测试策略 |
|------|---------|
| `config chmod` | 调用 `security.FixConfigPerm` 的命令构造 |
| `mirror test` | 参数解析 + 并发调用 `ConcurrentMirrorTest` |
| `instance add/list/switch/remove` | Store CRUD 的 CLI 参数组装 |
| `doctor --verbose` | VerboseCheck 结构体构造 |

#### 关键提示

- Cobra 命令的测试模式：`cmd.SetArgs([]string{"...", "..."})` + `cmd.Execute()`
- `config chmod` 最简单——参数解析后调用 FixConfigPerm
- 不要 mock Docker 调用——只测命令层的参数解析流程

---

## P2 — 功能完善（4 项）

---

### T9: Config Validate() 方法

**文件**：`internal/core/config.go`  
**工作量**：⭐

#### 实现

在 `NewAPIConfig` 上新增 `Validate() error`：

```go
func (c *NewAPIConfig) Validate() error {
    if c.Port <= 0 || c.Port > 65535 {
        return apperr.New(apperr.CodeConfigLoad, "port 必须在 1-65535 之间", "", nil)
    }
    if c.MaxBackups < 0 {
        return apperr.New(apperr.CodeConfigLoad, "max_backups 必须 >= 0", "", nil)
    }
    if c.HealthTimeout < 0 {
        return apperr.New(apperr.CodeConfigLoad, "health_timeout 必须 >= 0", "", nil)
    }
    return nil
}
```

然后在 `LoadConfig` 末尾调用：
```go
if err := cfg.NewAPI.Validate(); err != nil {
    return nil, err
}
```

#### 验证
- `internal/apperr` 的 CodeConfigLoad 错误码已有建议文案
- 写测试：Port=0 报错、Port=65536 报错、正常配置通过
- `go test ./internal/core/...` 确认不影响现有测试

---

### T10: audit list 轮转文件合并

**文件**：`internal/audit/audit.go`  
**工作量**：⭐⭐

#### 现状
`List()` 方法用 ring buffer 读主 `audit.log`，但轮转文件（`audit.log.1`、`audit.log.2`）被忽略。

#### 修改

在 `List()` 中读主文件后，追加读取 `.1`、`.2`、`.3`、`.4`、`.5` 文件（与 `rotate()` 逻辑对齐）：

```go
// Read rotated files (most recent first: .1 > .2 > ... > .5)
for i := 1; i <= 5; i++ {
    rotatedPath := r.path + fmt.Sprintf(".%d", i)
    file, err := os.Open(rotatedPath)
    if err != nil {
        continue // file doesn't exist, skip
    }
    defer file.Close()
    // decode and append entries
}
```

注意：当 `opt.Last > 0` 时，需要跨主文件+所有轮转文件做 ring buffer。

#### 验证
- 创建测试用的 audit.log + audit.log.1 + audit.log.2
- 验证 `List({Last: 5})` 返回跨文件的最新 5 条
- 验证旋转文件名格式与 `rotate()` 一致

---

### T11: doctor --verbose --json 统一序列化

**文件**：`internal/cli/doctor.go`  
**工作量**：⭐

#### 现状
`printDoctorJSON` 已改用 `json.MarshalIndent`。但 `printDoctorTable` 中 verbose 输出用手拼。

#### 修改
在 `printDoctorTable` 中，verbose 模式时也输出 FilePath/Command/Expected/Actual 字段（当前已有 RawOutput）。

#### 验证
- `go test ./internal/cli/...` 通过
- 手动验证：`newapi-tools doctor --verbose` 输出格式正常

---

### T12: gendocs pre-commit hook

**文件**：新增 `hooks/pre-commit`  
**工作量**：⭐

#### 实现

```bash
#!/bin/sh
# pre-commit hook: regenerate error code docs if apperr changed
if git diff --cached --name-only | grep -q "internal/apperr/apperr.go"; then
    echo "Regenerating error code documentation..."
    go run cmd/gendocs/main.go > docs/errors.md
    git add docs/errors.md
fi
```

#### 安装

```bash
# 需要先安装 hook
cp hooks/pre-commit .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```

#### 验证
- 修改 `internal/apperr/apperr.go`（加一个新错误码）
- `git add` 后运行 `git commit`，确认自动触发 gendocs
- 确认 `docs/errors.md` 自动更新

---

## 执行顺序建议

```
T5 (docker test)  ← 最影响覆盖率门禁
  ↓
T7 (security test) ← 最快完成
  ↓
T9 (Config Validate) ← 简单功能
  ↓
T6 (selfupdate test) ← 较复杂但重要
  ↓
T10 (audit rotated) ← 中等难度
  ↓
T12 (pre-commit hook) ← 快速收尾
```

每完成一项：`git add -A && git commit -m "类别: 任务描述" && git push origin handoff`

完成后通知 WorkBuddy 进行 Code Review。
