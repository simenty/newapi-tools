# 项目交接手册

> **项目**：NewAPI Tools (`github.com/simenty/newapi-tools`)  
> **版本**：v3.4.0  
> **交接时间**：2026-06-01  
> **接手工具**：TRAE AI IDE  

---

## 一、项目概述

面向零基础用户的 **new-api Docker 管理 CLI 工具**。Go 重写（V3），替代 Shell V2 版本。

- **GitHub**：https://github.com/simenty/newapi-tools
- **文档站**：https://simenty.github.io/newapi-tools/
- **Go 版本**：1.23，模块路径 `github.com/simenty/newapi-tools`
- **许可证**：MIT

### 核心定位

一句话：**帮用户用 Docker Compose 一键部署/管理 new-api，不用写 docker 命令。**

---

## 二、目录说明

```
newapi-tools/
├── cmd/
│   ├── newapi/main.go           # 入口
│   └── gendocs/main.go          # 错误码文档生成器（独立工具）
├── internal/
│   ├── apperr/                  # 结构化错误码（18 个 + suggestions）
│   ├── audit/                   # 审计日志：JSON Lines 写入 + 轮转 + 查询（环形缓冲）
│   ├── cli/                     # Cobra 命令（12 个命令）
│   │   ├── install.go           # 安装（7步交互向导 + 自动注册实例）
│   │   ├── status.go            # 状态（ls别名 + --instance + --watch）
│   │   ├── backup.go            # 备份 + MySQL dump
│   │   ├── restore.go           # 恢复（路径注入防护）
│   │   ├── update.go            # 容器更新 + CLI 自更新
│   │   ├── doctor.go            # 诊断（12项 + --fix + --verbose）
│   │   ├── config.go            # 配置（show/set/init/chmod）
│   │   ├── mirror.go            # 镜像管理（并发测速 + 表格输出）
│   │   ├── instance.go          # 多实例管理
│   │   ├── audit_list.go        # 审计日志查询
│   │   ├── version.go           # 版本信息
│   │   └── root.go              # 根命令 + 审计日志记录
│   ├── core/                    # 配置加载（Viper） + 版本常量
│   ├── docker/                  # Docker CLI 封装 + Compose + 镜像管理
│   ├── i18n/                    # 国际化（zh-CN / en）
│   ├── instance/                # 多实例元数据存储（instances.json CRUD）
│   ├── osutil/                  # OS 适配（Debian/RHEL/Fedora/Arch）
│   ├── plugin/                  # 插件系统（Shell / Go 插件）
│   ├── registry/                # 命令注册表
│   ├── security/                # 安全检查 + chmod 修复
│   ├── selfupdate/              # CLI 自更新（GitHub API + SHA256 + 原子替换）
│   └── ui/                      # 终端输出（Table / PrintStep / Logger）
├── plugins/newapi/              # 内置 Shell 插件（7 个 stub 脚本）
├── configs/                     # 默认配置
├── docs/                        # MkDocs 文档站源文件
├── .github/workflows/
│   ├── ci.yml                   # 三平台测试（vet → test → coverage ≥30% → build）
│   ├── release.yml              # tag 发布（5 平台构建 → 自动 release notes）
│   └── docs.yml                 # MkDocs 自动部署 GitHub Pages
├── Makefile                     # 构建/测试/文档命令
├── .golangci.yml                # golangci-lint 配置
├── .gitignore                   # 忽略规则
├── go.mod / go.sum              # Go 依赖
├── README.md                    # 中文 README
└── README_EN.md                 # 英文 README
```

---

## 三、依赖与构建

### 直接依赖（3 个）

| 包 | 版本 | 用途 |
|----|------|------|
| `spf13/cobra` | v1.8.1 | CLI 框架 |
| `spf13/viper` | v1.19.0 | 配置管理 |
| `gopkg.in/yaml.v3` | v3.0.1 | YAML 序列化 |

间接依赖 15 个（cobra 和 viper 的传递依赖），**无 Docker SDK**——所有 Docker 操作通过 `exec.Command` 调用 CLI。

### 开发命令（Makefile）

```bash
make build    # 编译到 dist/newapi-tools
make test     # go test ./...
make vet      # go vet ./...
make lint     # golangci-lint run ./...
make coverage # 生成 coverage.html
make docs     # go run cmd/gendocs/main.go → docs/errors.md
make check    # vet + test
make clean    # 清理 dist/
make run      # go run ./cmd/newapi/
```

### 测试

```bash
# 全量测试
go test ./...

# 指定包
go test ./internal/cli/ -v
go test ./internal/docker/...

# 不缓存
go test ./... -count=1
```

当前 **16 个包全部 PASS**，无失败测试。

### 交叉编译

```bash
# Linux amd64
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
  -ldflags="-s -w -X github.com/simenty/newapi-tools/internal/core.Version=v3.4.0" \
  -o dist/newapi-tools-linux-amd64 ./cmd/newapi/

# Windows
GOOS=windows GOARCH=amd64 go build ...
```

---

## 四、CI/CD 流水线

### CI（`.github/workflows/ci.yml`）

| 触发 | 矩阵 | 步骤 |
|------|------|------|
| push/PR → main | ubuntu/macos/windows × go 1.23 | vet → test → coverage gate (≥30%) → build |
| | 额外 lint job | ubuntu-latest + golangci-lint |
| | 额外 docs-check job | 运行 gendocs 并检查 docs/errors.md 变更 |

### 发布（`.github/workflows/release.yml`）

| 触发 | 流程 |
|------|------|
| push tag `v*` | 验证 tag 格式 → test → 5 平台构建（linux amd64/arm64, darwin amd64/arm64, windows amd64）→ tar.gz/zip → checksums → 自动 release notes → gh release create |

**幂等设计**：先 delete 再 create，支持重跑。

### 文档（`.github/workflows/docs.yml`）

push main（docs/ 或 mkdocs.yml 变更）→ MkDocs gh-deploy → GitHub Pages。

**发布流程**：
```bash
# 1. 更新版本号
# 编辑 internal/core/version.go

# 2. 提交代码
git add -A
git commit -m "feat: xxx"

# 3. 打 tag 发布
git tag v3.4.0
git push origin v3.4.0
# → GitHub Actions 自动构建 Release
```

---

## 五、架构设计要点

### 分层架构

```
cmd/ → 入口
internal/cli/ → 命令层（参数解析 + UI 输出）
internal/ 其他包 → 业务逻辑层
```

设计原则：
- **命令层不做业务逻辑**，只做参数解析和 UI 输出
- **internal 包通过函数/结构体调用**，不用接口抽象（项目规模小，保持简单）
- **新增包不改已有包公开 API**，仅扩展

### 关键数据流

```
用户输入 → cobra.RunE → 参数解析 → 业务函数 → 结构化错误(apperr) → UI输出
                                  ↓
                              audit.Log() ← 审计日志（自动记录每个命令）
```

### 错误码体系（`internal/apperr`）

```
规范：[A][NNN]
A = 类别: D=Docker, C=Config, I=Install/Instance, S=System, M=Mirror, B=Backup, P=Plugin, U=Update, X=Doctor

使用方式：
- apperr.New(code, msg, suggestion, cause) → 创建新错误
- apperr.Wrap(code, suggestion, cause) → 包装已有错误
- apperr.GetSuggestion(code) → 获取修复建议
```

### 多实例隔离

- 实例元数据：`~/.config/newapi-tools/instances.json`
- Compose 隔离：`COMPOSE_PROJECT_NAME=newapi-<name>`
- 实例目录：`/opt/newapi-<name>`（默认）

---

## 六、当前版本状态（v3.4.0）

### 已完成功能

| 功能 | 命令 | 状态 |
|------|------|------|
| 安装部署 | `install` | ✅ 7步交互向导 + 自动注册实例 |
| 状态查看 | `status` / `ls` | ✅ 表格/JSON双输出 + --watch + --instance |
| 备份恢复 | `backup` / `restore` | ✅ tar.gz + MySQL dump + 路径注入防护 |
| 容器更新 | `update` | ✅ Docker image + 自更新(--self) + 版本检查(--check) |
| 故障诊断 | `doctor` | ✅ 12项检查 + --fix + --verbose |
| 配置管理 | `config` | ✅ show/set/init/chmod |
| 镜像加速 | `mirror` | ✅ 6内置源 + 并发测速 + 自动选择 |
| 多实例管理 | `instance` | ✅ add/list/switch/remove |
| 审计日志 | `audit` | ✅ JSON Lines + 轮转 + 查询过滤 |
| CLI自更新 | `update --self` | ✅ GitHub API + SHA256 + 原子替换 + 跨分区回滚 |
| 国际化 | 全局 | ✅ zh-CN / en |
| CI/CD | GitHub Actions | ✅ 三平台测试 + 自动发布 + 文档部署 |

### 未完成/可改进

| 项目 | 优先级 | 说明 |
|------|--------|------|
| Config `Validate()` | P3 | 无前置校验，Port=0 或 MaxBackups=-99 可静默传入 |
| `doctor --verbose --json` 序列化 | P3 | 已改用 json.Marshal，但 printDoctorJSON 整体可进一步统一 |
| Docker SDK 集成 | P4 | 当前 exec.Command 调 CLI 够用，长远可考虑 |
| `audit list` 环形缓冲对齐 rotated 文件 | P3 | 当前 ring buffer 只读主文件，未合并 .1/.2 等轮转文件 |
| 新增错误码时自动触发 gendocs | P3 | 当前 gendocs 需手动执行 `make docs`，可考虑 pre-commit hook |

### 已知问题（无未修复 P0/P1）

v3.3.4 + v3.3.5 的安全审计和代码审计已修复全部 P0/P1 问题。当前无已知严重 Bug。

---

## 七、开发规范

### 代码风格
- **Go 标准项目布局**：`cmd/` + `internal/`
- **错误处理**：走 `apperr` 统一体系
- **日志**：走 `slog`（`ui.L()` 统一入口）
- **表格输出**：走 `ui.Table`（`NewTable()` → `AddRow()` → `Render()`）
- **步骤进度**：走 `ui.PrintStep(step, total, message)`
- **配置**：走 Viper，默认 `~/.config/newapi-tools/newapi-tools.yml`

### 提交规范

```
前缀：feat: / fix: / ci: / docs: / refactor: / chore: / test:
发布 tag：vX.Y.Z（语义版本）
Changelog：自动从 git log 生成
```

### 测试要求
- 覆盖率门禁：**≥30%**（CI 强制执行）
- 测试框架：Go 标准 `testing` 包
- CI 会运行 `golangci-lint`（errcheck/govet/staticcheck/unused/gosimple/ineffassign/gocyclo/gosec）

---

## 八、TRAE 接手指引

### 1. 首次打开

```bash
# clone 仓库
git clone https://github.com/simenty/newapi-tools.git
cd newapi-tools

# 切接手分支
git checkout handoff
```

在 TRAE 中打开 `newapi-tools/` 目录，启用源码管理（Source Control）。

### 2. 先跑通环境

```bash
# 编译
make build

# 运行
./dist/newapi-tools version

# 测试
make test
make vet
make lint
```

### 3. 推荐工作流程

```
WorkBuddy (规划)           TRAE (实现)
    │                        │
    ├─ 拆需求/整理上下文      │
    ├─ 规划任务清单           │
    └─ 输出任务描述 ─────────→ ├─ 切换到 handoff 分支
                              ├─ 实现一个小任务
                              ├─ go build + go test
                              ├─ git commit
                              ├─ git push
                              └─ 通知 WorkBuddy 复盘
```

### 4. 每个任务的最小单元

```
改动一处 → 编译 → 测试 → 提交 → 推送
```

不要跨任务修改。先修 Bug，再补测试，再加功能。

---

## 九、联系方式

- **GitHub Issues**：https://github.com/simenty/newapi-tools/issues
- **上游项目**：https://github.com/Calcium-Ion/new-api
- **项目维护者**：simenty（GitHub）
