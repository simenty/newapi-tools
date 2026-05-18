# NewAPI Tools V3.0

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
```

## Commands

| Command | Description |
|---------|-------------|
| `install` | Deploy new-api with Docker Compose (pulls image, generates config, starts containers) |
| `status` | Show container status. Use `--json` for JSON output |
| `backup` | Backup new-api data (configs + MySQL dump + data dir) to a `.tar.gz` archive |
| `restore` | Restore from a backup archive. Use `--file latest` to auto-pick newest |
| `update` | Update new-api to latest image. Auto-backup before update (use `--backup=false` to skip) |
| `doctor` | Run 10 diagnostic checks: Docker, containers, ports, disk, config files |
| `version` | Print version and build info |

### Examples

```bash
# Install with custom port
newapi-tools install --port 8080

# Create a backup
newapi-tools backup --output /var/backups

# Restore latest backup
newapi-tools restore --file latest --force

# Update to a specific tag
newapi-tools update --image calciumion/new-api:v0.3.x

# Full diagnostic report in JSON
newapi-tools doctor --json

# Use custom config file
newapi-tools --config /etc/newapi-tools.yml status
```

## Project Structure

```
newapi-tools/
├── cmd/newapi/          # Entry point (main.go)
├── internal/
│   ├── cli/             # Cobra commands (install/status/backup/restore/update/doctor)
│   ├── core/            # Config, version constants
│   ├── docker/          # Docker CLI wrapper + compose operations
│   ├── osutil/          # OS adapters (Debian/RHEL/Fedora/Arch)
│   ├── plugin/          # Plugin system (ShellPlugin/GoPlugin/Loader)
│   ├── registry/        # Command registry
│   └── ui/              # Output formatting (table, logger)
├── plugins/newapi/      # newapi Shell plugin (metadata.yml + scripts/)
├── configs/             # Default config files
└── scripts/             # Build & release scripts
```

## Configuration

Default config location: `~/.newapi-tools.yml`

```yaml
newapi:
  home: /opt/newapi          # Installation directory
  port: 3000                 # Host port for new-api
  docker_image: calciumion/new-api:latest
  backup_dir: /opt/newapi/backups

docker:
  compose_cmd: "docker compose"

log:
  level: info                # debug | info | warn | error
  format: text               # text | json
```

Override via CLI flags:
```bash
newapi-tools --log-level debug --config /path/to/config.yml status
```

## Development

```bash
make build    # Build binary → dist/newapi-tools
make test     # Run all tests (71 tests across 6 packages)
make run      # Run locally
make lint     # golangci-lint
```

### Running Tests

```bash
go test ./...                  # All packages
go test ./internal/cli/ -v     # CLI tests only
go test ./... -count=1         # Force no cache
```

## Plugin System

Plugins are discovered from `./plugins/` at startup:

- **Shell plugins**: directory with `metadata.yml` + `scripts/` → executed via `bash`
- **Go plugins**: directory with `plugin.go` → compiled as Go code

The bundled `plugins/newapi/` plugin provides 7 commands as shell scripts, with `install` and `status` already promoted to native Go implementations.

## Version History

| Version | Highlights |
|---------|-----------|
| V3.0-rc | All 7 commands Go-native: install/status/backup/restore/update/doctor + 71 tests |
| V3.0-a3 | install + status Go-native, newapi Shell plugin, 52 tests |
| V3.0-a2 | Plugin system, Registry, Docker wrapper, OS adapter |
| V3.0-a1 | Go project skeleton, Cobra CLI, Viper config, slog |
| V2.4    | Shell V2 with plugin system (archived on `v2` branch) |

## License

MIT
