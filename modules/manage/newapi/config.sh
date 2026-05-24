#!/bin/bash
# NewAPI 配置向导 v2.2
# V2.2: 路径更新为 manage/newapi/
set -eo pipefail

# shellcheck source=lib/_init.sh
source "${TOOLKIT_ROOT}/lib/_init.sh"

# 仅在直接执行时进行权限检查
if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    check_root
fi

# 退出如果是被 source
if [[ "${BASH_SOURCE[0]}" != "$0" ]]; then
    return 0 2>/dev/null || true
fi

# ---------- 配置类别 ----------
declare -a CONFIG_CATEGORIES=(
    "basic:基础配置"
    "network:网络配置"
    "backup:备份配置"
    "monitor:监控配置"
    "expert:专家配置"
)

# ---------- 显示当前配置 ----------
show_current_config() {
    local category="$1"
    echo -e "${UI_BOLD}${UI_CYAN}╔════════════════════════════════════════════════════════════╗${UI_PLAIN}"
    echo -e "${UI_BOLD}${UI_CYAN}║                    当前配置 ($category)                      ║${UI_PLAIN}"
    echo -e "${UI_BOLD}${UI_CYAN}╚════════════════════════════════════════════════════════════╝${UI_PLAIN}"
    echo ""

    case "$category" in
        basic)
            echo "  工具集路径:     ${TOOLKIT_ROOT}"
            echo "  安装模式:       $(get_config "basic.install_mode" "standard")"
            echo "  自动更新:       $(get_config "basic.auto_update" "true")"
            echo "  交互模式:       $(get_mode)"
            ;;
        network)
            echo "  DNS 服务器:     $(get_config "network.dns_primary" "223.5.5.5") / $(get_config "network.dns_secondary" "223.6.6.6")"
            echo "  NPM 镜像:       $(get_config "npm.registry" "https://registry.npmjs.org")"
            echo "  GitHub 代理:    $(get_config "network.github_proxy" "无")"
            ;;
        backup)
            echo "  备份目录:       $(get_config "backup.dir" "/opt/newapi/backups")"
            echo "  备份保留数:     $(get_config "backup.retention" "7")"
            echo "  自动备份:       $(get_config "backup.auto_enabled" "false")"
            echo "  备份压缩:       $(get_config "backup.compression" "gzip")"
            ;;
        monitor)
            echo "  健康检查间隔:   $(get_config "monitor.check_interval" "5") 分钟"
            echo "  CPU 告警阈值:   $(get_config "monitor.alert_cpu_threshold" "80")%"
            echo "  内存告警阈值:   $(get_config "monitor.alert_mem_threshold" "80")%"
            echo "  Webhook 通知:   $(get_config "monitor.webhook_enabled" "false")"
            ;;
        expert)
            echo "  调试模式:       $(get_config "expert.debug_mode" "false")"
            echo "  详细日志:       $(get_config "expert.verbose_log" "false")"
            echo "  跳过确认:       $(get_config "expert.skip_confirm" "false")"
            echo "  并行执行:       $(get_config "expert.parallel_exec" "false")"
            ;;
    esac

    echo ""
}

# ---------- 编辑基础配置 ----------
edit_basic_config() {
    show_current_config "basic"

    # 安装模式
    local install_mode
    install_mode=$(get_config "basic.install_mode" "standard")
    echo -e "${UI_BOLD}安装模式:${UI_PLAIN}"
    echo "  1) standard - 标准安装"
    echo "  2) minimal  - 最小安装（仅 NewAPI）"
    echo "  3) full     - 完整安装（NewAPI + 所有依赖）"
    read -r -p "选择安装模式 [当前: $install_mode]: " choice
    case "$choice" in
        1) set_config "basic.install_mode" "standard" ;;
        2) set_config "basic.install_mode" "minimal" ;;
        3) set_config "basic.install_mode" "full" ;;
    esac

    # 自动更新
    local auto_update
    auto_update=$(get_config "basic.auto_update" "true")
    if ask_yn "启用自动更新？" "$auto_update"; then
        set_config "basic.auto_update" "true"
    else
        set_config "basic.auto_update" "false"
    fi

    # 交互模式
    echo -e "${UI_BOLD}交互模式:${UI_PLAIN}"
    local current_mode
    current_mode=$(get_mode)
    echo "  当前: $current_mode (n=新手, e=专家)"
    read -r -p "切换模式 [直接输入 n 或 e]: " choice
    case "$choice" in
        n|N) switch_mode_to "novice" ;;
        e|E) switch_mode_to "expert" ;;
    esac

    ui_success "基础配置已保存"
}

# ---------- 编辑网络配置 ----------
edit_network_config() {
    show_current_config "network"

    # DNS 配置
    echo -e "${UI_BOLD}DNS 配置:${UI_PLAIN}"
    local dns_primary
    dns_primary=$(get_config "network.dns_primary" "223.5.5.5")
    read -r -p "主 DNS 服务器 [$dns_primary]: " input
    [[ -n "$input" ]] && set_config "network.dns_primary" "$input"

    local dns_secondary
    dns_secondary=$(get_config "network.dns_secondary" "223.6.6.6")
    read -r -p "备 DNS 服务器 [$dns_secondary]: " input
    [[ -n "$input" ]] && set_config "network.dns_secondary" "$input"

    # NPM 镜像
    echo -e "${UI_BOLD}NPM 镜像:${UI_PLAIN}"
    local npm_registry
    npm_registry=$(get_config "npm.registry" "https://registry.npmjs.org")
    echo "  常用选项:"
    echo "    1) https://registry.npmjs.org (官方)"
    echo "    2) https://registry.npmmirror.com (淘宝)"
    echo "    3) https://npmmirror.com/mirrors (腾讯)"
    read -r -p "选择或输入 NPM 镜像 [$npm_registry]: " choice
    case "$choice" in
        1) set_config "npm.registry" "https://registry.npmjs.org" ;;
        2) set_config "npm.registry" "https://registry.npmmirror.com" ;;
        3) set_config "npm.registry" "https://npmmirror.com/mirrors" ;;
        "") ;;  # 保持不变
        *) set_config "npm.registry" "$choice" ;;
    esac

    # GitHub 代理
    echo -e "${UI_BOLD}GitHub 代理:${UI_PLAIN}"
    local github_proxy
    github_proxy=$(get_config "network.github_proxy" "")
    read -r -p "GitHub 代理地址 [留空则不使用]: " input
    if [[ -z "$input" ]]; then
        set_config "network.github_proxy" ""
        echo "  不使用代理"
    else
        set_config "network.github_proxy" "$input"
    fi

    ui_success "网络配置已保存"
}

# ---------- 编辑备份配置 ----------
edit_backup_config() {
    show_current_config "backup"

    # 备份目录
    echo -e "${UI_BOLD}备份配置:${UI_PLAIN}"
    local backup_dir
    backup_dir=$(get_config "backup.dir" "/opt/newapi/backups")
    read -r -p "备份目录 [$backup_dir]: " input
    [[ -n "$input" ]] && set_config "backup.dir" "$input"

    # 备份保留数
    local retention
    retention=$(get_config "backup.retention" "7")
    read -r -p "备份保留数量(天) [$retention]: " input
    [[ -n "$input" ]] && set_config "backup.retention" "$input"

    # 自动备份
    local auto_backup
    auto_backup=$(get_config "backup.auto_enabled" "false")
    if ask_yn "启用自动备份？" "$auto_backup"; then
        set_config "backup.auto_enabled" "true"
    else
        set_config "backup.auto_enabled" "false"
    fi

    # 备份压缩
    echo -e "${UI_BOLD}备份压缩:${UI_PLAIN}"
    local compression
    compression=$(get_config "backup.compression" "gzip")
    echo "  1) gzip  - 标准压缩（兼容性好）"
    echo "  2) pigz  - 并行压缩（速度快，需安装）"
    echo "  3) none  - 不压缩（占用空间大）"
    read -r -p "选择压缩方式 [$compression]: " choice
    case "$choice" in
        1) set_config "backup.compression" "gzip" ;;
        2) set_config "backup.compression" "pigz" ;;
        3) set_config "backup.compression" "none" ;;
    esac

    ui_success "备份配置已保存"
}

# ---------- 编辑监控配置 ----------
edit_monitor_config() {
    show_current_config "monitor"

    # 健康检查间隔
    echo -e "${UI_BOLD}监控配置:${UI_PLAIN}"
    local check_interval
    check_interval=$(get_config "monitor.check_interval" "5")
    read -r -p "健康检查间隔(分钟) [$check_interval]: " input
    [[ -n "$input" ]] && set_config "monitor.check_interval" "$input"

    # CPU 告警阈值
    local cpu_threshold
    cpu_threshold=$(get_config "monitor.alert_cpu_threshold" "80")
    read -r -p "CPU 告警阈值(%) [$cpu_threshold]: " input
    [[ -n "$input" ]] && set_config "monitor.alert_cpu_threshold" "$input"

    # 内存告警阈值
    local mem_threshold
    mem_threshold=$(get_config "monitor.alert_mem_threshold" "80")
    read -r -p "内存告警阈值(%) [$mem_threshold]: " input
    [[ -n "$input" ]] && set_config "monitor.alert_mem_threshold" "$input"

    # Webhook 通知
    local webhook_enabled
    webhook_enabled=$(get_config "monitor.webhook_enabled" "false")
    if ask_yn "启用 Webhook 通知？" "$webhook_enabled"; then
        set_config "monitor.webhook_enabled" "true"
        read -r -p "Webhook URL: " webhook_url
        [[ -n "$webhook_url" ]] && set_config "monitor.webhook_url" "$webhook_url"
    else
        set_config "monitor.webhook_enabled" "false"
    fi

    ui_success "监控配置已保存"
}

# ---------- 编辑专家配置 ----------
edit_expert_config() {
    show_current_config "expert"

    echo -e "${UI_YELLOW}警告: 专家配置可能影响系统稳定性！${UI_PLAIN}"
    echo ""

    # 调试模式
    local debug_mode
    debug_mode=$(get_config "expert.debug_mode" "false")
    if ask_yn "启用调试模式？" "$debug_mode"; then
        set_config "expert.debug_mode" "true"
        set_config "expert.verbose_log" "true"
    else
        set_config "expert.debug_mode" "false"
    fi

    # 跳过确认
    local skip_confirm
    skip_confirm=$(get_config "expert.skip_confirm" "false")
    if ask_yn "跳过操作确认？" "$skip_confirm"; then
        echo -e "${UI_YELLOW}警告: 跳过确认可能导致误操作！${UI_PLAIN}"
        set_config "expert.skip_confirm" "true"
    else
        set_config "expert.skip_confirm" "false"
    fi

    # 并行执行
    local parallel_exec
    parallel_exec=$(get_config "expert.parallel_exec" "false")
    if ask_yn "启用并行执行？" "$parallel_exec"; then
        set_config "expert.parallel_exec" "true"
    else
        set_config "expert.parallel_exec" "false"
    fi

    ui_success "专家配置已保存"
}

# ---------- 重置配置 ----------
reset_config() {
    echo -e "${UI_RED}警告: 将重置所有配置为默认值！${UI_PLAIN}"
    if ! ask_yn "确定要继续吗？" "n"; then
        echo "操作已取消"
        return
    fi

    # 清除配置（保留状态文件）
    if [[ -f "${TOOLKIT_ROOT}/config.env" ]]; then
        rm -f "${TOOLKIT_ROOT}/config.env"
    fi

    ui_success "配置已重置为默认值"
    ui_info "请重新运行 newapi-tools config 进行配置"
}

# ---------- 导出配置 ----------
export_config() {
    local export_file="${1:-${TOOLKIT_ROOT}/config-export-$(date '+%Y%m%d-%H%M%S').env}"

    echo -e "${UI_BOLD}导出配置到:${UI_PLAIN} $export_file"

    {
        echo "# NewAPI Tools 配置导出"
        echo "# 导出时间: $(date '+%Y-%m-%d %H:%M:%S')"
        echo ""
        echo "# ===== 基础配置 ====="
        echo "BASIC_INSTALL_MODE=$(get_config "basic.install_mode" "standard")"
        echo "BASIC_AUTO_UPDATE=$(get_config "basic.auto_update" "true")"
        echo ""
        echo "# ===== 网络配置 ====="
        echo "DNS_PRIMARY=$(get_config "network.dns_primary" "223.5.5.5")"
        echo "DNS_SECONDARY=$(get_config "network.dns_secondary" "223.6.6.6")"
        echo "NPM_REGISTRY=$(get_config "npm.registry" "https://registry.npmjs.org")"
        echo ""
        echo "# ===== 备份配置 ====="
        echo "BACKUP_DIR=$(get_config "backup.dir" "/opt/newapi/backups")"
        echo "BACKUP_RETENTION=$(get_config "backup.retention" "7")"
        echo "BACKUP_AUTO_ENABLED=$(get_config "backup.auto_enabled" "false")"
        echo ""
        echo "# ===== 监控配置 ====="
        echo "MONITOR_CHECK_INTERVAL=$(get_config "monitor.check_interval" "5")"
        echo "MONITOR_ALERT_CPU=$(get_config "monitor.alert_cpu_threshold" "80")"
        echo "MONITOR_ALERT_MEM=$(get_config "monitor.alert_mem_threshold" "80")"
    } > "$export_file"

    ui_success "配置已导出到: $export_file"
}

# ---------- 导入配置 ----------
import_config() {
    local import_file="$1"

    if [[ -z "$import_file" ]]; then
        echo -e "${UI_BOLD}可用配置文件:${UI_PLAIN}"
        find "${TOOLKIT_ROOT}" -maxdepth 1 -name "config-export-*.env" -type f 2>/dev/null | while read -r f; do
            echo "  - $(basename "$f")"
        done
        echo ""
        read -r -p "输入配置文件路径: " import_file
    fi

    if [[ ! -f "$import_file" ]]; then
        ui_error "文件不存在: $import_file"
        return 1
    fi

    echo -e "${UI_YELLOW}警告: 将从 $import_file 导入配置！${UI_PLAIN}"
    if ! ask_yn "确定要继续吗？" "n"; then
        echo "操作已取消"
        return
    fi

    # 解析并导入配置
    while IFS='=' read -r key value; do
        [[ "$key" =~ ^#.*$ || -z "$key" ]] && continue
        key=$(echo "$key" | tr '[:upper:]' '[:lower:]' | sed 's/_/-/g')
        set_config "$key" "$value"
    done < "$import_file"

    ui_success "配置已从 $import_file 导入"
}

# ---------- 显示菜单 ----------
show_config_menu() {
    echo -e "${GREEN}==========================================${PLAIN}"
    echo -e "${GREEN}   NewAPI Tools 配置向导                 ${PLAIN}"
    echo -e "${GREEN}==========================================${PLAIN}"
    echo -e " ${YELLOW}1${PLAIN})  基础配置 (安装模式、自动更新)"
    echo -e " ${YELLOW}2${PLAIN})  网络配置 (DNS、NPM 镜像)"
    echo -e " ${YELLOW}3${PLAIN})  备份配置 (目录、保留策略)"
    echo -e " ${YELLOW}4${PLAIN})  监控配置 (告警阈值、Webhook)"
    echo -e " ${YELLOW}5${PLAIN})  专家配置 (调试模式等)"
    echo -e ""
    echo -e " ${YELLOW}e${PLAIN})  导出当前配置"
    echo -e " ${YELLOW}i${PLAIN})  导入配置文件"
    echo -e " ${YELLOW}r${PLAIN})  重置配置为默认值"
    echo -e " ${YELLOW}s${PLAIN})  显示所有当前配置"
    echo -e " ${YELLOW}q${PLAIN})  返回主菜单"
    echo -e "${GREEN}==========================================${PLAIN}"
}

# ---------- 主程序 ----------
main() {
    echo ""
    ui_header "配置向导"
    echo ""

    novice_prompt "配置向导帮助您自定义 NewAPI Tools 的各项参数。"

    while true; do
        show_config_menu
        read -r -p "请选择 [1-5/e/i/r/s/q]: " choice

        case "$choice" in
            1) edit_basic_config ;;
            2) edit_network_config ;;
            3) edit_backup_config ;;
            4) edit_monitor_config ;;
            5) edit_expert_config ;;
            e|E) export_config ;;
            i|I) import_config ;;
            r|R) reset_config ;;
            s|S)
                for cat in basic network backup monitor expert; do
                    show_current_config "$cat"
                done
                ;;
            q|Q) break ;;
            *) log_error "无效选项" ;;
        esac

        echo ""
        [[ "$choice" != "q" && "$choice" != "Q" ]] && read -r -p "按回车键继续..."
    done
}

main "$@"
