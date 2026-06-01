<div align="center">
  <h1>🛠 NewAPI Tools</h1>
  <p>Docker management platform for <a href="https://github.com/Calcium-Ion/new-api">new-api</a></p>

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
  <a href="#-quick-start">🚀 Quick Start</a> ·
  <a href="#-commands">📋 Commands</a> ·
  <a href="#-project-structure">📁 Structure</a> ·
  <a href="#-configuration">⚙️ Config</a> ·
  <a href="#-development">🔧 Dev</a> ·
  <a href="#-disclaimer">⚠️ Disclaimer</a>
</p>

---

## 🚀 Quick Start

```bash
# Build
make build

# Install new-api
./dist/newapi-tools install

# Check status
./dist/newapi-tools status

# Run diagnostics
./dist/newapi-tools doctor

# Speed up image pulls (China mainland)
./dist/newapi-tools mirror add tuna

# Check for CLI updates
./dist/newapi-tools update --check
```

---

## 📋 Commands

| Command | Description |
|---------|-------------|
| `install` | Deploy new-api with Docker Compose (`--interactive` for wizard) |
| `status` / `ls` | Show container status. `--json` for JSON, `--instance <name>` for multi-instance |
| `backup` | Backup configs, MySQL dump, data dir to `.tar.gz` |
| `restore` | Restore from backup. `--file latest` auto-picks newest |
| `update` | Update Docker image; `--self` updates CLI; `--check` inspects only |
| `doctor` | 12 diagnostic checks. `--fix` auto-repair, `--verbose` for detail |
| `config` | Config management: `set`, `init` (wizard), `chmod` (secure perms) |
| `mirror` | Docker registry mirror management |
| `instance` | Multi-instance management: `add`, `list`, `switch`, `remove` |
| `audit` | Audit log query: `list --last N`, `--cmd`, `--since`, `--json` |
| `version` | Print version and build info |

<details>
<summary><b>📖 Usage Examples (click to expand)</b></summary>

```bash
# Install
newapi-tools install --port 8080
newapi-tools install --domain newapi.example.com --mirror tuna
newapi-tools install --interactive

# Backup & Restore
newapi-tools backup
newapi-tools restore --file latest --force

# Self-update
newapi-tools update --check
newapi-tools update --self

# Diagnostics
newapi-tools doctor --json
newapi-tools doctor --fix
newapi-tools doctor --verbose

# Config
newapi-tools config
newapi-tools config set newapi.port 8080
newapi-tools config init
newapi-tools config chmod

# Audit
newapi-tools audit list --last 10
newapi-tools audit list --cmd install

# Multi-instance
newapi-tools instance add prod --port 8080
newapi-tools instance list
newapi-tools instance switch prod
newapi-tools --instance prod status

# Mirror
newapi-tools mirror builtin
newapi-tools mirror test tuna aliyun
newapi-tools mirror add tuna
newapi-tools mirror apply
```
</details>

> **Built-in mirrors**: `tuna`, `aliyun`, `ustc`, `163`, `azure`, `daocloud`

---

## 📁 Project Structure

```
newapi-tools/
├── cmd/
│   ├── newapi/          # Entry point (main.go)
│   └── gendocs/         # Error code doc generator
├── internal/
│   ├── apperr/          # Structured error codes & suggestions
│   ├── audit/           # Audit logging (JSON Lines + ring buffer query)
│   ├── cli/             # Cobra commands (12 commands)
│   ├── core/            # Config, version constants
│   ├── docker/          # Docker CLI wrapper + compose + mirror management
│   ├── i18n/            # Internationalization (zh-CN / en)
│   ├── instance/        # Multi-instance metadata store
│   ├── osutil/          # OS adapters (Debian/RHEL/Fedora/Arch)
│   ├── plugin/          # Plugin system (ShellPlugin/GoPlugin)
│   ├── registry/        # Command registry
│   ├── security/        # Permissions checking & fixing
│   ├── selfupdate/      # CLI self-update (GitHub Releases)
│   └── ui/              # Output formatting (table, logger, progress)
├── plugins/newapi/      # newapi Shell plugin
├── configs/             # Default config files
├── docs/                # MkDocs documentation site
└── .github/workflows/   # CI + Release + Docs automation
```

---

## ⚙️ Configuration

Default path: `~/.config/newapi-tools/newapi-tools.yml`

```yaml
newapi:
  home: /opt/newapi          # Installation directory
  port: 3000                 # Host port for new-api
  docker_image: calciumion/new-api:latest
  backup_dir: /opt/newapi/backups
  domain: ""                 # Custom domain (optional)
  health_timeout: 120        # Health check timeout (seconds)
  max_backups: 10            # Max backups to keep
docker:
  compose_cmd: "docker compose"
log:
  level: info                # debug | info | warn | error
  format: text               # text | json
instance:
  active: ""                 # Current active instance name
```

Override via CLI flags:

```bash
newapi-tools config set newapi.port 8080
newapi-tools config init
newapi-tools --log-level debug status
newapi-tools --config /path/to/config.yml status
```

---

## 🔧 Development

```bash
make build    # Build → dist/newapi-tools
make test     # Run tests (16 packages)
make run      # Run locally
make lint     # golangci-lint
make vet      # go vet ./...
make coverage # Generate test coverage report
make docs     # Generate error code docs
make check    # vet + test
```

<details>
<summary><b>📋 Testing & Cross-compile (click to expand)</b></summary>

```bash
# Run tests
go test ./...                  # All packages
go test ./internal/cli/ -v     # CLI only
go test ./... -count=1         # No cache

# Cross-compile for Linux
GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o dist/newapi-tools-linux-amd64 ./cmd/newapi/
```
</details>

---

## 🔌 Plugin System

Plugins discovered from `./plugins/` at startup:

- **Shell plugins**: `metadata.yml` + `scripts/` → executed via `bash`
- **Go plugins**: `plugin.go` → compiled as Go code

The bundled `plugins/newapi/` plugin provides 7 shell script commands, with `install` and `status` promoted to native Go.

---

## ⚠️ Disclaimer

This tool executes real Docker CLI commands that can modify container and host configurations. Running high-risk operations (such as `install --force`, `restore`, `update`) without verified backups, maintenance windows, or a clear rollback plan may result in container downtime, service interruption, configuration corruption, or irreversible data loss. **All data loss, recovery costs, and third-party restoration expenses are the sole responsibility of the operator.**

Before executing `backup`, `restore`, `update`, `doctor --fix`, or similar commands, it is recommended to read the relevant documentation in the [docs/](docs/) directory.

---

## 📄 License

[MIT](LICENSE)

---

<p align="center">
  <a href="#-quick-start">🚀 Quick Start</a> ·
  <a href="#-commands">📋 Commands</a> ·
  <a href="#-project-structure">📁 Structure</a> ·
  <a href="#-configuration">⚙️ Config</a> ·
  <a href="#-development">🔧 Dev</a> ·
  <a href="#-disclaimer">⚠️ Disclaimer</a>
</p>

<p align="center">
  <a href="README.md">🇨🇳 中文版本 →</a>
</p>
