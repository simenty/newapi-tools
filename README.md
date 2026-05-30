# NewAPI Tools V3.2

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
```

## Commands

| Command | Description |
|---------|-------------|
| `install` | Deploy new-api with Docker Compose (pulls image, generates config, starts containers) |
| `status` | Show container status. Use `--json` for JSON output |
| `backup` | Backup new-api data (configs + MySQL dump + data dir) to a `.tar.gz` archive |
| `restore` | Restore from a backup archive. Use `--file latest` to auto-pick newest |
| `update` | Update new-api to latest image. Auto-backup before update (use `--backup=false` to skip) |
| `doctor` | Run 10 diagnostic checks: Docker, containers, ports, disk, config files. Use `--fix` to auto-repair |
| `config` | View or modify configuration. Subcommands: `set <key> <value>`, `init` (interactive wizard) |
| `mirror` | Manage Docker registry mirrors to speed up image pulls in China |
| `version` | Print version and build info |

### Examples

```bash
# Install with custom port
newapi-tools install --port 8080

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

# Full diagnostic report in JSON
newapi-tools doctor --json

# Auto-fix detected issues
newapi-tools doctor --fix

# View current config
newapi-tools config

# Set a config value and persist
newapi-tools config set newapi.port 8080

# Interactive config wizard
newapi-tools config init

# Use custom config file
newapi-tools --config /etc/newapi-tools.yml status
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
├── internal/
│   ├── cli/             # Cobra commands (install/status/backup/restore/update/doctor/config/mirror/version)
│   ├── core/            # Config, version constants
│   ├── docker/          # Docker CLI wrapper + compose + mirror management
│   ├── osutil/          # OS adapters (Debian/RHEL/Fedora/Arch)
│   ├── plugin/          # Plugin system (ShellPlugin/GoPlugin/Loader)
│   ├── registry/        # Command registry
│   └── ui/              # Output formatting (table, logger)
├── plugins/newapi/      # newapi Shell plugin (metadata.yml + scripts/)
├── configs/             # Default config files
└── docs/                # Architecture & design docs
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
make test     # Run all tests (83 tests across 7 packages)
make run      # Run locally
make lint     # golangci-lint
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
| V3.2.0  | Self-update (`update --check` / `--self`), Audit log (`audit list`), Multi-instance management (`instance add/list/switch/remove`), Auto-generated error code docs |
| V3.1.0  | - |
| V3.0.0  | mirror command (add/remove/list/apply/test/reset/builtin), 6 built-in CN mirrors, 83 tests |
| V3.0-rc | 9 commands Go-native (incl. config + doctor --fix), 78 tests, Git v2/main split |
| V3.0-a3 | install + status Go-native, newapi Shell plugin, 52 tests |
| V3.0-a2 | Plugin system, Registry, Docker wrapper, OS adapter |
| V3.0-a1 | Go project skeleton, Cobra CLI, Viper config, slog |
| V2.4    | Shell V2 with plugin system (archived on `v2` branch) |

## License

MIT
