#!/bin/bash
set -eo pipefail
# 状态管理模块 v2.0 —— 支持断点续装、文件锁、状态历史
# 使用 JSON 文件跟踪系统状态，避免因中断导致重复操作

# common.sh 和 security.sh 已通过 lib/_init.sh 按依赖顺序加载

STATE_FILE="${STATE_FILE:-${TOOLKIT_ROOT}/state.json}"
STATE_LOCK_FILE="${STATE_LOCK_FILE:-${TOOLKIT_ROOT}/state.lock}"
STATE_HISTORY_FILE="${STATE_HISTORY_FILE:-${TOOLKIT_ROOT}/state-history.jsonl}"

# ---------- 文件锁函数（跨平台兼容）----------
# 锁持有计数器（支持重入）
_STATE_LOCK_COUNT=0

acquire_lock() {
    local lock_timeout="${1:-10}"

    # 优先使用 flock（Linux）
    if command -v flock &>/dev/null; then
        # 重入：如果已经持有锁，增加计数并直接返回
        if [[ $_STATE_LOCK_COUNT -gt 0 ]]; then
            _STATE_LOCK_COUNT=$((_STATE_LOCK_COUNT + 1))
            return 0
        fi

        # 关闭可能从父进程继承的 FD 200（防止子进程持有锁导致死锁）
        # 子进程继承 FD 200 后，即使父进程 release_lock，子进程仍持有文件描述符，
        # 导致 flock 锁不释放，后续 acquire_lock 会永久阻塞
        exec 200>&- 2>/dev/null || true

        # 重新打开锁文件
        exec 200>"$STATE_LOCK_FILE"
        if flock -w "$lock_timeout" 200; then
            _STATE_LOCK_COUNT=1
            return 0
        fi
        # 超时：显式释放锁并关闭 FD 200（避免锁泄漏）
        flock -u 200 2>/dev/null || true
        exec 200>&- 2>/dev/null || true
        return 1
    fi

    # 降级方案：使用 mkdir 作为锁（POSIX 兼容）
    local lock_dir="${STATE_LOCK_FILE}.dir"
    local waited=0
    while [[ $waited -lt "$lock_timeout" ]]; do
        if mkdir "$lock_dir" 2>/dev/null; then
            _STATE_LOCK_COUNT=1
            return 0
        fi
        # 检查锁是否过期（超过 30 秒强制清除）
        if [[ -d "$lock_dir" ]]; then
            local lock_age
            lock_age=$(( $(date +%s) - $(stat -c %Y "$lock_dir" 2>/dev/null || echo 0) ))
            if [[ $lock_age -gt 30 ]]; then
                rmdir "$lock_dir" 2>/dev/null || true
            fi
        fi
        sleep 0.5
        waited=$((waited + 1))
    done
    return 1
}

release_lock() {
    # 重入：减少计数，只有计数为 0 时才真正释放
    if [[ $_STATE_LOCK_COUNT -gt 0 ]]; then
        _STATE_LOCK_COUNT=$((_STATE_LOCK_COUNT - 1))
        if [[ $_STATE_LOCK_COUNT -gt 0 ]]; then
            return 0
        fi
    fi

    # 释放 flock 文件描述符
    if command -v flock &>/dev/null; then
        # 先显式释放锁，再关闭 FD（确保锁在 FD 关闭前被释放）
        flock -u 200 2>/dev/null || true
        exec 200>&- 2>/dev/null || true
    fi

    # 清理 mkdir 锁
    local lock_dir="${STATE_LOCK_FILE}.dir"
    rmdir "$lock_dir" 2>/dev/null || true
}

# ---------- 状态历史记录 ----------
record_history() {
    local action="$1"
    local key="${2:-}"
    local value="${3:-}"

    # 确保历史文件存在
    touch "$STATE_HISTORY_FILE" 2>/dev/null || return 0

    # 写入历史记录（JSONL 格式）
    local timestamp
    timestamp=$(date '+%Y-%m-%d %H:%M:%S')

    {
        echo "{\"timestamp\":\"$timestamp\",\"action\":\"$action\",\"key\":\"$key\",\"value\":\"$value\",\"host\":\"$(hostname)\"}"
    } >> "$STATE_HISTORY_FILE"

    # 保留最近 1000 条记录
    local line_count
    line_count=$(wc -l < "$STATE_HISTORY_FILE" 2>/dev/null || echo "0")
    if [[ $line_count -gt 1000 ]]; then
        tail -n 500 "$STATE_HISTORY_FILE" > "${STATE_HISTORY_FILE}.tmp" 2>/dev/null
        mv "${STATE_HISTORY_FILE}.tmp" "$STATE_HISTORY_FILE" 2>/dev/null || true
    fi
}

# ---------- 读取状态历史 ----------
get_state_history() {
    local limit="${1:-10}"

    if [[ ! -f "$STATE_HISTORY_FILE" ]]; then
        echo "暂无历史记录"
        return
    fi

    echo "最近 $limit 条状态变更记录："
    echo ""
    tail -n "$limit" "$STATE_HISTORY_FILE" | while IFS= read -r line; do
        local timestamp action key value
        timestamp=$(echo "$line" | sed 's/.*"timestamp":"\([^"]*\)".*/\1/')
        action=$(echo "$line" | sed 's/.*"action":"\([^"]*\)".*/\1/')
        key=$(echo "$line" | sed 's/.*"key":"\([^"]*\)".*/\1/')
        value=$(echo "$line" | sed 's/.*"value":"\([^"]*\)".*/\1/')
        echo "  [$timestamp] $action: $key = $value"
    done
}

# ---------- 初始化状态文件 ----------
init_state() {
    if [[ ! -f "$STATE_FILE" ]]; then
        cat > "$STATE_FILE" << 'EOF'
{
  "version": "2.0",
  "last_update": "",
  "installation": {
    "status": "not_started",
    "step": "",
    "completed_steps": [],
    "config": {}
  },
  "system": {
    "docker_installed": false,
    "dns_configured": false,
    "apt_sourced": false,
    "health_status": "unknown",
    "last_diag_result": "unknown",
    "last_diag_time": ""
  },
  "newapi": {
    "installed": false,
    "version": "",
    "compose_file": "",
    "env_file": ""
  }
}
EOF
        # 直接更新时间戳，避免递归调用
        if command -v jq &>/dev/null; then
            local tmp_file
            tmp_file=$(secure_temp_file "init")
            jq ".last_update = \"$(date '+%Y-%m-%d %H:%M:%S')\"" "$STATE_FILE" > "$tmp_file" 2>/dev/null && mv "$tmp_file" "$STATE_FILE"
        fi
        log_info "状态文件已初始化: $STATE_FILE"
    fi
}

# ---------- 读取状态值 ----------
get_state() {
    local key="$1"
    local default="${2:-}"
    
    if [[ ! -f "$STATE_FILE" ]]; then
        echo "$default"
        return 1
    fi
    
    local value=""
    
    # 尝试使用 jq 解析 JSON（支持嵌套键）
    if command -v jq &>/dev/null; then
        value=$(jq -r ".$key // empty" "$STATE_FILE" 2>/dev/null || echo "")
    fi
    
    # 降级方案：使用 grep 简单解析（只支持一级键）
    if [[ -z "$value" ]] && command -v grep &>/dev/null; then
        value=$(grep "\"$key\"" "$STATE_FILE" | sed 's/.*: *"*\([^,"]*\)"*.*/\1/' | head -1)
    fi
    
    # 如果没找到，返回默认值
    if [[ -z "$value" ]]; then
        echo "$default"
        return 1
    fi
    
    echo "$value"
    return 0
}

# ---------- 更新状态值（带文件锁）----------
set_state() {
    local key="$1"
    local value="$2"

    # 确保状态文件存在
    init_state >/dev/null

    # 获取锁（10秒超时）
    if ! acquire_lock 10; then
        log_warn "无法获取状态文件锁，使用无锁模式"
    fi

    # 设置退出时自动释放锁的 trap（防止异常退出导致锁泄漏）
    local _prev_trap
    _prev_trap=$(trap -p EXIT 2>/dev/null | sed "s/trap -- '//;s/' EXIT$//")
    trap '_set_state_cleanup' EXIT

    local tmp_file
    local result=1

    # 尝试使用 jq 安全更新（支持创建新键）
    if command -v jq &>/dev/null; then
        tmp_file=$(secure_temp_file "set")
        # 使用 --arg 避免命令注入
        if jq --arg key "$key" --arg val "$value" '.[$key] = $val' "$STATE_FILE" > "$tmp_file" 2>/dev/null; then
            mv "$tmp_file" "$STATE_FILE"
            result=0
            log_debug "状态已更新: $key = $value"
        else
            rm -f "$tmp_file"
            log_warn "jq 更新失败，尝试其他方法"
        fi
    fi

    # 降级方案：使用 sed 简单替换或追加
    if [[ $result -ne 0 ]] && grep -q "\"$key\"" "$STATE_FILE" 2>/dev/null; then
        # 键存在，替换值
        # 转义特殊字符，防止命令注入
        local key_escaped value_escaped
        key_escaped=$(escape_sed_pattern "$key")
        value_escaped=$(escape_sed_pattern "$value")
        # 对于替换部分，还需要转义 & 为 \&
        value_escaped=$(echo "$value_escaped" | sed 's/&/\\&/g')
        sed -i "s|\"$key_escaped\": *\"[^\"]*\"|\"$key_escaped\": \"$value_escaped\"|" "$STATE_FILE"
        result=0
        log_debug "状态已更新（sed）: $key = $value"
    elif [[ $result -ne 0 ]]; then
        # 键不存在，追加到 JSON 对象中
        # 简单处理：在最后一个 } 前插入新键值对
        # 转义特殊字符，防止命令注入
        key_escaped=$(escape_sed_pattern "$key")
        value_escaped=$(escape_sed_pattern "$value")
        value_escaped=$(echo "$value_escaped" | sed 's/&/\\&/g')
        sed -i "s/}$/,\n  \"$key_escaped\": \"$value_escaped\"\n}/" "$STATE_FILE" 2>/dev/null
        result=0
        log_debug "状态已创建（sed）: $key = $value"
    fi

    # 记录历史
    if [[ $result -eq 0 ]]; then
        record_history "set" "$key" "$value"
    fi

    # 释放锁
    release_lock

    # 恢复之前的 trap
    if [[ -n "$_prev_trap" ]]; then
        trap "$_prev_trap" EXIT
    else
        trap - EXIT
    fi

    return $result
}

# ---------- set_state 退出清理函数 ----------
_set_state_cleanup() {
    # 确保 flock 在异常退出时被释放
    if [[ $_STATE_LOCK_COUNT -gt 0 ]]; then
        _STATE_LOCK_COUNT=0
        if command -v flock &>/dev/null; then
            flock -u 200 2>/dev/null || true
            exec 200>&- 2>/dev/null || true
        fi
        local lock_dir="${STATE_LOCK_FILE}.dir"
        rmdir "$lock_dir" 2>/dev/null || true
    fi
}

# ---------- 标记步骤完成 ----------
mark_step_completed() {
    local step="$1"
    
    # 确保状态文件存在
    init_state >/dev/null
    
    local completed_steps
    completed_steps=$(get_state "installation.completed_steps" "[]")
    
    if [[ ! "$completed_steps" =~ "\"$step\"" ]]; then
        if command -v jq &>/dev/null; then
            local tmp_file
            tmp_file=$(secure_temp_file "step")
            jq ".installation.completed_steps += [\"$step\"]" "$STATE_FILE" > "$tmp_file" 2>/dev/null && mv "$tmp_file" "$STATE_FILE"
        else
            # 降级方案：读取后重写（简单但可靠）
            local steps_json
            steps_json=$(grep "\"completed_steps\"" "$STATE_FILE" | sed 's/.*: *\(\[.*\]\).*/\1/')
            if [[ -z "$steps_json" || "$steps_json" == "[]" ]]; then
                # 转义特殊字符，防止命令注入
                local step_escaped
                step_escaped=$(escape_sed_pattern "$step")
                sed -i "s/\"completed_steps\": \[\]/\"completed_steps\": [\"$step_escaped\"]/" "$STATE_FILE" 2>/dev/null || true
            else
                # 数组非空，追加（简单处理，可能不完美）
                # 转义特殊字符，防止命令注入
                step_escaped=$(escape_sed_pattern "$step")
                sed -i "s/\"completed_steps\": \[/\"completed_steps\": [\"$step_escaped\",/" "$STATE_FILE" 2>/dev/null || true
            fi
        fi
        log_info "步骤已完成: $step"
    fi
}

# ---------- 检查步骤是否完成 ----------
is_step_completed() {
    local step="$1"
    
    if [[ ! -f "$STATE_FILE" ]]; then
        return 1
    fi
    
    # 尝试使用 jq
    if command -v jq &>/dev/null; then
        local completed_steps
        completed_steps=$(jq -r ".installation.completed_steps[]?" "$STATE_FILE" 2>/dev/null | grep -Fx "$step")
        if [[ -n "$completed_steps" ]]; then
            return 0
        fi
    # 降级方案：直接 grep 状态文件
    elif command -v grep &>/dev/null; then
        if grep -q "\"$step\"" "$STATE_FILE" 2>/dev/null; then
            return 0
        fi
    fi
    
    return 1
}

# ---------- 更新安装状态 ----------
update_install_status() {
    local status="$1"
    local current_step="${2:-}"
    
    set_state "installation.status" "$status"
    
    if [[ -n "$current_step" ]]; then
        set_state "installation.step" "$current_step"
    fi
    
    log_info "安装状态: $status${current_step:+ ($current_step)}"
}

# ---------- 清除状态 ----------
clear_state() {
    rm -f "$STATE_FILE"
    log_info "状态文件已清除"
}

# ---------- 显示状态摘要 ----------
show_state_summary() {
    if [[ ! -f "$STATE_FILE" ]]; then
        echo "状态文件不存在"
        return 1
    fi
    
    echo "========== 系统状态摘要 =========="
    echo "最后更新: $(get_state 'last_update')"
    echo "安装状态: $(get_state 'installation.status')"
    echo "当前步骤: $(get_state 'installation.step')"
    echo "Docker:   $(get_state 'system.docker_installed')"
    echo "NewAPI:   $(get_state 'newapi.installed')"
    echo "=================================="
}

# ---------- 导出函数 ----------
export -f init_state get_state set_state
export -f mark_step_completed is_step_completed
export -f update_install_status clear_state show_state_summary
export -f acquire_lock release_lock record_history get_state_history
export -f _set_state_cleanup
