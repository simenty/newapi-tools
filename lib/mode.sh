#!/bin/bash
set -eo pipefail
# 新手/专家模式模块 —— 渐进式复杂度
# 新手模式：详细提示、自动选择、进度显示
# 专家模式：所有选项、跳过提示、直接执行

# ---------- 模式定义 ----------
MODE_FILE="${CONFIG_DIR:-${TOOLKIT_ROOT}/config}/mode"

# ---------- 获取当前模式 ----------
get_mode() {
    local mode=""
    
    # 1. 从配置文件读取
    mode=$(get_config "toolkit.mode" "")
    
    # 2. 如果配置文件没有，从模式文件读取
    if [[ -z "$mode" && -f "$MODE_FILE" ]]; then
        mode=$(cat "$MODE_FILE")
    fi
    
    # 3. 默认是新手模式
    if [[ -z "$mode" ]]; then
        mode="novice"
    fi
    
    echo "$mode"
}

# ---------- 设置模式 ----------
set_mode() {
    local mode="$1"
    
    if [[ "$mode" != "novice" && "$mode" != "expert" ]]; then
        ui_error "无效模式: $mode（必须是 novice 或 expert）"
        return 1
    fi
    
    # 保存到配置文件
    set_config "toolkit.mode" "$mode"
    
    # 保存到模式文件（兼容旧版本）
    echo "$mode" > "$MODE_FILE"
    
    # 导出到环境变量
    export NOVICE_MODE=0
    [[ "$mode" == "novice" ]] && export NOVICE_MODE=1
    
    ui_success "已切换到 ${mode} 模式"
}

# ---------- 显示模式选择菜单 ----------
show_mode_menu() {
    local current_mode
    current_mode=$(get_mode)
    
    echo -e "${UI_BOLD}${UI_CYAN}╔════════════════════════════════════════════════════════════╗${UI_PLAIN}"
    echo -e "${UI_BOLD}${UI_CYAN}║                        选择使用模式                          ║${UI_PLAIN}"
    echo -e "${UI_BOLD}${UI_CYAN}╚════════════════════════════════════════════════════════════╝${UI_PLAIN}"
    echo ""
    echo "当前模式: ${UI_BOLD}${current_mode}${UI_PLAIN}"
    echo ""
    echo -e "${UI_GREEN}1)${UI_PLAIN} 新手模式 (Novice Mode)"
    echo "   - 显示详细的操作提示和说明"
    echo "   - 自动选择推荐配置（可修改）"
    echo "   - 显示操作进度和状态面板"
    echo "   - 适合：第一次使用的用户"
    echo ""
    echo -e "${UI_GREEN}2)${UI_PLAIN} 专家模式 (Expert Mode)"
    echo "   - 显示所有高级选项"
    echo "   - 跳过非必要的确认提示"
    echo "   - 直接执行，无额外输出"
    echo "   - 适合：熟悉 NewAPI 的用户"
    echo ""
    echo -e "${UI_GREEN}3)${UI_PLAIN} 保持当前模式"
    echo ""
}

# ---------- 切换模式交互 ----------
switch_mode() {
    show_mode_menu
    
    read -r -p "请选择模式 [1-3]: " choice
    
    case "$choice" in
        1)
            set_mode "novice"
            ;;
        2)
            set_mode "expert"
            ;;
        3)
            ui_info "保持当前模式: $(get_mode)"
            ;;
        *)
            ui_error "无效选择"
            switch_mode  # 重新显示菜单
            ;;
    esac
}

# ---------- 新手模式：显示详细提示 ----------
novice_prompt() {
    [[ "$(get_mode)" != "novice" ]] && return 0
    
    local message="$1"
    echo -e "${UI_CYAN}💡 提示: ${message}${UI_PLAIN}"
    echo ""
}

# ---------- 新手模式：显示操作步骤 ----------
novice_step() {
    [[ "$(get_mode)" != "novice" ]] && return 0
    
    local step="$1"
    local total="$2"
    local description="$3"
    
    echo ""
    echo -e "${UI_BOLD}${UI_BLUE}┌──────────────────────────────────────────────────────────────┐${UI_PLAIN}"
    echo -e "${UI_BOLD}${UI_BLUE}│ 步骤 [$step/$total]  $description${UI_PLAIN}"
    echo -e "${UI_BOLD}${UI_BLUE}└──────────────────────────────────────────────────────────────┘${UI_PLAIN}"
    echo ""
    
    # 显示进度条
    show_progress "$step" "$total" "总体进度"
    echo ""
}

# ---------- 专家模式：快速确认 ----------
expert_confirm() {
    [[ "$(get_mode)" != "expert" ]] && return 0
    
    # 专家模式默认同意，除非用户指定
    [[ "${FORCE_YES:-0}" == "1" ]] && return 0
    
    ask_yn "$1" "y"
}

# ---------- 根据模式决定是否显示 ----------
if_novice() {
    [[ "$(get_mode)" == "novice" ]] && return 0 || return 1
}

if_expert() {
    [[ "$(get_mode)" == "expert" ]] && return 0 || return 1
}

# ---------- 模式化输出 ----------
mode_echo() {
    local level="$1"  # novice | expert | all
    local message="$2"
    
    case "$level" in
        novice)
            if_novice && echo "$message"
            ;;
        expert)
            if_expert && echo "$message"
            ;;
        all)
            echo "$message"
            ;;
    esac
}

# ---------- 初始化模式 ----------
init_mode() {
    # 确保配置目录存在
    mkdir -p "${CONFIG_DIR:-${TOOLKIT_ROOT}/config}"
    
    # 如果模式文件不存在，创建默认
    if [[ ! -f "$MODE_FILE" ]]; then
        set_mode "novice"
    fi
    
    # 导出当前模式到环境变量
    export NOVICE_MODE=0
    [[ "$(get_mode)" == "novice" ]] && export NOVICE_MODE=1
}

# ---------- 显示模式帮助 ----------
show_mode_help() {
    local mode
    mode=$(get_mode)
    
    echo -e "${UI_BOLD}当前模式: $mode${UI_PLAIN}"
    echo ""
    
    if [[ "$mode" == "novice" ]]; then
        echo "新手模式已启用，将显示："
        echo "  ✓ 详细的操作提示"
        echo "  ✓ 推荐配置说明"
        echo "  ✓ 操作进度显示"
        echo "  ✓ 错误原因解释"
        echo ""
        echo "提示：熟悉后可以使用 'newapi-tools mode expert' 切换到专家模式"
    else
        echo "专家模式已启用，将："
        echo "  ✓ 显示所有高级选项"
        echo "  ✓ 跳过非必要提示"
        echo "  ✓ 直接执行命令"
        echo "  ✓ 最小化输出"
        echo ""
        echo "提示：如需详细提示，可以使用 'newapi-tools mode novice' 切换到新手模式"
    fi
}

# ---------- 导出函数 ----------
export -f get_mode set_mode
export -f show_mode_menu switch_mode
export -f novice_prompt novice_step
export -f expert_confirm
export -f if_novice if_expert mode_echo
export -f init_mode show_mode_help
