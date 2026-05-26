# 快速开始

## 安装

=== "Linux (amd64)"

    ```bash
    curl -fsSL -O https://github.com/simenty/newapi-tools/releases/latest/download/newapi_v3.1.0_Linux_x86_64.zip
    unzip newapi_v3.1.0_Linux_x86_64.zip
    sudo mv newapi_v3.1.0_linux_amd64 /usr/local/bin/newapi-tools
    sudo chmod +x /usr/local/bin/newapi-tools
    ```

=== "Linux (arm64)"

    ```bash
    curl -fsSL -O https://github.com/simenty/newapi-tools/releases/latest/download/newapi_v3.1.0_Linux_arm64.zip
    unzip newapi_v3.1.0_Linux_arm64.zip
    sudo mv newapi_v3.1.0_linux_arm64 /usr/local/bin/newapi-tools
    sudo chmod +x /usr/local/bin/newapi-tools
    ```

=== "macOS (Intel)"

    ```bash
    curl -fsSL -O https://github.com/simenty/newapi-tools/releases/latest/download/newapi_v3.1.0_Darwin_x86_64.zip
    unzip newapi_v3.1.0_Darwin_x86_64.zip
    sudo mv newapi_v3.1.0_darwin_amd64 /usr/local/bin/newapi-tools
    sudo chmod +x /usr/local/bin/newapi-tools
    ```

=== "Windows (amd64)"

    从 [GitHub Releases](https://github.com/simenty/newapi-tools/releases/latest) 下载 `newapi_v3.1.0_Windows_x86_64.zip`，解压后将 `newapi_v3.1.0_windows_amd64.exe` 重命名为 `newapi-tools.exe` 并加入 PATH。

## 前置条件

- Docker 20.10+ 和 Docker Compose V2
- 操作系统：Debian 10+、Ubuntu 20.04+、CentOS 8+、RHEL 8+、Fedora 35+、Arch Linux

## 部署 new-api

```bash
# 一键部署（自动拉取镜像、生成配置、启动 new-api + MySQL + Redis）
newapi-tools install

# 交互式向导（自定义端口/数据库/Redis）
newapi-tools install --interactive

# 自定义端口
newapi-tools install --port 8080

# 指定镜像源加速（国内推荐）
newapi-tools install --mirror tuna
```

!!! tip "自动镜像加速"
    如果没有手动指定 `--mirror`，install 会自动测试所有内置镜像源并应用最快的一个。
    用 `--no-auto-mirror` 可以跳过自动检测。

## 检查状态

```bash
# 查看容器状态
newapi-tools status

# 实时监控（每 2 秒刷新）
newapi-tools status --watch --interval 2

# 显示所有关联容器（含 CPU/MEM 统计）
newapi-tools status --all

# JSON 格式输出（方便脚本处理）
newapi-tools status --json
```

## 日常操作

```bash
# 备份数据
newapi-tools backup

# 更新到最新版本
newapi-tools update

# 健康检查
newapi-tools doctor

# 自动修复常见问题
newapi-tools doctor --fix
```

## 配置

```bash
# 查看当前配置
newapi-tools config

# 修改配置项
newapi-tools config set newapi.port 8080

# 交互式配置向导
newapi-tools config init
```

## 下一步

- [命令参考](../commands/install.md)：每个命令的详细用法
- [镜像加速](mirror.md)：国内 Docker Hub 拉取优化
- [常见问题](faq.md)：排查和解决问题
