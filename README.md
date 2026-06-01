<div align="center">
  <h1>🛠 NewAPI Tools</h1>
  <p>Docker 管理平台 · 用于部署和管理 <a href="https://github.com/Calcium-Ion/new-api">new-api</a> 服务</p>

  <p>
    <a href="https://github.com/simenty/newapi-tools/blob/main/LICENSE">
      <img src="https://img.shields.io/github/license/simenty/newapi-tools?style=flat-square" alt="License">
    </a>
    <a href="https://github.com/simenty/newapi-tools/releases">
      <img src="https://img.shields.io/github/v/release/simenty/newapi-tools?style=flat-square" alt="Release">
    </a>
    <a href="https://github.com/simenty/newapi-tools/actions/workflows/ci.yml">
      <img src="https://img.shields.io/github/actions/workflow/status/simenty/newapi-tools/ci.yml?style=flat-square&label=CI" alt="CI">
    </a>
    <a href="https://github.com/simenty/newapi-tools/actions/workflows/release.yml">
      <img src="https://img.shields.io/github/actions/workflow/status/simenty/newapi-tools/release.yml?style=flat-square&label=Release" alt="Release">
    </a>
    <a href="https://goreportcard.com/report/github.com/simenty/newapi-tools">
      <img src="https://goreportcard.com/badge/github.com/simenty/newapi-tools?style=flat-square" alt="Go Report Card">
    </a>
    <img src="https://img.shields.io/github/go-mod/go-version/simenty/newapi-tools?style=flat-square" alt="Go Version">
  </p>

  <p>
    <a href="README.md">🇨🇳 中文</a> · <a href="README_EN.md">🇬🇧 English</a>
  </p>
</div>

<br>

<p align="center">
  <a href="#-快速开始">🚀 快速开始</a> ·
  <a href="#-命令列表">📋 命令列表</a> ·
  <a href="#-项目结构">📁 项目结构</a> ·
  <a href="#-配置文件">⚙️ 配置文件</a> ·
  <a href="#-开发">🔧 开发</a> ·
  <a href="#-免责声明">⚠️ 免责声明</a>
</p>

---

## 🚀 快速开始

```bash
# 构建
make build

# 安装 new-api
./dist/newapi-tools install

# 查看状态
./dist/newapi-tools status

# 运行诊断
./dist/newapi-tools doctor

# 配置镜像加速（中国大陆用户）
./dist/newapi-tools mirror add tuna

# 检查 CLI 更新
./dist/newapi-tools update --check
```

---

## 📋 命令列表

| 命令 | 说明 |
|------|------|
| `install` | 部署 new-api，支持 `--interactive` 交互式向导 |
| `status` / `ls` | 查看容器状态。`--json` 输出 JSON，`--instance <name>` 指定实例 |
| `backup` | 备份配置、MySQL 数据、数据目录到 `.tar.gz` |
| `restore` | 从备份恢复。`--file latest` 自动选最新备份 |
| `update` | 更新容器镜像；`--self` 自更新 CLI 本身；`--check` 仅检查版本 |
| `doctor` | 12 项诊断检查（Docker/容器/端口/磁盘/配置/权限），`--fix` 自动修复 |
| `config` | 配置管理：`set`、`init`（向导）、`chmod`（权限修复） |
| `mirror` | Docker 镜像加速管理：添加/删除/测速/应用 |
| `instance` | 多实例管理：添加/列表/切换/删除 |
| `audit` | 审计日志查询：`list --last N`、`--cmd`、`--since`、`--json` |
| `version` | 打印版本和构建信息 |

<details>
<summary><b>📖 常用示例（点击展开）</b></summary>

```bash
# 安装
newapi-tools install --port 8080
newapi-tools install --domain newapi.example.com --mirror tuna
newapi-tools install --interactive

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
newapi-tools mirror apply
```
</details>

> **内置镜像源**：`tuna`（清华）、`aliyun`（阿里云）、`ustc`（中科大）、`163`（网易）、`azure`（微软）、`daocloud`（道客）

---

## 📁 项目结构

```
newapi-tools/
├── cmd/
│   ├── newapi/          # 入口 main.go
│   └── gendocs/         # 错误码文档生成器
├── internal/
│   ├── apperr/          # 结构化错误码与建议
│   ├── audit/           # 审计日志（JSON Lines + 环形缓冲查询）
│   ├── cli/             # Cobra 命令（12 个命令）
│   ├── core/            # 配置加载、版本常量
│   ├── docker/          # Docker CLI 封装、Compose、镜像管理
│   ├── i18n/            # 国际化（zh-CN / en）
│   ├── instance/        # 多实例元数据存储
│   ├── osutil/          # 操作系统适配
│   ├── plugin/          # 插件系统
│   ├── registry/        # 命令注册表
│   ├── security/        # 安全权限检查与修复
│   ├── selfupdate/      # CLI 自更新（GitHub Releases）
│   └── ui/              # 终端输出工具（表格/进度/日志/错误）
├── plugins/newapi/      # newapi Shell 插件
├── configs/             # 默认配置文件
├── docs/                # MkDocs 文档站
└── .github/workflows/   # CI + 发布 + 文档自动化
```

---

## ⚙️ 配置文件

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

## 🔧 开发

```bash
make build    # 编译 → dist/newapi-tools
make test     # 运行测试（16 个包）
make run      # 本地运行
make lint     # golangci-lint
make vet      # go vet ./...
make coverage # 测试覆盖率报告
make docs     # 生成错误码文档
make check    # vet + test
```

<details>
<summary><b>📋 测试 & 交叉编译（点击展开）</b></summary>

```bash
# 运行测试
go test ./...                  # 全部包
go test ./internal/cli/ -v     # 仅 CLI 包
go test ./... -count=1         # 不缓存

# 交叉编译 Linux
GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o dist/newapi-tools-linux-amd64 ./cmd/newapi/
```
</details>

---

## 🔌 插件系统

启动时自动扫描 `./plugins/` 目录：

- **Shell 插件**：目录含 `metadata.yml` + `scripts/`，通过 bash 执行
- **Go 插件**：目录含 `plugin.go`，编译时注册

内置 `plugins/newapi/` 插件提供 7 个 Shell 脚本命令，`install` 和 `status` 已升级为原生 Go 实现。

---

## ⚠️ 免责声明

这是一个会真实调用 Docker CLI 命令并修改宿主机容器配置的运维工具。如果你在没有经过验证的备份、没有维护窗口、没有明确回滚方案的前提下执行高风险动作（如 `install --force`、`restore`、`update` 等），可能导致容器失联、业务中断、配置损坏或不可逆的数据损失。**所有数据损失、恢复成本与第三方恢复费用均由实际操作人自行承担。**

执行 `backup`、`restore`、`update`、`doctor --fix` 等命令前，建议先完整阅读 [docs/](docs/) 目录下的相关文档。

---

## 📄 许可证

[MIT](LICENSE)

---

<p align="center">
  <a href="#-快速开始">🚀 快速开始</a> ·
  <a href="#-命令列表">📋 命令列表</a> ·
  <a href="#-项目结构">📁 项目结构</a> ·
  <a href="#-配置文件">⚙️ 配置文件</a> ·
  <a href="#-开发">🔧 开发</a> ·
  <a href="#-免责声明">⚠️ 免责声明</a>
</p>

<p align="center">
  <a href="README_EN.md">🇬🇧 English Version →</a>
</p>
