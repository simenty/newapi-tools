#!/bin/bash
set -eo pipefail
# lib/plugin.sh — 插件框架（加载/卸载/校验/生命周期）
# 提供插件发现、加载、卸载、校验、列表等完整生命周期管理
# 所有公开函数使用 plugin_ 前缀，内部函数使用 _ 前缀

# 防重入守卫
[[ "${_PLUGIN_SH_LOADED:-0}" -eq 1 ]] && return 0
export _PLUGIN_SH_LOADED=1

# ---------- 全局注册表 ----------
declare -A PLUGIN_REGISTRY    # 插件ID → "metadata.yml路径:状态"
declare -A PLUGIN_COMMANDS    # 插件ID → "cmd1 cmd2 ..."（该插件注册的命令列表）
declare -A PLUGIN_HOOKS       # 插件ID → "hook1:func1 hook2:func2 ..."（该插件注册的钩子列表）

# ---------- 常量 ----------
_PLUGIN_SEARCH_DIRS=()        # 插件搜索路径（在 _init_plugin_dirs 中初始化）
_TOOLKIT_VERSION="2.4"        # 当前工具集版本号（用于兼容性检查）

# ---------- 初始化搜索路径 ----------
_init_plugin_dirs() {
    local modules_dir="${MODULES_DIR:-${TOOLKIT_ROOT}/modules}"

    # 按优先级添加搜索路径
    _PLUGIN_SEARCH_DIRS=(
        "${modules_dir}/manage"     # 现有结构：manage/newapi/, manage/base/
        "${modules_dir}/deploy"     # 部署模块也作为插件
        "${modules_dir}/monitor"    # 监控模块也作为插件
        "${modules_dir}/init"       # 初始化模块也作为插件
        "${modules_dir}/plugins"    # V3.0 结构（预留）
    )
}

# ==============================================================================
# _discover_plugins — 扫描所有插件目录并加载插件
# 启动时由 _init.sh 调用，执行全量插件发现
# ==============================================================================
_discover_plugins() {
    _init_plugin_dirs

    local search_dir sub_dir
    local discovered=0
    local loaded=0
    local skipped=0

    for search_dir in "${_PLUGIN_SEARCH_DIRS[@]}"; do
        # 跳过不存在的目录
        if [[ ! -d "$search_dir" ]]; then
            continue
        fi

        for sub_dir in "$search_dir"/*/; do
            # 跳过非目录或不存在（glob 无匹配时）
            [[ ! -d "$sub_dir" ]] && continue

            # 去掉末尾斜杠
            sub_dir="${sub_dir%/}"
            local dir_name
            dir_name=$(basename "$sub_dir")

            discovered=$((discovered + 1))

            if [[ -f "${sub_dir}/metadata.yml" ]]; then
                # V2.4+ 模式：有 metadata.yml，走标准插件加载
                if plugin_load "$sub_dir"; then
                    loaded=$((loaded + 1))
                else
                    skipped=$((skipped + 1))
                fi
            elif _has_sh_files "$sub_dir"; then
                # 兼容模式：无 metadata.yml 但有 .sh 文件，按约定注册
                _register_by_convention "$sub_dir"
                loaded=$((loaded + 1))
            fi
        done
    done

    log_debug "插件发现完成: 发现 ${discovered} 个目录, 加载 ${loaded} 个, 跳过 ${skipped} 个"
}

# ---------- 检查目录下是否有 .sh 文件 ----------
_has_sh_files() {
    local dir="$1"
    local sh_count
    sh_count=$(find "$dir" -maxdepth 1 -name "*.sh" -type f 2>/dev/null | wc -l)
    [[ "$sh_count" -gt 0 ]]
}

# ==============================================================================
# plugin_load — 加载插件（解析 metadata.yml 并注册命令和钩子）
# 参数: plugin_dir — 插件目录路径
# 返回: 0=成功, 1=失败
# ==============================================================================
plugin_load() {
    local plugin_dir="$1"
    local metadata_path="${plugin_dir}/metadata.yml"

    # 检查 yq 是否可用
    if ! command -v yq &>/dev/null; then
        log_warn "yq 未安装，无法解析 metadata.yml: ${metadata_path}（提示: 安装 yq 以启用插件元数据功能）"
        # 降级为约定模式
        _register_by_convention "$plugin_dir"
        return 0
    fi

    # 校验 metadata.yml 完整性
    if ! plugin_verify "$metadata_path"; then
        log_warn "插件元数据校验失败，跳过: ${plugin_dir}"
        return 1
    fi

    # 提取必填字段
    local plugin_id display_name version
    plugin_id=$(yq '.name' "$metadata_path" 2>/dev/null)
    display_name=$(yq '.display_name // .name' "$metadata_path" 2>/dev/null)
    version=$(yq '.version' "$metadata_path" 2>/dev/null)

    # 检查是否已加载（防重入）
    if [[ -n "${PLUGIN_REGISTRY[$plugin_id]:-}" ]]; then
        log_warn "插件 ${plugin_id} 已加载，跳过重复加载"
        return 0
    fi

    # 版本兼容性检查
    local min_version
    min_version=$(yq '.min_toolkit_version // ""' "$metadata_path" 2>/dev/null)
    if [[ -n "$min_version" && "$min_version" != "null" ]]; then
        if ! _check_version_compat "$min_version" "$_TOOLKIT_VERSION"; then
            log_warn "插件 ${plugin_id} (v${version}) 需要工具集 v${min_version}+，当前 v${_TOOLKIT_VERSION}，跳过"
            return 1
        fi
    fi

    # 注册命令
    local cmd_count=0
    local cmd_names=""
    local commands_yaml
    commands_yaml=$(yq '.commands // []' "$metadata_path" 2>/dev/null)

    if [[ "$commands_yaml" != "null" && "$commands_yaml" != "[]" && -n "$commands_yaml" ]]; then
        local cmd_count_total
        cmd_count_total=$(yq '.commands | length' "$metadata_path" 2>/dev/null)

        local i
        for (( i = 0; i < cmd_count_total; i++ )); do
            local cmd_name cmd_script cmd_desc cmd_help
            cmd_name=$(yq ".commands[$i].name" "$metadata_path" 2>/dev/null)
            cmd_script=$(yq ".commands[$i].script" "$metadata_path" 2>/dev/null)
            cmd_desc=$(yq ".commands[$i].desc" "$metadata_path" 2>/dev/null)
            cmd_help=$(yq ".commands[$i].help // \"\"" "$metadata_path" 2>/dev/null)

            # 脚本路径解析：相对路径基于插件目录
            if [[ "$cmd_script" != /* ]]; then
                cmd_script="${plugin_dir}/${cmd_script}"
            fi

            register_cmd "$cmd_name" "$cmd_script" "$cmd_desc" "$plugin_id" "$cmd_help"
            cmd_names="${cmd_names}${cmd_name} "
            cmd_count=$((cmd_count + 1))
        done
    fi

    # 注册钩子
    local hook_count=0
    local hook_names=""
    local hooks_yaml
    hooks_yaml=$(yq '.hooks // []' "$metadata_path" 2>/dev/null)

    if [[ "$hooks_yaml" != "null" && "$hooks_yaml" != "[]" && -n "$hooks_yaml" ]]; then
        local hook_count_total
        hook_count_total=$(yq '.hooks | length' "$metadata_path" 2>/dev/null)

        for (( i = 0; i < hook_count_total; i++ )); do
            local hook_name hook_func hook_priority
            hook_name=$(yq ".hooks[$i].name" "$metadata_path" 2>/dev/null)
            hook_func=$(yq ".hooks[$i].function" "$metadata_path" 2>/dev/null)
            hook_priority=$(yq ".hooks[$i].priority // 10" "$metadata_path" 2>/dev/null)

            register_hook "$hook_name" "$plugin_id" "$hook_func" "$hook_priority"
            hook_names="${hook_names}${hook_name}:${hook_func} "
            hook_count=$((hook_count + 1))
        done
    fi

    # 更新插件注册表
    PLUGIN_REGISTRY["$plugin_id"]="${metadata_path}:loaded"
    PLUGIN_COMMANDS["$plugin_id"]="${cmd_names% }"    # 去掉末尾空格
    PLUGIN_HOOKS["$plugin_id"]="${hook_names% }"

    log_info "插件已加载: ${display_name} (${plugin_id}) v${version} — ${cmd_count} 命令, ${hook_count} 钩子"
    return 0
}

# ==============================================================================
# plugin_unload — 卸载插件
# 参数: plugin_id — 插件ID
# 返回: 0=成功, 1=插件未找到
# ==============================================================================
plugin_unload() {
    local plugin_id="$1"

    # 检查插件是否存在
    if [[ -z "${PLUGIN_REGISTRY[$plugin_id]:-}" ]]; then
        log_warn "插件 ${plugin_id} 未加载，无法卸载"
        return 1
    fi

    # 从 CMD_REGISTRY 删除该插件注册的命令
    local cmd_names="${PLUGIN_COMMANDS[$plugin_id]:-}"
    if [[ -n "$cmd_names" ]]; then
        local cmd
        for cmd in $cmd_names; do
            if [[ -n "${CMD_REGISTRY[$cmd]:-}" ]]; then
                unset "CMD_REGISTRY[$cmd]"
                log_debug "已卸载命令: ${cmd} (来自插件 ${plugin_id})"
            fi
        done
    fi

    # 从 HOOK_REGISTRY 删除该插件注册的钩子
    local hook_entries="${PLUGIN_HOOKS[$plugin_id]:-}"
    if [[ -n "$hook_entries" ]]; then
        local entry
        for entry in $hook_entries; do
            # entry 格式: "hook_name:func_name"
            local hook_name="${entry%%:*}"
            local hook_key="${hook_name}:${plugin_id}"
            if [[ -n "${HOOK_REGISTRY[$hook_key]:-}" ]]; then
                unset "HOOK_REGISTRY[$hook_key]"
                log_debug "已卸载钩子: ${hook_name}:${plugin_id}"
            fi
        done
    fi

    # 从插件注册表删除
    unset "PLUGIN_REGISTRY[$plugin_id]"
    unset "PLUGIN_COMMANDS[$plugin_id]"
    unset "PLUGIN_HOOKS[$plugin_id]"

    log_info "插件已卸载: ${plugin_id}"
    return 0
}

# ==============================================================================
# plugin_verify — 校验 metadata.yml 完整性
# 参数: metadata_path — metadata.yml 文件路径
# 返回: 0=通过, 1=失败
# ==============================================================================
plugin_verify() {
    local metadata_path="$1"

    # 检查文件是否存在
    if [[ ! -f "$metadata_path" ]]; then
        log_warn "metadata.yml 不存在: ${metadata_path}"
        return 1
    fi

    # 检查 yq 是否可用
    if ! command -v yq &>/dev/null; then
        log_warn "yq 不可用，无法校验 metadata.yml: ${metadata_path}"
        return 1
    fi

    # 检查必填字段: name
    local name
    name=$(yq '.name' "$metadata_path" 2>/dev/null)
    if [[ -z "$name" || "$name" == "null" ]]; then
        log_warn "metadata.yml 缺少必填字段 'name': ${metadata_path}"
        return 1
    fi

    # 检查必填字段: version
    local version
    version=$(yq '.version' "$metadata_path" 2>/dev/null)
    if [[ -z "$version" || "$version" == "null" ]]; then
        log_warn "metadata.yml 缺少必填字段 'version': ${metadata_path}"
        return 1
    fi

    # 检查 name 是否为合法 ID（小写字母、数字、连字符）
    if [[ ! "$name" =~ ^[a-z][a-z0-9-]*$ ]]; then
        log_warn "metadata.yml 的 name 字段不合法（须为小写字母开头，仅含小写字母/数字/连字符）: ${name}"
        return 1
    fi

    return 0
}

# ==============================================================================
# plugin_list — 列出所有已加载插件
# 输出: 格式化的插件列表
# ==============================================================================
plugin_list() {
    local count=${#PLUGIN_REGISTRY[@]}

    if [[ $count -eq 0 ]]; then
        echo "当前无已加载插件"
        return 0
    fi

    echo "已加载插件 (${count}):"
    echo "-------------------------------------------------------"
    printf "%-20s %-12s %-10s %s\n" "插件ID" "版本" "状态" "命令数"
    echo "-------------------------------------------------------"

    local plugin_id
    for plugin_id in "${!PLUGIN_REGISTRY[@]}"; do
        local entry="${PLUGIN_REGISTRY[$plugin_id]}"
        local status="${entry##*:}"
        local metadata_path="${entry%:*}"

        # 从 metadata.yml 获取版本号
        local version="?"
        if [[ -f "$metadata_path" ]] && command -v yq &>/dev/null; then
            version=$(yq '.version' "$metadata_path" 2>/dev/null)
            [[ "$version" == "null" ]] && version="?"
        fi

        # 计算命令数
        local cmd_count=0
        local cmd_names="${PLUGIN_COMMANDS[$plugin_id]:-}"
        if [[ -n "$cmd_names" ]]; then
            cmd_count=$(echo "$cmd_names" | wc -w)
        fi

        printf "%-20s %-12s %-10s %s\n" "$plugin_id" "v${version}" "$status" "$cmd_count"
    done

    echo "-------------------------------------------------------"
}

# ==============================================================================
# plugin_info — 查看单个插件详细信息
# 参数: plugin_id — 插件ID
# ==============================================================================
plugin_info() {
    local plugin_id="$1"

    if [[ -z "${PLUGIN_REGISTRY[$plugin_id]:-}" ]]; then
        log_warn "插件 ${plugin_id} 未找到"
        return 1
    fi

    local entry="${PLUGIN_REGISTRY[$plugin_id]}"
    local metadata_path="${entry%:*}"
    local status="${entry##*:}"

    echo "========== 插件信息 =========="
    echo "插件ID:   ${plugin_id}"
    echo "状态:     ${status}"
    echo "元数据:   ${metadata_path}"

    # 如果有 metadata.yml，输出详细信息
    if [[ -f "$metadata_path" ]] && command -v yq &>/dev/null; then
        local display_name version min_ver
        display_name=$(yq '.display_name // .name' "$metadata_path" 2>/dev/null)
        version=$(yq '.version' "$metadata_path" 2>/dev/null)
        min_ver=$(yq '.min_toolkit_version // "无"' "$metadata_path" 2>/dev/null)
        [[ "$min_ver" == "null" ]] && min_ver="无"

        echo "名称:     ${display_name}"
        echo "版本:     ${version}"
        echo "最低工具集版本: ${min_ver}"
    fi

    # 显示注册的命令
    local cmd_names="${PLUGIN_COMMANDS[$plugin_id]:-}"
    if [[ -n "$cmd_names" ]]; then
        echo ""
        echo "注册命令:"
        local cmd
        for cmd in $cmd_names; do
            local desc
            desc=$(get_cmd_desc "$cmd" 2>/dev/null || echo "")
            printf "  %-18s %s\n" "$cmd" "$desc"
        done
    fi

    # 显示注册的钩子
    local hook_entries="${PLUGIN_HOOKS[$plugin_id]:-}"
    if [[ -n "$hook_entries" ]]; then
        echo ""
        echo "注册钩子:"
        local entry2
        for entry2 in $hook_entries; do
            echo "  ${entry2}"
        done
    fi

    echo "=============================="
}

# ==============================================================================
# get_plugin_dir — 获取插件目录路径
# 参数: plugin_id — 插件ID
# 输出: 插件目录的绝对路径，未找到则输出空
# ==============================================================================
get_plugin_dir() {
    local plugin_id="$1"
    local modules_dir="${MODULES_DIR:-${TOOLKIT_ROOT}/modules}"

    # 兼容多种目录结构
    local candidate_dirs=(
        "${modules_dir}/manage/${plugin_id}"
        "${modules_dir}/plugins/${plugin_id}"
        "${modules_dir}/deploy/${plugin_id}"
        "${modules_dir}/monitor/${plugin_id}"
        "${modules_dir}/init/${plugin_id}"
    )

    local dir
    for dir in "${candidate_dirs[@]}"; do
        if [[ -d "$dir" ]]; then
            echo "$dir"
            return 0
        fi
    done

    # 尝试从注册表反查（metadata.yml 的父目录）
    local entry="${PLUGIN_REGISTRY[$plugin_id]:-}"
    if [[ -n "$entry" ]]; then
        local metadata_path="${entry%:*}"
        if [[ -f "$metadata_path" ]]; then
            echo "$(dirname "$metadata_path")"
            return 0
        fi
    fi

    # 未找到
    return 1
}

# ==============================================================================
# _register_by_convention — 约定式注册（无 metadata.yml 时的兼容模式）
# 遍历目录下所有 .sh 文件，文件名去掉 .sh 后缀即为命令名
# 参数: plugin_dir — 插件目录路径
# ==============================================================================
_register_by_convention() {
    local plugin_dir="$1"
    local plugin_id
    plugin_id=$(basename "$plugin_dir")

    # 检查是否已注册
    if [[ -n "${PLUGIN_REGISTRY[$plugin_id]:-}" ]]; then
        log_debug "插件 ${plugin_id} 已注册，跳过约定式注册"
        return 0
    fi

    local cmd_names=""
    local sh_file

    for sh_file in "$plugin_dir"/*.sh; do
        # glob 无匹配时，*.sh 会保留字面值
        [[ ! -f "$sh_file" ]] && continue

        local cmd_name
        cmd_name=$(basename "$sh_file" .sh)

        # 跳过以 -base 结尾的文件（基类脚本，不作为命令注册）
        [[ "$cmd_name" == *"-base" ]] && continue

        # 跳过已被其他插件注册的命令（内置命令优先）
        if [[ -n "${CMD_REGISTRY[$cmd_name]:-}" ]]; then
            log_debug "约定式注册跳过已注册命令: ${cmd_name}（由插件 $(get_cmd_plugin_id "$cmd_name") 注册）"
            continue
        fi

        # 为命令生成描述（默认使用文件名）
        local desc="${plugin_id} ${cmd_name}"

        register_cmd "$cmd_name" "$sh_file" "$desc" "$plugin_id"
        cmd_names="${cmd_names}${cmd_name} "
    done

    # 只有当注册了命令或目录有内容时才加入注册表
    if [[ -n "$cmd_names" ]]; then
        # 更新插件注册表（无 metadata.yml 时记录目录路径）
        PLUGIN_REGISTRY["$plugin_id"]="${plugin_dir}:loaded_compat"
        PLUGIN_COMMANDS["$plugin_id"]="${cmd_names% }"
        PLUGIN_HOOKS["$plugin_id"]=""

        log_debug "约定式注册完成: ${plugin_id} — 命令: ${cmd_names% }"
    fi
}

# ==============================================================================
# _check_version_compat — 检查版本兼容性
# 参数: required_min — 要求的最低版本, current — 当前版本
# 返回: 0=兼容, 1=不兼容
# 使用简单的主版本号比较（x.y 格式）
# ==============================================================================
_check_version_compat() {
    local required_min="$1"
    local current="$2"

    # 空值视为兼容
    [[ -z "$required_min" ]] && return 0

    # 提取主版本号和次版本号
    local req_major req_minor cur_major cur_minor
    IFS='.' read -r req_major req_minor <<< "${required_min}"
    IFS='.' read -r cur_major cur_minor <<< "${current}"

    # 默认次版本号为 0
    req_major="${req_major:-0}"
    req_minor="${req_minor:-0}"
    cur_major="${cur_major:-0}"
    cur_minor="${cur_minor:-0}"

    # 比较主版本号
    if [[ $cur_major -gt $req_major ]]; then
        return 0
    elif [[ $cur_major -lt $req_major ]]; then
        return 1
    fi

    # 主版本号相同，比较次版本号
    if [[ $cur_minor -ge $req_minor ]]; then
        return 0
    else
        return 1
    fi
}

# ---------- 导出函数 ----------
export -f plugin_load plugin_unload plugin_verify
export -f plugin_list plugin_info get_plugin_dir
export -f _discover_plugins _register_by_convention _check_version_compat
