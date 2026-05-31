<div align="right">
  <a href="README.md">🇨🇳 中文</a> | <a href="README_EN.md">🇬🇧 English</a>
</div>

# NewAPI Tools V3.4

Docker 管理平台，用于部署和管理 [new-api](https://github.com/Calcium-Ion/new-api) 服务。Go 重写。

---

## 快速开始

```bash
# 构建
make build

# 安装 new-api
./dist/newapi-tools install

# 查看状态
./dist/newapi-tools status

# 运行诊断
./dist/newapi-tools doctor

# 查看/编辑配置
./dist/newapi-tools config
./dist/newapi-tools config set newapi.port 8080

# 配置镜像加速（中国大陆）
./dist/newapi-tools mirror add tuna

# 检查 CLI 更新
./dist/newapi-tools update --check
```

---

## 命令列表

| 命令 | 说明 |
|------|------|
| `install` | 部署 new-api（支持 `--interactive` 交互式向导） |
| `status` / `ls` | 查看容器状态。`--json` JSON输出，`--instance <name>` 指定实例 |
| `backup` | 备份配置、MySQL 数据、数据目录到 `.tar.gz` |
| `restore` | 从备份恢复。`--file latest` 自动选最新备份 |
| `update` | 更新容器镜像；`--self` 自更新 CLI 本身；`--check` 仅检查版本 |
| `doctor` | 12 项诊断检查（Docker/容器/端口/磁盘/配置/权限）。`--fix` 自动修复，`--verbose` 详细输出 |
| `config` | 配置管理。子命令：`set`、`init`（交互向导）、`chmod`（修复文件权限） |
| `mirror` | Docker 镜像加速管理。`add/remove/list/apply/test/reset/builtin` |
| `instance` | 多实例管理。`add/list/switch/remove` |
| `audit` | 审计日志查询。`list --last N`、`--cmd`、`--since`、`--json` |
| `version` | 打印版本和构建信息 |

### 常用示例

```bash
# 安装
newapi-tools install --port 8080
newapi-tools install --domain newapi.example.com --mirror tuna

# 备份与恢复
newapi-tools backup
newapi-tools restore --file latest --force

# 自更新
newapi-tools update --check
newapi-tools update --self

# 诊断
newapi-tools doctor --json
newapi-tools doctor --fix
newapi-tools doctor --verbose

# 配置
newapi-tools config
newapi-tools config set newapi.port 8080
newapi-tools config init
newapi-tools config chmod

# 审计
newapi-tools audit list --last 10
newapi-tools audit list --cmd install

# 多实例
newapi-tools instance add prod --port 8080
newapi-tools instance list
newapi-tools instance switch prod
newapi-tools --instance prod status

# 镜像加速
newapi-tools mirror builtin
newapi-tools mirror test tuna aliyun
newapi-tools mirror add tuna
newapi-tools mirror list
newapi-tools mirror apply
```

内置镜像源：`tuna`（清华）、`aliyun`（阿里云）、`ustc`（中科大）、`163`（网易）、`azure`（微软）、`daocloud`（道客）

---

## 项目结构

```
newapi-tools/
├── cmd/newapi/          # 入口 main.go
├── cmd/gendocs/         # 错误码文档生成器
├── internal/
│   ├── apperr/          # 结构化错误码与建议
│   ├── audit/           # 审计日志（JSON Lines + 环形缓冲查询）
│   ├── cli/             # Cobra 命令（12 个命令）
│   ├── core/            # 配置加载、版本常量
│   ├── docker/          # Docker CLI 封装、Compose 操作、镜像管理
│   ├── i18n/            # 国际化（zh-CN / en）
│   ├── instance/        # 多实例元数据存储
│   ├── osutil/          # 操作系统适配（Debian/RHEL/Fedora/Arch）
│   ├── plugin/          # 插件系统（ShellPlugin/GoPlugin）
│   ├── registry/        # 命令注册表
│   ├── security/        # 安全权限检查与修复
│   ├── selfupdate/      # CLI 自更新（GitHub Releases）
│   └── ui/              # 终端输出工具（表格/进度条/日志/错误）
├── plugins/newapi/      # newapi Shell 插件
├── configs/             # 默认配置文件
├── docs/                # MkDocs 文档站
├── .github/workflows/   # CI + 发布 + 文档自动化
```

---

## 配置文件

默认路径：`~/.config/newapi-tools/newapi-tools.yml`

```yaml
newapi:
  home: /opt/newapi          # 安装目录
  port: 3000                 # 映射端口
  docker_image: calciumion/new-api:latest
  backup_dir: /opt/newapi/backups
  domain: ""                 # 自定义域名（可选）
  health_timeout: 120        # 健康检查超时（秒）
  max_backups: 10            # 最大备份保留数

docker:
  compose_cmd: "docker compose"

log:
  level: info                # debug | info | warn | error
  format: text               # text | json

instance:
  active: ""                 # 当前活跃实例名
```

也可通过 CLI 参数覆盖：
```bash
newapi-tools config set newapi.port 8080
newapi-tools config init
newapi-tools --log-level debug status
newapi-tools --config /path/to/config.yml status
```

---

## 开发

```bash
make build    # 编译 → dist/newapi-tools
make test     # 运行测试（16 个包）
make run      # 本地运行
make lint     # golangci-lint
make vet      # go vet
make coverage # 测试覆盖率报告
make docs     # 生成错误码文档
```

### 运行测试

```bash
go test ./...                  # 全部包
go test ./internal/cli/ -v     # 仅 CLI 包
go test ./... -count=1         # 不缓存
```

### 交叉编译

```bash
GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o dist/newapi-tools-linux-amd64 ./cmd/newapi/
```

---

## 插件系统

启动时自动扫描 `./plugins/` 目录：

- **Shell 插件**：目录含 `metadata.yml` + `scripts/`，通过 bash 执行
- **Go 插件**：目录含 `plugin.go`，编译时注册

内置 `plugins/newapi/` 插件提供 7 个 Shell 脚本命令，`install` 和 `status` 已升级为原生 Go 实现。

---

## 版本历史

| 版本 | 亮点 |
|------|------|
| V3.4.0 | 审计报告修复（跨分区回滚、JSON 统一序列化、环形缓冲、CI 文档检查） |
| V3.3.5 | 修复 resolveAssetName semver 对比、doctor 错误码 X001、审计双日志 |
| V3.3.4 | 安全审计：路径遍历防护、权限锁定、Shell 注入防护、竞态条件修复 |
| V3.3.0 | Domain/MaxBackups 配置扩展、Check 结构体重构、自动回滚 |
| V3.2.0 | 自更新（`--check`/`--self`）、审计日志查询、多实例管理、错误码文档自动生成 |
| V3.1.0 | 国际化框架、结构化错误处理、审计日志、安全检查、交互式安装向导 |
| V3.0.0 | 镜像管理（6 个内置 CN 源）、83 个测试 |
| V3.0-rc | 9 个 Go 原生命令、78 个测试、Git v2/main 分支分离 |
| V3.0-a3 | install + status 原生 Go、newapi Shell 插件、52 个测试 |
| V3.0-a2 | 插件系统、注册表、Docker 封装、OS 适配 |
| V3.0-a1 | Go 项目骨架、Cobra CLI、Viper 配置、slog |
| V2.4 | Shell V2 插件系统（已归档到 `v2` 分支） |

---

## 许可证

MIT

---

<h1 align="center">English</h1>

<p align="center"><a href="README_EN.md">📖 Read the full English version →</a></p>

**NewAPI Tools V3.4** — Docker management platform for [new-api](https://github.com/Calcium-Ion/new-api), rewritten in Go.

| Feature | Description |
|---------|-------------|
| `install` | Deploy new-api with Docker Compose (interactive wizard) |
| `status` / `ls` | Show container status (`--json`, `--instance`) |
| `backup` / `restore` | Backup & restore with MySQL dump |
| `update` | Docker image update & CLI self-update (`--self`/`--check`) |
| `doctor` | 12 diagnostic checks with auto-fix (`--fix`/`--verbose`) |
| `config` | Config management (`set`/`init`/`chmod`) |
| `mirror` | Docker registry mirror management |
| `instance` | Multi-instance management |
| `audit` | Command audit log query |
| `version` | Print version info |

For complete English documentation, see [README_EN.md](README_EN.md).
