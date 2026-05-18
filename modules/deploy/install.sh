#!/bin/bash
# 安装部署路由脚本 v2.2
# 新增：FLAVOR 概念支持（newapi / one-api / sub2api）

# 守卫：如果是被 source，直接 return（不执行安装逻辑，不设置 set -e）
if [[ "${BASH_SOURCE[0]}" != "$0" ]]; then
    return 0
fi

set -eo pipefail

# ---------- 加载依赖（统一入口）----------
# shellcheck source=lib/_init.sh
source "${TOOLKIT_ROOT}/lib/_init.sh"

# ---------- 解析命令行参数 ----------
FLAVOR="newapi"  # 默认 FLAVOR

while [[ $# -gt 0 ]]; do
    case "$1" in
        --flavor)
            FLAVOR="$2"
            shift 2
            ;;
        --flavor=*)
            FLAVOR="${1#*=}"
            shift
            ;;
        --help|-h)
            echo "用法: newapi-tools install [--flavor <flavor>]"
            echo ""
            echo "支持的平台（FLAVOR）："
            echo "  newapi   - NewAPI（默认）"
            echo "  one-api  - One API"
            echo "  sub2api  - Sub2API"
            echo ""
            echo "示例："
            echo "  newapi-tools install"
            echo "  newapi-tools install --flavor newapi"
            echo "  newapi-tools install --flavor one-api"
            echo "  newapi-tools install --flavor sub2api"
            exit 0
            ;;
        *)
            log_error "未知参数: $1"
            exit 1
            ;;
    esac
done

# ---------- 路由到 FLAVOR 专用脚本 ----------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

case "$FLAVOR" in
    newapi)
        NEWAPI_INSTALL_SH="${SCRIPT_DIR}/newapi/install.sh"
        if [[ -f "$NEWAPI_INSTALL_SH" ]]; then
            log_info "使用 FLAVOR: newapi"
            # 仅在执行时才 exec，source 时只定义路由逻辑
            if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
                exec bash "$NEWAPI_INSTALL_SH" "$@"
            fi
        else
            log_error "NewAPI 安装脚本不存在: $NEWAPI_INSTALL_SH"
            log_info "请先创建 modules/deploy/newapi/ 目录和 install.sh"
            exit 1
        fi
        ;;
    one-api)
        ONE_API_INSTALL_SH="${SCRIPT_DIR}/one-api/install.sh"
        if [[ -f "$ONE_API_INSTALL_SH" ]]; then
            log_info "使用 FLAVOR: one-api"
            if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
                exec bash "$ONE_API_INSTALL_SH" "$@"
            fi
        else
            log_error "One API 安装脚本不存在: $ONE_API_INSTALL_SH"
            exit 1
        fi
        ;;
    sub2api)
        SUB2API_INSTALL_SH="${SCRIPT_DIR}/sub2api/install.sh"
        if [[ -f "$SUB2API_INSTALL_SH" ]]; then
            log_info "使用 FLAVOR: sub2api"
            if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
                exec bash "$SUB2API_INSTALL_SH" "$@"
            fi
        else
            log_error "Sub2API 安装脚本不存在: $SUB2API_INSTALL_SH"
            exit 1
        fi
        ;;
    *)
        log_error "不支持的 FLAVOR: $FLAVOR"
        log_info "支持的 FLAVOR: newapi, one-api, sub2api"
        exit 1
        ;;
esac
