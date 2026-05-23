# NewAPI Tools

Docker 管理平台，专为 [new-api](https://github.com/Calcium-Ion/new-api) 设计，Go 语言编写。

## 特性

- **一键部署**：`newapi-tools install` 自动拉镜像、生成配置、启动容器
- **状态监控**：`newapi-tools status` 查看容器运行状态，支持 JSON 输出
- **数据备份**：`newapi-tools backup` 备份数据、配置和数据库
- **快速恢复**：`newapi-tools restore --file latest` 从备份恢复
- **在线更新**：`newapi-tools update` 拉取最新镜像并热更新
- **健康诊断**：`newapi-tools doctor` 10 项自动检查，`--fix` 自动修复
- **镜像加速**：`newapi-tools mirror` 管理 Docker Hub 镜像源，国内拉取不再慢
- **自动检测镜像源**：install/update 时自动测速并应用最快镜像源

## 快速开始

```bash
# 安装（Linux / macOS）
curl -fsSL https://github.com/simenty/newapi-tools/releases/latest/download/newapi-tools_linux_amd64.tar.gz | tar xz
sudo mv newapi-tools /usr/local/bin/

# 部署 new-api
newapi-tools install

# 查看状态
newapi-tools status
```

详细安装说明请看 [快速开始](guide/getting-started.md)。

## 支持平台

| 平台 | 架构 |
|------|------|
| Linux | amd64, arm64 |
| macOS | amd64, arm64 |
| Windows | amd64 |
