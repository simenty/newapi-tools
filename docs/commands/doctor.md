# doctor

运行 10 项健康检查，诊断 new-api 部署状态。

## 用法

```bash
newapi-tools doctor [flags]
```

## 标志

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--fix` | false | 自动修复可修复的问题 |

## 检查项

| 检查 | 说明 |
|------|------|
| Docker 守护进程 | Docker 是否运行且可访问 |
| Docker Compose | docker compose 命令是否可用 |
| new-api 容器 | new-api 容器是否运行 |
| mysql 容器 | MySQL 容器是否运行 |
| redis 容器 | Redis 容器是否运行 |
| 端口监听 | 配置端口是否在监听 |
| 安装目录 | home 目录是否存在 |
| docker-compose.yml | 配置文件是否存在 |
| .env 文件 | 环境变量文件是否存在 |
| 磁盘空间 | 安装目录所在磁盘剩余空间是否充足（>1GB） |

## 示例

```bash
# 只检查，不修复
newapi-tools doctor

# 检查并自动修复
newapi-tools doctor --fix
```

## 输出示例

```
[PASS] Docker daemon is running
[PASS] Docker Compose is available
[FAIL] new-api container is not running
[FAIL] mysql container is not running
[PASS] redis container is running
...

Failures: 2
Run 'newapi-tools doctor --fix' to attempt auto-repair.
```

## --fix 能修复什么

- 创建缺失的安装目录
- 启动未运行的容器（执行 `docker compose up -d`）

!!! note "提示"
    `--fix` 无法修复 Docker 未安装、配置文件缺失等根本性问题，这些需要手动处理。
