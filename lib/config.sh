#!/bin/bash
set -eo pipefail
# 配置统一管理模块 —— 替代分散的 .env 和 toolkit.conf
# 使用 YAML 格式配置文件，支持分层配置

# common.sh 和 security.sh 已通过 lib/_init.sh 按依赖顺序加载

CONFIG_DIR="${CONFIG_DIR:-${TOOLKIT_ROOT}/config}"
MAIN_CONFIG="${CONFIG_DIR}/config.yaml"
DEFAULT_CONFIG="${CONFIG_DIR}/config.default.yaml"

# ---------- 初始化配置目录 ----------
init_config() {
    mkdir -p "$CONFIG_DIR"
    
    # 创建默认配置（如果不存在）
    if [[ ! -f "$DEFAULT_CONFIG" ]]; then
        cat > "$DEFAULT_CONFIG" << 'EOF'
# NewAPI 运维工具集 - 默认配置
# 本文件包含所有可配置项的默认值，请勿直接修改
# 如需自定义，请编辑 config.yaml（会自动覆盖默认值）

# 基本信息
toolkit:
  name: "NewAPI Tools"
  version: "2.0"
  mode: "novice"  # novice | expert

# 部署配置
deploy:
  flavor: "newapi"   # newapi | one-api | sub2api
  version: "latest"
  work_dir: "/home/new-api"
  
  # 数据库配置
  mysql:
    image: "mysql:8.0"
    root_password: ""  # 自动生成
    database: "new-api"
    user: "newapi"
    password: ""  # 自动生成
  
  # Redis 配置
  redis:
    image: "redis:7.2"
    password: ""  # 自动生成
  
  # NewAPI 配置（flavor=newapi 时使用）
  newapi:
    image: "calciumion/new-api:latest"
    port: 3000
    session_secret: ""  # 自动生成
  
  # One API 配置（flavor=one-api 时使用）
  one_api:
    image: "justsong/one-api:latest"
    port: 3000
    session_secret: ""  # 自动生成
    sqlite_enabled: true   # 默认使用 SQLite（无需 MySQL）

  # Sub2API 配置（flavor=sub2api 时使用）
  sub2api:
    image: "weishaw/sub2api:latest"
    port: 8080
    session_secret: ""  # 自动生成

  # NPM 配置
  npm:
    enabled: true
    image: "jc21/nginx-proxy-manager:latest"
    port: 81

# 备份配置
backup:
  dir: "/opt/newapi-tools/backups"
  retention_days: 7
  compress: true
  encrypt: false

# 通知配置
notification:
  webhook_url: ""
  webhook_type: "auto"  # auto | feishu | dingtalk | slack | custom
  notify_on_success: true
  notify_on_error: true

# 日志配置
log:
  dir: "/opt/newapi-tools/logs"
  level: "info"  # debug | info | warn | error
  retention_days: 30

# 监控配置
monitor:
  health_check_interval: 300  # 秒
  auto_restart: false
  alert_cpu_threshold: 80  # %
  alert_mem_threshold: 80  # %
  alert_disk_threshold: 85  # %
  alert_container_check: true
EOF
        log_info "默认配置文件已创建: $DEFAULT_CONFIG"
    fi
    
    # 创建用户配置（如果不存在）
    if [[ ! -f "$MAIN_CONFIG" ]]; then
        cat > "$MAIN_CONFIG" << 'EOF'
# NewAPI 运维工具集 - 用户配置
# 在此文件中覆盖默认配置
# 只需写需要修改的配置项

# 示例：修改部署目录
# deploy:
#   work_dir: "/data/new-api"

# 示例：启用 Webhook 通知
# notification:
#   webhook_url: "https://your-webhook-url"
EOF
        log_info "用户配置文件已创建: $MAIN_CONFIG"
    fi

    # 初始化兼容环境变量（原 env.sh 功能）
    _init_compat_env
}

# ---------- 读取配置值 ----------
get_config() {
    local key="$1"
    local default_value="${2:-}"
    local value=""
    
    # 优先级：环境变量 > 用户配置 > 默认配置 > 默认值
    
    # 0. 环境变量（最高优先级）
    local env_var
    env_var=$(echo "$key" | tr '.' '_' | tr '[:lower:]' '[:upper:]')
    if [[ -n "${!env_var:-}" ]]; then
        echo "${!env_var}"
        return 0
    fi
    
    # 1. 从用户配置读取（尝试多种方法，全部失败才降级）
    value=""
    if command -v yq &>/dev/null; then
        value=$(yq ".$key // empty" "$MAIN_CONFIG" 2>/dev/null || echo "")
    fi
    
    # 如果 yq 失败或不存在，尝试 Python
    if [[ -z "$value" ]] && command -v python3 &>/dev/null; then
        value=$(python3 -c "
import yaml
import sys
try:
    with open('$MAIN_CONFIG', 'r') as f:
        config = yaml.safe_load(f) or {}
    keys = '$key'.split('.')
    for k in keys:
        config = config.get(k, {})
    print(config if config and config != {} else '')
except:
    sys.exit(1)
" 2>/dev/null || echo "")
    fi
    
    # 如果 Python 也失败，尝试 grep（只支持一级配置）
    if [[ -z "$value" ]] && command -v grep &>/dev/null; then
        value=$(grep "^$key:" "$MAIN_CONFIG" 2>/dev/null | cut -d':' -f2- | sed 's/^ *//;s/"//g' || echo "")
    fi
    
    # 2. 如果用户配置中没有，从默认配置读取
    if [[ -z "$value" ]]; then
        if command -v yq &>/dev/null; then
            value=$(yq ".$key // empty" "$DEFAULT_CONFIG" 2>/dev/null || echo "")
        fi
        
        if [[ -z "$value" ]] && command -v python3 &>/dev/null; then
            value=$(python3 -c "
import yaml
try:
    with open('$DEFAULT_CONFIG', 'r') as f:
        config = yaml.safe_load(f) or {}
    keys = '$key'.split('.')
    for k in keys:
        config = config.get(k, {})
    print(config if config and config != {} else '')
except:
    pass
" 2>/dev/null || echo "")
        fi
        
        if [[ -z "$value" ]] && command -v grep &>/dev/null; then
            value=$(grep "^$key:" "$DEFAULT_CONFIG" 2>/dev/null | cut -d':' -f2- | sed 's/^ *//;s/"//g' || echo "")
        fi
    fi
    
    # 3. 如果都没有，使用默认值
    if [[ -z "$value" ]]; then
        value="$default_value"
    fi
    
    # 4. 去除可能的引号
    value=$(echo "$value" | sed 's/^"//;s/"$//')
    
    echo "$value"
}

# ---------- 写入配置值 ----------
set_config() {
    local key="$1"
    local value="$2"
    local key_path="$key"
    
    # 确保配置目录存在
    init_config
    
    # 尝试使用 yq 写入
    if command -v yq &>/dev/null; then
        local tmp_file
        tmp_file=$(secure_temp_file "yq")
        if yq eval ".$key = \"$value\"" "$MAIN_CONFIG" > "$tmp_file" 2>/dev/null; then
            mv "$tmp_file" "$MAIN_CONFIG"
            log_info "配置已更新: $key = $value"
            return 0
        else
            log_warn "yq 写入失败，尝试其他方法"
            rm -f "$tmp_file"
        fi
    fi
    
    # 降级方案 1：使用 Python（写入临时文件执行，避免 heredoc 问题）
    if command -v python3 &>/dev/null; then
        local py_tmp
        py_tmp=$(secure_temp_file "py")
        cat > "$py_tmp" << 'PYEOF'
import yaml
import sys

config_file = sys.argv[1]
key = sys.argv[2]
value = sys.argv[3]

with open(config_file, 'r') as f:
    config = yaml.safe_load(f) or {}

keys = key.split('.')
current = config
for i, k in enumerate(keys):
    if i == len(keys) - 1:
        current[k] = value
    else:
        current = current.setdefault(k, {})

with open(config_file, 'w') as f:
    yaml.dump(config, f, default_flow_style=False)
PYEOF
        
        if python3 "$py_tmp" "$MAIN_CONFIG" "$key" "$value" 2>/dev/null; then
            rm -f "$py_tmp"
            log_info "配置已更新（Python）: $key = $value"
            return 0
        else
            rm -f "$py_tmp"
            log_warn "Python 写入失败，尝试其他方法"
        fi
    fi
    
    # 降级方案 2：简单文本替换（只支持一级配置）
    if grep -q "^$key:" "$MAIN_CONFIG" 2>/dev/null; then
        # 转义特殊字符，防止命令注入
        local key_escaped value_escaped
        key_escaped=$(escape_sed_pattern "$key")
        value_escaped=$(escape_sed_pattern "$value")
        sed -i "s|^$key_escaped:.*|$key_escaped: $value_escaped|" "$MAIN_CONFIG"
        log_info "配置已更新（文本替换）: $key = $value"
        return 0
    else
        # 配置项不存在，追加到文件
        echo "$key: $value" >> "$MAIN_CONFIG"
        log_info "配置已追加: $key = $value"
        return 0
    fi
    
    log_error "无法写入配置: $key = $value"
    return 1
}

# ---------- 验证配置 ----------
validate_config() {
    local errors=0
    
    # 检查必要配置
    local required_keys=(
        "deploy.mysql.root_password"
        "deploy.mysql.password"
        "deploy.redis.password"
        "deploy.newapi.session_secret"
    )
    
    for key in "${required_keys[@]}"; do
        local value
        value=$(get_config "$key")
        if [[ -z "$value" ]]; then
            ui_error "配置缺失: $key"
            errors=$((errors + 1))
        fi
    done
    
    # 检查端口冲突
    local port
    port=$(get_config "deploy.newapi.port" "3000")
    if netstat -tlnp 2>/dev/null | grep -q ":$port "; then
        ui_warn "端口 $port 已被占用"
        errors=$((errors + 1))
    fi
    
    return $errors
}

# ---------- 显示当前配置 ----------
show_config() {
    echo -e "${UI_BOLD}${UI_CYAN}╔════════════════════════════════════════════════════════════╗${UI_PLAIN}"
    echo -e "${UI_BOLD}${UI_CYAN}║                        当前配置                              ║${UI_PLAIN}"
    echo -e "${UI_BOLD}${UI_CYAN}╚════════════════════════════════════════════════════════════╝${UI_PLAIN}"
    echo ""
    
    if command -v yq &>/dev/null; then
        echo "【用户配置】"
        yq '.' "$MAIN_CONFIG" 2>/dev/null || cat "$MAIN_CONFIG"
    else
        cat "$MAIN_CONFIG"
    fi
    
    echo ""
    echo "【配置位置】"
    echo "  用户配置: $MAIN_CONFIG"
    echo "  默认配置: $DEFAULT_CONFIG"
    echo ""
}

# ---------- 兼容性环境变量（原 lib/env.sh，V2.2 迁入）----------
# 从 config.yaml 读取值，如未设置则使用默认值
# 这些变量被其他模块广泛引用，必须保持可用
# ⚠️ 在 _init.sh 加载 config.sh 后自动调用，不依赖 init_config()
_init_compat_env() {
    # get_config 在 key 不存在时返回空字符串，需用 :+ 判断 + 硬编码默认值
    local _val

    _val=$(get_config 'deploy.newapi_home' 2>/dev/null) || _val=""
    : "${NEWAPI_HOME:=${_val:-/home/new-api}}"

    _val=$(get_config 'backup.retention_days' 2>/dev/null) || _val=""
    : "${BACKUP_RETENTION_DAYS:=${_val:-30}}"

    _val=$(get_config 'system.ssh_port' 2>/dev/null) || _val=""
    : "${SSH_PORT:=${_val:-2222}}"

    _val=$(get_config 'system.docker_compose_cmd' 2>/dev/null) || _val=""
    : "${DOCKER_COMPOSE_CMD:=${_val:-docker compose}}"

    _val=$(get_config 'system.log_dir' 2>/dev/null) || _val=""
    : "${LOG_DIR:=${_val:-${TOOLKIT_ROOT}/logs}}"

    unset _val

    export NEWAPI_HOME BACKUP_RETENTION_DAYS SSH_PORT DOCKER_COMPOSE_CMD LOG_DIR

    # 确保关键目录存在
    mkdir -p "${NEWAPI_HOME}/backups" "${LOG_DIR}" 2>/dev/null || true
}

# 加载时立即初始化兼容变量（不依赖 init_config 调用）
_init_compat_env

# ---------- 导出函数 ----------
export -f init_config get_config set_config
export -f validate_config show_config
