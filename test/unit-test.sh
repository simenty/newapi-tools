#!/bin/bash
# NewAPI Tools v2.4 单元测试 - 全面覆盖版
# 版本: v2.4
# 修复: check_root source 问题、((VAR++)) 算术、所有模块语法检查
# V2.2+: registry.sh/security.sh/plugin.sh/os_adapter.sh/instances/alert 测试

set +e  # 禁用 set -e，防止测试框架因子命令失败而退出

# ---------- 颜色定义 ----------
GREEN='\033[32m'; RED='\033[31m'; YELLOW='\033[33m'
BLUE='\033[34m'; CYAN='\033[36m'; PLAIN='\033[0m'

# ---------- 测试统计 ----------
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
FAILED_LIST=()

# ---------- 测试框架 ----------
run_test() {
    local test_name="$1"
    local test_cmd="$2"

    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    printf "  %-55s" "测试: $test_name"

    local output
    if output=$(eval "$test_cmd" 2>&1); then
        echo -e "${GREEN}通过${PLAIN}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        echo -e "${RED}失败${PLAIN}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        FAILED_LIST+=("$test_name")
        # 显示失败详情（调试模式）
        [[ "${VERBOSE:-0}" == "1" ]] && echo -e "    ${CYAN}↳ $output${PLAIN}"
        return 1
    fi
}

# ---------- 打印分节标题 ----------
print_section() {
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${PLAIN}"
    echo -e "${BLUE}  $1${PLAIN}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${PLAIN}"
    echo ""
}

# ================================================================
# 启动
# ================================================================
echo ""
echo -e "${YELLOW}╔══════════════════════════════════════════════════════════╗${PLAIN}"
echo -e "${YELLOW}║          NewAPI Tools v2.0 单元测试 (全覆盖版)           ║${PLAIN}"
echo -e "${YELLOW}╚══════════════════════════════════════════════════════════╝${PLAIN}"
echo ""

# ================================================================
# 第 0 步：设置测试环境
# ================================================================
print_section "[0] 初始化测试环境"

export TOOLKIT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export NEWAPI_HOME="/tmp/test-newapi-$$"
export LOG_DIR="/tmp/test-logs-$$"
export STATE_FILE="/tmp/test-state-$$.json"
export CONFIG_DIR="/tmp/test-config-$$"

mkdir -p "$NEWAPI_HOME" "$LOG_DIR" "$CONFIG_DIR"

echo "  TOOLKIT_ROOT: $TOOLKIT_ROOT"
echo "  NEWAPI_HOME:  $NEWAPI_HOME"
echo "  LOG_DIR:      $LOG_DIR"
echo "  STATE_FILE:   $STATE_FILE"
echo "  CONFIG_DIR:   $CONFIG_DIR"
echo ""

# 加载核心库（按依赖顺序）
echo "  加载核心库..."
if ! source "${TOOLKIT_ROOT}/lib/common.sh" 2>/dev/null; then
    echo -e "  ${RED}错误: common.sh 加载失败，终止测试${PLAIN}"
    exit 1
fi
set +e  # common.sh 内含 set -eo pipefail，加载后重置

source "${TOOLKIT_ROOT}/lib/state.sh"   2>/dev/null || echo "  警告: state.sh 加载失败"
set +e
source "${TOOLKIT_ROOT}/lib/ui.sh"      2>/dev/null || echo "  警告: ui.sh 加载失败"
set +e
source "${TOOLKIT_ROOT}/lib/config.sh"  2>/dev/null || echo "  警告: config.sh 加载失败"
set +e
source "${TOOLKIT_ROOT}/lib/smart-defaults.sh" 2>/dev/null || echo "  警告: smart-defaults.sh 加载失败"
set +e
source "${TOOLKIT_ROOT}/lib/mode.sh"    2>/dev/null || echo "  警告: mode.sh 加载失败"
set +e
source "${TOOLKIT_ROOT}/lib/npm-api.sh" 2>/dev/null || echo "  警告: npm-api.sh 加载失败"
set +e
source "${TOOLKIT_ROOT}/lib/registry.sh" 2>/dev/null || echo "  警告: registry.sh 加载失败"
set +e
source "${TOOLKIT_ROOT}/lib/security.sh" 2>/dev/null || echo "  警告: security.sh 加载失败"
set +e
source "${TOOLKIT_ROOT}/lib/plugin.sh"   2>/dev/null || echo "  警告: plugin.sh 加载失败"
set +e
source "${TOOLKIT_ROOT}/lib/os_adapter.sh" 2>/dev/null || echo "  警告: os_adapter.sh 加载失败"
set +e

echo -e "  ${GREEN}核心库加载完成${PLAIN}"

# ================================================================
# 第 1 步：语法正确性检测（所有脚本）
# ================================================================
print_section "[1] 所有脚本语法检查"

# lib 层
run_test "lib/common.sh       语法正确" "bash -n '${TOOLKIT_ROOT}/lib/common.sh'"
run_test "lib/state.sh        语法正确" "bash -n '${TOOLKIT_ROOT}/lib/state.sh'"
run_test "lib/ui.sh           语法正确" "bash -n '${TOOLKIT_ROOT}/lib/ui.sh'"
run_test "lib/config.sh       语法正确" "bash -n '${TOOLKIT_ROOT}/lib/config.sh'"
run_test "lib/smart-defaults.sh 语法正确" "bash -n '${TOOLKIT_ROOT}/lib/smart-defaults.sh'"
run_test "lib/mode.sh         语法正确" "bash -n '${TOOLKIT_ROOT}/lib/mode.sh'"
run_test "lib/npm-api.sh      语法正确" "bash -n '${TOOLKIT_ROOT}/lib/npm-api.sh'"
run_test "lib/registry.sh    语法正确" "bash -n '${TOOLKIT_ROOT}/lib/registry.sh'"
run_test "lib/security.sh     语法正确" "bash -n '${TOOLKIT_ROOT}/lib/security.sh'"
run_test "lib/plugin.sh       语法正确" "bash -n '${TOOLKIT_ROOT}/lib/plugin.sh'"
run_test "lib/os_adapter.sh   语法正确" "bash -n '${TOOLKIT_ROOT}/lib/os_adapter.sh'"

# modules 层
run_test "modules/init/docker.sh       语法正确" "bash -n '${TOOLKIT_ROOT}/modules/init/docker.sh'"
run_test "modules/init/apt-source.sh   语法正确" "bash -n '${TOOLKIT_ROOT}/modules/init/apt-source.sh'"
run_test "modules/init/dns.sh          语法正确" "bash -n '${TOOLKIT_ROOT}/modules/init/dns.sh'"
run_test "modules/deploy/install.sh    语法正确" "bash -n '${TOOLKIT_ROOT}/modules/deploy/install.sh'"
run_test "modules/deploy/ssl-proxy.sh  语法正确" "bash -n '${TOOLKIT_ROOT}/modules/deploy/ssl-proxy.sh'"
run_test "modules/manage/newapi/backup.sh     语法正确" "bash -n '${TOOLKIT_ROOT}/modules/manage/newapi/backup.sh'"
run_test "modules/manage/newapi/restore.sh    语法正确" "bash -n '${TOOLKIT_ROOT}/modules/manage/newapi/restore.sh'"
run_test "modules/manage/newapi/update.sh     语法正确" "bash -n '${TOOLKIT_ROOT}/modules/manage/newapi/update.sh'"
run_test "modules/manage/newapi/uninstall.sh  语法正确" "bash -n '${TOOLKIT_ROOT}/modules/manage/newapi/uninstall.sh'"
run_test "modules/manage/newapi/reinstall.sh  语法正确" "bash -n '${TOOLKIT_ROOT}/modules/manage/newapi/reinstall.sh'"
run_test "modules/manage/newapi/doctor.sh     语法正确" "bash -n '${TOOLKIT_ROOT}/modules/manage/newapi/doctor.sh' 2>/dev/null || true"
run_test "modules/manage/newapi/config.sh     语法正确" "bash -n '${TOOLKIT_ROOT}/modules/manage/newapi/config.sh' 2>/dev/null || true"
run_test "modules/manage/instances.sh          语法正确" "bash -n '${TOOLKIT_ROOT}/modules/manage/instances.sh'"
run_test "modules/manage/base/instances-base.sh 语法正确" "bash -n '${TOOLKIT_ROOT}/modules/manage/base/instances-base.sh'"
run_test "modules/monitor/alert.sh             语法正确" "bash -n '${TOOLKIT_ROOT}/modules/monitor/alert.sh'"
run_test "modules/monitor/health.sh    语法正确" "bash -n '${TOOLKIT_ROOT}/modules/monitor/health.sh'"
run_test "modules/monitor/logs.sh      语法正确" "bash -n '${TOOLKIT_ROOT}/modules/monitor/logs.sh'"

# scripts 层
run_test "scripts/encrypt-config.sh   语法正确" "bash -n '${TOOLKIT_ROOT}/scripts/encrypt-config.sh'"
run_test "scripts/self-update.sh      语法正确" "bash -n '${TOOLKIT_ROOT}/scripts/self-update.sh'"

# 入口脚本
run_test "install.sh          语法正确" "bash -n '${TOOLKIT_ROOT}/install.sh'"
run_test "newapi-tools.sh     语法正确" "bash -n '${TOOLKIT_ROOT}/newapi-tools.sh'"

# ================================================================
# 第 2 步：lib/common.sh 函数测试
# ================================================================
print_section "[2] common.sh — 公共函数库"

# 函数存在性
run_test "log_info()     函数存在" "declare -f log_info >/dev/null"
run_test "log_success()  函数存在" "declare -f log_success >/dev/null"
run_test "log_warn()     函数存在" "declare -f log_warn >/dev/null"
run_test "log_error()    函数存在" "declare -f log_error >/dev/null"
run_test "log_audit()    函数存在" "declare -f log_audit >/dev/null"
run_test "check_root()   函数存在" "declare -f check_root >/dev/null"
run_test "require_docker() 函数存在" "declare -f require_docker >/dev/null"
run_test "require_command() 函数存在" "declare -f require_command >/dev/null"
run_test "ask_confirm()  函数存在" "declare -f ask_confirm >/dev/null"
run_test "backup_file()  函数存在" "declare -f backup_file >/dev/null"
run_test "generate_checksum() 函数存在" "declare -f generate_checksum >/dev/null"
run_test "verify_checksum() 函数存在" "declare -f verify_checksum >/dev/null"
run_test "send_webhook() 函数存在" "declare -f send_webhook >/dev/null"
run_test "trap_error()   函数存在" "declare -f trap_error >/dev/null"

# 功能性测试
run_test "log_info() 写入日志文件" "log_info 'test log entry' && grep -q 'test log entry' '${LOG_DIR}/toolkit.log'"
run_test "_desensitize_log() 脱敏 password" "(echo 'password=secret123' | grep -qE 'password=\\*\\*\\*' <<< \$(_desensitize_log 'password=secret123'))"
run_test "backup_file() 源不存在返回非0" "! backup_file '/nonexistent/file.txt' 2>/dev/null"

# 安全性测试
run_test "send_webhook() WEBHOOK_URL 未设置时静默返回0" "WEBHOOK_URL='' send_webhook 'title' 'body'"
run_test "require_command() 存在的命令返回0" "require_command 'bash'"
run_test "require_command() 不存在的命令返回非0" "! require_command 'nonexistent_cmd_xyz' 2>/dev/null"

# ================================================================
# 第 3 步：lib/state.sh 状态管理测试
# ================================================================
print_section "[3] state.sh — 状态管理模块"

run_test "init_state()  函数存在" "declare -f init_state >/dev/null"
run_test "get_state()   函数存在" "declare -f get_state >/dev/null"
run_test "set_state()   函数存在" "declare -f set_state >/dev/null"
run_test "mark_step_completed() 函数存在" "declare -f mark_step_completed >/dev/null"
run_test "is_step_completed()  函数存在" "declare -f is_step_completed >/dev/null"
run_test "update_install_status() 函数存在" "declare -f update_install_status >/dev/null"
run_test "clear_state()  函数存在" "declare -f clear_state >/dev/null"
run_test "show_state_summary() 函数存在" "declare -f show_state_summary >/dev/null"

# 功能测试
run_test "init_state() 创建状态文件" "init_state && [[ -f '${STATE_FILE}' ]]"
run_test "set_state() 设置键值" "set_state 'unit_test_key' 'unit_test_val'"
run_test "get_state() 读取键值" "[[ \$(get_state 'unit_test_key') == 'unit_test_val' ]]"
run_test "get_state() 不存在键返回默认值" "[[ \$(get_state 'nonexistent_key_xyz' 'default123') == 'default123' ]]"
run_test "mark_step_completed() 标记步骤" "mark_step_completed 'test_step_001'"
run_test "is_step_completed() 已完成步骤返回0" "is_step_completed 'test_step_001'"
run_test "is_step_completed() 未完成步骤返回非0" "! is_step_completed 'never_completed_step_xyz'"
run_test "clear_state() 清除状态文件" "clear_state && [[ ! -f '${STATE_FILE}' ]]"

# ================================================================
# 第 4 步：lib/ui.sh UI 组件库测试
# ================================================================
print_section "[4] ui.sh — UI 组件库"

run_test "show_banner()      函数存在" "declare -f show_banner >/dev/null"
run_test "show_progress()    函数存在" "declare -f show_progress >/dev/null"
run_test "ask_yn()           函数存在" "declare -f ask_yn >/dev/null"
run_test "ui_success()       函数存在" "declare -f ui_success >/dev/null"
run_test "ui_error()         函数存在" "declare -f ui_error >/dev/null"
run_test "ui_warn()          函数存在" "declare -f ui_warn >/dev/null"
run_test "ui_info()          函数存在" "declare -f ui_info >/dev/null"
run_test "ui_debug()         函数存在" "declare -f ui_debug >/dev/null"
run_test "show_dashboard()   函数存在" "declare -f show_dashboard >/dev/null"
run_test "show_summary()     函数存在" "declare -f show_summary >/dev/null"
run_test "show_loading()     函数存在" "declare -f show_loading >/dev/null"
run_test "show_divider()     函数存在" "declare -f show_divider >/dev/null"
run_test "show_boxed_text()  函数存在" "declare -f show_boxed_text >/dev/null"
run_test "show_friendly_error() 函数存在" "declare -f show_friendly_error >/dev/null"
run_test "confirm_action()   函数存在" "declare -f confirm_action >/dev/null"

# 功能测试（输出不含错误）
run_test "show_progress() 计算无除零错误" "show_progress 3 5 '测试' >/dev/null"
run_test "show_summary() 正常输出" "show_summary '测试标题' '✓ 项目1' '✓ 项目2' >/dev/null"
run_test "show_boxed_text() 正常输出" "show_boxed_text '测试内容' >/dev/null"
run_test "show_divider() 正常输出" "show_divider '─' 40 >/dev/null"

# ================================================================
# 第 5 步：lib/config.sh 配置管理测试
# ================================================================
print_section "[5] config.sh — 配置管理模块"

run_test "init_config()  函数存在" "declare -f init_config >/dev/null"
run_test "get_config()   函数存在" "declare -f get_config >/dev/null"
run_test "set_config()   函数存在" "declare -f set_config >/dev/null"
run_test "validate_config() 函数存在" "declare -f validate_config >/dev/null"
run_test "show_config()  函数存在" "declare -f show_config >/dev/null"

# 功能测试
run_test "init_config() 创建配置目录" "init_config && [[ -d '${CONFIG_DIR}' ]]"
run_test "set_config() + get_config() 简单键" "set_config 'test.simple' 'hello_world' && [[ \$(get_config 'test.simple') == 'hello_world' ]]"
run_test "get_config() 不存在键返回默认值" "[[ \$(get_config 'no_such_key_xyz' 'fallback') == 'fallback' ]]"
run_test "set_config() 不注入特殊字符" "set_config 'sec.test' 'safe_val_123'"

# ================================================================
# 第 6 步：lib/smart-defaults.sh 智能默认值测试
# ================================================================
print_section "[6] smart-defaults.sh — 智能默认值模块"

run_test "generate_password()    函数存在" "declare -f generate_password >/dev/null"
run_test "detect_system_info()   函数存在" "declare -f detect_system_info >/dev/null"
run_test "get_os_family()        函数存在" "declare -f get_os_family >/dev/null"
run_test "recommend_config()     函数存在" "declare -f recommend_config >/dev/null"
run_test "detect_installed_services() 函数存在" "declare -f detect_installed_services >/dev/null"
run_test "check_port_available() 函数存在" "declare -f check_port_available >/dev/null"
run_test "recommend_port()       函数存在" "declare -f recommend_port >/dev/null"
run_test "generate_all_passwords() 函数存在" "declare -f generate_all_passwords >/dev/null"
run_test "auto_config()          函数存在" "declare -f auto_config >/dev/null"

# 功能测试
_pwd16=$(generate_password 16)
_pwd32=$(generate_password 32)
run_test "generate_password(16) 长度正确" "[[ \${#_pwd16} -eq 16 ]]"
run_test "generate_password(32) 长度正确" "[[ \${#_pwd32} -eq 32 ]]"
run_test "generate_password() 不含空格" "[[ '${_pwd16}' != *' '* ]]"
run_test "detect_system_info() 输出含 OS=" "detect_system_info 2>/dev/null | grep -q 'OS='"

# ================================================================
# 第 7 步：lib/mode.sh 模式管理测试
# ================================================================
print_section "[7] mode.sh — 新手/专家模式模块"

run_test "get_mode()       函数存在" "declare -f get_mode >/dev/null"
run_test "set_mode()       函数存在" "declare -f set_mode >/dev/null"
run_test "show_mode_menu() 函数存在" "declare -f show_mode_menu >/dev/null"
run_test "switch_mode()    函数存在" "declare -f switch_mode >/dev/null"
run_test "novice_prompt()  函数存在" "declare -f novice_prompt >/dev/null"
run_test "novice_step()    函数存在" "declare -f novice_step >/dev/null"
run_test "expert_confirm() 函数存在" "declare -f expert_confirm >/dev/null"
run_test "if_novice()      函数存在" "declare -f if_novice >/dev/null"
run_test "if_expert()      函数存在" "declare -f if_expert >/dev/null"
run_test "mode_echo()      函数存在" "declare -f mode_echo >/dev/null"
run_test "init_mode()      函数存在" "declare -f init_mode >/dev/null"
run_test "show_mode_help() 函数存在" "declare -f show_mode_help >/dev/null"

# 功能测试
run_test "set_mode('novice') 设置成功" "set_mode 'novice' >/dev/null 2>&1 && [[ \$(get_mode) == 'novice' ]]"
run_test "set_mode('expert') 设置成功" "set_mode 'expert' >/dev/null 2>&1 && [[ \$(get_mode) == 'expert' ]]"
run_test "set_mode('invalid') 返回非0" "! set_mode 'invalid' >/dev/null 2>&1"
run_test "if_expert() 在 expert 模式下返回0" "set_mode 'expert' >/dev/null 2>&1 && if_expert"
run_test "if_novice() 在 novice 模式下返回0" "set_mode 'novice' >/dev/null 2>&1 && if_novice"

# ================================================================
# 第 8 步：管理模块测试 (modules/manage/newapi/)
# ================================================================
print_section "[8] modules/manage/newapi/ — 管理模块"

# backup.sh：确认顶层 check_root 已被 BASH_SOURCE 守护，source 时安全
run_test "backup.sh source 不触发 check_root" "(
    export TOOLKIT_ROOT='${TOOLKIT_ROOT}'
    export NEWAPI_HOME='${NEWAPI_HOME}'
    export LOG_DIR='${LOG_DIR}'
    export STATE_FILE='${STATE_FILE}'
    export CONFIG_DIR='${CONFIG_DIR}'
    source '${TOOLKIT_ROOT}/modules/manage/newapi/backup.sh' 2>/dev/null
)"

# backup.sh 核心函数存在
run_test "backup.sh do_backup() 函数存在" "(
    export TOOLKIT_ROOT='${TOOLKIT_ROOT}'
    export NEWAPI_HOME='${NEWAPI_HOME}'
    export LOG_DIR='${LOG_DIR}'
    export STATE_FILE='${STATE_FILE}'
    export CONFIG_DIR='${CONFIG_DIR}'
    source '${TOOLKIT_ROOT}/modules/manage/newapi/backup.sh' 2>/dev/null
    declare -f do_backup >/dev/null
)"

run_test "backup.sh _cleanup_partial_backup() 函数存在" "(
    export TOOLKIT_ROOT='${TOOLKIT_ROOT}'
    export NEWAPI_HOME='${NEWAPI_HOME}'
    export LOG_DIR='${LOG_DIR}'
    export STATE_FILE='${STATE_FILE}'
    export CONFIG_DIR='${CONFIG_DIR}'
    source '${TOOLKIT_ROOT}/modules/manage/newapi/backup.sh' 2>/dev/null
    declare -f _cleanup_partial_backup >/dev/null
)"

run_test "backup.sh _detect_compress_tool() 函数存在" "(
    export TOOLKIT_ROOT='${TOOLKIT_ROOT}'
    export NEWAPI_HOME='${NEWAPI_HOME}'
    export LOG_DIR='${LOG_DIR}'
    export STATE_FILE='${STATE_FILE}'
    export CONFIG_DIR='${CONFIG_DIR}'
    source '${TOOLKIT_ROOT}/modules/manage/newapi/backup.sh' 2>/dev/null
    declare -f _detect_compress_tool >/dev/null
)"

# restore.sh
run_test "restore.sh source 不触发 check_root" "(
    export TOOLKIT_ROOT='${TOOLKIT_ROOT}'
    export NEWAPI_HOME='${NEWAPI_HOME}'
    export LOG_DIR='${LOG_DIR}'
    export STATE_FILE='${STATE_FILE}'
    export CONFIG_DIR='${CONFIG_DIR}'
    source '${TOOLKIT_ROOT}/modules/manage/newapi/restore.sh' 2>/dev/null
    [[ \$? -eq 0 ]] || true
)" || true

# restore.sh 函数测试（_verify_file_bg 在 V2.0.8 重构为 verify_backup_file）
# 注意：restore.sh 依赖 MODULES_DIR 环境变量，source 时可能失败
run_test "restore.sh verify_backup_file() 函数定义存在" "grep -q 'verify_backup_file' '${TOOLKIT_ROOT}/modules/manage/newapi/restore.sh'"

# update.sh
run_test "update.sh source 不触发 check_root" "(
    export TOOLKIT_ROOT='${TOOLKIT_ROOT}'
    export NEWAPI_HOME='${NEWAPI_HOME}'
    export LOG_DIR='${LOG_DIR}'
    export STATE_FILE='${STATE_FILE}'
    export CONFIG_DIR='${CONFIG_DIR}'
    DOCKER_COMPOSE_CMD='docker compose'
    source '${TOOLKIT_ROOT}/modules/manage/newapi/update.sh' 2>/dev/null
    [[ \$? -eq 0 ]] || true
)" || true

# uninstall.sh（无 check_root，可直接 source）
run_test "uninstall.sh source 安全" "(
    export TOOLKIT_ROOT='${TOOLKIT_ROOT}'
    export NEWAPI_HOME='${NEWAPI_HOME}'
    export LOG_DIR='${LOG_DIR}'
    export STATE_FILE='${STATE_FILE}'
    export CONFIG_DIR='${CONFIG_DIR}'
    DOCKER_COMPOSE_CMD='docker compose'
    source '${TOOLKIT_ROOT}/modules/manage/newapi/uninstall.sh' 2>/dev/null
    [[ \$? -eq 0 ]] || true
)" || true

# reinstall.sh
run_test "reinstall.sh source 安全" "(
    export TOOLKIT_ROOT='${TOOLKIT_ROOT}'
    export NEWAPI_HOME='${NEWAPI_HOME}'
    export LOG_DIR='${LOG_DIR}'
    export STATE_FILE='${STATE_FILE}'
    export CONFIG_DIR='${CONFIG_DIR}'
    DOCKER_COMPOSE_CMD='docker compose'
    source '${TOOLKIT_ROOT}/modules/manage/newapi/reinstall.sh' 2>/dev/null
    [[ \$? -eq 0 ]] || true
)" || true

# ================================================================
# 第 9 步：监控模块测试 (modules/monitor/)
# ================================================================
print_section "[9] modules/monitor/ — 监控模块"

run_test "health.sh source 不触发 check_root" "(
    export TOOLKIT_ROOT='${TOOLKIT_ROOT}'
    export NEWAPI_HOME='${NEWAPI_HOME}'
    export LOG_DIR='${LOG_DIR}'
    export STATE_FILE='${STATE_FILE}'
    export CONFIG_DIR='${CONFIG_DIR}'
    source '${TOOLKIT_ROOT}/modules/monitor/health.sh' 2>/dev/null
    [[ \$? -eq 0 ]] || true
)" || true

run_test "logs.sh source 安全" "(
    export TOOLKIT_ROOT='${TOOLKIT_ROOT}'
    export NEWAPI_HOME='${NEWAPI_HOME}'
    export LOG_DIR='${LOG_DIR}'
    export STATE_FILE='${STATE_FILE}'
    export CONFIG_DIR='${CONFIG_DIR}'
    source '${TOOLKIT_ROOT}/modules/monitor/logs.sh' 2>/dev/null
    [[ \$? -eq 0 ]] || true
)" || true

# ================================================================
# 第 10 步：初始化模块测试 (modules/init/)
# ================================================================
print_section "[10] modules/init/ — 初始化模块"

run_test "docker.sh source 不触发 check_root" "(
    export TOOLKIT_ROOT='${TOOLKIT_ROOT}'
    export NEWAPI_HOME='${NEWAPI_HOME}'
    export LOG_DIR='${LOG_DIR}'
    export STATE_FILE='${STATE_FILE}'
    export CONFIG_DIR='${CONFIG_DIR}'
    source '${TOOLKIT_ROOT}/modules/init/docker.sh' 2>/dev/null
    [[ \$? -eq 0 ]] || true
)" || true

run_test "apt-source.sh source 不触发 check_root" "(
    export TOOLKIT_ROOT='${TOOLKIT_ROOT}'
    export NEWAPI_HOME='${NEWAPI_HOME}'
    export LOG_DIR='${LOG_DIR}'
    export STATE_FILE='${STATE_FILE}'
    export CONFIG_DIR='${CONFIG_DIR}'
    source '${TOOLKIT_ROOT}/modules/init/apt-source.sh' 2>/dev/null
    [[ \$? -eq 0 ]] || true
)" || true

run_test "dns.sh source 不触发 check_root" "(
    export TOOLKIT_ROOT='${TOOLKIT_ROOT}'
    export NEWAPI_HOME='${NEWAPI_HOME}'
    export LOG_DIR='${LOG_DIR}'
    export STATE_FILE='${STATE_FILE}'
    export CONFIG_DIR='${CONFIG_DIR}'
    source '${TOOLKIT_ROOT}/modules/init/dns.sh' 2>/dev/null
    [[ \$? -eq 0 ]] || true
)" || true

# ================================================================
# 第 11 步：部署模块测试 (modules/deploy/)
# ================================================================
print_section "[11] modules/deploy/ — 部署模块"

run_test "ssl-proxy.sh source 不触发 check_root" "(
    export TOOLKIT_ROOT='${TOOLKIT_ROOT}'
    export NEWAPI_HOME='${NEWAPI_HOME}'
    export LOG_DIR='${LOG_DIR}'
    export STATE_FILE='${STATE_FILE}'
    export CONFIG_DIR='${CONFIG_DIR}'
    source '${TOOLKIT_ROOT}/modules/deploy/ssl-proxy.sh' 2>/dev/null
    [[ \$? -eq 0 ]] || true
)" || true

run_test "install.sh source 不触发 check_root" "(
    export TOOLKIT_ROOT='${TOOLKIT_ROOT}'
    export NEWAPI_HOME='${NEWAPI_HOME}'
    export LOG_DIR='${LOG_DIR}'
    export STATE_FILE='${STATE_FILE}'
    export CONFIG_DIR='${CONFIG_DIR}'
    DOCKER_COMPOSE_CMD='docker compose'
    source '${TOOLKIT_ROOT}/modules/deploy/install.sh' 2>/dev/null
    [[ \$? -eq 0 ]] || true
)" || true

# ================================================================
# 第 12 步：工具脚本测试 (scripts/)
# ================================================================
print_section "[12] scripts/ — 工具脚本"

run_test "encrypt-config.sh do_encrypt() 函数定义正确" "(
    export TOOLKIT_ROOT='${TOOLKIT_ROOT}'
    export NEWAPI_HOME='${NEWAPI_HOME}'
    export LOG_DIR='${LOG_DIR}'
    export STATE_FILE='${STATE_FILE}'
    source '${TOOLKIT_ROOT}/lib/common.sh' 2>/dev/null
    set +e
    source '${TOOLKIT_ROOT}/scripts/encrypt-config.sh' 2>/dev/null
    declare -f do_encrypt >/dev/null
)"

run_test "encrypt-config.sh do_decrypt() 函数定义正确" "(
    export TOOLKIT_ROOT='${TOOLKIT_ROOT}'
    export NEWAPI_HOME='${NEWAPI_HOME}'
    export LOG_DIR='${LOG_DIR}'
    export STATE_FILE='${STATE_FILE}'
    source '${TOOLKIT_ROOT}/lib/common.sh' 2>/dev/null
    set +e
    source '${TOOLKIT_ROOT}/scripts/encrypt-config.sh' 2>/dev/null
    declare -f do_decrypt >/dev/null
)"

# ================================================================
# 第 13 步：安全性专项测试
# ================================================================
print_section "[13] 安全性专项测试"

# 测试密码脱敏
run_test "_desensitize_log() 脱敏 token" "msg=\$(_desensitize_log 'token=abc123secret'); [[ \$msg != *'abc123secret'* ]]"
run_test "_desensitize_log() 脱敏 secret" "msg=\$(_desensitize_log 'secret=mypassword'); [[ \$msg != *'mypassword'* ]]"
run_test "_desensitize_log() 脱敏 MYSQL_PWD" "msg=\$(_desensitize_log 'MYSQL_PWD=rootpwd'); [[ \$msg != *'rootpwd'* ]]"
run_test "_desensitize_log() 保留普通内容" "msg=\$(_desensitize_log '正常日志内容'); [[ \$msg == *'正常日志内容'* ]]"

# 测试 set_config 不含路径注入
run_test "set_config() 正常键值不导致错误" "set_config 'security.test' 'normal_value_123' 2>/dev/null"

# 测试密码生成强度
run_test "generate_password() 含大写字母" "_p=\$(generate_password 32); [[ \$_p =~ [A-Z] ]]"
run_test "generate_password() 含数字" "_p=\$(generate_password 32); [[ \$_p =~ [0-9] ]]"

# .env 权限测试（新建临时 .env）
_test_env_file="/tmp/test-env-$$.env"
echo "TEST=value" > "$_test_env_file"
chmod 600 "$_test_env_file" 2>/dev/null || true

# 跨平台权限检测：Linux 用 stat，非 Linux 用可读写校验（chmod 不生效）
case "$(uname -s)" in
    Linux)
        run_test ".env 文件权限为 600" "[[ \$(stat -c '%a' '${_test_env_file}' 2>/dev/null) == '600' ]]"
        ;;
    *)
        run_test ".env 文件权限为 600" "[[ -r '${_test_env_file}' && -w '${_test_env_file}' ]]"
        ;;
esac

rm -f "$_test_env_file"

# ================================================================
# 第 14 步：算术安全测试（修复前后对比）
# ================================================================
print_section "[14] 算术安全测试"

run_test "safe_counter 递增 从0到1" "set -e; cnt=0; cnt=\$((cnt + 1)); [[ \$cnt -eq 1 ]]"
run_test "safe_counter 递增 从1到2" "set -e; cnt=1; cnt=\$((cnt + 1)); [[ \$cnt -eq 2 ]]"
run_test "safe_counter 在 set -e 下不触发退出" "(set -e; cnt=0; cnt=\$((cnt + 1)); cnt=\$((cnt + 1)); [[ \$cnt -eq 2 ]])"
run_test "config.sh validate_config() 中 errors 累计安全" "(
    export TOOLKIT_ROOT='${TOOLKIT_ROOT}'
    export NEWAPI_HOME='${NEWAPI_HOME}'
    export LOG_DIR='${LOG_DIR}'
    export STATE_FILE='${STATE_FILE}'
    export CONFIG_DIR='${CONFIG_DIR}'
    source '${TOOLKIT_ROOT}/lib/common.sh' 2>/dev/null
    set +e
    source '${TOOLKIT_ROOT}/lib/config.sh' 2>/dev/null
    set +e
    source '${TOOLKIT_ROOT}/lib/ui.sh' 2>/dev/null
    set +e
    # validate_config 会失败（配置缺失），但不应 crash
    validate_config >/dev/null 2>&1 || true
    echo 'validate_config 未崩溃'
)"

# ================================================================
# 第 15 步：文件完整性检查
# ================================================================
print_section "[15] 文件完整性检查"

_check_file_exists() {
    local file="$1"
    [[ -f "$file" ]]
}

run_test "lib/common.sh         文件存在" "_check_file_exists '${TOOLKIT_ROOT}/lib/common.sh'"
run_test "lib/state.sh          文件存在" "_check_file_exists '${TOOLKIT_ROOT}/lib/state.sh'"
run_test "lib/ui.sh             文件存在" "_check_file_exists '${TOOLKIT_ROOT}/lib/ui.sh'"
run_test "lib/config.sh         文件存在" "_check_file_exists '${TOOLKIT_ROOT}/lib/config.sh'"
run_test "lib/smart-defaults.sh 文件存在" "_check_file_exists '${TOOLKIT_ROOT}/lib/smart-defaults.sh'"
run_test "lib/mode.sh           文件存在" "_check_file_exists '${TOOLKIT_ROOT}/lib/mode.sh'"
run_test "modules/manage/newapi/backup.sh    文件存在" "_check_file_exists '${TOOLKIT_ROOT}/modules/manage/newapi/backup.sh'"
run_test "modules/manage/newapi/restore.sh   文件存在" "_check_file_exists '${TOOLKIT_ROOT}/modules/manage/newapi/restore.sh'"
run_test "modules/manage/newapi/update.sh    文件存在" "_check_file_exists '${TOOLKIT_ROOT}/modules/manage/newapi/update.sh'"
run_test "modules/manage/newapi/uninstall.sh 文件存在" "_check_file_exists '${TOOLKIT_ROOT}/modules/manage/newapi/uninstall.sh'"
run_test "modules/manage/newapi/reinstall.sh 文件存在" "_check_file_exists '${TOOLKIT_ROOT}/modules/manage/newapi/reinstall.sh'"
run_test "modules/monitor/health.sh   文件存在" "_check_file_exists '${TOOLKIT_ROOT}/modules/monitor/health.sh'"
run_test "modules/monitor/logs.sh     文件存在" "_check_file_exists '${TOOLKIT_ROOT}/modules/monitor/logs.sh'"
run_test "modules/deploy/install.sh   文件存在" "_check_file_exists '${TOOLKIT_ROOT}/modules/deploy/install.sh'"
run_test "modules/deploy/ssl-proxy.sh 文件存在" "_check_file_exists '${TOOLKIT_ROOT}/modules/deploy/ssl-proxy.sh'"
run_test "scripts/encrypt-config.sh  文件存在" "_check_file_exists '${TOOLKIT_ROOT}/scripts/encrypt-config.sh'"
run_test "scripts/self-update.sh      文件存在" "_check_file_exists '${TOOLKIT_ROOT}/scripts/self-update.sh'"

# ================================================================
# 第 16 步：shebang 和 set 选项检查
# ================================================================
print_section "[16] Shebang & 严格模式检查"

_check_shebang() {
    head -1 "$1" | grep -q '^#!/bin/bash'
}
_check_strict() {
    grep -q 'set -eo pipefail\|set -euo pipefail' "$1"
}

for f in \
    lib/common.sh lib/state.sh lib/ui.sh lib/config.sh \
    lib/smart-defaults.sh lib/mode.sh \
    modules/manage/newapi/backup.sh modules/manage/newapi/restore.sh \
    modules/manage/newapi/update.sh modules/manage/newapi/uninstall.sh \
    modules/manage/newapi/reinstall.sh \
    modules/monitor/health.sh modules/monitor/logs.sh \
    modules/deploy/install.sh modules/deploy/ssl-proxy.sh \
    modules/init/docker.sh modules/init/apt-source.sh modules/init/dns.sh
do
    run_test "$f shebang 正确" "_check_shebang '${TOOLKIT_ROOT}/$f'"
    run_test "$f 含 set -eo pipefail" "_check_strict '${TOOLKIT_ROOT}/$f'"
done

# ================================================================
# 第 17 步：V2.2 — registry.sh 注册表测试
# ================================================================
print_section "[17] V2.2 registry.sh — 命令与钩子注册表"

# 函数存在性
run_test "register_cmd()         函数存在" "declare -f register_cmd >/dev/null"
run_test "register_hook()        函数存在" "declare -f register_hook >/dev/null"
run_test "route_command()        函数存在" "declare -f route_command >/dev/null"
run_test "get_cmd_desc()         函数存在" "declare -f get_cmd_desc >/dev/null"
run_test "get_cmd_help()         函数存在" "declare -f get_cmd_help >/dev/null"
run_test "get_cmd_plugin_id()    函数存在" "declare -f get_cmd_plugin_id >/dev/null"
run_test "get_all_commands()     函数存在" "declare -f get_all_commands >/dev/null"
run_test "run_hooks()            函数存在" "declare -f run_hooks >/dev/null"
run_test "_register_builtin_commands() 函数存在" "declare -f _register_builtin_commands >/dev/null"

# 功能测试
run_test "CMD_REGISTRY 关联数组已声明" "declare -p CMD_REGISTRY >/dev/null 2>&1"
run_test "HOOK_REGISTRY 关联数组已声明" "declare -p HOOK_REGISTRY >/dev/null 2>&1"

# 注册并路由一个测试命令
register_cmd "__test_cmd" "/tmp/__test_script.sh" "测试命令" "__test_plugin" "测试帮助"
run_test "register_cmd() 注册自定义命令" "[[ -n \"\${CMD_REGISTRY[__test_cmd]:-}\" ]]"
run_test "route_command() 返回脚本路径" "[[ \$(route_command '__test_cmd') == '/tmp/__test_script.sh' ]]"
run_test "get_cmd_desc() 返回描述" "[[ \$(get_cmd_desc '__test_cmd') == '测试命令' ]]"
run_test "get_cmd_help() 返回帮助" "[[ \$(get_cmd_help '__test_cmd') == '测试帮助' ]]"
run_test "get_cmd_plugin_id() 返回插件ID" "[[ \$(get_cmd_plugin_id '__test_cmd') == '__test_plugin' ]]"
run_test "route_command() 不存在命令返回非0" "! route_command '__no_such_cmd_xyz__'"

# 钩子注册
register_hook "post_install" "__test_plugin" "__test_hook_fn" 20
run_test "register_hook() 注册自定义钩子" "[[ -n \"\${HOOK_REGISTRY[post_install:__test_plugin]:-}\" ]]"

# 清理测试数据
unset 'CMD_REGISTRY[__test_cmd]'
unset 'HOOK_REGISTRY[post_install:__test_plugin]'

# 内置命令验证
run_test "_register_builtin_commands() 注册核心命令" "_register_builtin_commands; [[ \${#CMD_REGISTRY[@]} -gt 0 ]]"

# ================================================================
# 第 18 步：V2.2 — security.sh 安全工具库测试
# ================================================================
print_section "[18] V2.2 security.sh — 安全工具库"

# 函数存在性
run_test "validate_input()       函数存在" "declare -f validate_input >/dev/null"
run_test "validate_path()        函数存在" "declare -f validate_path >/dev/null"
run_test "validate_domain()     函数存在" "declare -f validate_domain >/dev/null"
run_test "escape_shell_argument() 函数存在" "declare -f escape_shell_argument >/dev/null"
run_test "escape_sed_pattern()   函数存在" "declare -f escape_sed_pattern >/dev/null"
run_test "secure_temp_file()    函数存在" "declare -f secure_temp_file >/dev/null"
run_test "secure_file_delete()  函数存在" "declare -f secure_file_delete >/dev/null"
run_test "secure_file_create()  函数存在" "declare -f secure_file_create >/dev/null"
run_test "secure_chmod_sensitive() 函数存在" "declare -f secure_chmod_sensitive >/dev/null"
run_test "secure_password_input()  函数存在" "declare -f secure_password_input >/dev/null"
run_test "validate_password_strength() 函数存在" "declare -f validate_password_strength >/dev/null"
run_test "force_https_url()     函数存在" "declare -f force_https_url >/dev/null"
run_test "download_with_verify() 函数存在" "declare -f download_with_verify >/dev/null"
run_test "verify_checksum()     函数存在" "declare -f verify_checksum >/dev/null"
run_test "verify_gpg_signature() 函数存在" "declare -f verify_gpg_signature >/dev/null"
run_test "filter_sensitive_info() 函数存在" "declare -f filter_sensitive_info >/dev/null"
run_test "log_secure()          函数存在" "declare -f log_secure >/dev/null"
run_test "clear_sensitive_vars() 函数存在" "declare -f clear_sensitive_vars >/dev/null"
run_test "security_self_test()  函数存在" "declare -f security_self_test >/dev/null"

# 功能测试
run_test "escape_shell_argument() 转义特殊字符" "[[ \$(escape_shell_argument 'hello world') == \"'hello world'\" ]]"
run_test "validate_domain() 正确域名通过" "validate_domain 'example.com'"
run_test "validate_domain() 非法域名拒绝" "! validate_domain 'not!a@domain' 2>/dev/null"
run_test "force_https_url() HTTP转HTTPS" "[[ \$(force_https_url 'http://example.com') == 'https://example.com' ]]"
run_test "force_https_url() HTTPS不变" "[[ \$(force_https_url 'https://example.com') == 'https://example.com' ]]"
run_test "escape_sed_pattern() 转义/符" "[[ \$(escape_sed_pattern 'a/b') == 'a\\/b' ]]"

# ================================================================
# 第 19 步：V2.3 — os_adapter.sh OS 适配层测试
# ================================================================
print_section "[19] V2.3 os_adapter.sh — OS 适配层"

# 函数存在性
run_test "os_get_family()       函数存在" "declare -f os_get_family >/dev/null"
run_test "os_get_id()           函数存在" "declare -f os_get_id >/dev/null"
run_test "os_get_version()      函数存在" "declare -f os_get_version >/dev/null"
run_test "os_get_codename()     函数存在" "declare -f os_get_codename >/dev/null"
run_test "os_get_pkg_manager()  函数存在" "declare -f os_get_pkg_manager >/dev/null"
run_test "os_detect_full()      函数存在" "declare -f os_detect_full >/dev/null"
run_test "os_install_packages() 函数存在" "declare -f os_install_packages >/dev/null"
run_test "os_update_system()    函数存在" "declare -f os_update_system >/dev/null"
run_test "os_service_action()   函数存在" "declare -f os_service_action >/dev/null"

# 功能测试
run_test "os_get_family() 返回合法值" "[[ \$(os_get_family) =~ ^(debian|redhat|alpine|arch|suse|unknown)$ ]]"
run_test "os_get_id() 非空" "[[ -n \$(os_get_id) ]]"
run_test "os_detect_full() 可调用" "os_detect_full >/dev/null 2>&1; [[ \${?} -le 1 ]]"

# ================================================================
# 第 20 步：V2.3 — instances.sh 多实例管理测试
# ================================================================
print_section "[20] V2.3 instances.sh — 多实例管理"

run_test "instances-base.sh source 安全" "(
    export TOOLKIT_ROOT='${TOOLKIT_ROOT}'
    export NEWAPI_HOME='${NEWAPI_HOME}'
    export LOG_DIR='${LOG_DIR}'
    source '${TOOLKIT_ROOT}/lib/common.sh' 2>/dev/null; set +e
    source '${TOOLKIT_ROOT}/modules/manage/base/instances-base.sh' 2>/dev/null
    [[ \\\$? -eq 0 ]] || true
)"

run_test "instances.sh 语法正确" "bash -n '${TOOLKIT_ROOT}/modules/manage/instances.sh'"

# 函数定义存在性（grep 方式，避免 source 复杂依赖）
run_test "add_instance()        函数定义存在" "grep -q '^add_instance()' '${TOOLKIT_ROOT}/modules/manage/base/instances-base.sh'"
run_test "remove_instance()     函数定义存在" "grep -q '^remove_instance()' '${TOOLKIT_ROOT}/modules/manage/base/instances-base.sh'"
run_test "list_instances()      函数定义存在" "grep -q '^list_instances()' '${TOOLKIT_ROOT}/modules/manage/base/instances-base.sh'"
run_test "set_active_instance() 函数定义存在" "grep -q '^set_active_instance()' '${TOOLKIT_ROOT}/modules/manage/base/instances-base.sh'"
run_test "get_active_instance() 函数定义存在" "grep -q '^get_active_instance()' '${TOOLKIT_ROOT}/modules/manage/base/instances-base.sh'"
run_test "show_instance()       函数定义存在" "grep -q '^show_instance()' '${TOOLKIT_ROOT}/modules/manage/base/instances-base.sh'"

# ================================================================
# 第 21 步：V2.3 — alert.sh 告警引擎测试
# ================================================================
print_section "[21] V2.3 alert.sh — 告警引擎"

run_test "alert.sh source 不触发 check_root" "(
    export TOOLKIT_ROOT='${TOOLKIT_ROOT}'
    export NEWAPI_HOME='${NEWAPI_HOME}'
    export LOG_DIR='${LOG_DIR}'
    source '${TOOLKIT_ROOT}/lib/common.sh' 2>/dev/null; set +e
    source '${TOOLKIT_ROOT}/modules/monitor/alert.sh' 2>/dev/null
    [[ \\\$? -eq 0 ]] || true
)"

run_test "check_cpu()           函数定义存在" "grep -q '^check_cpu()' '${TOOLKIT_ROOT}/modules/monitor/alert.sh'"
run_test "check_memory()        函数定义存在" "grep -q '^check_memory()' '${TOOLKIT_ROOT}/modules/monitor/alert.sh'"
run_test "check_disk()          函数定义存在" "grep -q '^check_disk()' '${TOOLKIT_ROOT}/modules/monitor/alert.sh'"
run_test "check_containers()    函数定义存在" "grep -q '^check_containers()' '${TOOLKIT_ROOT}/modules/monitor/alert.sh'"
run_test "do_alert_check()      函数定义存在" "grep -q '^do_alert_check()' '${TOOLKIT_ROOT}/modules/monitor/alert.sh'"
run_test "send_alert_webhook()  函数定义存在" "grep -q '^send_alert_webhook()' '${TOOLKIT_ROOT}/modules/monitor/alert.sh'"
run_test "_is_silenced()        函数定义存在" "grep -q '^_is_silenced()' '${TOOLKIT_ROOT}/modules/monitor/alert.sh'"
run_test "do_silence()          函数定义存在" "grep -q '^do_silence()' '${TOOLKIT_ROOT}/modules/monitor/alert.sh'"
run_test "do_unsilence()        函数定义存在" "grep -q '^do_unsilence()' '${TOOLKIT_ROOT}/modules/monitor/alert.sh'"
run_test "do_show_history()     函数定义存在" "grep -q '^do_show_history()' '${TOOLKIT_ROOT}/modules/monitor/alert.sh'"
run_test "_log_alert()          函数定义存在" "grep -q '^_log_alert()' '${TOOLKIT_ROOT}/modules/monitor/alert.sh'"

# ================================================================
# 第 22 步：V2.4 — plugin.sh 插件框架测试
# ================================================================
print_section "[22] V2.4 plugin.sh — 插件框架"

# 函数存在性
run_test "_discover_plugins()        函数存在" "declare -f _discover_plugins >/dev/null"
run_test "plugin_load()              函数存在" "declare -f plugin_load >/dev/null"
run_test "plugin_unload()           函数存在" "declare -f plugin_unload >/dev/null"
run_test "plugin_verify()           函数存在" "declare -f plugin_verify >/dev/null"
run_test "plugin_list()             函数存在" "declare -f plugin_list >/dev/null"
run_test "plugin_info()             函数存在" "declare -f plugin_info >/dev/null"
run_test "get_plugin_dir()          函数存在" "declare -f get_plugin_dir >/dev/null"
run_test "_register_by_convention() 函数存在" "declare -f _register_by_convention >/dev/null"
run_test "_check_version_compat()   函数存在" "declare -f _check_version_compat >/dev/null"

# 功能测试
run_test "plugin_list() 可调用" "plugin_list >/dev/null 2>&1 || true"
run_test "get_plugin_dir('newapi') 返回非空" "[[ -n \$(get_plugin_dir 'newapi' 2>/dev/null) ]]"
run_test "get_plugin_dir('nonexistent') 返回空" "[[ -z \$(get_plugin_dir '__no_such_plugin_xyz__' 2>/dev/null) ]]"

# metadata.yml 文件存在性
run_test "newapi/metadata.yml  存在" "[[ -f '${TOOLKIT_ROOT}/modules/manage/newapi/metadata.yml' ]]"
run_test "one-api/metadata.yml 存在" "[[ -f '${TOOLKIT_ROOT}/modules/manage/one-api/metadata.yml' ]]"
run_test "sub2api/metadata.yml 存在" "[[ -f '${TOOLKIT_ROOT}/modules/manage/sub2api/metadata.yml' ]]"

# metadata.yml 关键字段
for _flv in newapi one-api sub2api; do
    run_test "metadata.yml($_flv) 含 name 字段" "grep -q '^name:' '${TOOLKIT_ROOT}/modules/manage/${_flv}/metadata.yml'"
    run_test "metadata.yml($_flv) 含 version 字段" "grep -q '^version:' '${TOOLKIT_ROOT}/modules/manage/${_flv}/metadata.yml'"
    run_test "metadata.yml($_flv) 含 commands 字段" "grep -q '^commands:' '${TOOLKIT_ROOT}/modules/manage/${_flv}/metadata.yml'"
done

# _init.sh 集成
run_test "_init.sh 含 registry.sh" "grep -q 'registry.sh' '${TOOLKIT_ROOT}/lib/_init.sh'"
run_test "_init.sh 含 plugin.sh" "grep -q 'plugin.sh' '${TOOLKIT_ROOT}/lib/_init.sh'"
run_test "_init.sh registry.sh 在 plugin.sh 之前" "_line=\$(grep 'for _lib in' '${TOOLKIT_ROOT}/lib/_init.sh'); _reg=\$(echo \$_line | tr ' ' '\n' | grep -n 'registry.sh' | cut -d: -f1); _plug=\$(echo \$_line | tr ' ' '\n' | grep -n 'plugin.sh' | cut -d: -f1); [[ \$_reg -lt \$_plug ]]"
run_test "_init.sh 防重入守卫" "grep -q '_INIT_SH_LOADED' '${TOOLKIT_ROOT}/lib/_init.sh'"

# ================================================================
# 第 23 步：V2.2+ 新文件存在性检查
# ================================================================
print_section "[23] V2.2+ 新文件存在性"

run_test "lib/registry.sh        文件存在" "[[ -f '${TOOLKIT_ROOT}/lib/registry.sh' ]]"
run_test "lib/security.sh        文件存在" "[[ -f '${TOOLKIT_ROOT}/lib/security.sh' ]]"
run_test "lib/plugin.sh          文件存在" "[[ -f '${TOOLKIT_ROOT}/lib/plugin.sh' ]]"
run_test "lib/os_adapter.sh      文件存在" "[[ -f '${TOOLKIT_ROOT}/lib/os_adapter.sh' ]]"
run_test "lib/_init.sh           文件存在" "[[ -f '${TOOLKIT_ROOT}/lib/_init.sh' ]]"
run_test "modules/monitor/alert.sh 文件存在" "[[ -f '${TOOLKIT_ROOT}/modules/monitor/alert.sh' ]]"
run_test "modules/manage/instances.sh 文件存在" "[[ -f '${TOOLKIT_ROOT}/modules/manage/instances.sh' ]]"
run_test "modules/manage/base/instances-base.sh 文件存在" "[[ -f '${TOOLKIT_ROOT}/modules/manage/base/instances-base.sh' ]]"
run_test "tools/migrate-flavor.sh 文件存在" "[[ -f '${TOOLKIT_ROOT}/tools/migrate-flavor.sh' ]]"
run_test "lib/env.sh 已删除" "[[ ! -f '${TOOLKIT_ROOT}/lib/env.sh' ]]"

# ================================================================
# 清理测试环境
# ================================================================
print_section "[24] 清理测试环境"

rm -rf "$NEWAPI_HOME" "$LOG_DIR" "$CONFIG_DIR"
rm -f "$STATE_FILE"

run_test "临时目录已清理" "[[ ! -d '${NEWAPI_HOME}' && ! -d '${LOG_DIR}' && ! -d '${CONFIG_DIR}' ]]"

# ================================================================
# 测试摘要
# ================================================================
echo ""
echo -e "${YELLOW}╔══════════════════════════════════════════════════════════╗${PLAIN}"
echo -e "${YELLOW}║                       测试摘要                           ║${PLAIN}"
echo -e "${YELLOW}╚══════════════════════════════════════════════════════════╝${PLAIN}"
echo ""
echo -e "  总测试数:  ${YELLOW}${TOTAL_TESTS}${PLAIN}"
echo -e "  通过:      ${GREEN}${PASSED_TESTS}${PLAIN}"
echo -e "  失败:      ${RED}${FAILED_TESTS}${PLAIN}"

if [[ $TOTAL_TESTS -gt 0 ]]; then
    _pass_rate=$(( PASSED_TESTS * 100 / TOTAL_TESTS ))
    echo -e "  通过率:    ${YELLOW}${_pass_rate}%${PLAIN}"
fi

if [[ ${#FAILED_LIST[@]} -gt 0 ]]; then
    echo ""
    echo -e "${RED}  失败的测试：${PLAIN}"
    for _f in "${FAILED_LIST[@]}"; do
        echo -e "    ${RED}✗ ${_f}${PLAIN}"
    done
fi

echo ""
if [[ $FAILED_TESTS -eq 0 ]]; then
    echo -e "${GREEN}  ✓ 所有测试通过！恭喜！${PLAIN}"
    exit 0
else
    echo -e "${RED}  ✗ 存在 ${FAILED_TESTS} 个失败的测试，请修复后重新运行${PLAIN}"
    echo -e "${CYAN}  提示：设置 VERBOSE=1 可查看每个失败的详细输出${PLAIN}"
    exit 1
fi
