#!/bin/bash
set -eo pipefail
# UI 组件库 —— 提升用户体验
# 提供进度条、彩色输出、ASCII艺术、状态面板等友好界面

# ---------- 颜色定义（增强版）----------
UI_GREEN='\033[32m'; UI_RED='\033[31m'; UI_YELLOW='\033[33m'
UI_BLUE='\033[34m'; UI_CYAN='\033[36m'; UI_MAGENTA='\033[35m'
UI_BOLD='\033[1m'; UI_DIM='\033[2m'; UI_PLAIN='\033[0m'

# ---------- 显示 ASCII 艺术横幅 ----------
show_banner() {
    clear
    echo -e "${UI_CYAN}${UI_BOLD}"
    cat << 'EOF'
╔══════════════════════════════════════════════════════════════╗
║                                                              ║
║   ███╗   ██╗███████╗██╗   ██╗██████╗  █████╗ ███████╗    ║
║   ████╗  ██║██╔════╝██║   ██║██╔══██╗██╔══██╗██╔════╝    ║
║   ██╔██╗ ██║█████╗  ██║   ██║██████╔╝███████║███████╗    ║
║   ██║╚██╗██║██╔══╝  ██║   ██║██╔══██╗██╔══██║╚════██║    ║
║   ██║ ╚████║███████╗╚██████╔╝██║  ██║██║  ██║███████║    ║
║   ╚═╝  ╚═══╝╚══════╝ ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝╚══════╝    ║
║                                                              ║
║       运维工具集  v2.0                                        ║
║       🚀 更智能 · 更简单 · 更强大                            ║
║                                                              ║
╚══════════════════════════════════════════════════════════════╝
EOF
    echo -e "${UI_PLAIN}"
    echo ""
}

# ---------- 进度条 ----------
show_progress() {
    local current="$1"
    local total="$2"
    local desc="${3:-处理中}"
    local width=50
    
    # 防止除零错误
    if [[ "$total" -eq 0 || -z "$total" ]]; then
        printf "\r${UI_BLUE}%s [....................] 0%%${UI_PLAIN}" "$desc"
        return 0
    fi
    
    local percentage=$((current * 100 / total))
    local completed=$((width * current / total))
    local remaining=$((width - completed))
    
    # 构建进度条
    local bar="["
    for ((i=0; i<completed; i++)); do bar+="="; done
    [[ $completed -gt 0 && $completed -lt $width ]] && bar+=">"
    for ((i=0; i<remaining; i++)); do bar+=" "; done
    bar+="]"
    
    # 显示进度条
    printf "\r${UI_BLUE}%s %s %d%%${UI_PLAIN}" "$desc" "$bar" "$percentage"
    
    # 完成时换行
    [[ $current -eq $total ]] && echo "" || true
}

# ---------- 简化的是/否提问 ----------
ask_yn() {
    local question="$1"
    local default="${2:-n}"
    local prompt
    
    if [[ "$default" == "y" ]]; then
        prompt="${UI_YELLOW}${question} [Y/n]: ${UI_PLAIN}"
    else
        prompt="${UI_YELLOW}${question} [y/N]: ${UI_PLAIN}"
    fi
    
    read -r -p "$prompt" answer
    answer=${answer:-$default}
    
    case "$answer" in
        [Yy]|[Yy][Ee][Ss]) return 0 ;;
        *) return 1 ;;
    esac
}

# ---------- 成功/失败/警告/信息 提示（增强版）----------
ui_success() {
    echo -e "${UI_GREEN}✓ $*${UI_PLAIN}"
    log_success "$*"
}

ui_error() {
    echo -e "${UI_RED}✗ $*${UI_PLAIN}"
    log_error "$*"
}

ui_warn() {
    echo -e "${UI_YELLOW}⚠ $*${UI_PLAIN}"
    log_warn "$*"
}

ui_info() {
    echo -e "${UI_BLUE}ℹ $*${UI_PLAIN}"
    log_info "$*"
}

ui_debug() {
    echo -e "${UI_DIM}  $*${UI_PLAIN}"
    [[ "${DEBUG:-0}" == "1" ]] && log_info "[DEBUG] $*"
}

# ---------- 显示系统状态面板 ----------
show_dashboard() {
    echo -e "${UI_BOLD}${UI_CYAN}╔════════════════════════════════════════════════════════════╗${UI_PLAIN}"
    echo -e "${UI_BOLD}${UI_CYAN}║                        系统状态面板                          ║${UI_PLAIN}"
    echo -e "${UI_BOLD}${UI_CYAN}╚════════════════════════════════════════════════════════════╝${UI_PLAIN}"
    echo ""
    
    # 系统信息
    echo -e "${UI_BOLD}【系统信息】${UI_PLAIN}"
    echo "  主机名:   $(hostname)"
    echo "  操作系统: $(cat /etc/os-release 2>/dev/null | grep PRETTY_NAME | cut -d'"' -f2 || echo '未知')"
    echo "  内核版本: $(uname -r)"
    echo "  内存:     $(free -h 2>/dev/null | awk '/^Mem:/ {print $2}' || echo '未知')"
    echo "  磁盘:     $(df -h / 2>/dev/null | awk 'NR==2 {print $2 " (已用 " $5 ")"}' || echo '未知')"
    echo ""
    
    # Docker 状态
    echo -e "${UI_BOLD}【Docker 状态】${UI_PLAIN}"
    if command -v docker &>/dev/null; then
        ui_success "Docker 已安装: $(docker --version | cut -d' ' -f3 | tr -d ',')"
        if docker info &>/dev/null; then
            ui_success "Docker 服务运行中"
            echo "  容器总数: $(docker ps -a --format '{{.Names}}' 2>/dev/null | wc -l)"
            echo "  运行中:   $(docker ps --format '{{.Names}}' 2>/dev/null | wc -l)"
        else
            ui_warn "Docker 服务未运行"
        fi
    else
        ui_error "Docker 未安装"
    fi
    echo ""
    
    # NewAPI 状态
    echo -e "${UI_BOLD}【NewAPI 状态】${UI_PLAIN}"
    local compose_file="${NEWAPI_HOME:-/home/new-api}/docker-compose.yml"
    if [[ -f "$compose_file" ]]; then
        ui_success "NewAPI 已部署"
        cd "$(dirname "$compose_file")" 2>/dev/null && {
            echo "  服务状态:"
            docker compose ps 2>/dev/null | awk 'NR>2 {print "    - " $1 ": " $4}' || echo "    无法获取服务状态"
        }
    else
        ui_warn "NewAPI 未部署"
    fi
    echo ""
    
    # 最近备份
    echo -e "${UI_BOLD}【最近备份】${UI_PLAIN}"
    local backup_dir="${TOOLKIT_ROOT:-/opt/newapi-tools}/backups"
    if [[ -d "$backup_dir" ]]; then
        local latest_backup
        latest_backup=$(find "$backup_dir" -type f -name "*.sql.gz" -o -name "*.tar.gz" 2>/dev/null | sort -r | head -1)
        if [[ -n "$latest_backup" ]]; then
            echo "  最新备份: $(basename "$latest_backup")"
            echo "  备份时间: $(stat -c %y "$latest_backup" 2>/dev/null | cut -d'.' -f1 || echo '未知')"
        else
            echo "  暂无备份"
        fi
    else
        echo "  备份目录不存在"
    fi
    echo ""
}

# ---------- 显示欢迎信息（增强版）----------
show_welcome() {
    show_banner
    
    echo -e "${UI_BOLD}欢迎使用 NewAPI 运维工具集！${UI_PLAIN}"
    echo ""
    echo "本工具将帮助您："
    echo "  ${UI_GREEN}1.${UI_PLAIN} 快速部署 NewAPI 及其依赖服务（MySQL + Redis + NPM）"
    echo "  ${UI_GREEN}2.${UI_PLAIN} 配置 SSL 证书与 Nginx 反向代理"
    echo "  ${UI_GREEN}3.${UI_PLAIN} 管理数据备份与快速恢复"
    echo "  ${UI_GREEN}4.${UI_PLAIN} 监控系统健康状态与日志"
    echo "  ${UI_GREEN}5.${UI_PLAIN} 一键更新 NewAPI 版本（含自动回滚）"
    echo ""
    
    # 显示新手/专家模式提示
    if [[ "${NOVICE_MODE:-0}" == "1" ]]; then
        echo -e "${UI_CYAN}ℹ 当前为新手模式，将显示详细提示${UI_PLAIN}"
        echo -e "${UI_DIM}   提示：输入 'm' 可以切换到专家模式${UI_PLAIN}"
    else
        echo -e "${UI_CYAN}ℹ 当前为专家模式，将最小化输出${UI_PLAIN}"
        echo -e "${UI_DIM}   提示：输入 'm' 可以切换到新手模式${UI_PLAIN}"
    fi
    
    echo ""
    
    # 快速状态检查
    if command -v docker &>/dev/null && docker ps &>/dev/null; then
        if docker ps --format '{{.Names}}' 2>/dev/null | grep -qw "new-api"; then
            echo -e "${UI_GREEN}  ✓ NewAPI 已运行${UI_PLAIN}"
        else
            echo -e "${UI_YELLOW}  ⚠ NewAPI 未运行${UI_PLAIN}"
        fi
    else
        echo -e "${UI_YELLOW}  ⚠ Docker 未运行${UI_PLAIN}"
    fi
    
    echo ""
    read -r -p "按回车键继续..."
}

# ---------- 显示操作结果摘要（增强版）----------
show_summary() {
    local title="$1"
    shift
    local lines=("$@")
    
    echo ""
    echo -e "${UI_BOLD}${UI_GREEN}╔════════════════════════════════════════════════════════════╗${UI_PLAIN}"
    
    # 计算标题长度并居中
    local title_len=${#title}
    local padding=$(( (60 - title_len) / 2 ))
    printf "${UI_BOLD}${UI_GREEN}║%*s%s%*s║${UI_PLAIN}\n" $padding "" "$title" $((60 - title_len - padding)) ""
    
    echo -e "${UI_BOLD}${UI_GREEN}╚════════════════════════════════════════════════════════════╝${UI_PLAIN}"
    echo ""
    
    for line in "${lines[@]}"; do
        # 根据内容自动添加图标
        if [[ "$line" =~ ^✓ ]]; then
            echo -e "  ${UI_GREEN}$line${UI_PLAIN}"
        elif [[ "$line" =~ ^✗ ]]; then
            echo -e "  ${UI_RED}$line${UI_PLAIN}"
        elif [[ "$line" =~ ^⚠ ]]; then
            echo -e "  ${UI_YELLOW}$line${UI_PLAIN}"
        elif [[ "$line" =~ ^ℹ ]]; then
            echo -e "  ${UI_BLUE}$line${UI_PLAIN}"
        elif [[ -z "$line" ]]; then
            echo ""
        else
            echo "  $line"
        fi
    done
    
    echo ""
}

# ---------- 加载动画 ----------
show_loading() {
    local message="${1:-处理中...}"
    local duration="${2:-5}"
    local chars="⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"
    local chars_array=()
    
    # 将字符转为数组
    for ((i=0; i<${#chars}; i++)); do
        chars_array+=("${chars:$i:1}")
    done
    
    local end_time=$((SECONDS + duration))
    
    while [[ $SECONDS -lt $end_time ]]; do
        for char in "${chars_array[@]}"; do
            printf "\r${UI_CYAN}%s${UI_PLAIN} %s" "$char" "$message"
            sleep 0.1
        done
    done
    
    printf "\r%s\n" "$message - 完成"
}

# ---------- 显示分隔线 ----------
show_divider() {
    local char="${1:-─}"
    local length="${2:-60}"
    local color="${3:-${UI_BLUE}}"
    
    printf "${color}"
    printf "%${length}s" | tr ' ' "$char"
    printf "${UI_PLAIN}\n"
}

# ---------- 显示带框的文本 ----------
show_boxed_text() {
    local text="$1"
    local color="${2:-${UI_CYAN}}"
    local text_len=${#text}
    local width=$((text_len + 4))
    
    printf "${color}"
    printf "╔%${width}s╗\n" | tr ' ' '═'
    printf "║  %s  ║\n" "$text"
    printf "╚%${width}s╝\n" | tr ' ' '═'
    printf "${UI_PLAIN}"
}

# ---------- 友好错误提示（将技术错误转为人话）----------
show_friendly_error() {
    local error_msg="$1"
    local solution="${2:-}"
    
    echo ""
    echo -e "${UI_RED}╔════════════════════════════════════════════════════════════╗${UI_PLAIN}"
    echo -e "${UI_RED}║  ✗ 出错了！                                           ║${UI_PLAIN}"
    echo -e "${UI_RED}╚════════════════════════════════════════════════════════════╝${UI_PLAIN}"
    echo ""
    echo -e "${UI_YELLOW}问题描述：${UI_PLAIN}"
    echo "  $error_msg"
    echo ""
    
    if [[ -n "$solution" ]]; then
        echo -e "${UI_GREEN}解决方案：${UI_PLAIN}"
        echo "  $solution"
        echo ""
    fi
    
    # 常见错误和解决方案
    if echo "$error_msg" | grep -qi "permission denied"; then
        echo -e "${UI_CYAN}可能的原因：${UI_PLAIN}"
        echo "  1. 没有使用 root 用户运行"
        echo "  2. 文件/目录权限不足"
        echo ""
        echo -e "${UI_GREEN}解决方案：${UI_PLAIN}"
        echo "  - 使用 root 用户: sudo bash $0"
        echo "  - 检查权限: ls -la [文件/目录]"
        echo "  - 修改权限: chmod 777 [文件/目录]"
    elif echo "$error_msg" | grep -qi "command not found"; then
        echo -e "${UI_CYAN}可能的原因：${UI_PLAIN}"
        echo "  1. 未安装该命令"
        echo "  2. 命令不在 PATH 中"
        echo ""
        echo -e "${UI_GREEN}解决方案：${UI_PLAIN}"
        echo "  - 安装命令: apt install [包名]"
        echo "  - 检查 PATH: echo \$PATH"
    elif echo "$error_msg" | grep -qi "connection refused\|network"; then
        echo -e "${UI_CYAN}可能的原因：${UI_PLAIN}"
        echo "  1. 网络连接问题"
        echo "  2. 防火墙阻止"
        echo "  3. 服务未启动"
        echo ""
        echo -e "${UI_GREEN}解决方案：${UI_PLAIN}"
        echo "  - 检查网络: ping -c 1 baidu.com"
        echo "  - 检查防火墙: ufw status"
        echo "  - 启动服务: systemctl start [服务名]"
    fi
    
    echo ""
}

# ---------- 增强确认对话框 ----------
confirm_action() {
    local message="$1"
    local require_yes="${2:-false}"
    
    if [[ "$require_yes" == "true" ]]; then
        # 危险操作，需要输入 YES
        echo -e "${UI_RED}${UI_BOLD}⚠ 危险操作警告 ⚠${UI_PLAIN}"
        echo -e "${UI_YELLOW}$message${UI_PLAIN}"
        echo ""
        read -r -p "请输入 YES 确认: " answer
        if [[ "$answer" == "YES" ]]; then
            return 0
        else
            ui_info "操作已取消"
            return 1
        fi
    else
        # 普通确认
        ask_yn "$message" "n"
        return $?
    fi
}

# ---------- 导出函数 ----------
export -f show_banner show_progress ask_yn
export -f ui_success ui_error ui_warn ui_info ui_debug
export -f show_dashboard show_welcome show_summary
export -f show_loading show_divider show_boxed_text
export -f show_friendly_error confirm_action
