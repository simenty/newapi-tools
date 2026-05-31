<div align="right">
  <a href="README.md">🇨🇳 中文</a> | <a href="README_EN.md">🇬🇧 English</a>
</div>

# NewAPI Tools V3.4

Docker management platform for [new-api](https://github.com/Calcium-Ion/new-api), rewritten in Go.

---

## Quick Start

```bash
# Build
make build

# Install new-api
./dist/newapi-tools install

# Check status
./dist/newapi-tools status

# Run diagnostics
./dist/newapi-tools doctor

# View / edit configuration
./dist/newapi-tools config
./dist/newapi-tools config set newapi.port 8080

# Speed up image pulls (China mainland)
./dist/newapi-tools mirror add tuna

# Check for CLI updates
./dist/newapi-tools update --check
```

---

## Commands

| Command | Description |
|---------|-------------|
| `install` | Deploy new-api with Docker Compose (interactive wizard with `--interactive`) |
| `status` / `ls` | Show container status. `--json` for JSON, `--instance <name>` for multi-instance |
| `backup` | Backup configs, MySQL dump, data dir to `.tar.gz` |
| `restore` | Restore from backup. `--file latest` to auto-pick newest |
| `update` | Update Docker image; `--self` to update CLI binary; `--check` to inspect |
| `doctor` | 12 diagnostic checks. `--fix` auto-repair, `--verbose` for detail |
| `config` | Config management: `set`, `init` (wizard), `chmod` (secure permissions) |
| `mirror` | Docker registry mirror management |
| `instance` | Multi-instance management: `add`, `list`, `switch`, `remove` |
| `audit` | Audit log query: `list --last N`, `--cmd`, `--since`, `--json` |
| `version` | Print version and build info |

### Examples

```bash
# Install
newapi-tools install --port 8080
newapi-tools install --domain newapi.example.com --mirror tuna

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

Built-in mirrors: `tuna`, `aliyun`, `ustc`, `163`, `azure`, `daocloud`

---

## Project Structure

```
newapi-tools/
├── cmd/newapi/          # Entry point (main.go)
├── cmd/gendocs/         # Error code doc generator
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
├── plugins/newapi/      # newapi Shell plugin (metadata.yml + scripts/)
├── configs/             # Default config files
├── docs/                # MkDocs documentation site
└── .github/workflows/   # CI + Release + Docs automation
```

---

## Configuration

Default: `~/.config/newapi-tools/newapi-tools.yml`

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

## Development

```bash
make build    # Build → dist/newapi-tools
make test     # Run tests (16 packages)
make run      # Run locally
make lint     # golangci-lint
make vet      # go vet ./...
make coverage # Generate test coverage report
make docs     # Generate error code docs
```

### Running Tests

```bash
go test ./...                  # All packages
go test ./internal/cli/ -v     # CLI only
go test ./... -count=1         # No cache
```

### Cross-compile for Linux

```bash
GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o dist/newapi-tools-linux-amd64 ./cmd/newapi/
```

---

## Plugin System

Plugins discovered from `./plugins/` at startup:

- **Shell plugins**: `metadata.yml` + `scripts/` → executed via `bash`
- **Go plugins**: `plugin.go` → compiled as Go code

The bundled `plugins/newapi/` plugin provides 7 shell script commands, with `install` and `status` promoted to native Go.

---

## Version History

| Version | Highlights |
|---------|-----------|
| V3.4.0  | Audit report fixes: cross-partition EXDEV fallback, `--json` uses `encoding/json`, audit ring buffer, CI docs check |
| V3.3.5  | Fix resolveAssetName semver comparison, doctor error code X001, audit double logging |
| V3.3.4  | Security audit: path traversal fix, permission lockdown, shell injection prevention, race condition fixes |
| V3.3.0  | Domain/MaxBackups config, `Check` struct refactor, auto-rollback |
| V3.2.0  | Self-update (`--check`/`--self`), Audit log query, Multi-instance management, Error code docs |
| V3.1.0  | i18n framework, structured error handling, audit logging, security checks, interactive install wizard |
| V3.0.0  | mirror command, 6 built-in CN mirrors, 83 tests |
| V3.0-rc | 9 Go-native commands, 78 tests, Git v2/main split |
| V3.0-a3 | install + status Go-native, newapi Shell plugin, 52 tests |
| V3.0-a2 | Plugin system, Registry, Docker wrapper, OS adapter |
| V3.0-a1 | Go project skeleton, Cobra CLI, Viper config, slog |
| V2.4    | Shell V2 plugin system (archived on `v2` branch) |

---

## License

MIT
