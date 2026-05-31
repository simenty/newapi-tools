# NewAPI Tools V3.4

Docker management platform for [new-api](https://github.com/Calcium-Ion/new-api), rewritten in Go.

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

# Check for self-update
./dist/newapi-tools update --check
```

## Commands

| Command | Description |
|---------|-------------|
| `install` | Deploy new-api with Docker Compose (interactive wizard with `--interactive`) |
| `status` / `ls` | Show container status. Use `--json` for JSON output, `--instance <name>` for multi-instance |
| `backup` | Backup new-api data (configs + MySQL dump + data dir) to a `.tar.gz` archive |
| `restore` | Restore from a backup archive. Use `--file latest` to auto-pick newest |
| `update` | Update new-api image, or `--self` to update the CLI binary itself, `--check` to inspect |
| `doctor` | Run 12 diagnostic checks: Docker, containers, ports, disk, config files, permissions. Use `--fix` to auto-repair, `--verbose` for detail |
| `config` | View or modify configuration. Subcommands: `set`, `init` (interactive wizard), `chmod` (secure file perms) |
| `mirror` | Manage Docker registry mirrors to speed up image pulls in China |
| `instance` | Manage multiple new-api instances: `add`, `list`, `switch`, `remove` |
| `audit` | View command audit log: `list --last N`, `--cmd`, `--since`, `--json` |
| `version` | Print version and build info |

### Examples

```bash
# Install with custom port
newapi-tools install --port 8080

# Install with custom domain and health timeout
newapi-tools install --domain newapi.example.com --health-timeout 300

# Install using a specific registry mirror
newapi-tools install --mirror tuna

# Create a backup
newapi-tools backup --output /var/backups

# Restore latest backup
newapi-tools restore --file latest --force

# Update to a specific tag
newapi-tools update --image calciumion/new-api:v0.3.x

# Update using a registry mirror
newapi-tools update --mirror tuna

# Check for CLI updates
newapi-tools update --check

# Self-update the CLI binary
newapi-tools update --self

# Full diagnostic report in JSON
newapi-tools doctor --json

# Auto-fix detected issues
newapi-tools doctor --fix

# Verbose diagnostic output
newapi-tools doctor --verbose

# View current config
newapi-tools config

# Set a config value and persist
newapi-tools config set newapi.port 8080
newapi-tools config set newapi.domain newapi.example.com

# Interactive config wizard
newapi-tools config init

# Fix config file permissions (chmod 600)
newapi-tools config chmod

# View audit log (last 10 entries)
newapi-tools audit list --last 10

# View audit log filtered by command
newapi-tools audit list --cmd install

# Use custom config file
newapi-tools --config /etc/newapi-tools.yml status

# Add a new instance
newapi-tools instance add prod --port 8080 --domain newapi.example.com

# List all instances
newapi-tools instance list

# Switch to another instance
newapi-tools instance switch prod

# Use a specific instance for a single command
newapi-tools --instance prod status
```

### Mirror Examples

```bash
# List built-in mirrors
newapi-tools mirror builtin

# Add TUNA mirror (Tsinghua University)
newapi-tools mirror add tuna

# Add a custom mirror URL
newapi-tools mirror add https://my-mirror.example.com

# List currently configured mirrors
newapi-tools mirror list

# Test if a mirror is reachable
newapi-tools mirror test tuna

# Apply mirror list to /etc/docker/daemon.json and reload Docker
newapi-tools mirror apply

# Remove a mirror
newapi-tools mirror remove tuna

# Reset to empty (no mirrors)
newapi-tools mirror reset
```

Built-in mirror shortcuts: `tuna`, `aliyun`, `ustc`, `163`, `azure`, `daocloud`

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

## Configuration

Default config location: `~/.config/newapi-tools/newapi-tools.yml`

```yaml
newapi:
  home: /opt/newapi          # Installation directory
  port: 3000                 # Host port for new-api
  docker_image: calciumion/new-api:latest
  backup_dir: /opt/newapi/backups
  domain: ""                 # Custom domain for new-api (optional)
  health_timeout: 120        # Health check timeout in seconds
  max_backups: 10            # Maximum number of backups to keep

docker:
  compose_cmd: "docker compose"

log:
  level: info                # debug | info | warn | error
  format: text               # text | json

instance:
  active: ""                 # Current active instance name
```

Override via CLI flags or `config set`:
```bash
newapi-tools config set newapi.port 8080
newapi-tools config init                    # Interactive wizard
newapi-tools --log-level debug status
newapi-tools --config /path/to/config.yml status
```

## Development

```bash
make build    # Build binary → dist/newapi-tools
make test     # Run all tests (16 packages)
make run      # Run locally
make lint     # golangci-lint
make vet      # go vet ./...
make coverage # Generate test coverage report
make docs     # Generate error code docs
```

### Running Tests

```bash
go test ./...                  # All packages
go test ./internal/cli/ -v     # CLI tests only
go test ./... -count=1         # Force no cache
```

### Cross-compile for Linux

```bash
GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o dist/newapi-tools-linux-amd64 ./cmd/newapi/
```

## Plugin System

Plugins are discovered from `./plugins/` at startup:

- **Shell plugins**: directory with `metadata.yml` + `scripts/` → executed via `bash`
- **Go plugins**: directory with `plugin.go` → compiled as Go code

The bundled `plugins/newapi/` plugin provides 7 commands as shell scripts, with `install` and `status` already promoted to native Go implementations.

## Version History

| Version | Highlights |
|---------|-----------|
| V3.4.0  | Audit report fixes: cross-partition EXDEV fallback, `--json` now uses `encoding/json`, audit ring buffer, CI docs check |
| V3.3.5  | Fix resolveAssetName semver comparison, doctor error code X001, audit double logging |
| V3.3.4  | Security audit: path traversal fix, permission lockdown, shell injection prevention, race condition fixes |
| V3.3.0  | Domain/MaxBackups config, `Check` struct refactor, auto-rollback |
| V3.2.0  | Self-update (`update --check` / `--self`), Audit log (`audit list`), Multi-instance management (`instance add/list/switch/remove`), Auto-generated error code docs |
| V3.1.0  | i18n framework, structured error handling, audit logging, security checks, interactive install wizard |
| V3.0.0  | mirror command (add/remove/list/apply/test/reset/builtin), 6 built-in CN mirrors, 83 tests |
| V3.0-rc | 9 commands Go-native (incl. config + doctor --fix), 78 tests, Git v2/main split |
| V3.0-a3 | install + status Go-native, newapi Shell plugin, 52 tests |
| V3.0-a2 | Plugin system, Registry, Docker wrapper, OS adapter |
| V3.0-a1 | Go project skeleton, Cobra CLI, Viper config, slog |
| V2.4    | Shell V2 with plugin system (archived on `v2` branch) |

## License

MIT
