#!/bin/bash
# 系统健康检查 v2.0
# 新增：UI 增强、配置统一管理、新手/专家模式、详细诊断
set -eo pipefail

# shellcheck source=lib/_init.sh
source "${TOOLKIT_ROOT}/lib/_init.sh"

# 仅在直接执行时进行权限检查（source 时跳过，便于单元测试）
if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    check_root
fi

# ---------- 以下为主执行逻辑，仅在直接运行时执行 ----------
if [[ "${BASH_SOURCE[0]}" != "$0" ]]; then
    return 0 2>/dev/null || true
fi

# ---------- 读取配置阈值 ----------
CPU_THRESHOLD=$(get_config "monitor.alert_cpu_threshold" "80")
MEM_THRESHOLD=$(get_config "monitor.alert_mem_threshold" "80")
DISK_THRESHOLD=85

OVERALL_HEALTH=0

# ---------- 显示新手提示 ----------
novice_prompt "健康检查将扫描系统资源、Docker 容器状态、NewAPI 服务状态。如有异常会给出修复建议。"

# ---------- 系统资源状态 ----------
ui_info "扫描系统资源状态..."
echo ""
echo -e "${UI_BOLD}${UI_CYAN}╔════════════════════════════════════════════════════════════╗${UI_PLAIN}"
echo -e "${UI_BOLD}${UI_CYAN}║                     系统资源状态                           ║${UI_PLAIN}"
echo -e "${UI_BOLD}${UI_CYAN}╚════════════════════════════════════════════════════════════╝${UI_PLAIN}"
echo ""

# 主机名
echo -e "${UI_BOLD}主机名:${UI_PLAIN}   $(hostname)"

# 运行时间
echo -e "${UI_BOLD}运行时间:${UI_PLAIN} $(uptime -p 2>/dev/null || uptime)"

# 内存使用
MEM_INFO=$(free -h | awk '/Mem/{print $3 " / " $2 " (" int($3/$2*100) "%)"}')
MEM_PERCENT=$(free | awk '/Mem/{print int($3/$2*100)}')
if [[ $MEM_PERCENT -ge $MEM_THRESHOLD ]]; then
    echo -e "${UI_BOLD}内存使用:${UI_RED} $MEM_INFO (警告：超过 ${MEM_THRESHOLD}%)${UI_PLAIN}"
    OVERALL_HEALTH=1
else
    echo -e "${UI_BOLD}内存使用:${UI_PLAIN} $MEM_INFO"
fi

# 磁盘使用
DISK_INFO=$(df -h / | awk 'NR==2{print $3 " / " $2 " (" $5 ")"}')
DISK_PERCENT=$(df / | awk 'NR==2{print $5}' | tr -d '%')
if [[ $DISK_PERCENT -ge $DISK_THRESHOLD ]]; then
    echo -e "${UI_BOLD}磁盘使用:${UI_RED} $DISK_INFO (警告：超过 ${DISK_THRESHOLD}%)${UI_PLAIN}"
    OVERALL_HEALTH=1
else
    echo -e "${UI_BOLD}磁盘使用:${UI_PLAIN} $DISK_INFO"
fi

# CPU 负载
CPU_LOAD=$(cat /proc/loadavg 2>/dev/null | awk '{print $1", "$2", "$3}' || echo "未知")
CPU_CORES=$(nproc 2>/dev/null || echo "1")
LOAD_PERCENT=$(echo "$CPU_LOAD" | awk -v cores="$CPU_CORES" '{print int($1/cores*100)}')
if [[ $LOAD_PERCENT -ge $CPU_THRESHOLD ]]; then
    echo -e "${UI_BOLD}CPU 负载:${UI_RED} $CPU_LOAD (警告：超过 ${CPU_THRESHOLD}%)${UI_PLAIN}"
    OVERALL_HEALTH=1
else
    echo -e "${UI_BOLD}CPU 负载:${UI_PLAIN} $CPU_LOAD (${CPU_CORES} 核心)"
fi

echo ""

# ---------- Docker 容器状态 ----------
ui_info "检查 Docker 容器状态..."
echo ""
echo -e "${UI_BOLD}${UI_CYAN}╔════════════════════════════════════════════════════════════╗${UI_PLAIN}"
echo -e "${UI_BOLD}${UI_CYAN}║                     Docker 容器状态                         ║${UI_PLAIN}"
echo -e "${UI_BOLD}${UI_CYAN}╚════════════════════════════════════════════════════════════╝${UI_PLAIN}"
echo ""

if ! command -v docker &>/dev/null; then
    ui_error "Docker 未安装"
    OVERALL_HEALTH=1
    exit $OVERALL_HEALTH
fi

if ! docker ps &>/dev/null; then
    ui_error "Docker 服务未运行或无权访问"
    ui_info "修复建议：systemctl start docker"
    OVERALL_HEALTH=1
    exit $OVERALL_HEALTH
fi

# 显示所有容器状态
echo -e "${UI_BOLD}容器列表：${UI_PLAIN}"
printf "  %-20s %-10s %-20s\n" "容器名" "状态" "端口"
echo "  ------------------------------------------------------------"

while IFS= read -r line; do
    CONTAINER=$(echo "$line" | awk '{print $1}')
    STATUS=$(echo "$line" | awk '{print $2}')
    PORTS=$(echo "$line" | awk '{print $3}')
    
    # 判断状态并显示对应颜色
    if [[ "$STATUS" == "healthy" || "$STATUS" == "Up" ]]; then
        STATUS_ICON="${UI_GREEN}✓${UI_PLAIN}"
    else
        STATUS_ICON="${UI_RED}✗${UI_PLAIN}"
        OVERALL_HEALTH=1
    fi
    
    printf "  %-20s %s %-10s %-20s\n" "$CONTAINER" "$STATUS_ICON" "$STATUS" "$PORTS"
done < <(docker ps --format '{{.Names}}\t{{.Status}}\t{{.Ports}}' 2>/dev/null || echo "")

echo ""

# ---------- NewAPI 容器健康检查 ----------
echo -e "${UI_BOLD}${UI_CYAN}╔════════════════════════════════════════════════════════════╗${UI_PLAIN}"
echo -e "${UI_BOLD}${UI_CYAN}║                    NewAPI 健康状态                          ║${UI_PLAIN}"
echo -e "${UI_BOLD}${UI_CYAN}╚════════════════════════════════════════════════════════════╝${UI_PLAIN}"
echo ""

if docker inspect --format='{{.State.Health.Status}}' new-api 2>/dev/null | grep -q "healthy"; then
    ui_success "NewAPI 容器健康状态：正常"
    
    # 显示额外信息（新手模式）
    if_novice && {
        echo ""
        echo -e "${UI_DIM}详细信息：${UI_PLAIN}"
        echo "  - 容器名: new-api"
        echo "  - 镜像: $(docker inspect --format='{{.Config.Image}}' new-api 2>/dev/null || echo '未知')"
        echo "  - 启动时间: $(docker inspect --format='{{.State.StartedAt}}' new-api 2>/dev/null | cut -d'.' -f1 || echo '未知')"
        echo "  - 重启次数: $(docker inspect --format='{{.RestartCount}}' new-api 2>/dev/null || echo '未知')"
        echo ""
    }
else
    ui_error "NewAPI 容器健康检查未通过！"
    OVERALL_HEALTH=1
    
    # 显示最近日志帮助诊断
    echo ""
    echo -e "${UI_YELLOW}--- 最近日志（最后 20 行）---${UI_PLAIN}"
    docker logs --tail 20 new-api 2>&1 | tail -20
    echo -e "${UI_YELLOW}--- 日志结束 ---${UI_PLAIN}"
    echo ""
    
    # 给出修复建议
    ui_info "修复建议："
    echo "  1. 查看完整日志: docker logs new-api"
    echo "  2. 重启容器: docker restart new-api"
    echo "  3. 重建容器: cd ${NEWAPI_HOME} && docker compose up -d"
    echo "  4. 检查配置: cat ${NEWAPI_HOME}/.env"
    echo ""
    
    # 发送 Webhook 告警
    send_webhook "NewAPI 健康检查失败" "服务器 $(hostname) 的 NewAPI 容器状态异常，请及时处理。查看日志: docker logs new-api"
fi

# ---------- 其他容器状态 ----------
echo -e "${UI_BOLD}${UI_CYAN}╔════════════════════════════════════════════════════════════╗${UI_PLAIN}"
echo -e "${UI_BOLD}${UI_CYAN}║                    依赖服务状态                             ║${UI_PLAIN}"
echo -e "${UI_BOLD}${UI_CYAN}╚════════════════════════════════════════════════════════════╝${UI_PLAIN}"
echo ""

for container in mysql redis npm; do
    if docker inspect --format='{{.State.Status}}' "$container" 2>/dev/null | grep -qv "running"; then
        ui_warn "容器 $container 未运行"
        OVERALL_HEALTH=1
        
        # 给出修复建议
        if_novice && {
            echo "  修复建议: docker start $container"
            echo ""
        }
    else
        ui_success "容器 $container 运行正常"
    fi
done

echo ""

# ---------- 总体健康状态 ----------
echo -e "${UI_BOLD}${UI_CYAN}╔════════════════════════════════════════════════════════════╗${UI_PLAIN}"
echo -e "${UI_BOLD}${UI_CYAN}║                     总体健康状态                             ║${UI_PLAIN}"
echo -e "${UI_BOLD}${UI_CYAN}╚════════════════════════════════════════════════════════════╝${UI_PLAIN}"
echo ""

if [[ $OVERALL_HEALTH -eq 0 ]]; then
    ui_success "✓ 所有检查通过，系统健康状态正常"
    
    # 更新状态
    set_state "system.health_status" "healthy"
    
    # 显示成功摘要（新手模式）
    if_novice && {
        show_summary "健康状态摘要" \
            "✓ 系统资源正常（内存、磁盘、CPU）" \
            "✓ Docker 容器全部运行" \
            "✓ NewAPI 健康检查通过" \
            "✓ 所有依赖服务正常"
    }
else
    ui_error "✗ 存在异常，请检查上述警告和错误"
    
    # 更新状态
    set_state "system.health_status" "unhealthy"
    
    # 显示失败摘要（新手模式）
    if_novice && {
        show_summary "健康状态摘要" \
            "✗ 存在异常，需要处理的项" \
            "请查看上述具体错误信息" \
            "建议按照修复建议逐项处理"
    }
fi

echo ""

# 专家模式：只返回退出码
if_expert && {
    exit $OVERALL_HEALTH
}

# 新手模式：询问是否查看日志
if_novice && {
    if [[ $OVERALL_HEALTH -ne 0 ]]; then
        if ask_yn "是否查看 NewAPI 完整日志？" "y"; then
            bash "${TOOLKIT_ROOT}/modules/monitor/logs.sh"
        fi
    fi
}

exit $OVERALL_HEALTH
