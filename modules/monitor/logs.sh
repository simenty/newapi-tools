#!/bin/bash
# 查看 NewAPI 运行日志 v2.0
# 新增：UI 增强、智能选择容器、日志级别高亮、配置统一管理
set -eo pipefail

# shellcheck source=lib/_init.sh
source "${TOOLKIT_ROOT}/lib/_init.sh"

# ---------- 以下为主执行逻辑，仅在直接运行时执行（source 时跳过，便于单元测试）----------
if [[ "${BASH_SOURCE[0]}" != "$0" ]]; then
    return 0 2>/dev/null || true
fi

LINES=100
FOLLOW=false
CONTAINER="new-api"

# ---------- 解析参数 ----------
while [[ $# -gt 0 ]]; do
    case "$1" in
        -n|--lines)
            LINES="$2"
            shift 2
            ;;
        -f|--follow)
            FOLLOW=true
            shift
            ;;
        -c|--container)
            CONTAINER="$2"
            shift 2
            ;;
        -h|--help)
            cat << EOF
用法: newapi-tools logs [选项]

选项：
  -n, --lines N      显示最近 N 行（默认 100）
  -f, --follow       实时追踪日志（Ctrl+C 退出）
  -c, --container C  指定容器名（默认 new-api）
  -h, --help         显示帮助

示例：
  newapi-tools logs               # 查看最近 100 行
  newapi-tools logs -n 50        # 查看最近 50 行
  newapi-tools logs -f           # 实时追踪日志
  newapi-tools logs -c mysql    # 查看 mysql 容器日志
EOF
            exit 0
            ;;
        *)
            log_error "未知参数: $1"
            exit 1
            ;;
    esac
done

# ---------- 显示新手提示 ----------
novice_prompt "日志查看器可以查看容器日志。NewAPI 主日志使用 'new-api' 容器，数据库日志使用 'mysql' 容器。"

# ---------- 智能选择容器（如果未指定）----------
if [[ "$CONTAINER" == "new-api" ]]; then
    # 检查哪些容器在运行
    RUNNIN_CONTAINERS=$(docker ps --format '{{.Names}}' 2>/dev/null || echo "")
    
    if echo "$RUNNIN_CONTAINERS" | grep -qw "new-api"; then
        CONTAINER="new-api"
    elif echo "$RUNNIN_CONTAINERS" | grep -qw "mysql"; then
        ui_info "new-api 容器未运行，已自动切换到 mysql 容器"
        CONTAINER="mysql"
    elif echo "$RUNNIN_CONTAINERS" | grep -qw "redis"; then
        ui_info "new-api 容器未运行，已自动切换到 redis 容器"
        CONTAINER="redis"
    else
        ui_error "没有运行中的容器"
        ui_info "请先启动容器: cd ${NEWAPI_HOME} && docker compose up -d"
        exit 1
    fi
fi

# ---------- 检查容器是否存在 ----------
if ! docker ps --format '{{.Names}}' 2>/dev/null | grep -qw "$CONTAINER"; then
    ui_error "容器 $CONTAINER 未运行"
    ui_info "运行中的容器："
    docker ps --format '  - {{.Names}}' 2>/dev/null || echo "  (无)"
    exit 1
fi

# ---------- 显示日志 ----------
ui_success "正在查看容器: $CONTAINER"

if [[ "${FOLLOW:-false}" == "true" ]]; then
    echo -e "${UI_BLUE}>>> 实时日志（Ctrl+C 退出）...${UI_PLAIN}"
    echo ""
    
    # 实时追踪（带颜色高亮）
    docker logs --tail="$LINES" -f "$CONTAINER" 2>&1 | while IFS= read -r line; do
        # 高亮关键字
        if echo "$line" | grep -qi "error\|fatal\|panic"; then
            echo -e "${UI_RED}${line}${UI_PLAIN}"
        elif echo "$line" | grep -qi "warn"; then
            echo -e "${UI_YELLOW}${line}${UI_PLAIN}"
        elif echo "$line" | grep -qi "success\|completed"; then
            echo -e "${UI_GREEN}${line}${UI_PLAIN}"
        else
            echo "$line"
        fi
    done
else
    echo -e "${UI_BLUE}>>> 最近 ${LINES} 行日志：${UI_PLAIN}"
    echo ""
    
    # 显示最近 N 行（带颜色高亮）
    docker logs --tail="$LINES" "$CONTAINER" 2>&1 | while IFS= read -r line; do
        # 高亮关键字
        if echo "$line" | grep -qi "error\|fatal\|panic"; then
            echo -e "${UI_RED}${line}${UI_PLAIN}"
        elif echo "$line" | grep -qi "warn"; then
            echo -e "${UI_YELLOW}${line}${UI_PLAIN}"
        elif echo "$line" | grep -qi "success\|completed"; then
            echo -e "${UI_GREEN}${line}${UI_PLAIN}"
        else
            echo "$line"
        fi
    done
fi

echo ""
ui_info "日志查看完成"

# 询问是否查看更多（新手模式）
if_novice && {
    if ask_yn "是否查看其他容器日志？" "n"; then
        echo ""
        echo "可用的容器："
        docker ps --format '  - {{.Names}}' 2>/dev/null || echo "  (无)"
        echo ""
        read -r -p "请输入容器名: " next_container
        
        if [[ -n "$next_container" ]]; then
            bash "${TOOLKIT_ROOT}/modules/monitor/logs.sh" -c "$next_container"
        fi
    fi
}
