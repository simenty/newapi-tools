#!/bin/bash
set -eo pipefail
# 公共函数库 —— 资深开发优化版
# 修复：JSON 转义、Webhook 健壮性、trap 改进

# 设置默认 TOOLKIT_ROOT（如果未设置）
TOOLKIT_ROOT="${TOOLKIT_ROOT:-/tmp/newapi-tools}"

# 预定义日志函数（避免加载 security.sh 时 log_xxx 报 command not found）
# 完整版定义在第100行起，此处为兼容占位
log_debug()   { [[ "${DEBUG:-0}" == "1" ]] && echo "[DEBUG] $*" >&2 || true; }
log_info()    { echo "[INFO] $*"; }
log_success() { echo "[SUCCESS] $*"; }
log_warn()    { echo "[WARN] $*" >&2; }
log_error()   { echo "[ERROR] $*" >&2; }
log_audit()   { echo "[AUDIT] $*"; }

# security.sh 已通过 lib/_init.sh 按依赖顺序加载，此处不再交叉 source

# ---------- 颜色定义 ----------
GREEN='\033[32m'; RED='\033[31m'; YELLOW='\033[33m'; BLUE='\033[34m'; PLAIN='\033[0m'

# ---------- 日志目录（优先使用环境变量，回退到默认） ----------
LOG_DIR="${LOG_DIR:-${TOOLKIT_ROOT}/logs}"
LOG_FILE="${LOG_DIR}/toolkit.log"
AUDIT_FILE="${LOG_DIR}/audit.log"
mkdir -p "$LOG_DIR"

# ---------- 日志函数 ----------
# 日志脱敏：自动过滤密码、Token 等敏感信息
_desensitize_log() {
    local msg="$1"
    
    # 优先使用安全库中的增强脱敏函数
    if command -v filter_sensitive_info &>/dev/null; then
        msg=$(filter_sensitive_info "$msg")
    else
        # 降级方案：基本脱敏
        # 脱敏密码相关（排除单引号内的内容）
        msg=$(echo "$msg" | sed -E "s/(password|pwd|passwd)[=: ']+[^'[:space:]]+/\1=***/gi")
        # 脱敏 Token/Secret
        msg=$(echo "$msg" | sed -E "s/(token|secret|key)[=: ']+[^'[:space:]]+/\1=***/gi")
        # 脱敏 MySQL 连接字符串
        msg=$(echo "$msg" | sed -E "s/(MYSQL_PWD|SQL_DSN|REDIS_CONN)[=: ']+[^'[:space:]]+/\1=***/gi")
    fi
    
    echo "$msg"
}

# ---------- 日志轮转配置 ----------
setup_logrotate() {
    # 需要 root 权限
    if [[ $EUID -ne 0 ]]; then
        log_warn "配置 logrotate 需要 root 权限，跳过"
        return 1
    fi

    # 检查 logrotate 是否安装
    if ! command -v logrotate &>/dev/null; then
        log_info "logrotate 未安装，正在安装..."
        if command -v apt-get &>/dev/null; then
            apt-get install -y logrotate >/dev/null 2>&1 || true
        elif command -v yum &>/dev/null; then
            yum install -y logrotate >/dev/null 2>&1 || true
        else
            log_warn "无法自动安装 logrotate，请手动安装"
            return 1
        fi
    fi

    local logrotate_conf="/etc/logrotate.d/newapi-tools"
    local log_dir="$LOG_DIR"

    mkdir -p "$log_dir"

    cat > "$logrotate_conf" << 'EOF'
/opt/newapi-tools/logs/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    create 0644 root root
}
EOF

    log_success "日志轮转已配置: $logrotate_conf"
    log_info "策略: 每天轮转，保留 7 天，压缩旧日志"
}

log_info()    { local msg; msg=$(_desensitize_log "$*"); echo -e "[$(date '+%Y-%m-%d %H:%M:%S')] [INFO] $msg" | tee -a "$LOG_FILE"; }
log_success() { local msg; msg=$(_desensitize_log "$*"); echo -e "${GREEN}[$(date '+%Y-%m-%d %H:%M:%S')] [SUCCESS] $msg${PLAIN}" | tee -a "$LOG_FILE"; }
log_warn()    { local msg; msg=$(_desensitize_log "$*"); echo -e "${YELLOW}[$(date '+%Y-%m-%d %H:%M:%S')] [WARN] $msg${PLAIN}" | tee -a "$LOG_FILE" >&2; }
log_error()   { local msg; msg=$(_desensitize_log "$*"); echo -e "${RED}[$(date '+%Y-%m-%d %H:%M:%S')] [ERROR] $msg${PLAIN}" | tee -a "$LOG_FILE" >&2; }
log_debug()   { [[ "${DEBUG:-0}" == "1" ]] && { local msg; msg=$(_desensitize_log "$*"); echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')] [DEBUG] $msg${PLAIN}" >> "$LOG_FILE"; }; return 0; }
log_audit()   { local msg; msg=$(_desensitize_log "$*"); echo "[$(date '+%Y-%m-%d %H:%M:%S')] [AUDIT] $msg" >> "$AUDIT_FILE"; }

# ---------- 权限检查 ----------
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "此脚本必须使用 root 用户执行！"
        exit 1
    fi
}

# ---------- 环境依赖检查 ----------
require_docker() {
    if ! command -v docker &>/dev/null; then
        log_error "Docker 未安装，请先执行环境初始化（菜单选项 1）。"
        exit 1
    fi
    if ! docker info &>/dev/null; then
        log_error "Docker 守护进程未运行，请执行: systemctl start docker"
        exit 1
    fi
}

require_command() {
    local cmd="$1"
    local hint="${2:-}"
    if ! command -v "$cmd" &>/dev/null; then
        log_error "缺少依赖命令: $cmd"
        [[ -n "$hint" ]] && log_info "安装建议: $hint"
        return 1  # 返回而非退出，便于单元测试
    fi
}

# ---------- 二次确认（增强版：严格匹配 yes） ----------
ask_confirm() {
    local msg="${1:-确定要继续吗？}"
    log_audit "等待用户确认: $msg"
    read -r -p "$msg (必须输入 yes 确认): " confirm
    if [[ "$confirm" == "yes" ]]; then
        log_audit "用户确认通过"
        return 0
    fi
    log_info "操作已取消"
    return 1
}

# ---------- 文件备份 ----------
backup_file() {
    local src="$1"
    if [[ ! -f "$src" ]]; then
        log_warn "备份源文件不存在: $src"
        return 1
    fi
    local bak_dir="${TOOLKIT_ROOT}/backups/file_backups"
    mkdir -p "$bak_dir"
    local bak_file="${bak_dir}/$(basename "$src").bak.$(date +%s)"
    cp -a "$src" "$bak_file"
    log_info "已备份: $src → $bak_file"
}

# ---------- SHA-256 校验（替代不安全的 MD5） ----------
generate_checksum() {
    local file="$1"
    if [[ -f "$file" ]]; then
        sha256sum "$file" > "${file}.sha256"
        log_info "已生成校验文件: ${file}.sha256"
    else
        log_error "生成校验失败，文件不存在: $file"
        return 1
    fi
}

verify_checksum() {
    local file="$1"
    local sha256file="${file}.sha256"
    if [[ ! -f "$sha256file" ]]; then
        log_error "缺少校验文件: $sha256file，拒绝继续（安全策略）。"
        return 1
    fi
    if sha256sum -c "$sha256file" --status; then
        log_success "校验通过: $file"
        return 0
    else
        log_error "校验失败！文件可能已损坏: $file"
        return 1
    fi
}

# ---------- Webhook 通知（修复 JSON 转义） ----------
send_webhook() {
    local title="$1"
    local content="$2"
    local url="${WEBHOOK_URL:-}"
    
    if [[ -z "$url" ]]; then
        return 0   # 未配置 Webhook，静默跳过
    fi
    
    # 使用 jq 进行正确的 JSON 转义
    local escaped_title
    local escaped_content
    if command -v jq &>/dev/null; then
        escaped_title=$(jq -Rn --arg v "$title" '$v')
        escaped_content=$(jq -Rn --arg v "$content" '$v')
        # 移除 jq 输出的首尾双引号
        escaped_title="${escaped_title#\"}"
        escaped_title="${escaped_title%\"}"
        escaped_content="${escaped_content#\"}"
        escaped_content="${escaped_content%\"}"
    else
        # 降级方案：手动转义关键字符
        escaped_title=$(printf '%s' "$title" | sed 's/\\/\\\\/g; s/"/\\"/g')
        escaped_content=$(printf '%s' "$content" | sed 's/\\/\\\\/g; s/"/\\"/g')
    fi
    
    # 飞书/钉钉通用格式（text 类型）
    local payload
    payload=$(printf '{"msg_type":"text","content":{"text":"%s\n%s"}}' "$escaped_title" "$escaped_content")
    
    if curl -fsS -X POST \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "$url" &>/dev/null; then
        log_info "Webhook 通知已发送"
    else
        log_warn "Webhook 通知发送失败（不影响主流程）"
    fi
}

# ---------- 通用错误捕获 ----------
trap_error() {
    local exit_code=$?
    log_audit "脚本异常退出: $0，行号: $1，最后命令: $2，退出码: $exit_code"
    send_webhook "NewAPI 工具集异常" "服务器 $(hostname) 的脚本 $0 在第 $1 行异常退出，命令: $2"
    exit $exit_code
}

# ---------- 系统信息检测（带缓存） ----------
detect_system_info_cached() {
    local cache_dir="${STATE_DIR:-${TOOLKIT_ROOT}/state}"
    local cache_file="$cache_dir/system-info.cache"
    mkdir -p "$cache_dir"

    # 缓存有效期：1 小时（3600 秒）
    if [[ -f "$cache_file" ]]; then
        local now=$(date +%s)
        local cache_mtime=$(stat -c %Y "$cache_file" 2>/dev/null || echo 0)
        local age=$(( now - cache_mtime ))
        local os_mtime=0
        [[ -f /etc/os-release ]] && os_mtime=$(stat -c %Y /etc/os-release 2>/dev/null || echo 0)

        if [[ $age -lt 3600 && $cache_mtime -ge $os_mtime ]]; then
            source "$cache_file"
            log_debug "使用缓存的系统信息"
            return 0
        fi
    fi

    # 重新检测
    OS_NAME=$(grep -E "^PRETTY_NAME=" /etc/os-release 2>/dev/null | cut -d'"' -f2 || uname -s)
    OS_VERSION=$(grep -E "^VERSION_ID=" /etc/os-release 2>/dev/null | cut -d'"' -f2 || echo "unknown")
    OS_FAMILY=$(grep -E "^ID=" /etc/os-release 2>/dev/null | cut -d'"' -f2 || echo "unknown")
    KERNEL_VERSION=$(uname -r 2>/dev/null || echo "unknown")

    # 写入缓存
    cat > "$cache_file" << EOF
OS_NAME="$OS_NAME"
OS_VERSION="$OS_VERSION"
OS_FAMILY="$OS_FAMILY"
KERNEL_VERSION="$KERNEL_VERSION"
EOF
    log_debug "系统信息已缓存: $cache_file"
}

# ---------- [V1.0 兼容层 - 计划在 V3.0 移除] ----------
# 加载 .env 敏感变量（安全版）
# 只导出白名单变量，防止 .env 中的随意变量污染环境
# ⚠️ 此函数已弃用，新代码应使用 get_config() 函数
load_env_sensitive() {
    log_warn "load_env_sensitive() 已弃用，请使用 get_config() 替代"
    local env_file="${NEWAPI_HOME:-/home/new-api}/.env"
    if [[ ! -f "$env_file" ]]; then
        log_error "找不到 .env 文件: $env_file，请先部署 NewAPI。"
        exit 1
    fi
    # 只加载白名单变量
    while IFS='=' read -r key value; do
        # 跳过注释和空行
        [[ "$key" =~ ^#.*$ || -z "$key" ]] && continue
        # 只导出安全的变量名
        if [[ "$key" =~ ^(DB_ROOT_PASSWORD|SESSION_SECRET|SQL_DSN|REDIS_CONN_STRING)$ ]]; then
            export "$key=$value"
        fi
    done < "$env_file"
    log_info "已加载敏感环境变量（白名单过滤）"
}

# ---------- Docker 镜像预检 ----------
pull_image_if_needed() {
    local image="$1"
    local tag="${2:-latest}"

    if docker image inspect "${image}:${tag}" &>/dev/null; then
        log_info "镜像已存在，跳过拉取: ${image}:${tag}"
        return 0
    fi

    log_info "拉取镜像: ${image}:${tag}"
    docker pull "${image}:${tag}"
}
