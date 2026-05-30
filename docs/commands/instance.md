# instance

管理多个 new-api 实例。每个实例拥有独立的配置、Docker Compose 项目和安装目录。

## 用法

```bash
newapi-tools instance <subcommand> [flags]
```

## 子命令

| 子命令 | 说明 |
|--------|------|
| `instance add <name>` | 添加新实例 |
| `instance list` | 列出所有已配置实例 |
| `instance switch <name>` | 切换活跃实例 |
| `instance remove <name>` | 删除实例（不会删除数据和容器） |

## 实例配置

每个实例包含以下配置：
- `name`：唯一名称
- `home`：安装目录
- `port`：服务端口
- `docker_image`：使用的 Docker 镜像
- `domain`：自定义域名（可选）
- `health_timeout`：健康检查超时时间（秒）
- `max_backups`：最大保留备份数量

## 实例 add 标志

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--home` | `/opt/newapi-<name>` | 实例安装目录 |
| `--port` | `3000` | 实例监听端口 |
| `--image` | `calciumion/new-api:latest` | 实例使用的 Docker 镜像 |
| `--domain` | | 实例自定义域名（可选） |
| `--health-timeout` | `120` | 健康检查超时时间（秒） |
| `--max-backups` | `10` | 最大保留备份数量 |

## 全局标志

| 标志 | 说明 |
|------|------|
| `--instance <name>` | 临时为单个命令切换实例（不改变活跃实例） |

## 示例

```bash
# 添加新实例
newapi-tools instance add prod --port 8080 --domain newapi.example.com

# 列出所有实例
newapi-tools instance list

# 切换到新实例
newapi-tools instance switch prod

# 查看当前活跃实例的状态
newapi-tools status

# 使用特定实例查看状态（不改变活跃实例）
newapi-tools --instance dev status

# 为特定实例安装
newapi-tools --instance dev install --port 8081

# 删除实例（不会删除容器或数据）
newapi-tools instance remove dev
```

## 注意事项

- 切换实例会更新 `instance.active` 配置项
- 删除实例不会删除该实例的 Docker 容器或数据目录
- 所有实例共享同一个配置文件，但各实例配置独立存储在 `~/.config/newapi-tools/instances.json` 中
- 端口必须唯一，不能有两个实例使用相同端口
