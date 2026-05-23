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
| `docker.compose_cmd` | `docker compose` | Docker Compose 命令 |
| `log.level` | `info` | 日志级别（debug/info/warn/error） |
| `log.format` | `text` | 日志格式（text/json） |

## 示例

```bash
# 查看当前配置
newapi-tools config

# 修改端口
newapi-tools config set newapi.port 8080

# 修改安装目录
newapi-tools config set newapi.home /data/newapi

# 交互式配置向导
newapi-tools config init
```
