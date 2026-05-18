#!/bin/bash
# alert.sh — 告警规则引擎
# 支持 CPU/内存/磁盘/容器状态阈值检查 + Webhook 通知
# 用法: newapi-tools alert [--check] [--test-webhook]
set -eo pipefail

# 自动检测 TOOLKIT_ROOT（兼容直接执行和通过 newapi-tools.sh 调用）
TOOLKIT_ROOT="${TOOLKIT_ROOT:-$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")/../.." && pwd)}"
export TOOLKIT_ROOT

# shellcheck source=lib/_init.sh
source "${TOOLKIT_ROOT}/lib/_init.sh"

# 仅在直接执行时进行权限检查
if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    check_root
fi

# ---------- 以下为主执行逻辑，仅在直接运行时执行 ----------
if [[ "${BASH_SOURCE[0]}" != "$0" ]]; then
    return 0 2>/dev/null || true
fi

# ---------- 常量 ----------
ALERT_STATE_DIR="${STATE_DIR:-${TOOLKIT_ROOT}/state}"
ALERT_HISTORY_FILE="${ALERT_STATE_DIR}/alert-history.log"
ALERT_SILENCE_FILE="${ALERT_STATE_DIR}/alert-silence"
ALERT_COOLDOWN=300  # 默认静默期 5 分钟（秒）

# ---------- 从配置读取阈值 ----------
CPU_THRESHOLD=$(get_config "monitor.alert_cpu_threshold" "80")
MEM_THRESHOLD=$(get_config "monitor.alert_mem_threshold" "80")
DISK_THRESHOLD=$(get_config "monitor.alert_disk_threshold" "85")
CONTAINER_CHECK=$(get_config "monitor.alert_container_check" "true")
WEBHOOK_URL=$(get_config "notification.webhook_url" "")
WEBHOOK_TYPE=$(get_config "notification.webhook_type" "auto")  # auto/feishu/dingtalk/slack/custom

# ---------- 参数解析 ----------
ACTION="check"
while [[ $# -gt 0 ]]; do
    case "$1" in
        --check|-c)
            ACTION="check"
            shift
            ;;
        --test-webhook|-t)
            ACTION="test_webhook"
            shift
            ;;
        --history)
            ACTION="history"
            shift
            ;;
        --silence)
            # 用法: --silence 600  (静默 600 秒)
            ACTION="silence"
            SILENCE_SECONDS="${2:-300}"
            shift 2
            ;;
        --unsilence)
            ACTION="unsilence"
            shift
            ;;
        -h|--help)
            cat << EOF
用法: newapi-tools alert [选项]

选项：
  -c, --check         执行告警检查（默认）
  -t, --test-webhook  测试 Webhook 通知
  --history           查看告警历史
  --silence N         静默 N 秒（期间不发告警）
  --unsilence         取消静默
  -h, --help          显示帮助

配置项（config.yaml）：
  monitor.alert_cpu_threshold      CPU 告警阈值（默认 80%）
  monitor.alert_mem_threshold      内存告警阈值（默认 80%）
  monitor.alert_disk_threshold    磁盘告警阈值（默认 85%）
  monitor.alert_container_check   是否检查容器状态（默认 true）
  notification.webhook_url         Webhook 地址
  notification.webhook_type        Webhook 类型: auto/feishu/dingtalk/slack/custom
EOF
            exit 0
            ;;
        *)
            log_error "未知参数: $1"
            exit 1
            ;;
    esac
done

# ---------- 告警历史记录 ----------
_log_alert() {
    local level="$1"
    local message="$2"
    mkdir -p "$(dirname "$ALERT_HISTORY_FILE")"
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [${level}] ${message}" >> "$ALERT_HISTORY_FILE"
}

# ---------- 静默期检查 ----------
_is_silenced() {
    if [[ ! -f "$ALERT_SILENCE_FILE" ]]; then
        return 1  # 未静默
    fi

    local silence_until
    silence_until=$(cat "$ALERT_SILENCE_FILE" 2>/dev/null || echo "0")
    local now
    now=$(date +%s)

    if [[ $now -ge $silence_until ]]; then
        # 静默期已过，清理
        rm -f "$ALERT_SILENCE_FILE"
        return 1
    fi

    return 0  # 仍在静默期
}

# ---------- 增强版 Webhook 通知 ----------
# 支持: 飞书/钉钉/Slack/自定义
send_alert_webhook() {
    local title="$1"
    local content="$2"
    local alert_level="${3:-warn}"

    # 未配置 Webhook URL 则跳过
    if [[ -z "$WEBHOOK_URL" ]]; then
        log_debug "未配置 Webhook URL，跳过通知"
        return 0
    fi

    # 检查静默期
    if _is_silenced; then
        log_debug "告警静默中，跳过通知"
        return 0
    fi

    # 确定颜色
    local color="#FF0000"  # 默认红色
    case "$alert_level" in
        critical) color="#FF0000" ;;
        warn)     color="#FFA500" ;;
        info)     color="#0000FF" ;;
    esac

    local payload
    local content_type="Content-Type: application/json"

    # 根据 Webhook 类型构建 payload
    case "$WEBHOOK_TYPE" in
        feishu)
            # 飞书 Webhook 格式
            local escaped_title escaped_content
            escaped_title=$(printf '%s' "$title" | sed 's/"/\\"/g; s/\\/\\\\/g')
            escaped_content=$(printf '%s' "$content" | sed 's/"/\\"/g; s/\\/\\\\/g')
            payload=$(cat << JSONEOF
{
  "msg_type": "interactive",
  "card": {
    "header": {
      "title": { "tag": "plain_text", "content": "${escaped_title}" },
      "template": "${alert_level}"
    },
    "elements": [
      { "tag": "markdown", "content": "${escaped_content}" }
    ]
  }
}
JSONEOF
            )
            ;;
        dingtalk)
            # 钉钉 Webhook 格式
            local escaped_title escaped_content
            escaped_title=$(printf '%s' "$title" | sed 's/"/\\"/g; s/\\/\\\\/g')
            escaped_content=$(printf '%s' "$content" | sed 's/"/\\"/g; s/\\/\\\\/g')
            payload=$(cat << JSONEOF
{
  "msgtype": "markdown",
  "markdown": {
    "title": "${escaped_title}",
    "text": "### ${escaped_title}\n\n${escaped_content}"
  }
}
JSONEOF
            )
            ;;
        slack)
            # Slack Webhook 格式
            local escaped_title escaped_content
            escaped_title=$(printf '%s' "$title" | sed 's/"/\\"/g; s/\\/\\\\/g')
            escaped_content=$(printf '%s' "$content" | sed 's/"/\\"/g; s/\\/\\\\/g')
            payload=$(cat << JSONEOF
{
  "attachments": [
    {
      "color": "${color}",
      "title": "${escaped_title}",
      "text": "${escaped_content}",
      "footer": "NewAPI Tools Alert",
      "ts": $(date +%s)
    }
  ]
}
JSONEOF
            )
            ;;
        auto|*)
            # 自动检测或自定义格式：使用通用 text 格式
            local escaped_title escaped_content
            escaped_title=$(printf '%s' "$title" | sed 's/"/\\"/g; s/\\/\\\\/g')
            escaped_content=$(printf '%s' "$content" | sed 's/"/\\"/g; s/\\/\\\\/g')
            payload=$(cat << JSONEOF
{
  "msg_type": "text",
  "content": {
    "text": "${escaped_title}\n${escaped_content}"
  }
}
JSONEOF
            )
            ;;
    esac

    # 发送 Webhook
    local http_code
    http_code=$(curl -s -o /dev/null -w "%{http_code}" \
        -X POST \
        -H "$content_type" \
        -d "$payload" \
        --max-time 10 \
        "$WEBHOOK_URL" 2>/dev/null || echo "000")

    if [[ "$http_code" == "200" || "$http_code" == "204" ]]; then
        log_info "告警通知已发送: $title"
        _log_alert "notify" "已发送: $title (HTTP $http_code)"
    else
        log_warn "告警通知发送失败 (HTTP $http_code): $title"
        _log_alert "notify_fail" "发送失败: $title (HTTP $http_code)"
    fi
}

# ---------- CPU 使用率检查 ----------
check_cpu() {
    local cpu_cores
    cpu_cores=$(nproc 2>/dev/null || echo 1)
    local load_1min
    load_1min=$(cat /proc/loadavg 2>/dev/null | awk '{print $1}' || echo "0")
    # loadavg 是浮点数，用 awk 做浮点比较
    local usage_percent
    usage_percent=$(awk -v load="$load_1min" -v cores="$cpu_cores" 'BEGIN { printf "%d", load/cores*100 }')

    if [[ $usage_percent -ge $CPU_THRESHOLD ]]; then
        echo "CRITICAL"
        send_alert_webhook \
            "[告警] CPU 使用率过高: ${usage_percent}%" \
            "服务器: $(hostname)\nCPU 使用率: ${usage_percent}%（阈值: ${CPU_THRESHOLD}%）\n负载: $(cat /proc/loadavg | awk '{print $1, $2, $3}')\n核心数: ${cpu_cores}\n\n请检查: top 或 htop" \
            "critical"
        _log_alert "alert" "CPU ${usage_percent}% >= ${CPU_THRESHOLD}%"
    else
        echo "OK"
    fi
}

# ---------- 内存使用率检查 ----------
check_memory() {
    local mem_percent
    mem_percent=$(free | awk '/Mem/{printf "%d", $3/$2*100}')

    if [[ $mem_percent -ge $MEM_THRESHOLD ]]; then
        local mem_info
        mem_info=$(free -h | awk '/Mem/{print "已用: "$3" / 总计: "$2}')
        echo "CRITICAL"
        send_alert_webhook \
            "[告警] 内存使用率过高: ${mem_percent}%" \
            "服务器: $(hostname)\n内存使用率: ${mem_percent}%（阈值: ${MEM_THRESHOLD}%）\n${mem_info}\n\n请检查: free -h" \
            "critical"
        _log_alert "alert" "内存 ${mem_percent}% >= ${MEM_THRESHOLD}%"
    else
        echo "OK"
    fi
}

# ---------- 磁盘使用率检查 ----------
check_disk() {
    local disk_percent
    disk_percent=$(df / | awk 'NR==2{print $5}' | tr -d '%')

    if [[ $disk_percent -ge $DISK_THRESHOLD ]]; then
        local disk_info
        disk_info=$(df -h / | awk 'NR==2{print "已用: "$3" / 总计: "$2" ("$5")"}')
        echo "CRITICAL"
        send_alert_webhook \
            "[告警] 磁盘使用率过高: ${disk_percent}%" \
            "服务器: $(hostname)\n磁盘使用率: ${disk_percent}%（阈值: ${DISK_THRESHOLD}%）\n${disk_info}\n\n请检查: df -h 或 du -sh /*" \
            "critical"
        _log_alert "alert" "磁盘 ${disk_percent}% >= ${DISK_THRESHOLD}%"
    else
        echo "OK"
    fi
}

# ---------- 容器状态检查 ----------
check_containers() {
    if [[ "$CONTAINER_CHECK" != "true" ]]; then
        echo "SKIP"
        return 0
    fi

    local alert_count=0
    local alert_details=""

    # 检查 NewAPI 核心容器
    local containers=("new-api" "mysql" "redis" "npm")
    for container in "${containers[@]}"; do
        if ! docker inspect --format='{{.Name}}' "$container" &>/dev/null; then
            # 容器不存在，跳过（可能没有部署该服务）
            continue
        fi

        local status
        status=$(docker inspect --format='{{.State.Status}}' "$container" 2>/dev/null || echo "unknown")

        if [[ "$status" != "running" ]]; then
            alert_count=$((alert_count + 1))
            alert_details+="\n- 容器 $container 状态: $status"

            # 获取最近日志帮助诊断
            local last_logs
            last_logs=$(docker logs --tail 5 "$container" 2>&1 | tr '\n' ' ' | cut -c1-200)
            alert_details+="\n  最近日志: $last_logs"
        fi
    done

    if [[ $alert_count -gt 0 ]]; then
        echo "CRITICAL"
        send_alert_webhook \
            "[告警] ${alert_count} 个容器异常" \
            "服务器: $(hostname)\n异常容器: ${alert_count} 个${alert_details}\n\n请检查: docker ps -a" \
            "critical"
        _log_alert "alert" "${alert_count} 个容器异常"
    else
        echo "OK"
    fi
}

# ---------- 执行告警检查 ----------
do_alert_check() {
    echo -e "${UI_BOLD}${UI_CYAN}╔════════════════════════════════════════════════════════════╗${UI_PLAIN}"
    echo -e "${UI_BOLD}${UI_CYAN}║                    告警规则检查                             ║${UI_PLAIN}"
    echo -e "${UI_BOLD}${UI_CYAN}╚════════════════════════════════════════════════════════════╝${UI_PLAIN}"
    echo ""

    local total_checks=0
    local alert_count=0

    # CPU
    printf "  %-20s " "CPU 使用率"
    local cpu_result
    cpu_result=$(check_cpu)
    if [[ "$cpu_result" == "CRITICAL" ]]; then
        echo -e "${RED}告警${PLAIN}"
        alert_count=$((alert_count + 1))
    else
        local cpu_cores=$(nproc 2>/dev/null || echo 1)
        local load=$(cat /proc/loadavg 2>/dev/null | awk '{print $1}' || echo "0")
        local usage=$(awk -v load="$load" -v cores="$cpu_cores" 'BEGIN { printf "%d", load/cores*100 }')
        echo -e "${GREEN}${usage}% (阈值: ${CPU_THRESHOLD}%)${PLAIN}"
    fi
    total_checks=$((total_checks + 1))

    # 内存
    printf "  %-20s " "内存使用率"
    local mem_result
    mem_result=$(check_memory)
    if [[ "$mem_result" == "CRITICAL" ]]; then
        echo -e "${RED}告警${PLAIN}"
        alert_count=$((alert_count + 1))
    else
        local mem_pct=$(free | awk '/Mem/{printf "%d", $3/$2*100}')
        echo -e "${GREEN}${mem_pct}% (阈值: ${MEM_THRESHOLD}%)${PLAIN}"
    fi
    total_checks=$((total_checks + 1))

    # 磁盘
    printf "  %-20s " "磁盘使用率"
    local disk_result
    disk_result=$(check_disk)
    if [[ "$disk_result" == "CRITICAL" ]]; then
        echo -e "${RED}告警${PLAIN}"
        alert_count=$((alert_count + 1))
    else
        local disk_pct=$(df / | awk 'NR==2{print $5}' | tr -d '%')
        echo -e "${GREEN}${disk_pct}% (阈值: ${DISK_THRESHOLD}%)${PLAIN}"
    fi
    total_checks=$((total_checks + 1))

    # 容器
    printf "  %-20s " "容器状态"
    local container_result
    container_result=$(check_containers)
    if [[ "$container_result" == "CRITICAL" ]]; then
        echo -e "${RED}告警${PLAIN}"
        alert_count=$((alert_count + 1))
    elif [[ "$container_result" == "SKIP" ]]; then
        echo -e "${YELLOW}跳过${PLAIN}"
    else
        echo -e "${GREEN}正常${PLAIN}"
    fi
    total_checks=$((total_checks + 1))

    echo ""
    echo -e "${UI_BOLD}${UI_CYAN}╔════════════════════════════════════════════════════════════╗${UI_PLAIN}"
    echo -e "${UI_BOLD}${UI_CYAN}║                    检查结果                                 ║${UI_PLAIN}"
    echo -e "${UI_BOLD}${UI_CYAN}╚════════════════════════════════════════════════════════════╝${UI_PLAIN}"
    echo ""

    if [[ $alert_count -eq 0 ]]; then
        echo -e "  ${GREEN}所有检查通过，无告警${PLAIN}"
        _log_alert "info" "告警检查完成: ${total_checks} 项全部通过"
    else
        echo -e "  ${RED}${alert_count}/${total_checks} 项触发告警${PLAIN}"
        _log_alert "warn" "告警检查完成: ${alert_count}/${total_checks} 项触发告警"

        # 显示静默状态
        if _is_silenced; then
            local silence_until
            silence_until=$(cat "$ALERT_SILENCE_FILE" 2>/dev/null || echo "0")
            local remaining=$(( silence_until - $(date +%s) ))
            echo -e "  ${YELLOW}告警静默中，剩余 ${remaining} 秒${PLAIN}"
        fi
    fi

    echo ""
    echo -e "  阈值配置:"
    echo -e "    CPU:  ${CPU_THRESHOLD}%    内存: ${MEM_THRESHOLD}%    磁盘: ${DISK_THRESHOLD}%"
    echo -e "    容器检查: ${CONTAINER_CHECK}"
    if [[ -n "$WEBHOOK_URL" ]]; then
        echo -e "    Webhook: ${WEBHOOK_TYPE} (已配置)"
    else
        echo -e "    Webhook: ${YELLOW}未配置${PLAIN}"
    fi
    echo ""

    return $alert_count
}

# ---------- 测试 Webhook ----------
do_test_webhook() {
    echo -e "${UI_BOLD}测试 Webhook 通知...${UI_PLAIN}"
    echo ""

    if [[ -z "$WEBHOOK_URL" ]]; then
        log_error "未配置 Webhook URL"
        log_info "请在 config.yaml 中设置 notification.webhook_url"
        echo ""
        echo "示例配置:"
        echo "  notification:"
        echo "    webhook_url: \"https://your-webhook-url\""
        echo "    webhook_type: \"auto\"  # auto/feishu/dingtalk/slack/custom"
        exit 1
    fi

    log_info "Webhook URL: $WEBHOOK_URL"
    log_info "Webhook 类型: $WEBHOOK_TYPE"
    echo ""

    # 发送测试通知
    send_alert_webhook \
        "[测试] NewAPI Tools 告警测试" \
        "这是一条测试通知。\n\n服务器: $(hostname)\n时间: $(date '+%Y-%m-%d %H:%M:%S')\n\n如收到此消息，说明 Webhook 配置正确。" \
        "info"

    if [[ $? -eq 0 ]]; then
        log_success "测试通知已发送，请检查接收端"
    else
        log_error "测试通知发送失败，请检查 Webhook URL"
    fi
}

# ---------- 显示告警历史 ----------
do_show_history() {
    echo -e "${UI_BOLD}${UI_CYAN}╔════════════════════════════════════════════════════════════╗${UI_PLAIN}"
    echo -e "${UI_BOLD}${UI_CYAN}║                    告警历史                                 ║${UI_PLAIN}"
    echo -e "${UI_BOLD}${UI_CYAN}╚════════════════════════════════════════════════════════════╝${UI_PLAIN}"
    echo ""

    if [[ ! -f "$ALERT_HISTORY_FILE" ]]; then
        echo "  暂无告警记录"
        echo ""
        return 0
    fi

    # 显示最近 20 条记录
    local lines="${1:-20}"
    echo "  最近 ${lines} 条记录:"
    echo ""
    tail -"$lines" "$ALERT_HISTORY_FILE" | while IFS= read -r line; do
        # 高亮不同级别
        if echo "$line" | grep -q "\\[alert\\]"; then
            echo -e "  ${RED}${line}${PLAIN}"
        elif echo "$line" | grep -q "\\[warn\\]"; then
            echo -e "  ${YELLOW}${line}${PLAIN}"
        else
            echo -e "  ${line}"
        fi
    done
    echo ""
}

# ---------- 设置静默 ----------
do_silence() {
    local seconds="${SILENCE_SECONDS:-300}"
    local silence_until=$(( $(date +%s) + seconds ))
    echo "$silence_until" > "$ALERT_SILENCE_FILE"
    log_success "告警已静默 ${seconds} 秒（至 $(date -d "@${silence_until}" '+%H:%M:%S' 2>/dev/null || date '+%H:%M:%S')）"
}

# ---------- 取消静默 ----------
do_unsilence() {
    if [[ -f "$ALERT_SILENCE_FILE" ]]; then
        rm -f "$ALERT_SILENCE_FILE"
        log_success "告警静默已取消"
    else
        log_info "当前未设置静默"
    fi
}

# ---------- 主入口 ----------
mkdir -p "$ALERT_STATE_DIR"

case "$ACTION" in
    check)
        do_alert_check
        ;;
    test_webhook)
        do_test_webhook
        ;;
    history)
        do_show_history
        ;;
    silence)
        do_silence
        ;;
    unsilence)
        do_unsilence
        ;;
    *)
        do_alert_check
        ;;
esac
