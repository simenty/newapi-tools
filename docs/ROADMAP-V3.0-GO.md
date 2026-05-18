# NewAPI Tools V3.0 — Go 重写方案

> 版本：v1.0 | 日期：2026-05-17 | 状态：待确认

---

## 一、核心决策

| 项目 | 决策 | 理由 |
|------|------|------|
| 插件机制 | Plan B → Plan A 渐进迁移 | 先跑起来再优化，Shell 代码可复用 |
| 网关范围 | 只做 newapi，砍掉 one-api / sub2api | 聚焦核心，减少一半工作量 |
| Go 布局 | 标准 Go 项目布局 | 社区惯例，降低上手门槛 |
| Docker 交互 | Docker SDK 为主 + exec.Command 兜底 | Go core 类型安全，compose 场景用 CLI 兜底 |
| V2.4 Shell | 归档到 v2 分支，V3.0 干净起步 | 不背历史包袱 |

---

## 二、渐进式迁移：Plan B → Plan A

### 2.1 核心思路

```
阶段1 (Plan B)                    阶段2 (过渡)                    阶段3 (Plan A)
┌──────────────────┐           ┌──────────────────┐           ┌──────────────────┐
│  Go Core         │           │  Go Core         │           │  Go Core         │
│  ├ CLI (Cobra)   │           │  ├ CLI (Cobra)   │           │  ├ CLI (Cobra)   │
│  ├ Config        │           │  ├ Config        │           │  ├ Config        │
│  ├ Registry      │           │  ├ Registry      │           │  ├ Registry      │
│  ├ Plugin Loader │           │  ├ Plugin Loader │           │  ├ Plugin Loader │
│  └ OS Adapter    │           │  └ OS Adapter    │           │  └ OS Adapter    │
│                  │           │                  │           │                  │
│  Shell Plugin    │           │  Mixed Plugins   │           │  Go Plugin       │
│  └ newapi/*.sh   │──迁移──▶  │  ├ newapi (部分Go)│──迁移──▶  │  └ newapi (全Go)  │
│                  │           │  └ newapi (部分sh)│           │                  │
└──────────────────┘           └──────────────────┘           └──────────────────┘
  exec.Command调.sh               两种插件并存                    纯 Go 实现
```

### 2.2 插件接口设计（关键）

Go 侧定义统一 Plugin 接口，Shell 和 Go 插件都实现它：

```go
// internal/plugin/plugin.go
type Plugin interface {
    // 元数据
    Name() string
    Version() string
    Commands() []Command

    // 生命周期
    Init(ctx Context) error
    Execute(cmd string, args []string) error
    Shutdown() error
}

type Command struct {
    Name        string
    Description string
    Usage       string
}
```

**两种实现**：

```go
// ShellPlugin — 调 .sh 脚本（Plan B 阶段）
type ShellPlugin struct {
    dir     string        // plugins/newapi/scripts/
    meta    Metadata      // 解析 metadata.yml
}
func (p *ShellPlugin) Execute(cmd string, args []string) error {
    return exec.Command("bash", p.scriptPath(cmd), args...).Run()
}

// GoPlugin — 原生 Go（Plan A 阶段）
type GoPlugin struct {
    handler map[string]HandlerFunc
}
func (p *GoPlugin) Execute(cmd string, args []string) error {
    return p.handler[cmd](args)
}
```

**迁移方式**：每个命令独立迁移，不需要一次全搬：

```
newapi 插件迁移示例：
  install  → 先保留 .sh（ShellPlugin） → 重写为 Go（GoPlugin）
  backup   → 先保留 .sh               → 重写为 Go
  restore  → 先保留 .sh               → 重写为 Go
  ...每迁移一个命令，删除对应 .sh，插件从 ShellPlugin 切到 GoPlugin
```

### 2.3 插件发现机制

```
plugins/
  newapi/
    metadata.yml          # 必需，声明插件信息
    scripts/              # Shell 脚本（Plan B 阶段存在）
      install.sh
      backup.sh
      ...
    plugin.go             # Go 原生实现（迁移后出现）
```

发现规则：
1. 有 `plugin.go` → 加载为 GoPlugin
2. 无 `plugin.go` 但有 `scripts/` → 加载为 ShellPlugin
3. 两者都有 → GoPlugin 优先（兼容期）

---

## 三、项目结构（标准 Go 布局）

```
newapi-tools/
├── cmd/
│   └── newapi/                    # CLI 入口
│       └── main.go
├── internal/
│   ├── cli/                       # Cobra 命令定义
│   │   ├── root.go                # newapi-tools
│   │   ├── install.go             # install 子命令
│   │   ├── backup.go              # backup 子命令
│   │   ├── restore.go             # restore 子命令
│   │   ├── update.go              # update 子命令
│   │   ├── status.go              # status 子命令
│   │   └── doctor.go              # doctor 子命令
│   ├── core/
│   │   ├── config.go              # 配置管理
│   │   ├── registry.go            # 命令注册表
│   │   └── context.go             # 运行上下文
│   ├── docker/
│   │   ├── client.go              # Docker SDK 封装
│   │   ├── container.go           # 容器操作
│   │   └── compose.go             # compose 操作（exec 兜底）
│   ├── osutil/
│   │   ├── adapter.go             # OS 适配器接口
│   │   ├── debian.go             # Debian/Ubuntu
│   │   ├── rhel.go               # CentOS/Rocky/Alma
│   │   ├── fedora.go             # Fedora
│   │   └── arch.go              # Arch Linux
│   ├── plugin/
│   │   ├── plugin.go              # Plugin 接口
│   │   ├── loader.go              # 插件发现+加载
│   │   ├── shell.go              # ShellPlugin 适配器
│   │   └── goplugin.go           # GoPlugin 基础
│   └── ui/
│       ├── output.go              # 输出格式化
│       ├── progress.go            # 进度条
│       └── table.go               # 表格输出
├── plugins/
│   └── newapi/                   # newapi 插件
│       ├── metadata.yml
│       └── scripts/              # Shell 脚本（Plan B 阶段）
│           ├── install.sh
│           ├── backup.sh
│           ├── restore.sh
│           ├── update.sh
│           ├── config.sh
│           └── doctor.sh
├── configs/
│   └── newapi-tools.yml          # 默认配置
├── scripts/
│   ├── build.sh                  # 构建脚本
│   └── release.sh                # 发布脚本
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 四、Docker 交互策略

### 4.1 分层设计

```go
// internal/docker/client.go — Docker SDK 封装
type Client struct {
    client *docker.Client  // github.com/docker/docker/client
}

// 核心操作 — 用 Docker SDK
func (c *Client) ContainerList() ([]Container, error)
func (c *Client) ContainerInspect(id string) (Container, error)
func (c *Client) ContainerStart(id string) error
func (c *Client) ContainerStop(id string, timeout time.Duration) error
func (c *Client) ContainerRemove(id string) error
func (c *Client) ImagePull(ctx context.Context, ref string) error
func (c *Client) VolumeList() ([]Volume, error)
func (c *Client) NetworkList() ([]Network, error)

// internal/docker/compose.go — compose 操作用 exec 兜底
func ComposeUp(projectDir string) error {
    return exec.Command("docker", "compose", "-f", composeFile, "up", "-d").Run()
}
func ComposeDown(projectDir string) error {
func ComposePull(projectDir string) error {
```

### 4.2 使用原则

| 场景 | 方式 | 原因 |
|------|------|------|
| 容器 CRUD | Docker SDK | 类型安全，错误结构化 |
| 镜像拉取 | Docker SDK | 进度回调，取消支持 |
| 卷/网络管理 | Docker SDK | 简单查询 |
| docker compose | exec.Command | SDK 不支持 compose，CLI 是唯一可靠方式 |
| docker stats 监控 | Docker SDK | 流式 API，不用解析文本 |
| Shell 插件操作 | 直接 docker CLI | 不经过 Go 层 |

---

## 五、技术选型

| 组件 | 选型 | 版本 | 说明 |
|------|------|------|------|
| 语言 | Go | ≥ 1.22 | 泛型 + 增强 error handling |
| CLI 框架 | Cobra | v1.8+ | Go CLI 事实标准 |
| 配置 | Viper | v1.18+ | YAML/JSON/ENV 全支持 |
| 日志 | slog | 标准库 | Go 1.21+ 内置，零依赖 |
| Docker | docker/docker/client | v27+ | Docker Engine API |
| YAML 解析 | gopkg.in/yaml.v3 | v3 | metadata.yml 解析 |
| 表格输出 | tablewriter | v0.0.5 | 美观表格 |
| 进度条 | progressbar | v3.14 | 镜像拉取等长操作 |
| 交叉编译 | goreleaser | 最新 | 多平台构建发布 |

---

## 六、实施路线图

### 阶段总览

```
V3.0-a1 ──▶ V3.0-a2 ──▶ V3.0-a3 ──▶ V3.0-rc
(骨架)       (核心)       (插件)       (发布)
 5天          7天          7天          5天
──────────────────────────────────────────────
  合计约 24 天（~5 周，含调试）
```

### V3.0-a1：Go 项目骨架 + CLI 框架（5 天）

| 天 | 任务 | 产出 |
|----|------|------|
| 1 | Go 项目初始化，go mod init，标准目录结构 | 项目骨架 |
| 2 | Cobra CLI 框架，root + 6 个子命令骨架 | cmd/ + internal/cli/ |
| 3 | Viper 配置管理，configs/newapi-tools.yml | internal/core/config.go |
| 4 | slog 日志封装，输出格式化 | internal/ui/ |
| 5 | Makefile + 构建脚本 + goreleaser 配置 | Makefile |

**可运行里程碑**：`newapi-tools --version` + `newapi-tools install --help` 能跑

### V3.0-a2：核心层 — Registry + Plugin + Docker + OS（7 天）

| 天 | 任务 | 产出 |
|----|------|------|
| 1 | Plugin 接口设计 + ShellPlugin 适配器 | internal/plugin/ |
| 2 | 插件发现机制 + metadata.yml 解析 | internal/plugin/loader.go |
| 3 | Registry 命令注册表 + 路由 | internal/core/registry.go |
| 4 | Docker SDK 封装 + compose 兜底 | internal/docker/ |
| 5 | OS 适配器（Debian + RHEL） | internal/osutil/ |
| 6 | OS 适配器（Fedora + Arch） | internal/osutil/ |
| 7 | 集成测试：插件加载 → 命令路由 → 执行 | test/ |

**可运行里程碑**：Go 加载 Shell 插件并执行 `newapi-tools doctor`

### V3.0-a3：newapi 插件迁移 + 全功能验证（7 天）

| 天 | 任务 | 产出 |
|----|------|------|
| 1 | 迁移 V2.4 的 7 个 Shell 脚本到 plugins/newapi/scripts/ | Shell 插件就位 |
| 2 | 逐个验证 Shell 插件命令（install/backup/restore） | 功能验证 |
| 3 | 逐个验证 Shell 插件命令（update/config/doctor/status） | 功能验证 |
| 4 | Docker 交互优化：容器状态查询用 SDK 重写 | internal/docker/ 增强 |
| 5 | 前两个命令 Go 化迁移（install + status） | Go 迁移示范 |
| 6 | 端到端测试：全新部署 → 备份 → 还原 → 升级 | 集成测试 |
| 7 | Bug 修复 + 代码审查 + 文档 | 代码收尾 |

**可运行里程碑**：`newapi-tools install` 全流程可用（Go 核心调 Shell 插件）

### V3.0-rc：发布准备（5 天）

| 天 | 任务 | 产出 |
|----|------|------|
| 1 | goreleaser 交叉编译（linux/amd64 + linux/arm64） | 构建产物 |
| 2 | 安装脚本编写（一行 curl 安装） | install.sh |
| 3 | 全平台测试（Debian/Ubuntu/CentOS） | 测试报告 |
| 4 | README + 使用文档 | README.md |
| 5 | GitHub Release 发布 | v3.0.0 |

---

## 七、V2.4 归档计划

```bash
# 1. 创建 v2 分支保留完整历史
git checkout -b v2
git push origin v2

# 2. 主分支干净起步
git checkout main
# 清除所有 Shell 文件，只保留 docs/ 作为参考
git rm -r lib/ modules/ scripts/ test/ tools/
git rm *.sh *.py *.md  # 清理根目录杂物

# 3. 提交 Go 项目骨架
# 新的 .gitignore, go.mod, cmd/, internal/, plugins/, ...
```

归档后 V2.4 代码在 `v2` 分支随时可查，主分支是干净的 Go 项目。

---

## 八、命令设计

```
newapi-tools                          # 显示帮助
newapi-tools install                  # 安装 newapi
newapi-tools backup                  # 备份 newapi
newapi-tools restore                 # 还原 newapi
newapi-tools update                  # 升级 newapi
newapi-tools status                  # 查看状态
newapi-tools doctor                  # 诊断问题
newapi-tools config                  # 配置管理
newapi-tools version                 # 版本信息
```

去掉了 V2.4 的 `--flavor` 参数，因为只做 newapi，不需要多网关路由。

未来如果加新网关：
```
newapi-tools install --gateway one-api   # 预留扩展
```

---

## 九、与 V2.4 的对比

| 维度 | V2.4 (Shell) | V3.0 (Go) |
|------|-------------|-----------|
| 语言 | Bash 4+ | Go 1.22+ |
| 运行方式 | source + 函数调用 | 单二进制 |
| 依赖 | bash, yq, docker CLI, jq | docker CLI (compose) |
| 安装方式 | git clone + chmod | curl 一行 / 包管理器 |
| 跨平台 | 手动 os_adapter.sh | Go 交叉编译 |
| 插件机制 | source .sh + registry 数组 | Plugin 接口 + 自动发现 |
| 配置 | config.sh (shell 变量) | YAML + Viper |
| 错误处理 | set -e + $? 检查 | error wrapping + 结构化 |
| 测试 | 324 个 shell 单元测试 | Go test + 集成测试 |
| 输出 | echo + color codes | slog + tablewriter |
| 多网关 | newapi/one-api/sub2api | 只做 newapi |
| ARM 支持 | 需要手动适配 | goreleaser 交叉编译 |

---

## 十、风险与应对

| 风险 | 影响 | 应对 |
|------|------|------|
| Docker SDK API 版本不兼容 | 容器操作失败 | 版本协商 + 降级到 CLI |
| Shell 脚本在 Go 调用下行为不一致 | 功能回归 | 逐个命令对比测试 |
| compose 场景 exec.Command 失败 | 部署失败 | 保留直接 docker CLI 调用路径 |
| Go 学习曲线 | 开发进度慢 | 聚焦核心，不用花哨特性 |
| 砍 one-api/sub2api 后用户流失 | 用户不满意 | 插件系统预留扩展口，社区可贡献 |

---

*本文档随开发迭代持续更新。*
