# install

部署 new-api 及其依赖服务（MySQL + Redis）。

## 用法

```bash
newapi-tools install [flags]
```

## 说明

install 命令执行以下步骤：

1. 检查 Docker 是否可用
2. 检查是否已安装（如已运行，提示用 `--force` 重装）
3. 创建安装目录
4. 生成 `docker-compose.yml` 和 `.env` 配置文件
5. 拉取 Docker 镜像
6. 启动容器
7. 将实例配置添加到实例存储（如果指定了 `--instance`）

## 标志

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--port` | 3000 | new-api 监听端口 |
| `--image` | calciumion/new-api:latest | 使用的 Docker 镜像 |
| `--domain` | | new-api 自定义域名（可选） |
| `--health-timeout` | 120 | 健康检查超时时间（秒） |
| `--instance` | | 要安装的实例名称（用于多实例管理） |
| `--force` | false | 强制重装（移除已有容器） |
| `--mirror` | | 指定镜像源加速拉取（如 tuna、aliyun，或完整 URL） |
| `--no-auto-mirror` | false | 跳过自动检测最快镜像源 |

## 示例

```bash
# 默认安装
newapi-tools install

# 自定义端口和域名
newapi-tools install --port 8080 --domain newapi.example.com

# 使用清华源加速
newapi-tools install --mirror tuna

# 强制重装
newapi-tools install --force

# 跳过自动镜像检测
newapi-tools install --no-auto-mirror

# 安装到特定实例
newapi-tools install --instance prod
```

## 自动镜像加速

当没有指定 `--mirror` 且 daemon.json 中没有配置镜像源时，install 会并发测试所有内置镜像源（tuna、aliyun、ustc、163、azure、daocloud），选择延迟最低的自动应用。

这个行为可以通过 `--no-auto-mirror` 关闭。
