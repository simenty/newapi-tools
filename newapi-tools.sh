#!/bin/bash
# NewAPI 运维工具集 — 主控脚本 v2.4
# V2.4: 插件框架 + OS 适配层 + 多实例管理 + 告警引擎
set -eo pipefail

# ---------- 可靠获取工具集根目录（兼容软链接）----------
TOOLKIT_ROOT="$(cd "$(dirname "$(readlink -f "$0")")" && pwd)"
export TOOLKIT_ROOT

# ---------- 加载核心库（统一入口）----------
# shellcheck source=lib/_init.sh
source "${TOOLKIT_ROOT}/lib/_init.sh"

MODULES_DIR="${TOOLKIT_ROOT}/modules"
export MODULES_DIR

# ---------- 发现并加载插件（在 MODULES_DIR 和所有 lib 就绪后）----------
if declare -f _discover_plugins &>/dev/null; then
    _discover_plugins
fi

# ---------- 初始化 v2.0 功能 ----------
init_v2_features() {
    # 初始化状态管理
    init_state

    # 初始化配置
    init_config

    # 初始化模式
    init_mode

    log_info "v2.0 功能已初始化"
}

# 调用初始化
init_v2_features

# ---------- 命令行模式路由（注册表驱动）----------
if [[ $# -gt 0 ]]; then
    # 全局 --help / -h 支持
    if [[ "$1" == "--help" || "$1" == "-h" ]]; then
        echo "NewAPI 运维工具集 v2.4"
        echo ""
        echo "用法: newapi-tools [命令] [参数]"
        echo ""
        echo "可用命令:"
        while IFS= read -r cmd; do
            desc=$(get_cmd_desc "$cmd")
            printf "  %-15s %s\n" "$cmd" "$desc"
        done < <(get_all_commands)
        echo ""
        echo "查看命令帮助: newapi-tools <命令> --help"
        echo "查看全局帮助: newapi-tools --help"
        exit 0
    fi

    CMD="$1"
    shift

    # 从注册表获取脚本路径（路径中含变量引用，需 eval 展开）
    SCRIPT_PATH=$(eval echo "$(route_command "$CMD")")

    if [[ -z "$SCRIPT_PATH" ]]; then
        log_error "未知命令: $CMD"
        echo "可用命令: $(get_all_commands | tr '\n' ' ')"
        echo "查看帮助: newapi-tools <命令> --help"
        exit 1
    fi

    if [[ "$1" == "--help" || "$1" == "-h" ]]; then
        echo "用法: newapi-tools $CMD [参数]"
        desc=$(get_cmd_desc "$CMD")
        echo "  $desc"
        exit 0
    fi

    log_info "执行命令: $CMD ($SCRIPT_PATH)"
    exec bash "$SCRIPT_PATH" "$@"
fi

# ---------- 交互菜单（v2.0 增强版）----------
show_menu() {
    show_banner

    # 显示系统状态面板（新手模式）
    if_novice && show_dashboard

    echo -e "${GREEN}==========================================${PLAIN}"
    echo -e "${GREEN}   NewAPI 运维工具集  v2.4              ${PLAIN}"
    echo -e "${GREEN}==========================================${PLAIN}"
    echo -e " ${YELLOW}1${PLAIN})  环境准备 (DNS/换源/Docker)"
    echo -e " ${YELLOW}2${PLAIN})  安装部署 NewAPI"
    echo -e " ${YELLOW}3${PLAIN})  配置 SSL 与反向代理"
    echo -e " ${YELLOW}4${PLAIN})  更新 NewAPI (含备份与自动回滚)"
    echo -e " ${YELLOW}5${PLAIN})  数据备份 (手动备份)"
    echo -e " ${YELLOW}6${PLAIN})  备份恢复"
    echo -e " ${YELLOW}7${PLAIN})  系统健康检查"
    echo -e " ${YELLOW}8${PLAIN})  重装 / 卸载 NewAPI"
    echo -e " ${YELLOW}9${PLAIN})  查看运行日志"
    echo -e " ${YELLOW}c${PLAIN})  配置向导"
    echo -e " ${YELLOW}d${PLAIN})  系统诊断"
    echo -e " ${YELLOW}i${PLAIN})  多实例管理"
    echo -e " ${YELLOW}a${PLAIN})  告警检查"
    echo -e " ${YELLOW}0${PLAIN})  更新管理脚本自身"
    echo -e " ${YELLOW}m${PLAIN})  切换新手/专家模式 (当前: $(get_mode))"
    echo -e " ${YELLOW}s${PLAIN})  显示系统状态"
    echo -e " ${YELLOW}q${PLAIN})  退出"
    echo -e "${GREEN}==========================================${PLAIN}"
}

# 确认函数
require_confirmation() {
    local description="$1"
    echo -e "${YELLOW}即将执行：${description}${PLAIN}"
    ask_confirm "确定要继续吗？" || return 1
}

# ---------- 主循环 ----------
while true; do
    show_menu
    read -r -p "请输入选项 [0-9/a/c/d/i/m/s/q]: " choice
    case "$choice" in
        1)
            if require_confirmation "环境准备：依次执行 DNS 配置、换源与系统更新、安装 Docker。"; then
                bash "${MODULES_DIR}/init/dns.sh"
                bash "${MODULES_DIR}/init/apt-source.sh"
                bash "${MODULES_DIR}/init/docker.sh"
                # 更新状态
                mark_step_completed "dns"
                mark_step_completed "apt_source"
                mark_step_completed "docker"
                set_state "system.docker_installed" "true"
            fi
            ;;
        2)
            if require_confirmation "安装部署 NewAPI 全家桶 (MySQL+Redis+NPM)。"; then
                # 新手模式：显示智能配置
                if_novice && {
                    ui_info "正在为您生成智能配置..."
                    auto_config
                }
                bash "${MODULES_DIR}/deploy/install.sh"
                # 更新状态
                mark_step_completed "newapi_install"
                set_state "newapi.installed" "true"
            fi
            ;;
        3)
            if require_confirmation "自动配置 SSL 证书与 Nginx 反向代理（需要 NPM 账号密码）。"; then
                bash "${MODULES_DIR}/deploy/ssl-proxy.sh"
                mark_step_completed "ssl_proxy"
            fi
            ;;
        4)
            if require_confirmation "更新 NewAPI（将先自动备份，然后拉取最新镜像，若健康检查失败会自动回滚）。"; then
                bash "${MODULES_DIR}/manage/newapi/update.sh"
            fi
            ;;
        5)
            if require_confirmation "手动备份数据库与核心数据（生成 MD5 校验文件）。"; then
                bash "${MODULES_DIR}/manage/newapi/backup.sh" --manual
            fi
            ;;
        6)
            if require_confirmation "从备份列表中恢复数据（将覆盖当前数据，请谨慎）。"; then
                bash "${MODULES_DIR}/manage/newapi/restore.sh"
            fi
            ;;
        7)
            # 健康检查无副作用，直接执行
            bash "${MODULES_DIR}/monitor/health.sh"
            ;;
        8)
            echo -e "\n${YELLOW}子选项：1) 重装 NewAPI   2) 彻底卸载 NewAPI${PLAIN}"
            read -r -p "请选择子项: " sub
            case "$sub" in
                1)
                    if require_confirmation "重装 NewAPI：将先备份数据，然后清空所有容器与数据目录，重新部署。"; then
                        bash "${MODULES_DIR}/manage/newapi/reinstall.sh"
                    fi
                    ;;
                2)
                    if require_confirmation "彻底卸载 NewAPI：将永久删除所有容器、数据、备份及定时任务。"; then
                        bash "${MODULES_DIR}/manage/newapi/uninstall.sh"
                        set_state "newapi.installed" "false"
                    fi
                    ;;
                *)
                    log_error "无效选项"
                    ;;
            esac
            ;;
        9)
            bash "${MODULES_DIR}/monitor/logs.sh"
            ;;
        c|C)
            bash "${MODULES_DIR}/manage/newapi/config.sh"
            ;;
        d|D)
            bash "${MODULES_DIR}/manage/newapi/doctor.sh"
            ;;
        i|I)
            bash "${MODULES_DIR}/manage/instances.sh"
            ;;
        a|A)
            bash "${MODULES_DIR}/monitor/alert.sh"
            ;;
        0)
            if require_confirmation "从 GitHub 拉取最新工具集代码并更新自身。"; then
                bash "${TOOLKIT_ROOT}/scripts/self-update.sh"
            fi
            ;;
        m|M)
            # 切换新手/专家模式
            switch_mode
            ;;
        s|S)
            # 显示系统状态
            show_dashboard
            read -r -p "按回车键返回菜单..."
            continue
            ;;
        q|Q)
            echo "再见！"
            exit 0
            ;;
        *)
            log_error "请输入正确的选项 [0-9/a/c/d/i/m/s/q]"
            ;;
    esac
    echo ""
    read -r -p "按回车键返回菜单..."
done
