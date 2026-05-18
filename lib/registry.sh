#!/bin/bash
# lib/registry.sh — 命令与钩子注册表
# 用法：register_cmd "命令名" "脚本路径" "描述" ["plugin_id"] ["help_text"]
#       register_hook "hook名" "插件ID" "函数名" [优先级]

set -eo pipefail

# 命令注册表（关联数组）
# V2.4 扩展格式: "plugin_id|脚本路径|描述|帮助"
# 兼容旧格式: "脚本路径|描述"（plugin_id 默认 "_core"）
declare -A CMD_REGISTRY    # 命令名 → "plugin_id|脚本路径|描述|帮助"
# 钩子注册表（关联数组）
declare -A HOOK_REGISTRY   # "hook名:插件ID" → "函数名:优先级"

# 注册命令
# 用法: register_cmd "命令名" "脚本路径" "描述" ["plugin_id"] ["help_text"]
# plugin_id 默认为 "_core"（内置命令），保持向后兼容
register_cmd() {
    local cmd="$1"
    local script="$2"
    local desc="${3:-}"
    local plugin_id="${4:-_core}"
    local help_text="${5:-}"

    CMD_REGISTRY["$cmd"]="${plugin_id}|${script}|${desc}|${help_text}"
}

# 注册钩子
# 用法: register_hook "hook名" "插件ID" "函数名" [优先级]
register_hook() {
    local hook_name="$1"
    local plugin_id="$2"
    local func_name="$3"
    local priority="${4:-10}"

    HOOK_REGISTRY["${hook_name}:${plugin_id}"]="${func_name}:${priority}"
}

# 路由命令（返回脚本路径）
route_command() {
    local cmd="$1"
    local entry="${CMD_REGISTRY[$cmd]:-}"

    if [[ -z "$entry" ]]; then
        return 1
    fi

    # V2.4 格式: "plugin_id|脚本路径|描述|帮助"
    # 提取第二个字段（脚本路径）
    local remaining="${entry#*|}"       # 去掉 plugin_id|
    echo "${remaining%%|*}"             # 取脚本路径（到下一个 | 之前）
}

# 获取命令描述
get_cmd_desc() {
    local cmd="$1"
    local entry="${CMD_REGISTRY[$cmd]:-}"

    if [[ -z "$entry" ]]; then
        return 1
    fi

    # V2.4 格式: "plugin_id|脚本路径|描述|帮助"
    # 提取第三个字段（描述）
    local stripped="${entry#*|}"         # 去掉 plugin_id|
    stripped="${stripped#*|}"            # 去掉脚本路径|
    echo "${stripped%%|*}"               # 取描述（到下一个 | 之前）
}

# 获取所有已注册命令（排序）
get_all_commands() {
    printf '%s\n' "${!CMD_REGISTRY[@]}" | sort
}

# 获取命令所属插件ID
get_cmd_plugin_id() {
    local cmd="$1"
    local entry="${CMD_REGISTRY[$cmd]:-}"

    if [[ -z "$entry" ]]; then
        return 1
    fi

    # 第一个字段是 plugin_id
    echo "${entry%%|*}"
}

# 获取命令帮助文本
get_cmd_help() {
    local cmd="$1"
    local entry="${CMD_REGISTRY[$cmd]:-}"

    if [[ -z "$entry" ]]; then
        return 1
    fi

    # V2.4 格式: "plugin_id|脚本路径|描述|帮助"
    # 提取第四个字段（帮助）
    local stripped="${entry#*|}"         # 去掉 plugin_id|
    stripped="${stripped#*|}"            # 去掉脚本路径|
    stripped="${stripped#*|}"            # 去掉描述|
    echo "$stripped"
}

# 执行钩子（按优先级排序后依次执行）
run_hooks() {
    local hook_name="$1"
    shift

    # 收集匹配的钩子
    local hooks=()
    local key
    for key in "${!HOOK_REGISTRY[@]}"; do
        # 检查 key 是否以 hook_name: 开头
        if [[ "${key%%:*}" == "$hook_name" ]]; then
            local value="${HOOK_REGISTRY[$key]}"
            local priority="${value##*:}"
            local func="${value%%:*}"
            hooks+=("${priority}:${func}")
        fi
    done

    # 如果没有钩子，直接返回
    [[ ${#hooks[@]} -eq 0 ]] && return 0

    # 按优先级排序（数字越小越先执行）
    local sorted
    IFS=$'\n' sorted=($(printf '%s\n' "${hooks[@]}" | sort -n)); unset IFS

    # 依次执行
    local entry
    for entry in "${sorted[@]}"; do
        local func="${entry#*:}"
        "$func" "$@"
    done
}

# ---------- 注册核心命令 ----------
# V2.4: 只注册不属于特定插件的核心命令
# FLAVOR 插件命令（backup/restore/update/doctor 等）由 plugin.sh 从 metadata.yml 自动注册
# 路径在调用时由 MODULES_DIR 和 TOOLKIT_ROOT 展开
_register_builtin_commands() {
    # 部署命令（不属于特定 FLAVOR）
    register_cmd "install"     '${MODULES_DIR}/deploy/install.sh'             "部署 NewAPI 全家桶"     "deploy" "一键部署 NewAPI + MySQL + Redis + NPM"
    register_cmd "ssl"         '${MODULES_DIR}/deploy/ssl-proxy.sh'           "配置 SSL 证书与反向代理" "deploy" "配置 SSL 证书和 Nginx Proxy Manager"

    # 监控命令（通用，不区分 FLAVOR）
    register_cmd "health"      '${MODULES_DIR}/monitor/health.sh'             "健康检查"             "monitor" "检查所有服务的健康状态"
    register_cmd "logs"        '${MODULES_DIR}/monitor/logs.sh'               "查看运行日志"          "monitor" "查看 NewAPI 相关服务日志"
    register_cmd "alert"       '${MODULES_DIR}/monitor/alert.sh'              "告警检查"             "monitor" "检查告警规则并通知"

    # 多实例管理
    register_cmd "instances"   '${MODULES_DIR}/manage/instances.sh'            "多实例管理"           "manage" "管理多个 NewAPI 实例"

    # 工具集自身
    register_cmd "self-update" '${TOOLKIT_ROOT}/scripts/self-update.sh'       "更新工具集自身"        "_core" "从 GitHub 拉取最新工具集代码"
}

# 注册内置命令
_register_builtin_commands
