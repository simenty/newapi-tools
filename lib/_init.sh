#!/bin/bash
# lib/_init.sh — 统一初始化入口
# 用法：source "${TOOLKIT_ROOT}/lib/_init.sh"
# 一次性加载所有 lib，删除各 lib 内的交叉 source

set -eo pipefail

# 防重入守卫
if [[ "${_INIT_SH_LOADED:-0}" -eq 1 ]]; then
    return 0 2>/dev/null || true
fi
export _INIT_SH_LOADED=1

# 确保 TOOLKIT_ROOT 已设置
TOOLKIT_ROOT="${TOOLKIT_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
export TOOLKIT_ROOT

_LIB_DIR="${TOOLKIT_ROOT}/lib"

# 按依赖顺序加载
for _lib in common.sh security.sh state.sh ui.sh config.sh registry.sh plugin.sh os_adapter.sh smart-defaults.sh mode.sh npm-api.sh; do
    if [[ -f "${_LIB_DIR}/${_lib}" ]]; then
        source "${_LIB_DIR}/${_lib}"
    else
        echo "[ERROR] 核心库缺失: ${_LIB_DIR}/${_lib}" >&2
        return 1 2>/dev/null || exit 1
    fi
done

# 清理临时变量
unset _lib _LIB_DIR
