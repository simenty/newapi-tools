# NewAPI Tools v2.0 安装指南

## 📋 系统要求

### 操作系统
- Ubuntu 20.04 / 22.04 / 24.04（推荐 22.04）
- Debian 10 / 11 / 12
- 其他基于 Debian/Ubuntu 的发行版

### 硬件要求
- CPU：1 核（推荐 2 核以上）
- 内存：1GB（推荐 2GB 以上）
- 磁盘：10GB（推荐 20GB 以上）

### 网络要求
- 可以访问 Docker Hub（或配置镜像加速）
- 可以访问 GitHub（下载工具集）
- 配置域名需要 80/443 端口可访问

---

## 🚀 安装方法

### 方法 1：一键安装（推荐）

```bash
# 使用 curl
bash <(curl -fsSL https://raw.githubusercontent.com/simenty/newapi-tools/main/install.sh)

# 或使用 wget
bash <(wget -qO- https://raw.githubusercontent.com/simenty/newapi-tools/main/install.sh)
```

### 方法 2：手动安装

```bash
# 1. 克隆仓库
git clone https://github.com/simenty/newapi-tools.git /opt/newapi-tools

# 2. 运行安装器
bash /opt/newapi-tools/install.sh

# 3. 验证安装
newapi-tools --version
```

### 方法 3：从 Release 下载

```bash
# 1. 下载最新 Release
wget https://github.com/simenty/newapi-tools/releases/latest/download/newapi-tools.tar.gz

# 2. 解压
tar -xzf newapi-tools.tar.gz -C /opt/

# 3. 运行安装器
bash /opt/newapi-tools/install.sh
```

---

## ⚙️ 安装后的配置

### 1. 首次运行

```bash
newapi-tools
```

首次运行会：
- 初始化状态管理（`state.json`）
- 创建配置文件（`config/` 目录）
- 显示模式选择（新手/专家模式）

### 2. 选择模式

**新手模式**（推荐第一次使用的用户）：
- 显示详细提示
- 自动选择推荐配置
- 显示进度条和操作说明

**专家模式**（适合熟悉的用户）：
- 跳过非必要提示
- 显示所有高级选项
- 最小化输出

> 可以随时在主菜单输入 `m` 切换模式

### 3. 环境准备（菜单选项 1）

**步骤 1：配置 DNS**
- 推荐：阿里 DNS（223.5.5.5 / 223.6.6.6）
- 可选：腾讯 DNS、114 DNS、Google DNS

**步骤 2：更换软件源**
- 推荐：清华大学源
- 可选：阿里云源、腾讯云源、华为云源

**步骤 3：安装 Docker**
- 自动安装 Docker CE 最新版
- 配置 Docker 镜像加速（可选）

### 4. 部署 NewAPI（菜单选项 2）

v2.0 新增：
- ✓ 自动生成安全密码（MySQL、Redis、Session Secret）
- ✓ 智能推荐配置（根据系统内存自动调整）
- ✓ 5 步进度显示（新手模式）
- ✓ 断点续装（中断后重新运行可跳过已完成步骤）

**部署内容**：
- NewAPI（AI 接口管理）
- MySQL 8.0（数据库）
- Redis 7.2（缓存）
- Nginx Proxy Manager（反向代理）

**访问地址**：
- NewAPI: `http://服务器IP:3000`
- NPM 管理: `http://服务器IP:81`（默认账号: admin@example.com / changeme）

### 5. 配置域名和 SSL（菜单选项 3）

**前提条件**：
1. 域名 DNS 已解析到服务器 IP
2. 端口 80/443 未被占用
3. 已修改 NPM 默认密码

**操作步骤**：
1. 输入域名（如 `api.example.com`）
2. 输入 NPM 管理员邮箱
3. 输入 NPM 管理员密码
4. 等待 Let's Encrypt 证书签发（1-2 分钟）

**完成后访问**：`https://你的域名`

---

## 🔧 可选配置

### 1. 配置 Webhook 通知

编辑配置文件 `/opt/newapi-tools/config/config.yaml`：

```yaml
notification:
  webhook_url: "https://oapi.dingtalk.com/robot/send?access_token=xxx"  # 钉钉 Webhook
  notify_on_success: true
  notify_on_error: true
```

支持：
- 钉钉 Webhook
- 飞书 Webhook
- Slack Webhook
- 自定义 Webhook

### 2. 修改备份保留天数

编辑配置文件：

```yaml
backup:
  dir: "/opt/newapi-tools/backups"
  retention_days: 7  # 保留 7 天，改为你需要的天数
  compress: true
  encrypt: false
```

### 3. 配置资源告警阈值

编辑配置文件：

```yaml
monitor:
  health_check_interval: 300  # 健康检查间隔（秒）
  auto_restart: false
  alert_cpu_threshold: 80  # CPU 使用率告警阈值（%）
  alert_mem_threshold: 80  # 内存使用率告警阈值（%）
```

---

## 🐛 常见问题

### 1. 安装时提示 "permission denied"

**原因**：没有使用 root 用户运行。

**解决方案**：
```bash
# 切换到 root 用户
su - root

# 或使用 sudo
sudo bash install.sh
```

### 2. 安装时提示 "command not found: curl"

**原因**：系统缺少 curl 或 wget。

**解决方案**：
```bash
# Ubuntu/Debian
apt update
apt install -y curl wget
```

### 3. 安装后 `newapi-tools` 命令找不到

**原因**：软链接创建失败。

**解决方案**：
```bash
# 手动创建软链接
ln -sf /opt/newapi-tools/newapi-tools.sh /usr/local/bin/newapi-tools

# 或直接使用完整路径
bash /opt/newapi-tools/newapi-tools.sh
```

### 4. 部署时提示 "端口已被占用"

**原因**：3000 或 81 端口已被其他程序占用。

**解决方案**：
```bash
# 查看端口占用
netstat -tlnp | grep -E ':(3000|81) '

# 修改配置文件，使用其他端口
edit /opt/newapi-tools/config/config.yaml
# 修改：
#   deploy:
#     newapi:
#       port: 8080  # 改为 8080
#     npm:
#       port: 8081  # 改为 8081
```

### 5. 部署后无法访问 NewAPI

**可能原因 1**：防火墙未开放端口。

**解决方案**：
```bash
# Ubuntu（使用 ufw）
ufw allow 3000/tcp
ufw allow 80/tcp
ufw allow 443/tcp
ufw reload
```

**可能原因 2**：Docker 容器未正常启动。

**解决方案**：
```bash
# 检查容器状态
docker ps

# 查看 NewAPI 日志
docker logs new-api

# 重启容器
cd /home/new-api
docker compose restart
```

### 6. SSL 配置失败

**可能原因 1**：域名 DNS 未解析到本机 IP。

**解决方案**：
```bash
# 检查 DNS 解析
dig your-domain.com

# 应该返回你的服务器 IP
```

**可能原因 2**：端口 80/443 被占用。

**解决方案**：
```bash
# 查看端口占用
netstat -tlnp | grep -E ':(80|443) '

# 停止占用的程序，或配置 NPM 使用其他端口
```

**可能原因 3**：Let's Encrypt 请求频率限制。

**解决方案**：
- 等待 1 小时后重试
- 或使用 DNS 验证方式申请证书

---

## 📞 获取帮助

### 1. 查看帮助文档

```bash
# 主帮助
newapi-tools --help

# 子命令帮助
newapi-tools backup --help
newapi-tools restore --help
newapi-tools update --help
```

### 2. 查看日志

```bash
# 工具日志
tail -f /opt/newapi-tools/logs/toolkit.log

# NewAPI 日志
docker logs -f new-api

# NPM 日志
docker logs -f npm
```

### 3. 提交 Issue

访问 GitHub Issues：
https://github.com/simenty/newapi-tools/issues

### 4. 加入社区

（如果有 QQ 群或 Discord 服务器，可以在这里添加）

---

## 📝 附录：完整安装示例

### 示例 1：全新服务器，从头安装

```bash
# 1. 切换到 root
su - root

# 2. 一键安装 newapi-tools
bash <(curl -fsSL https://raw.githubusercontent.com/simenty/newapi-tools/main/install.sh)

# 3. 启动工具
newapi-tools

# 4. 选择新手模式

# 5. 菜单选项 1：环境准备（DNS → 换源 → 安装 Docker）

# 6. 菜单选项 2：部署 NewAPI

# 7. 等待部署完成（5-10 分钟）

# 8. 访问 http://服务器IP:3000 完成 NewAPI 初始化

# 9. 菜单选项 3：配置域名和 SSL

# 10. 访问 https://你的域名 完成！
```

### 示例 2：已有 Docker，只安装工具

```bash
# 1. 安装 newapi-tools
git clone https://github.com/simenty/newapi-tools.git /opt/newapi-tools
bash /opt/newapi-tools/install.sh

# 2. 启动工具（会自动检测 Docker）
newapi-tools

# 3. 跳过菜单选项 1（Docker 已安装）

# 4. 菜单选项 2：部署 NewAPI
```

---

**祝安装顺利！ 🎉**
