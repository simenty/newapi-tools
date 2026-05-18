# NewAPI Tools v2.0 用户使用指南

## 🎯 简介

NewAPI Tools v2.0 是一个现代化的 NewAPI 运维工具集，提供：
- ✓ 一键部署 NewAPI 全家桶（MySQL + Redis + NPM）
- ✓ 智能配置生成（自动生成安全密码）
- ✓ 状态管理（支持断点续装）
- ✓ 新手/专家模式（渐进式复杂度）
- ✓ 友好的 UI 界面（进度条、彩色输出）
- ✓ SSL 证书自动申请与配置
- ✓ 数据备份与快速恢复（含自动回滚）
- ✓ 系统健康监控

---

## 🚀 快速开始

### 1. 安装 newapi-tools

```bash
# 下载并安装
bash <(curl -fsSL https://raw.githubusercontent.com/simenty/newapi-tools/main/install.sh)

# 或手动安装
git clone https://github.com/simenty/newapi-tools.git /opt/newapi-tools
bash /opt/newapi-tools/install.sh
```

### 2. 启动工具

```bash
newapi-tools
```

首次运行会显示欢迎界面和模式选择。

---

## 🎨 新手模式 vs 专家模式

### 新手模式（默认）
- ✓ 显示详细的操作提示和说明
- ✓ 自动选择推荐配置（可修改）
- ✓ 显示操作进度和状态面板
- ✓ 错误时显示友好提示和修复建议
- 适合：第一次使用的用户

### 专家模式
- ✓ 显示所有高级选项
- ✓ 跳过非必要的确认提示
- ✓ 直接执行，最小化输出
- ✓ 适合：熟悉 NewAPI 的用户

### 切换模式
在主菜单输入 `m` 可以随时切换模式。

---

## 📖 功能详解

### 1. 环境准备（菜单选项 1）

**功能**：配置 DNS、更换软件源、安装 Docker

**新手模式体验**：
- 显示 3 个子步骤的进度条
- 自动推荐最适合的 DNS 和软件源
- 每个步骤都有详细解释

**专家模式**：直接执行，无额外提示

---

### 2. 安装部署 NewAPI（菜单选项 2）

**功能**：一键部署 NewAPI + MySQL + Redis + NPM

**v2.0 改进**：
- ✓ 自动生成安全密码（MySQL、Redis、Session Secret）
- ✓ 智能检测系统配置，推荐最优参数
- ✓ 5 步进度显示（新手模式）
- ✓ 断点续装（如果中断，重新运行会跳过已完成步骤）

**部署流程**：
1. 准备配置（生成 .env 和 docker-compose.yml）
2. 拉取 Docker 镜像
3. 启动容器
4. 等待服务初始化
5. 验证健康状态

**访问地址**：
- NewAPI: `http://服务器IP:3000`
- NPM 管理: `http://服务器IP:81`（默认账号: admin@example.com / changeme）

---

### 3. 配置 SSL 与反向代理（菜单选项 3）

**功能**：自动申请 Let's Encrypt 证书并配置 Nginx 反向代理

**使用前准备**：
1. 确保域名 DNS 已解析到服务器 IP
2. 确保端口 80/443 未被占用
3. 首次使用需要修改 NPM 默认密码

**操作步骤**：
1. 输入域名（如 `api.example.com`）
2. 输入 NPM 管理员邮箱
3. 输入 NPM 管理员密码
4. 等待证书签发（1-2 分钟）

**完成后访问**：`https://你的域名`

---

### 4. 更新 NewAPI（菜单选项 4）

**功能**：一键更新 NewAPI 到最新版本

**v2.0 改进**：
- ✓ 更新前自动备份
- ✓ 更新后自动健康检查
- ✓ 健康检查失败自动回滚
- ✓ 4 步进度显示

**安全机制**：
1. 更新前自动备份数据
2. 记录当前镜像 ID
3. 拉取新镜像并重启容器
4. 等待 90 秒健康检查
5. 如果失败，自动回滚到旧版本

---

### 5. 数据备份（菜单选项 5）

**功能**：备份 NewAPI 数据库和数据卷

**备份内容**：
- MySQL 数据库（`.sql` 文件）
- 数据卷（`data/` 和 `npm/` 目录，`.tar.gz` 文件）
- MD5 校验文件（确保备份完整性）

**备份位置**：`/home/new-api/backups/`

**自动清理**：超过 7 天的旧备份会自动删除（可在配置中修改保留天数）

---

### 6. 备份恢复（菜单选项 6）

**功能**：从备份文件恢复数据

**v2.0 改进**：
- ✓ 显示备份文件列表（带大小和修改时间）
- ✓ 自动校验备份完整性（MD5 校验）
- ✓ 恢复前二次确认
- ✓ 恢复后自动重启容器

**使用步骤**：
1. 选择要恢复的备份序号
2. 确认恢复操作
3. 等待恢复完成
4. 检查服务健康状态

---

### 7. 系统健康检查（菜单选项 7）

**功能**：检查系统资源、Docker 容器、NewAPI 服务状态

**检查项**：
- 系统资源（内存、磁盘、CPU 负载）
- Docker 容器状态（new-api、mysql、redis、npm）
- NewAPI 健康状态（Docker 健康检查）
- 错误日志（最后 20 行）

**新手模式**：显示详细的健康状态面板和建议

**专家模式**：只返回退出码（0=健康，1=异常）

---

### 8. 重装 / 卸载（菜单选项 8）

**重装 NewAPI**：
1. 自动备份当前数据
2. 删除所有容器和数据
3. 重新部署

**卸载 NewAPI**：
1. 删除所有容器、网络、数据卷
2. 删除备份文件
3. 清理定时任务
4. （可选）删除日志和配置文件

---

## 🚆 状态管理（v2.0 新功能）

v2.0 引入了状态管理（`state.json`），记录：
- 安装进度（已完成的步骤）
- 系统状态（Docker 是否安装、DNS 是否配置）
- NewAPI 状态（是否安装、版本号）

**好处**：
- ✓ 断点续装：如果安装中断，重新运行会跳过已完成步骤
- ✓ 避免重复操作：工具会检测已完成的操作
- ✓ 更好的错误处理：知道当前在哪一步失败

**查看状态**：
```bash
cat /opt/newapi-tools/state.json
```

**清除状态**（从头开始）：
```bash
newapi-tools  # 菜单选项 8（卸载）会清除状态
```

---

## 🎛️ 配置管理（v2.0 新功能）

v2.0 使用 YAML 格式的配置文件：
- **用户配置**：`/opt/newapi-tools/config/config.yaml`（你的自定义配置）
- **默认配置**：`/opt/newapi-tools/config/config.default.yaml`（所有默认值）

**优先级**：用户配置 > 默认配置 > 内置默认值

**常用配置项**：
```yaml
# 部署配置
deploy:
  newapi:
    port: 3000  # NewAPI 端口
  npm:
    port: 81  # NPM 管理端口

# 备份配置
backup:
  dir: "/opt/newapi-tools/backups"
  retention_days: 7  # 备份保留天数

# 通知配置
notification:
  webhook_url: "https://your-webhook-url"  # 飞书/钉钉 Webhook
```

---

## 🔧 命令行用法

除了交互菜单，还可以直接使用命令行：

```bash
# 备份
newapi-tools backup

# 恢复（交互式）
newapi-tools restore

# 更新
newapi-tools update

# 安装部署
newapi-tools install

# 配置 SSL
newapi-tools ssl

# 健康检查
newapi-tools health

# 查看日志
newapi-tools logs

# 更新工具集自身
newapi-tools self-update

# 查看帮助
newapi-tools --help
newapi-tools backup --help
```

---

## 📊 新手模式体验

### 启动时的欢迎界面
```
╔═════════════╗
║   NewAPI 运维工具集  v2.0                              ║
║       🚀 更智能 · 更简单 · 更强大                            ║
╚═════════════╝

欢迎使用 NewAPI 运维工具集！

本工具将帮助您：
  1. 快速部署 NewAPI 及其依赖服务（MySQL + Redis + NPM）
  2. 配置 SSL 证书与 Nginx 反向代理
  3. 管理数据备份与快速恢复
  4. 监控系统健康状态与日志
  5. 一键更新 NewAPI 版本（含自动回滚）

  ℹ 当前为新手模式，将显示详细提示

按回车键继续...
```

### 安装时的进度显示
```
>>> 步骤 [1/5] 准备配置文件
[===========>                    ] 20%

>>> 步骤 [2/5] 创建目录和配置文件
[===================>            ] 40%

>>> 步骤 [3/5] 生成 Docker Compose 配置
[===========================>    ] 60%

>>> 步骤 [4/5] 拉取镜像并启动容器
[=================================>  ] 80%

>>> 步骤 [5/5] 等待服务初始化
[=====================================] 100%
```

### 错误时的友好提示
```
╔═════════════╗
║  ✗ 出错了！                                           ║
╚═════════════╝

问题描述：
  Docker 未运行

可能的原因：
  1. 没有使用 root 用户运行
  2. Docker 服务未启动

解决方案：
  - 使用 root 用户: sudo bash $0
  - 检查 Docker 状态: systemctl status docker
  - 启动 Docker: systemctl start docker
```

---

## 🔧 专家模式体验

专家模式下，输出简洁，适合熟悉的用户：

```bash
$ newapi-tools install
[1/5] 准备配置... 完成
[2/5] 创建目录... 完成
[3/5] 生成配置... 完成
[4/5] 拉取镜像... 完成
[5/5] 启动服务... 完成

=== 部署完成 ===
  NewAPI 地址  : <ADDRESS_REDACTED>
  NPM 管理地址  : http://192.168.1.100:81
```

---

## 📦 常见问题

### 1. 安装后无法访问 NewAPI？
**可能原因**：
- 防火墙未开放端口 3000
- Docker 容器未正常启动

**解决方案**：
```bash
# 检查容器状态
docker ps

# 查看 NewAPI 日志
docker logs new-api

# 开放防火墙端口（Ubuntu）
ufw allow 3000/tcp
ufw allow 80/tcp
ufw allow 443/tcp
```

### 2. SSL 配置失败？
**可能原因**：
- 域名 DNS 未解析到本机 IP
- 端口 80/443 被占用
- Let's Encrypt 请求频率限制

**解决方案**：
```bash
# 检查 DNS 解析
dig your-domain.com

# 检查端口占用
netstat -tlnp | grep -E ':(80|443) '

# 手动配置 NPM
访问 http://服务器IP:81
```

### 3. 更新后 NewAPI 无法启动？
**自动回滚**：如果健康检查失败，工具会自动回滚到旧版本。

**手动回滚**：
```bash
cd /home/new-api
docker compose down
docker tag 旧镜像ID calciumion/new-api:latest
docker compose up -d
```

### 4. 如何修改 NewAPI 端口？
编辑配置文件 `/opt/newapi-tools/config/config.yaml`：
```yaml
deploy:
  newapi:
    port: 8080  # 修改为 8080
```

然后重新部署：
```bash
newapi-tools install
```

---

## 📞 获取帮助

- GitHub Issues: https://github.com/simenty/newapi-tools/issues
- 文档: https://docs.newapi.pro/

---

**祝使用愉快！ 🎉**
