# config

查看和修改 newapi-tools 配置。

## 用法

```bash
newapi-tools config [subcommand] [flags]
```

## 子命令

| 子命令 | 说明 |
|--------|------|
| `config` | 显示当前配置（无子命令） |
| `config set <key> <value>` | 设置配置项 |
| `config init` | 交互式配置向导 |

## 配置文件位置

```
~/.config/newapi-tools/newapi-tools.yml
```

## 可配置项

| 键 | 默认值 | 说明 |
|----|--------|------|
| `newapi.home` | `/opt/newapi` | new-api 安装目录 |
| `newapi.port` | `3000` | new-api 监听端口 |
| `newapi.docker_image` | `calciumion/new-api:latest` | Docker 镜像 |
| `newapi.backup_dir` | `<home>/backups` | 备份目录 |
| `newapi.domain` | `""` | new-api 自定义域名（可选） |
| `newapi.health_timeout` | `120` | 健康检查超时时间（秒） |
| `newapi.max_backups` | `10` | 最大保留备份数量 |
| `docker.compose_cmd` | `docker compose` | Docker Compose 命令 |
| `log.level` | `info` | 日志级别（debug/info/warn/error） |
| `log.format` | `text` | 日志格式（text/json） |
| `instance.active` | `""` | 当前活跃实例的名称 |

## 说明

- 当存在活跃实例时，`config` 命令会自动使用该实例的配置
- `config set` 会同时更新配置文件和活跃实例的配置（如果存在）
- 使用 `--instance <name>` 标志可以在单个命令上临时切换到其他实例

## 示例

```bash
# 查看当前配置
newapi-tools config

# 修改端口
newapi-tools config set newapi.port 8080

# 设置自定义域名
newapi-tools config set newapi.domain newapi.example.com

# 修改健康检查超时
newapi-tools config set newapi.health_timeout 300

# 设置最大保留备份数量
newapi-tools config set newapi.max_backups 20

# 修改安装目录
newapi-tools config set newapi.home /data/newapi

# 交互式配置向导
newapi-tools config init
```
