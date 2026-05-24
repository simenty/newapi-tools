#!/bin/bash
# NewAPI 系统诊断工具 v2.2
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

# ---------- 诊断结果收集 ----------
declare -A DIAG_RESULTS
DIAG_SCORE=100
DIAG_ISSUES=0

# ---------- 并行诊断支持 ----------
DIAG_TMP_DIR=""
_CURRENT_DIAG_RESULT_FILE=""

_cleanup_diag_tmp() {
    [[ -n "$DIAG_TMP_DIR" && -d "$DIAG_TMP_DIR" ]] && rm -rf "$DIAG_TMP_DIR" 2>/dev/null
}
trap _cleanup_diag_tmp EXIT

_run_diag_bg() {
    local diag_name="$1"
    shift
    (
        export _CURRENT_DIAG_RESULT_FILE="$DIAG_TMP_DIR/${diag_name}_results"
        : > "$_CURRENT_DIAG_RESULT_FILE"
        "$@" > "$DIAG_TMP_DIR/${diag_name}.out" 2>&1
    ) &
    echo $!
}

_aggregate_results() {
    for rf in "$DIAG_TMP_DIR"/*_results; do
        [[ -f "$rf" ]] || continue
        while read -r cn cr sv; do
            DIAG_RESULTS["$cn"]="$cr"
            case "$sv" in
                critical)
                    if [[ "$cr" != "OK" && "$cr" != "PASS" ]]; then
                        DIAG_SCORE=$((DIAG_SCORE - 30))
                        DIAG_ISSUES=$((DIAG_ISSUES + 1))
                    fi
                    ;;
                warning)
                    if [[ "$cr" != "OK" && "$cr" != "PASS" ]]; then
                        DIAG_SCORE=$((DIAG_SCORE - 15))
                        DIAG_ISSUES=$((DIAG_ISSUES + 1))
                    fi
                    ;;
            esac
        done < "$rf"
    done
}

_show_parallel_output() {
    for name in "$@"; do
        local f="$DIAG_TMP_DIR/${name}.out"
        [[ -f "$f" ]] && cat "$f"
    done
}

# ---------- 诊断函数 ----------
diag_check() {
    local check_name="$1"
    local check_result="$2"
    local severity="$3"  # critical, warning, info

    DIAG_RESULTS["$check_name"]="$check_result"

    # 并行模式：将结果写入文件，由父进程聚合
    if [[ -n "$_CURRENT_DIAG_RESULT_FILE" ]]; then
        echo "$check_name $check_result $severity" >> "$_CURRENT_DIAG_RESULT_FILE"
        return 0
    fi

    case "$severity" in
        critical)
            if [[ "$check_result" != "OK" && "$check_result" != "PASS" ]]; then
                DIAG_SCORE=$((DIAG_SCORE - 30))
                DIAG_ISSUES=$((DIAG_ISSUES + 1))
            fi
            ;;
        warning)
            if [[ "$check_result" != "OK" && "$check_result" != "PASS" ]]; then
                DIAG_SCORE=$((DIAG_SCORE - 15))
                DIAG_ISSUES=$((DIAG_ISSUES + 1))
            fi
            ;;
        info)
            # info 级别不影响分数
            ;;
    esac
}

# ---------- 系统环境诊断 ----------
diagnose_environment() {
    echo -e "${UI_BOLD}${UI_CYAN}╔════════════════════════════════════════════════════════════╗${UI_PLAIN}"
    echo -e "${UI_BOLD}${UI_CYAN}║                     系统环境诊断                           ║${UI_PLAIN}"
    echo -e "${UI_BOLD}${UI_CYAN}╚════════════════════════════════════════════════════════════╝${UI_PLAIN}"
    echo ""

    # 使用缓存的系统信息（避免重复读 /etc/os-release）
    detect_system_info_cached

    if [[ -n "$OS_NAME" ]]; then
        diag_check "os_version" "OK" "info"
        echo -e "  ${UI_GREEN}✓${UI_PLAIN} 操作系统: $OS_NAME"
    else
        diag_check "os_version" "FAIL" "warning"
        echo -e "  ${UI_RED}✗${UI_PLAIN} 操作系统: 无法检测"
    fi

    if [[ -n "$KERNEL_VERSION" ]]; then
        diag_check "kernel" "OK" "info"
        echo -e "  ${UI_GREEN}✓${UI_PLAIN} 内核版本: $KERNEL_VERSION"
    fi

    local uptime_info
    uptime_info=$(uptime -p 2>/dev/null || uptime)
    diag_check "uptime" "OK" "info"
    echo -e "  ${UI_GREEN}✓${UI_PLAIN} 运行时间: $uptime_info"

    echo ""
}

# ---------- 网络连接诊断 ----------
diagnose_network() {
    echo -e "${UI_BOLD}${UI_CYAN}╔════════════════════════════════════════════════════════════╗${UI_PLAIN}"
    echo -e "${UI_BOLD}${UI_CYAN}║                     网络连接诊断                           ║${UI_PLAIN}"
    echo -e "${UI_BOLD}${UI_CYAN}╚════════════════════════════════════════════════════════════╝${UI_PLAIN}"
    echo ""

    # DNS 解析
    if command -v nslookup &>/dev/null; then
        if nslookup google.com &>/dev/null; then
            diag_check "dns" "OK" "critical"
            echo -e "  ${UI_GREEN}✓${UI_PLAIN} DNS 解析: 正常"
        else
            diag_check "dns" "FAIL" "critical"
            echo -e "  ${UI_RED}✗${UI_PLAIN} DNS 解析: 失败"
        fi
    elif command -v ping &>/dev/null; then
        if ping -c 1 -W 2 8.8.8.8 &>/dev/null; then
            diag_check "dns" "OK" "critical"
            echo -e "  ${UI_GREEN}✓${UI_PLAIN} DNS/网络: 正常"
        else
            diag_check "dns" "FAIL" "critical"
            echo -e "  ${UI_RED}✗${UI_PLAIN} DNS/网络: 失败"
        fi
    fi

    # GitHub 连接
    if command -v curl &>/dev/null; then
        if curl -s --connect-timeout 5 -o /dev/null https://github.com; then
            diag_check "github" "OK" "info"
            echo -e "  ${UI_GREEN}✓${UI_PLAIN} GitHub 连接: 正常"
        else
            diag_check "github" "FAIL" "warning"
            echo -e "  ${UI_YELLOW}⚠${UI_PLAIN} GitHub 连接: 失败（可能影响更新功能）"
        fi
    fi

    # NPM 镜像连接
    local npm_registry
    npm_registry=$(get_config "npm.registry" "https://registry.npmjs.org")
    if command -v curl &>/dev/null; then
        if curl -s --connect-timeout 5 -o /dev/null "$npm_registry"; then
            diag_check "npm_registry" "OK" "info"
            echo -e "  ${UI_GREEN}✓${UI_PLAIN} NPM 镜像: 正常 ($npm_registry)"
        else
            diag_check "npm_registry" "FAIL" "warning"
            echo -e "  ${UI_YELLOW}⚠${UI_PLAIN} NPM 镜像: 失败 ($npm_registry)"
        fi
    fi

    echo ""
}

# ---------- Docker 环境诊断 ----------
diagnose_docker() {
    echo -e "${UI_BOLD}${UI_CYAN}╔════════════════════════════════════════════════════════════╗${UI_PLAIN}"
    echo -e "${UI_BOLD}${UI_CYAN}║                     Docker 环境诊断                        ║${UI_PLAIN}"
    echo -e "${UI_BOLD}${UI_CYAN}╚════════════════════════════════════════════════════════════╝${UI_PLAIN}"
    echo ""

    # Docker 安装
    if ! command -v docker &>/dev/null; then
        diag_check "docker_installed" "FAIL" "critical"
        echo -e "  ${UI_RED}✗${UI_PLAIN} Docker: 未安装"
        echo -e "    ${UI_DIM}修复: newapi-tools → 选项 1 安装环境${UI_PLAIN}"
        return
    fi
    diag_check "docker_installed" "OK" "critical"
    echo -e "  ${UI_GREEN}✓${UI_PLAIN} Docker: 已安装 ($(docker --version 2>/dev/null | cut -d',' -f1))"

    # Docker 服务状态
    if ! systemctl is-active docker &>/dev/null; then
        diag_check "docker_service" "FAIL" "critical"
        echo -e "  ${UI_RED}✗${UI_PLAIN} Docker 服务: 未运行"
        echo -e "    ${UI_DIM}修复: sudo systemctl start docker${UI_PLAIN}"
    else
        diag_check "docker_service" "OK" "critical"
        echo -e "  ${UI_GREEN}✓${UI_PLAIN} Docker 服务: 运行中"
    fi

    # Docker 权限
    if ! docker ps &>/dev/null; then
        diag_check "docker_permission" "FAIL" "critical"
        echo -e "  ${UI_RED}✗${UI_PLAIN} Docker 权限: 无权访问"
        echo -e "    ${UI_DIM}修复: sudo usermod -aG docker \$USER && 重新登录${UI_PLAIN}"
    else
        diag_check "docker_permission" "OK" "critical"
        echo -e "  ${UI_GREEN}✓${UI_PLAIN} Docker 权限: 正常"
    fi

    # Docker Compose
    if command -v docker-compose &>/dev/null; then
        diag_check "docker_compose" "OK" "info"
        echo -e "  ${UI_GREEN}✓${UI_PLAIN} Docker Compose: 已安装 ($(docker-compose --version 2>/dev/null))"
    elif docker compose version &>/dev/null; then
        diag_check "docker_compose" "OK" "info"
        echo -e "  ${UI_GREEN}✓${UI_PLAIN} Docker Compose: 已安装 (v2 $(docker compose version 2>/dev/null | awk '{print $4}'))"
    else
        diag_check "docker_compose" "WARN" "warning"
        echo -e "  ${UI_YELLOW}⚠${UI_PLAIN} Docker Compose: 未安装"
    fi

    echo ""
}

# ---------- NewAPI 服务诊断 ----------
diagnose_newapi() {
    echo -e "${UI_BOLD}${UI_CYAN}╔════════════════════════════════════════════════════════════╗${UI_PLAIN}"
    echo -e "${UI_BOLD}${UI_CYAN}║                    NewAPI 服务诊断                         ║${UI_PLAIN}"
    echo -e "${UI_BOLD}${UI_CYAN}╚════════════════════════════════════════════════════════════╝${UI_PLAIN}"
    echo ""

    # 检查 NewAPI 安装状态
    local newapi_installed=false
    if [[ -d "${NEWAPI_HOME:-/opt/newapi}" ]]; then
        newapi_installed=true
        diag_check "newapi_home" "OK" "critical"
        echo -e "  ${UI_GREEN}✓${UI_PLAIN} 安装目录: ${NEWAPI_HOME:-/opt/newapi} 存在"
    else
        diag_check "newapi_home" "NOT_FOUND" "critical"
        echo -e "  ${UI_RED}✗${UI_PLAIN} 安装目录: 未找到"
        echo -e "    ${UI_DIM}请先运行: newapi-tools → 选项 2 安装部署${UI_PLAIN}"
    fi

    # 检查配置文件
    if [[ -f "${NEWAPI_HOME:-/opt/newapi}/.env" ]]; then
        diag_check "newapi_env" "OK" "critical"
        echo -e "  ${UI_GREEN}✓${UI_PLAIN} 配置文件: .env 存在"
    else
        diag_check "newapi_env" "NOT_FOUND" "critical"
        echo -e "  ${UI_RED}✗${UI_PLAIN} 配置文件: .env 不存在"
    fi

    # 检查容器状态
    if ! $newapi_installed; then
        echo ""
        return
    fi

    local containers=("new-api" "mysql" "redis" "npm")
    local all_running=true

    for container in "${containers[@]}"; do
        local status
        status=$(docker inspect --format='{{.State.Status}}' "$container" 2>/dev/null || echo "not_found")

        case "$status" in
            running)
                echo -e "  ${UI_GREEN}✓${UI_PLAIN} 容器 $container: 运行中"
                ;;
            not_found)
                echo -e "  ${UI_YELLOW}⚠${UI_PLAIN} 容器 $container: 不存在"
                all_running=false
                ;;
            *)
                echo -e "  ${UI_RED}✗${UI_PLAIN} 容器 $container: $status"
                all_running=false
                ;;
        esac
    done

    if $all_running; then
        diag_check "newapi_containers" "OK" "critical"
    else
        diag_check "newapi_containers" "FAIL" "critical"
    fi

    # 健康检查
    local health_status
    health_status=$(docker inspect --format='{{.State.Health.Status}}' new-api 2>/dev/null || echo "no_healthcheck")
    if [[ "$health_status" == "healthy" ]]; then
        diag_check "newapi_health" "OK" "critical"
        echo -e "  ${UI_GREEN}✓${UI_PLAIN} 健康检查: 通过"
    elif [[ "$health_status" == "no_healthcheck" ]]; then
        diag_check "newapi_health" "WARN" "warning"
        echo -e "  ${UI_YELLOW}⚠${UI_PLAIN} 健康检查: 未配置"
    else
        diag_check "newapi_health" "FAIL" "critical"
        echo -e "  ${UI_RED}✗${UI_PLAIN} 健康检查: 失败"
    fi

    echo ""
}

# ---------- 资源状态诊断 ----------
diagnose_resources() {
    echo -e "${UI_BOLD}${UI_CYAN}╔════════════════════════════════════════════════════════════╗${UI_PLAIN}"
    echo -e "${UI_BOLD}${UI_CYAN}║                     资源状态诊断                           ║${UI_PLAIN}"
    echo -e "${UI_BOLD}${UI_CYAN}╚════════════════════════════════════════════════════════════╝${UI_PLAIN}"
    echo ""

    # 磁盘空间
    local disk_usage
    disk_usage=$(df -h / | awk 'NR==2{print $5}')
    local disk_percent
    disk_percent=$(df / | awk 'NR==2{print $5}' | tr -d '%')

    if [[ $disk_percent -ge 90 ]]; then
        diag_check "disk_space" "CRITICAL" "critical"
        echo -e "  ${UI_RED}✗${UI_PLAIN} 磁盘空间: ${disk_usage} (严重不足!)"
    elif [[ $disk_percent -ge 80 ]]; then
        diag_check "disk_space" "WARNING" "warning"
        echo -e "  ${UI_YELLOW}⚠${UI_PLAIN} 磁盘空间: ${disk_usage} (不足)"
    else
        diag_check "disk_space" "OK" "info"
        echo -e "  ${UI_GREEN}✓${UI_PLAIN} 磁盘空间: ${disk_usage}"
    fi

    # 内存
    local mem_percent
    mem_percent=$(free | awk '/Mem/{print int($3/$2*100)}')
    if [[ $mem_percent -ge 90 ]]; then
        diag_check "memory" "CRITICAL" "critical"
        echo -e "  ${UI_RED}✗${UI_PLAIN} 内存使用: ${mem_percent}% (严重不足!)"
    elif [[ $mem_percent -ge 80 ]]; then
        diag_check "memory" "WARNING" "warning"
        echo -e "  ${UI_YELLOW}⚠${UI_PLAIN} 内存使用: ${mem_percent}% (较高)"
    else
        diag_check "memory" "OK" "info"
        echo -e "  ${UI_GREEN}✓${UI_PLAIN} 内存使用: ${mem_percent}%"
    fi

    # CPU 负载
    local load_avg
    load_avg=$(cat /proc/loadavg 2>/dev/null | awk '{print $1}' || echo "0")
    local cpu_cores
    cpu_cores=$(nproc 2>/dev/null || echo "1")
    local load_percent
    load_percent=$(echo "$load_avg $cpu_cores" | awk '{print int($1/$2*100)}')

    if [[ $load_percent -ge 80 ]]; then
        diag_check "cpu_load" "WARNING" "warning"
        echo -e "  ${UI_YELLOW}⚠${UI_PLAIN} CPU 负载: $load_avg (${load_percent}%)"
    else
        diag_check "cpu_load" "OK" "info"
        echo -e "  ${UI_GREEN}✓${UI_PLAIN} CPU 负载: $load_avg (${load_percent}%)"
    fi

    echo ""
}

# ---------- 配置文件诊断 ----------
diagnose_config() {
    echo -e "${UI_BOLD}${UI_CYAN}╔════════════════════════════════════════════════════════════╗${UI_PLAIN}"
    echo -e "${UI_BOLD}${UI_CYAN}║                     配置状态诊断                           ║${UI_PLAIN}"
    echo -e "${UI_BOLD}${UI_CYAN}╚════════════════════════════════════════════════════════════╝${UI_PLAIN}"
    echo ""

    # 状态文件
    if [[ -f "${TOOLKIT_ROOT}/state.json" ]]; then
        diag_check "state_file" "OK" "info"
        echo -e "  ${UI_GREEN}✓${UI_PLAIN} 状态文件: 存在"
    else
        diag_check "state_file" "NOT_FOUND" "info"
        echo -e "  ${UI_YELLOW}⚠${UI_PLAIN} 状态文件: 不存在（首次安装？）"
    fi

    # 备份目录
    local backup_dir
    backup_dir=$(get_config "backup.dir" "/opt/newapi/backups")
    if [[ -d "$backup_dir" ]]; then
        local backup_count
        backup_count=$(find "$backup_dir" -type f \( -name "*.tar.gz" -o -name "*.sql" \) 2>/dev/null | wc -l)
        diag_check "backup_dir" "OK" "info"
        echo -e "  ${UI_GREEN}✓${UI_PLAIN} 备份目录: $backup_dir ($backup_count 个备份)"
    else
        diag_check "backup_dir" "NOT_FOUND" "warning"
        echo -e "  ${UI_YELLOW}⚠${UI_PLAIN} 备份目录: $backup_dir 不存在"
    fi

    # 日志文件
    local log_file
    log_file="${TOOLKIT_ROOT}/logs/toolkit.log"
    if [[ -f "$log_file" ]]; then
        local log_size
        log_size=$(du -h "$log_file" 2>/dev/null | awk '{print $1}' || echo "未知")
        diag_check "log_file" "OK" "info"
        echo -e "  ${UI_GREEN}✓${UI_PLAIN} 日志文件: $log_size"
    else
        diag_check "log_file" "NOT_FOUND" "info"
        echo -e "  ${UI_YELLOW}⚠${UI_PLAIN} 日志文件: 不存在"
    fi

    echo ""
}

# ---------- 显示诊断报告 ----------
show_diagnostic_report() {
    echo -e "${UI_BOLD}${UI_CYAN}╔════════════════════════════════════════════════════════════╗${UI_PLAIN}"
    echo -e "${UI_BOLD}${UI_CYAN}║                      诊断报告摘要                          ║${UI_PLAIN}"
    echo -e "${UI_BOLD}${UI_CYAN}╚════════════════════════════════════════════════════════════╝${UI_PLAIN}"
    echo ""

    # 健康分数
    if [[ $DIAG_SCORE -ge 80 ]]; then
        echo -e "  ${UI_GREEN}诊断分数: $DIAG_SCORE/100${UI_PLAIN} - 系统状态良好"
    elif [[ $DIAG_SCORE -ge 50 ]]; then
        echo -e "  ${UI_YELLOW}诊断分数: $DIAG_SCORE/100${UI_PLAIN} - 存在 $DIAG_ISSUES 个问题"
    else
        echo -e "  ${UI_RED}诊断分数: $DIAG_SCORE/100${UI_PLAIN} - 需要立即处理 $DIAG_ISSUES 个问题"
    fi

    echo ""

    # 问题列表
    if [[ $DIAG_ISSUES -gt 0 ]]; then
        echo -e "  ${UI_BOLD}发现的问题:${UI_PLAIN}"
        for key in "${!DIAG_RESULTS[@]}"; do
            local value="${DIAG_RESULTS[$key]}"
            if [[ "$value" == "FAIL" || "$value" == "CRITICAL" || "$value" == "WARNING" || "$value" == "NOT_FOUND" ]]; then
                local severity_icon="${UI_YELLOW}⚠${UI_PLAIN}"
                local severity_text="警告"
                case "$value" in
                    FAIL|CRITICAL)
                        severity_icon="${UI_RED}✗${UI_PLAIN}"
                        severity_text="错误"
                        ;;
                    WARNING)
                        severity_text="警告"
                        ;;
                    NOT_FOUND)
                        severity_text="未找到"
                        ;;
                esac
                echo -e "    $severity_icon $severity_text: $key"
            fi
        done
        echo ""
    fi

    # 建议
    echo -e "  ${UI_BOLD}建议:${UI_PLAIN}"
    if [[ $DIAG_SCORE -lt 50 ]]; then
        echo "    1. 优先解决红色标记的错误"
        echo "    2. 运行 'newapi-tools' 查看完整菜单"
        echo "    3. 如需帮助，请查看日志: newapi-tools logs"
    elif [[ $DIAG_ISSUES -gt 0 ]]; then
        echo "    1. 关注黄色标记的警告"
        echo "    2. 定期运行诊断: newapi-tools doctor"
        echo "    3. 保持系统资源充足"
    else
        echo "    1. 系统状态良好，继续保持"
        echo "    2. 定期运行诊断检查: newapi-tools doctor"
        echo "    3. 建议定期备份数据: newapi-tools backup"
    fi

    echo ""
}

# ---------- 主程序 ----------
main() {
    echo ""
    ui_header "系统诊断工具"
    echo ""

    novice_prompt "诊断工具将全面检查系统环境、网络连接、Docker、NewAPI 服务和资源状态。"

    # 初始化并行诊断
    _init_parallel_diag() {
        DIAG_TMP_DIR=$(mktemp -d)
    }
    _init_parallel_diag

    # 并行执行无依赖的诊断
    echo -e "${UI_DIM}正在执行诊断...${UI_PLAIN}"
    local pids=()
    local diag_names=()

    _run_diag_bg "environment" diagnose_environment
    pids+=($!)
    diag_names+=("environment")

    _run_diag_bg "network" diagnose_network
    pids+=($!)
    diag_names+=("network")

    _run_diag_bg "resources" diagnose_resources
    pids+=($!)
    diag_names+=("resources")

    _run_diag_bg "config" diagnose_config
    pids+=($!)
    diag_names+=("config")

    # 等待所有并行诊断完成
    for pid in "${pids[@]}"; do
        wait $pid 2>/dev/null
    done

    # 按顺序显示输出
    for name in "${diag_names[@]}"; do
        local f="$DIAG_TMP_DIR/${name}.out"
        [[ -f "$f" ]] && cat "$f"
    done

    # 聚合评分
    _aggregate_results

    # 串行执行有依赖的诊断
    diagnose_docker
    diagnose_newapi

    # 显示报告
    show_diagnostic_report

    # 更新状态
    if [[ $DIAG_ISSUES -eq 0 ]]; then
        set_state "system.last_diag_result" "healthy"
    else
        set_state "system.last_diag_result" "issues_found"
    fi
    set_state "system.last_diag_time" "$(date '+%Y-%m-%d %H:%M:%S')"

    # 专家模式只返回退出码
    if_expert && {
        if [[ $DIAG_ISSUES -gt 0 ]]; then
            exit 1
        fi
        exit 0
    }

    # 新手模式询问
    if_novice && {
        if ask_yn "是否查看最近日志？" "n"; then
            bash "${TOOLKIT_ROOT}/modules/monitor/logs.sh"
        fi
    }

    exit $((DIAG_ISSUES > 0 ? 1 : 0))
}

main "$@"
