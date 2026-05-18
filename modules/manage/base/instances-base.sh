#!/bin/bash
# instances-base.sh — 多实例管理通用框架
# 只定义函数，不执行任何操作。被各 FLAVOR 的 instances.sh source 后调用。
# 用法: source "${MODULES_DIR}/manage/base/instances-base.sh"

set -eo pipefail

# TOOLKIT_ROOT 安全回退（被 source 时通常已设置）
TOOLKIT_ROOT="${TOOLKIT_ROOT:-$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")/../.." && pwd)}"
export TOOLKIT_ROOT

# shellcheck source=lib/_init.sh
source "${TOOLKIT_ROOT}/lib/_init.sh"

# ---------- 实例数据目录 ----------
INSTANCES_DIR="${CONFIG_DIR:-${TOOLKIT_ROOT}/config}/instances"

# ---------- 确保实例目录存在 ----------
_ensure_instances_dir() {
    mkdir -p "$INSTANCES_DIR"
}

# ---------- 列出所有实例 ----------
# 输出格式: 实例名 FLAVOR 目录 状态
list_instances() {
    _ensure_instances_dir

    local count=0
    local instance_file

    for instance_file in "$INSTANCES_DIR"/*.yaml; do
        [[ -f "$instance_file" ]] || continue
        local name flavor dir status
        name=$(basename "$instance_file" .yaml)

        # 读取实例元数据
        if command -v yq &>/dev/null; then
            flavor=$(yq '.flavor // "newapi"' "$instance_file" 2>/dev/null || echo "newapi")
            dir=$(yq '.directory // ""' "$instance_file" 2>/dev/null || echo "")
        else
            # 降级方案：grep
            flavor=$(grep "^flavor:" "$instance_file" 2>/dev/null | awk '{print $2}' | tr -d '"' || echo "newapi")
            dir=$(grep "^directory:" "$instance_file" 2>/dev/null | awk '{print $2}' | tr -d '"' || echo "")
        fi

        # 检查实例状态
        status=$(_check_instance_status "$dir" "$flavor")

        printf "%-20s %-10s %-30s %s\n" "$name" "$flavor" "$dir" "$status"
        count=$((count + 1))
    done

    if [[ $count -eq 0 ]]; then
        log_info "未找到任何实例"
    fi

    return 0
}

# ---------- 检查实例状态 ----------
# 参数: $1=目录, $2=FLAVOR
# 返回: running/stopped/unknown
_check_instance_status() {
    local dir="$1"
    local flavor="${2:-newapi}"

    if [[ -z "$dir" || ! -d "$dir" ]]; then
        echo "unknown"
        return 0
    fi

    # 检查 docker-compose.yml 是否存在
    if [[ ! -f "${dir}/docker-compose.yml" ]]; then
        echo "not_deployed"
        return 0
    fi

    # 检查容器是否在运行
    local container_name
    case "$flavor" in
        newapi)   container_name="new-api" ;;
        one-api)  container_name="one-api" ;;
        sub2api)  container_name="sub2api" ;;
        *)        container_name="$flavor" ;;
    esac

    if docker inspect --format='{{.State.Status}}' "$container_name" 2>/dev/null | grep -q "running"; then
        echo "running"
    else
        echo "stopped"
    fi
}

# ---------- 显示实例详情 ----------
# 参数: $1=实例名
show_instance() {
    local name="$1"
    local instance_file="${INSTANCES_DIR}/${name}.yaml"

    if [[ ! -f "$instance_file" ]]; then
        log_error "实例不存在: $name"
        return 1
    fi

    echo -e "${UI_BOLD}${UI_CYAN}╔════════════════════════════════════════════════════════════╗${UI_PLAIN}"
    echo -e "${UI_BOLD}${UI_CYAN}║                  实例详情: ${name}${UI_PLAIN}"
    echo -e "${UI_BOLD}${UI_CYAN}╚════════════════════════════════════════════════════════════╝${UI_PLAIN}"
    echo ""

    # 读取并显示实例配置
    if command -v yq &>/dev/null; then
        yq '.' "$instance_file" 2>/dev/null || cat "$instance_file"
    else
        cat "$instance_file"
    fi

    echo ""

    # 检查状态
    local flavor dir
    if command -v yq &>/dev/null; then
        flavor=$(yq '.flavor // "newapi"' "$instance_file" 2>/dev/null || echo "newapi")
        dir=$(yq '.directory // ""' "$instance_file" 2>/dev/null || echo "")
    else
        flavor=$(grep "^flavor:" "$instance_file" 2>/dev/null | awk '{print $2}' | tr -d '"' || echo "newapi")
        dir=$(grep "^directory:" "$instance_file" 2>/dev/null | awk '{print $2}' | tr -d '"' || echo "")
    fi

    local status
    status=$(_check_instance_status "$dir" "$flavor")

    echo -e "${UI_BOLD}运行状态:${UI_PLAIN} $status"
    echo -e "${UI_BOLD}配置文件:${UI_PLAIN} $instance_file"

    # 如果实例正在运行，显示容器信息
    if [[ "$status" == "running" ]]; then
        local container_name
        case "$flavor" in
            newapi)   container_name="new-api" ;;
            one-api)  container_name="one-api" ;;
            sub2api)  container_name="sub2api" ;;
            *)        container_name="$flavor" ;;
        esac

        echo ""
        echo -e "${UI_BOLD}容器信息:${UI_PLAIN}"
        docker inspect --format='  镜像: {{.Config.Image}}
  启动时间: {{.State.StartedAt}}
  重启次数: {{.RestartCount}}
  端口: {{range $p, $conf := .NetworkSettings.Ports}}{{if $conf}}{{$p}} {{end}}{{end}}' "$container_name" 2>/dev/null || true
    fi
}

# ---------- 添加实例 ----------
# 参数: $1=实例名, $2=FLAVOR, $3=目录
add_instance() {
    local name="$1"
    local flavor="${2:-newapi}"
    local dir="${3:-}"

    _ensure_instances_dir

    # 参数校验
    if [[ -z "$name" ]]; then
        log_error "实例名不能为空"
        return 1
    fi

    # 实例名只能包含字母、数字、连字符
    if [[ ! "$name" =~ ^[a-zA-Z0-9][a-zA-Z0-9_-]*$ ]]; then
        log_error "实例名只能包含字母、数字、连字符和下划线，且以字母或数字开头"
        return 1
    fi

    local instance_file="${INSTANCES_DIR}/${name}.yaml"

    if [[ -f "$instance_file" ]]; then
        log_error "实例已存在: $name"
        return 1
    fi

    # 如果未指定目录，使用默认值
    if [[ -z "$dir" ]]; then
        dir="/home/${name}"
    fi

    # 创建实例配置
    cat > "$instance_file" << EOF
# 实例配置: ${name}
# 创建时间: $(date '+%Y-%m-%d %H:%M:%S')

name: "${name}"
flavor: "${flavor}"
directory: "${dir}"

# 容器配置（由部署脚本自动填充）
containers: []
image: ""
port: 0
EOF

    log_success "实例已添加: $name (FLAVOR=$flavor, 目录=$dir)"
    log_info "配置文件: $instance_file"
}

# ---------- 删除实例 ----------
# 参数: $1=实例名
remove_instance() {
    local name="$1"

    if [[ -z "$name" ]]; then
        log_error "实例名不能为空"
        return 1
    fi

    local instance_file="${INSTANCES_DIR}/${name}.yaml"

    if [[ ! -f "$instance_file" ]]; then
        log_error "实例不存在: $name"
        return 1
    fi

    # 安全确认
    if ! ask_confirm "确定要删除实例 ${name}？此操作仅删除实例注册信息，不删除实际数据"; then
        log_info "操作已取消"
        return 0
    fi

    rm -f "$instance_file"
    log_success "实例已删除: $name"
}

# ---------- 获取当前活跃实例 ----------
# 返回: 实例名（从配置读取，默认 "default"）
get_active_instance() {
    get_config "deploy.active_instance" "default"
}

# ---------- 设置活跃实例 ----------
# 参数: $1=实例名
set_active_instance() {
    local name="$1"

    if [[ ! -f "${INSTANCES_DIR}/${name}.yaml" ]]; then
        log_error "实例不存在: $name"
        return 1
    fi

    set_config "deploy.active_instance" "$name"
    log_success "已切换到实例: $name"
}

# ---------- 导出函数 ----------
export -f list_instances show_instance add_instance remove_instance
export -f get_active_instance set_active_instance
