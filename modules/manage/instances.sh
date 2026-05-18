#!/bin/bash
# instances.sh — 多实例管理入口
# 用法: newapi-tools instances [list|show|add|remove|switch] [参数]
set -eo pipefail

# 自动检测 TOOLKIT_ROOT（兼容直接执行和通过 newapi-tools.sh 调用）
TOOLKIT_ROOT="${TOOLKIT_ROOT:-$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")/../.." && pwd)}"
export TOOLKIT_ROOT

# shellcheck source=lib/_init.sh
source "${TOOLKIT_ROOT}/lib/_init.sh"

# 确保 MODULES_DIR 已设置
MODULES_DIR="${MODULES_DIR:-${TOOLKIT_ROOT}/modules}"
export MODULES_DIR

# 仅在直接执行时进行权限检查
if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    check_root
fi

# ---------- 以下为主执行逻辑，仅在直接运行时执行 ----------
if [[ "${BASH_SOURCE[0]}" != "$0" ]]; then
    return 0 2>/dev/null || true
fi

# 加载实例管理框架
# shellcheck source=modules/manage/base/instances-base.sh
source "${MODULES_DIR}/manage/base/instances-base.sh"

# ---------- 参数解析 ----------
SUB_CMD="${1:-list}"
shift 2>/dev/null || true

case "$SUB_CMD" in
    list|ls|l)
        echo -e "${UI_BOLD}${UI_CYAN}╔════════════════════════════════════════════════════════════╗${UI_PLAIN}"
        echo -e "${UI_BOLD}${UI_CYAN}║                    实例列表                                 ║${UI_PLAIN}"
        echo -e "${UI_BOLD}${UI_CYAN}╚════════════════════════════════════════════════════════════╝${UI_PLAIN}"
        echo ""
        printf "  %-20s %-10s %-30s %s\n" "实例名" "FLAVOR" "目录" "状态"
        echo "  --------------------------------------------------------------------------------"
        list_instances
        echo ""
        ;;
    show|info|s)
        if [[ -z "${1:-}" ]]; then
            local active
            active=$(get_active_instance)
            show_instance "$active"
        else
            show_instance "$1"
        fi
        ;;
    add|a)
        local name="${1:-}"
        local flavor="${2:-newapi}"
        local dir="${3:-}"

        if [[ -z "$name" ]]; then
            log_error "用法: newapi-tools instances add <实例名> [FLAVOR] [目录]"
            echo "  FLAVOR: newapi (默认) | one-api | sub2api"
            echo "  示例: newapi-tools instances add prod newapi /home/new-api"
            exit 1
        fi
        add_instance "$name" "$flavor" "$dir"
        ;;
    remove|rm|delete|d)
        if [[ -z "${1:-}" ]]; then
            log_error "用法: newapi-tools instances remove <实例名>"
            exit 1
        fi
        remove_instance "$1"
        ;;
    switch|use|sw)
        if [[ -z "${1:-}" ]]; then
            local active
            active=$(get_active_instance)
            log_info "当前活跃实例: $active"
            echo ""
            printf "  %-20s %-10s %-30s %s\n" "实例名" "FLAVOR" "目录" "状态"
            echo "  --------------------------------------------------------------------------------"
            list_instances
            exit 0
        fi
        set_active_instance "$1"
        ;;
    help|-h|--help)
        cat << EOF
用法: newapi-tools instances [命令] [参数]

命令：
  list                列出所有实例（默认）
  show [实例名]       显示实例详情（默认显示当前活跃实例）
  add <名称> [FLAVOR] [目录]   添加实例
  remove <名称>       删除实例
  switch <名称>       切换活跃实例

FLAVOR: newapi (默认) | one-api | sub2api

示例：
  newapi-tools instances list
  newapi-tools instances add prod newapi /home/new-api
  newapi-tools instances add staging one-api /home/one-api
  newapi-tools instances switch prod
  newapi-tools instances remove staging
EOF
        ;;
    *)
        log_error "未知子命令: $SUB_CMD"
        echo "可用命令: list, show, add, remove, switch"
        echo "查看帮助: newapi-tools instances help"
        exit 1
        ;;
esac
