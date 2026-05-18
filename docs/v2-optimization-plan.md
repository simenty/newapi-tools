# NewAPI Tools v2.0 架构优化与用户体验升级方案

> **文档版本**: v1.0  
> **日期**: 2026-05-13  
> **作者**: Vincent  
> **目标**: 基于官方 NewAPI 文档理念，重新设计工具架构，提升用户体验，实现从"能用"到"好用"的跨越

---

## 一、现状分析

### 1.1 当前优势（v1.2）
✅ 代码质量高（`set -euo pipefail`、幂等性检查、错误处理）  
✅ 功能覆盖完整（环境初始化、部署、备份、更新、SSL）  
✅ 安全设计到位（`.env` 权限 600、密码不回显、二次确认）  
✅ 备份机制可靠（MD5 校验、自动回滚）  

### 1.2 当前不足
❌ **用户体验偏向专家**：交互式菜单信息密集，小白易被吓到  
❌ **缺少状态管理**：不知道当前系统处于什么状态（装了吗？配置了吗？）  
❌ **配置分散**：`.env`、`toolkit.conf`、`profiles/` 多个地方  
❌ **错误提示太技术化**：用户看到 `awk: line 2: == 0: syntax error` 会懵  
❌ **缺少进度反馈**：长时间操作（如拉取镜像）没有进度条  
❌ **不支持多种部署方式**：官方支持 6 种，目前只支持 Docker Compose  

---

## 二、核心设计理念（基于官方文档）

### 2.1 官方文档的核心理念
1. **多种部署方式**：适应不同用户群体（新手 → 专家 → 企业）
2. **合规优先**：明确公示合规要求
3. **生态思维**：不仅仅是一个工具，而是整个生态的一部分
4. **文档驱动**：完善的文档降低使用门槛

### 2.2 我们的设计理念
```
新手友好 ←─────────────────────→ 专家可用
   易用性    功能性    灵活性
```

**核心理念：**
1. **渐进式复杂度**：默认简化，高级功能可选
2. **状态可视化**：随时知道系统处于什么状态
3. **配置集中化**：一个地方搞定所有配置
4. **操作可逆**：任何操作前先备份，出错了能恢复
5. **社区驱动**：支持插件扩展

---

## 三、架构现代化方案

### 3.1 架构对比

#### 当前架构（v1.2）
```
newapi-tools.sh          # 主入口
├── lib/                # 公共函数
├── config/             # 配置文件（分散）
├── modules/            # 功能模块
└── scripts/            # 辅助脚本
```

**问题：**
- 状态不持久化（每次都重新检测）
- 配置分散在多个文件
- 难以扩展（新增功能需要改主脚本）

#### 目标架构（v2.0）
```
newapi-tools/
├── bin/
│   └── newapi-tools          # 主入口（统一）
├── lib/                      # 公共函数库
│   ├── common.sh            # 日志、确认、校验
│   ├── ui.sh               # UI 组件（新增）
│   ├── state.sh            # 状态管理（新增）
│   ├── config.sh           # 配置管理（新增）
│   └── plugins.sh         # 插件管理（新增）
├── config/
│   ├── system.yaml         # 系统配置（统一格式）
│   ├── newapi.yaml         # NewAPI 配置
│   └── security.yaml      # 安全配置
├── state/
│   └── state.json         # 状态文件（新增）
├── modules/                # 功能模块（保持不变）
├── plugins/                # 插件目录（新增）
│   ├── 1panel/           # 1Panel 部署插件
│   ├── bt-panel/         # 宝塔面板插件
│   └── wechat-notify/    # 微信通知插件
├── docs/                   # 文档（新增）
│   ├── FAQ.md
│   ├── TROUBLESHOOTING.md
│   └── video-tutorials.md
└── data/                   # 数据目录（新增）
    ├── backups/           # 备份存放
    ├── snapshots/         # 系统快照
    └── cache/             # 缓存
```

---

### 3.2 核心改进点

#### **改进 1：状态管理（State Management）**

**问题**：当前每次执行都重新检测系统状态，效率低且用户不知道当前状态。

**方案**：引入 `state.json` 状态文件

```json
// state/state.json
{
  "version": "2.0",
  "installed": true,
  "newapi_version": "latest",
  "deploy_method": "docker-compose",  // 部署方式
  "mysql_password_set": true,
  "ssl_enabled": false,
  "last_backup": "2026-05-13T01:30:00Z",
  "health_status": "healthy",
  "system": {
    "dns_configured": true,
    "docker_installed": true,
    "sources_replaced": true
  },
  "containers": {
    "new-api": "running",
    "mysql": "running",
    "redis": "running",
    "npm": "running"
  }
}
```

**好处：**
- 快速查询状态（不用每次都 `docker ps`）
- 支持断点续装（中断后继续执行）
- 可视化系统状态（`newapi-tools status`）

**实现：**
```bash
# lib/state.sh

STATE_FILE="${TOOLKIT_ROOT}/state/state.json"

# 初始化状态文件
init_state() {
  if [[ ! -f "$STATE_FILE" ]]; then
    cat > "$STATE_FILE" << 'EOF'
{
  "version": "2.0",
  "installed": false,
  "last_updated": ""
}
EOF
  fi
}

# 读取状态
get_state() {
  local key="$1"
  jq -r ".$key // empty" "$STATE_FILE"
}

# 更新状态
set_state() {
  local key="$1"
  local value="$2"
  local tmp_file=$(mktemp)
  jq ".$key = \"$value\"" "$STATE_FILE" > "$tmp_file"
  mv "$tmp_file" "$STATE_FILE"
  log_info "状态已更新: $key = $value"
}
```

---

#### **改进 2：配置统一管理（Configuration Management）**

**问题**：当前配置分散在 `.env`、`toolkit.conf`、`profiles/` 多个地方，用户不知道改哪个。

**方案**：统一使用 YAML 格式配置文件

```yaml
# config/system.yaml
dns:
  primary: "223.5.5.5"
  secondary: "223.6.6.6"

docker:
  mirror: "https://mirror.ccs.tencentyun.com"
  compose_command: "docker compose"

backup:
  retention_days: 30
  remote_backup: false
  remote_url: ""

notification:
  webhook_url: ""
  channels: ["feishu", "dingtalk"]
```

**实现：**
```bash
# lib/config.sh

# 读取配置（支持 YAML）
get_config() {
  local key="$1"
  local config_file="${TOOLKIT_ROOT}/config/system.yaml"
  yq -r ".$key // empty" "$config_file"
}

# 更新配置
set_config() {
  local key="$1"
  local value="$2"
  local config_file="${TOOLKIT_ROOT}/config/system.yaml"
  local tmp_file=$(mktemp)
  yq ".$key = \"$value\"" "$config_file" > "$tmp_file"
  mv "$tmp_file" "$config_file"
  log_info "配置已更新: $key = $value"
}
```

**依赖：** 需要安装 `yq`（YAML 处理工具）
```bash
# 自动安装 yq
install_yq() {
  if ! command -v yq &>/dev/null; then
    log_info "正在安装 yq（YAML 处理工具）..."
    wget -qO /usr/local/bin/yq https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64
    chmod +x /usr/local/bin/yq
  fi
}
```

---

#### **改进 3：UI 组件库（User Interface Library）**

**问题**：当前输出都是纯文本，缺少视觉反馈。

**方案**：引入 `lib/ui.sh`，提供统一的 UI 组件

```bash
# lib/ui.sh

# 显示标题横幅
show_banner() {
  clear
  cat << 'EOF'
   _   _      _   ____  ____    _
  | \ | | ___| |_|___ \| ___|__| |__
  |  \| |/ _ \ __|___) | |_ / _ \ '_ \
  | |\  |  __/ |_/ __/|  _|  __/ |_) |
  |_| \_|\___|\__|_____|_|  \___|_.__/

  NewAPI 运维工具集 v2.0
  https://github.com/simenty/newapi-tools

EOF
}

# 显示进度条
show_progress() {
  local msg="$1"
  local percent="$2"
  local width=50
  local filled=$((percent * width / 100))
  local empty=$((width - filled))
  
  printf "\r${BLUE}${msg}${PLAIN} ["
  printf "%0.s=" $(seq 1 $filled)
  printf "%0.s-" $(seq 1 $empty)
  printf "] %d%%" $percent
  
  if [[ $percent -eq 100 ]]; then
    echo -e "\n${GREEN}✓ 完成${PLAIN}"
  fi
}

# 显示成功/失败/警告/信息
ui_success() { echo -e "${GREEN}✓ $*${PLAIN}"; }
ui_error()   { echo -e "${RED}✗ $*${PLAIN}" >&2; }
ui_warn()    { echo -e "${YELLOW}⚠ $*${PLAIN}"; }
ui_info()    { echo -e "${BLUE}ℹ $*${PLAIN}"; }

# 显示步骤进度（1/5, 2/5, ...）
show_step() {
  local current="$1"
  local total="$2"
  local msg="$3"
  echo -e "\n${BLUE}[步骤 ${current}/${total}]${PLAIN} ${msg}"
}

# 询问是/否（简化版）
ask_yn() {
  local msg="$1"
  local default="${2:-n}"
  local prompt="[y/n]"
  
  [[ "$default" == "y" ]] && prompt="[Y/n]"
  [[ "$default" == "n" ]] && prompt="[y/N]"
  
  read -r -p "$msg $prompt " answer
  answer=${answer:-$default}
  
  [[ "$answer" =~ ^[Yy]$ ]]
}

# 显示菜单（增强版）
show_menu() {
  local title="$1"
  shift
  local options=("$@")
  
  clear
  echo -e "${GREEN}===== ${title} =====${PLAIN}\n"
  
  local i=1
  for opt in "${options[@]}"; do
    echo -e "  ${YELLOW}${i})${PLAIN} ${opt}"
    ((i++))
  done
  
  echo ""
}
```

**效果对比：**

| 当前输出 | 优化后输出 |
|---------|-------------|
| `正在安装 Docker...` | `ℹ [步骤 1/5] 正在安装 Docker...` + 进度条 |
| `错误：Docker 未安装` | `✗ Docker 未安装，正在自动安装...` |
| `安装完成` | `✓ Docker 安装完成（版本 24.0.5）` |

---

#### **改进 4：智能默认值（Smart Defaults）**

**问题**：当前需要用户手动输入所有信息（MySQL 密码、域名等），容易出错。

**方案**：自动生成合理默认值，用户只需确认或修改

```bash
# lib/smart-defaults.sh

# 生成强密码
generate_password() {
  local length="${1:-32}"
  openssl rand -base64 48 | tr -dc 'a-zA-Z0-9' | head -c "$length"
}

# 推测域名（基于 /etc/hostname 或公网 IP）
suggest_domain() {
  local hostname=$(hostname)
  local public_ip=$(curl -s ifconfig.me || echo "your-server-ip")
  
  # 如果 hostname 看起来像域名，直接使用
  if [[ "$hostname" =~ \. ]]; then
    echo "$hostname"
  else
    echo "api.example.com"
    ui_info "请将 api.example.com 替换为你的实际域名"
  fi
}

# 智能推荐配置
suggest_config() {
  local mem_total=$(free -m | awk '/Mem:/ {print $2}')
  local cpu_cores=$(nproc)
  
  ui_info "检测到系统配置："
  echo "  - 内存: ${mem_total}MB"
  echo "  - CPU: ${cpu_cores} 核"
  
  # 根据配置推荐
  if [[ $mem_total -lt 2048 ]]; then
    ui_warn "内存较小，建议关闭不必要的服务（如 NPM）"
    echo "推荐配置："
    echo "  - 不安装 Nginx Proxy Manager"
    echo "  - 使用 Caddy 作为反向代理（更轻量）"
  fi
}
```

**使用场景：**
```bash
# 安装部署时
read -r -p "MySQL 密码 [默认: $(generate_password 16)]: " MYSQL_PASS
MYSQL_PASS=${MYSQL_PASS:-$(generate_password 16)}

read -r -p "域名 [默认: $(suggest_domain)]: " DOMAIN
DOMAIN=${DOMAIN:-$(suggest_domain)}
```

---

#### **改进 5：新手模式 vs 专家模式**

**问题**：当前所有用户看到的是同一个菜单，新手会被高级选项吓到。

**方案**：根据用户输入自动判断或手动选择模式

```bash
# lib/mode.sh

CURRENT_MODE=""

# 检测用户水平
detect_user_level() {
  local experience=0
  
  # 检查用户是否知道某些命令
  command -v vim &>/dev/null && ((experience++))
  command -v git &>/dev/null && ((experience++))
  command -v docker &>/dev/null && ((experience++))
  grep -q "Docker" /etc/group 2>/dev/null && ((experience++))
  
  if [[ $experience -ge 3 ]]; then
    echo "expert"
  elif [[ $experience -ge 1 ]]; then
    echo "intermediate"
  else
    echo "beginner"
  fi
}

# 设置模式
set_mode() {
  local mode="$1"
  
  case "$mode" in
    beginner)
      CURRENT_MODE="beginner"
      # 简化菜单，减少选项
      export MENU_STYLE="simple"
      ;;
    expert)
      CURRENT_MODE="expert"
      # 显示所有高级选项
      export MENU_STYLE="full"
      ;;
    *)
      # 自动检测
      CURRENT_MODE=$(detect_user_level)
      ;;
  esac
  
  log_info "当前模式: $CURRENT_MODE"
}

# 根据模式显示不同内容
show_by_mode() {
  local beginner_msg="$1"
  local expert_msg="$2"
  
  if [[ "$CURRENT_MODE" == "beginner" ]]; then
    echo "$beginner_msg"
  else
    echo "$expert_msg"
  fi
}
```

**效果：**

| 功能 | 新手模式 | 专家模式 |
|------|---------|---------|
| 安装部署 | 一键安装（全默认） | 自定义每个参数 |
| 备份 | 自动备份（每日） | 手动备份 + 自定义策略 |
| 配置 | 图形化向导 | 直接编辑配置文件 |
| 日志 | 只看错误信息 | 完整日志 + 过滤选项 |

---

### 3.3 插件机制（Plugin System）

**问题**：当前所有功能都写在主仓库里，难以扩展。

**方案**：引入插件机制，让社区可以贡献功能

```bash
# lib/plugins.sh

PLUGINS_DIR="${TOOLKIT_ROOT}/plugins"

# 加载所有插件
load_plugins() {
  if [[ -d "$PLUGINS_DIR" ]]; then
    for plugin in "$PLUGINS_DIR"/*/plugin.sh; do
      if [[ -f "$plugin" ]]; then
        source "$plugin"
        log_info "已加载插件: $(basename $(dirname $plugin))"
      fi
    done
  fi
}

# 安装插件
install_plugin() {
  local plugin_url="$1"
  local plugin_name=$(basename "$plugin_url" .git)
  
  mkdir -p "$PLUGINS_DIR"
  git clone "$plugin_url" "$PLUGINS_DIR/$plugin_name"
  
  log_success "插件已安装: $plugin_name"
  log_info "请重新启动 newapi-tools 以加载插件"
}

# 插件示例：wechat-notify/plugin.sh
# plugins/wechat-notify/plugin.sh
plugin_name="wechat-notify"
plugin_version="1.0"
plugin_author="Vincent"

# 插件入口函数（会挂载到主菜单）
plugin_main() {
  echo "微信通知插件"
  read -r -p "输入企业微信 Webhook URL: " webhook_url
  set_config "notification.wechat_webhook" "$webhook_url"
  log_success "微信通知已配置"
}

# 注册插件到菜单
register_plugin() {
  # 这个函数在插件加载时被执行
  # 将插件添加到主菜单的 "插件" 子菜单
  echo "$plugin_name" >> "${TOOLKIT_ROOT}/data/plugins_menu.txt"
}
```

**插件示例：1Panel 部署插件**

```bash
# plugins/1panel/plugin.sh

plugin_main() {
  show_banner
  echo "1Panel 部署模式"
  echo "--------------"
  echo ""
  echo "1Panel 是一款现代化的 Linux 服务器运维管理面板"
  echo "使用 1Panel 部署 NewAPI 可以享受："
  echo "  - 图形化界面操作"
  echo "  - 一键安装应用"
  echo "  - 方便的日志查看"
  echo ""
  
  if ask_yn "是否继续？"; then
    # 安装 1Panel
    log_info "正在安装 1Panel..."
    curl -sSL https://resource.fit2cloud.com/1panel/quick_start.sh -o quick_start.sh
    bash quick_start.sh
    
    log_success "1Panel 安装完成！"
    log_info "请访问 https://$(hostname -I | awk '{print $1}'):$(grep PORT /opt/1panel/configs/1panel.conf | cut -d'=' -f2) 继续配置"
  fi
}
```

---

## 四、功能扩展（参考官方文档）

### 4.1 支持多种部署方式

官方文档支持 6 种部署方式，我们逐步实现：

| 部署方式 | 优先级 | 实现难度 | 说明 |
|---------|--------|---------|------|
| Docker Compose | ✅ 已完成 | - | 当前默认方式 |
| Docker 单容器 | 🔥 高 | 低 | 适合快速测试 |
| 1Panel 面板 | 🔥 高 | 中 | 新手友好 |
| 宝塔面板 | 中 | 中 | 国内用户多 |
| 集群部署 | 低 | 高 | 企业级功能 |
| 本地开发 | 低 | 中 | 开发者用 |

#### **实现：Docker 单容器部署**

```bash
# modules/deploy/docker-simple.sh

deploy_docker_simple() {
  show_step 1 3 "检查 Docker"
  require_docker
  
  show_step 2 3 "生成随机密码"
  local mysql_pass=$(generate_password 16)
  
  show_step 3 3 "启动容器"
  docker run -d \
    --name new-api-simple \
    -p 127.0.0.1:3000:3000 \
    -e SQL_DSN="root:${mysql_pass}@tcp(db:3306)/newapi" \
    -e REDIS_CONN_STRING="redis://redis:6379" \
    -e SESSION_SECRET="$(generate_password 32)" \
    --restart always \
    calciumion/new-api:latest
  
  log_success "NewAPI 单容器模式已启动"
  log_info "访问地址: <ADDRESS_REDACTED>
  ui_warn "注意：此模式不适合生产环境"
}
```

---

### 4.2 配置向导（Configuration Wizard）

**问题**：当前配置分散，用户不知道该改哪里。

**方案**：增加 `newapi-tools config` 命令，提供交互式配置向导

```bash
# modules/config/wizard.sh

config_wizard() {
  show_banner
  echo "配置向导"
  echo "=========="
  echo ""
  
  # 步骤 1：基本信息
  show_step 1 4 "基本信息"
  read -r -p "域名: " DOMAIN
  read -r -p "端口 [默认: 3000]: " PORT
  PORT=${PORT:-3000}
  
  # 步骤 2：数据库配置
  show_step 2 4 "数据库配置"
  read -r -p "MySQL 密码 [自动生成]: " MYSQL_PASS
  MYSQL_PASS=${MYSQL_PASS:-$(generate_password 16)}
  
  # 步骤 3：安全配置
  show_step 3 4 "安全配置"
  read -r -p "开启 HTTPS？(y/n) [默认: y]: " ENABLE_HTTPS
  ENABLE_HTTPS=${ENABLE_HTTPS:-y}
  
  # 步骤 4：备份策略
  show_step 4 4 "备份策略"
  read -r -p "自动备份 (y/n) [默认: y]: " AUTO_BACKUP
  AUTO_BACKUP=${AUTO_BACKUP:-y}
  
  if [[ "$AUTO_BACKUP" == "y" ]]; then
    read -r -p "备份保留天数 [默认: 30]: " RETENTION
    RETENTION=${RETENTION:-30}
  fi
  
  # 生成配置文件
  cat > config/newapi.yaml << EOF
domain: "$DOMAIN"
port: $PORT
mysql_password: "$MYSQL_PASS"
enable_https: $([[ "$ENABLE_HTTPS" == "y" ]] && echo "true" || echo "false")
backup:
  auto_backup: $([[ "$AUTO_BACKUP" == "y" ]] && echo "true" || echo "false")
  retention_days: $RETENTION
EOF
  
  log_success "配置已生成: config/newapi.yaml"
}
```

---

### 4.3 迁移工具（Migration Tools）

**场景**：用户之前用的是 One API，想迁移到 NewAPI。

```bash
# modules/migrate/one-api-to-new-api.sh

migrate_one_api_to_new_api() {
  show_banner
  echo "One API → NewAPI 迁移工具"
  echo "=========================="
  echo ""
  
  ui_warn "迁移前会自动备份 One API 数据，迁移失败可恢复"
  
  if ! ask_yn "确定要迁移吗？"; then
    return 0
  fi
  
  # 步骤 1：备份 One API
  show_step 1 4 "备份 One API 数据"
  docker exec one-api mysqldump -uroot -p"$ONE_API_MYSQL_PASS" oneapi > one-api-backup.sql
  generate_checksum "one-api-backup.sql"
  
  # 步骤 2：转换数据格式
  show_step 2 4 "转换数据格式"
  # NewAPI 和 One API 的数据库结构略有不同
  # 需要编写转换脚本（这里简化）
  log_info "正在转换渠道配置..."
  log_info "正在转换用户数据..."
  
  # 步骤 3：部署 NewAPI
  show_step 3 4 "部署 NewAPI"
  bash "${MODULES_DIR}/deploy/install.sh"
  
  # 步骤 4：导入数据
  show_step 4 4 "导入数据到 NewAPI"
  docker exec -i new-api mysql -uroot -p"$MYSQL_PASS" newapi < converted-data.sql
  
  log_success "迁移完成！"
  ui_info "请访问 http://${DOMAIN}:${PORT} 验证"
}
```

---

## 五、分阶段实施计划

### **Phase 1: 用户体验优化（1-2 周，立即可做）**

| 任务 | 优先级 | 工作量 | 说明 |
|------|--------|--------|------|
| 引入 `lib/ui.sh` | 🔥 高 | 0.5 天 | 进度条、彩色输出、emoji |
| 智能默认值 | 🔥 高 | 1 天 | 自动生成密码、推测域名 |
| 优化错误提示 | 🔥 高 | 1 天 | 技术错误 → 人话 |
| 增加 ASCII 艺术横幅 | 中 | 0.5 天 | 提升品牌感 |
| 新手/专家模式 | 中 | 2 天 | 自动检测用户水平 |

**目标：** 让小白也能轻松上手

---

### **Phase 2: 架构现代化（2-3 周）**

| 任务 | 优先级 | 工作量 | 说明 |
|------|--------|--------|------|
| 状态管理（`state.json`） | 🔥 高 | 2 天 | 跟踪系统状态 |
| 配置统一管理（YAML） | 🔥 高 | 3 天 | 统一配置文件格式 |
| 幂等性检查优化 | 高 | 1 天 | 基于状态文件 |
| 断点续装 | 中 | 2 天 | 中断后继续执行 |
| 插件机制 | 中 | 3 天 | 支持社区扩展 |

**目标：** 架构更清晰，易于维护和扩展

---

### **Phase 3: 功能扩展（3-4 周）**

| 任务 | 优先级 | 工作量 | 说明 |
|------|--------|--------|------|
| Docker 单容器部署 | 🔥 高 | 1 天 | 快速测试 |
| 1Panel 部署插件 | 🔥 高 | 3 天 | 新手友好 |
| 配置向导 | 高 | 2 天 | 交互式配置 |
| 迁移工具（One API） | 中 | 3 天 | 帮助用户迁移 |
| 远程备份（S3） | 中 | 3 天 | 备份到云存储 |
| 监控栈（Prometheus） | 低 | 5 天 | 可选安装 |

**目标：** 功能更完整，覆盖更多使用场景

---

### **Phase 4: 社区与生态（持续进行）**

| 任务 | 优先级 | 工作量 | 说明 |
|------|--------|--------|------|
| 完善文档（FAQ、视频） | 🔥 高 | 持续 | 降低使用门槛 |
| 插件市场 | 中 | 5 天 | 让用户可以分享插件 |
| GitHub Actions（CI/CD） | 中 | 2 天 | 自动测试、发布 |
| 多语言支持（i18n） | 低 | 5 天 | 支持英文、中文 |

**目标：** 建立社区生态

---

## 六、具体实施：Phase 1 详细设计

### 6.1 引入 `lib/ui.sh`

**文件：** `lib/ui.sh`

```bash
#!/bin/bash
# UI 组件库 —— 提供统一的用户界面组件

# 颜色定义（如果未定义）
: ${GREEN:='\033[32m'}
: ${RED:='\033[31m'}
: ${YELLOW:='\033[33m'}
: ${BLUE:='\033[34m'}
: ${PLAIN:='\033[0m'}

# 显示标题横幅
show_banner() {
  clear
  cat << 'EOF'
   _   _      _   ____  ____    _
  | \ | | ___| |_|___ \| ___|__| |__
  |  \| |/ _ \ __|___) | |_ / _ \ '_ \
  | |\  |  __/ |_/ __/|  _|  __/ |_) |
  |_| \_|\___|\__|_____|_|  \___|_.__/

  NewAPI 运维工具集 v2.0
  https://github.com/simenty/newapi-tools

EOF
}

# 显示进度条
show_progress() {
  local msg="$1"
  local percent="$2"
  local width=50
  local filled=$((percent * width / 100))
  local empty=$((width - filled))
  
  printf "\r${BLUE}${msg}${PLAIN} ["
  printf "%0.s=" $(seq 1 $filled 2>/dev/null || echo "")
  printf "%0.s-" $(seq 1 $empty 2>/dev/null || echo "")
  printf "] %d%%" $percent
  
  # 处理 seq 在某些系统不兼容的问题
  if [[ $percent -eq 100 ]]; then
    echo -e "\n${GREEN}✓ 完成${PLAIN}"
  fi
}

# 成功/失败/警告/信息 提示
ui_success() { echo -e "${GREEN}✓ $*${PLAIN}"; }
ui_error()   { echo -e "${RED}✗ $*${PLAIN}" >&2; }
ui_warn()    { echo -e "${YELLOW}⚠ $*${PLAIN}"; }
ui_info()    { echo -e "${BLUE}ℹ $*${PLAIN}"; }

# 显示步骤进度
show_step() {
  local current="$1"
  local total="$2"
  local msg="$3"
  echo -e "\n${BLUE}[步骤 ${current}/${total}]${PLAIN} ${msg}"
}

# 询问是/否（简化版）
ask_yn() {
  local msg="$1"
  local default="${2:-n}"
  local prompt=""
  
  if [[ "$default" == "y" ]]; then
    prompt="[Y/n]"
  else
    prompt="[y/N]"
  fi
  
  read -r -p "$msg $prompt " answer
  answer=${answer:-$default}
  
  [[ "$answer" =~ ^[Yy]$ ]]
}

# 显示菜单（增强版）
show_menu() {
  local title="$1"
  shift
  local options=("$@")
  
  clear
  echo -e "${GREEN}===== ${title} =====${PLAIN}\n"
  
  local i=1
  for opt in "${options[@]}"; do
    echo -e "  ${YELLOW}${i})${PLAIN} ${opt}"
    ((i++))
  done
  
  echo ""
}

# 显示状态板（Dashboard）
show_dashboard() {
  clear
  show_banner
  
  echo -e "${BLUE}----- 系统状态 -----${PLAIN}"
  
  # 读取状态文件
  if [[ -f "${TOOLKIT_ROOT}/state/state.json" ]]; then
    local installed=$(jq -r '.installed' "${TOOLKIT_ROOT}/state/state.json")
    local health=$(jq -r '.health_status' "${TOOLKIT_ROOT}/state/state.json")
    
    if [[ "$installed" == "true" ]]; then
      ui_success "NewAPI 已安装"
    else
      ui_warn "NewAPI 未安装"
    fi
    
    if [[ "$health" == "healthy" ]]; then
      ui_success "系统健康状态：正常"
    else
      ui_error "系统健康状态：异常"
    fi
  else
    ui_warn "状态文件不存在，请先执行环境初始化"
  fi
  
  echo ""
}
```

---

### 6.2 智能默认值

**文件：** `lib/smart-defaults.sh`

```bash
#!/bin/bash
# 智能默认值生成器

# 生成强密码
generate_password() {
  local length="${1:-32}"
  
  # 检查 openssl 是否可用
  if command -v openssl &>/dev/null; then
    openssl rand -base64 48 | tr -dc 'a-zA-Z0-9' | head -c "$length"
  else
    # 备用方案
    cat /dev/urandom | tr -dc 'a-zA-Z0-9' | head -c "$length"
  fi
}

# 推测域名
suggest_domain() {
  local hostname=$(hostname 2>/dev/null || echo "server")
  local public_ip=$(curl -s --max-time 3 ifconfig.me 2>/dev/null || echo "127.0.0.1")
  
  # 如果 hostname 看起来像域名，直接使用
  if [[ "$hostname" =~ \. ]] && [[ ! "$hostname" =~ ^localhost ]]; then
    echo "$hostname"
  else
    echo "api.example.com"
  fi
}

# 智能推荐配置
suggest_config() {
  local mem_total=$(free -m 2>/dev/null | awk '/Mem:/ {print $2}')
  local cpu_cores=$(nproc 2>/dev/null || echo "1")
  
  echo "检测到系统配置："
  echo "  - 内存: ${mem_total}MB"
  echo "  - CPU: ${cpu_cores} 核"
  echo ""
  
  # 根据配置推荐
  if [[ ${mem_total:-0} -lt 2048 ]]; then
    ui_warn "内存较小（< 2GB），建议："
    echo "  - 不安装 Nginx Proxy Manager"
    echo "  - 使用 Caddy 作为反向代理（更轻量）"
    echo "  - 或升级服务器内存"
  fi
  
  if [[ ${cpu_cores} -lt 2 ]]; then
    ui_warn "CPU 核心数较少，部署时间可能较长"
  fi
}
```

---

### 6.3 优化错误提示

**问题：** 当前错误信息太技术化

**方案：** 提供"技术错误 → 用户友好提示"的映射

```bash
# lib/error-handler.sh

# 用户友好的错误提示
show_friendly_error() {
  local error_msg="$1"
  local solution="${2:-}"
  
  echo -e "\n${RED}✗ 出错了${PLAIN}"
  echo ""
  echo "问题：$error_msg"
  
  if [[ -n "$solution" ]]; then
    echo ""
    echo "解决方法："
    echo "$solution" | while IFS= read -r line; do
      echo "  ${line}"
    done
  fi
  
  echo ""
  echo "如果问题仍然存在，请："
  echo "  1. 查看日志: cat ${LOG_FILE}"
  echo "  2. 开 Issue: https://github.com/simenty/newapi-tools/issues"
}

# 常见错误映射
handle_common_errors() {
  local error_output="$1"
  
  if echo "$error_output" | grep -q "permission denied"; then
    show_friendly_error \
      "权限不足" \
      "- 请使用 root 用户执行本脚本\n- 或使用 sudo bash $0"
    return 1
  fi
  
  if echo "$error_output" | grep -q "command not found"; then
    local missing_cmd=$(echo "$error_output" | grep "command not found" | awk '{print $1}')
    show_friendly_error \
      "缺少命令: $missing_cmd" \
      "- 执行: apt install $missing_cmd\n- 或参考文档手动安装"
    return 1
  fi
  
  if echo "$error_output" | grep -q "port is already allocated"; then
    show_friendly_error \
      "端口被占用" \
      "- 检查哪个程序占用了端口: netstat -tulpn | grep <端口号>\n- 停止占用端口的程序\n- 或修改 NewAPI 配置文件使用其他端口"
    return 1
  fi
  
  # 未知错误，显示原始信息
  show_friendly_error "$error_output"
  return 1
}
```

---

### 6.4 增加 ASCII 艺术横幅

**文件：** `lib/banner.sh`

```bash
#!/bin/bash
# ASCII 艺术横幅生成器

# 显示主横幅
show_main_banner() {
  clear
  cat << 'EOF'
   _   _      _   ____  ____    _
  | \ | | ___| |_|___ \| ___|__| |__
  |  \| |/ _ \ __|___) | |_ / _ \ '_ \
  | |\  |  __/ |_/ __/|  _|  __/ |_) |
  |_| \_|\___|\__|_____|_|  \___|_.__/

  NewAPI 运维工具集 v2.0
  https://github.com/simenty/newapi-tools

EOF
}

# 显示子模块横幅
show_section_banner() {
  local title="$1"
  local version="${2:-}"
  
  echo ""
  echo "=========================================="
  echo "  $title"
  [[ -n "$version" ]] && echo "  Version: $version"
  echo "=========================================="
  echo ""
}

# 显示成功安装的横幅
show_success_banner() {
  cat << 'EOF'

   _____   ____  ____    _____  _    _  _   _ 
  | ____| / ___||  _ \  |  ___|| |  | || \ | |
  |  _|  | |    | |_) | | |_   | |  | ||  \| |
  | |___ | |___ |  _ <  |  _|  | |__| || |\  |
  |_____| \____||_| \_\ |_|     \____/ |_| \_|

EOF
}
```

**生成工具：** 可以使用 http://patorjk.com/software/taag/ 生成更多样式

---

## 七、总结与下一步

### 7.1 核心改进总结

| 改进点 | 当前状态 | 目标状态 | 优先级 |
|--------|---------|---------|--------|
| 用户体验 | 专家级 | 新手友好 + 专家可用 | 🔥 高 |
| 状态管理 | 无 | 完整的状态跟踪 | 🔥 高 |
| 配置管理 | 分散 | 统一 YAML 配置 | 🔥 高 |
| 错误处理 | 技术化 | 用户友好提示 | 🔥 高 |
| 功能覆盖 | Docker Compose | 6 种部署方式 | 中 |
| 扩展性 | 无 | 插件机制 | 中 |
| 社区生态 | 无 | 文档 + 插件市场 | 低 |

### 7.2 下一步行动

**立即可做（本周）：**
1. ✅ 引入 `lib/ui.sh`（进度条、彩色输出）
2. ✅ 智能默认值（自动生成密码）
3. ✅ 优化错误提示（技术错误 → 人话）
4. ✅ 增加 ASCII 艺术横幅

**短期目标（本月）：**
1. 状态管理（`state.json`）
2. 配置统一管理（YAML）
3. 新手/专家模式

**长期目标（本季度）：**
1. 插件机制
2. 多种部署方式支持
3. 完善的文档和视频教程

---

## 八、附录：技术选型说明

### 8.1 为什么保留 Shell 而不是换成 Python/Go？

**理由：**
1. **依赖少**：Shell 脚本只需要 Bash，Python/Go 需要额外安装
2. **部署简单**：Shell 脚本可以直接 `curl | bash`，Python/Go 需要包管理
3. **透明性**：Shell 脚本用户可以看到每一行在干什么，更有安全感
4. **生态兼容**：服务器运维工具都是 Shell 写的（pve-tools、lnmt等）

**但是：**
- 复杂逻辑（如 JSON/YAML 处理）可以调用 `jq`/`yq`
- 如果需要更复杂的 UI，可以用 `whiptail` 或 `dialog`

### 8.2 为什么选择 YAML 而不是继续用 .env？

**理由：**
1. **结构化**：YAML 支持嵌套结构，`.env` 不支持
2. **可读性**：YAML 更适合人类阅读和编辑
3. **生态**：Kubernetes、Docker Compose、Ansible 都在用 YAML

**兼容性：**
- 保持对 `.env` 的向后兼容（读取时同时支持）
- 新配置写在 YAML 里

---

**文档结束**

> 下一步：选择 Phase 1 中的任意一个任务，我开始写代码！
