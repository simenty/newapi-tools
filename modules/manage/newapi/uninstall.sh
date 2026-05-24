#!/bin/bash
# NewAPI 彻底卸载模块 v2.2
# V2.2: 路径更新为 manage/newapi/
set -eo pipefail

# shellcheck source=lib/_init.sh
source "${TOOLKIT_ROOT}/lib/_init.sh"

# ---------- 以下为主执行逻辑，仅在直接运行时执行（source 时跳过，便于单元测试）----------
if [[ "${BASH_SOURCE[0]}" != "$0" ]]; then
    return 0 2>/dev/null || true
fi

log_audit "用户触发卸载流程"

# ---------- 显示新手提示 ----------
novice_prompt "卸载将永久删除所有容器、数据、备份和定时任务。此操作不可逆！"

echo -e "\n${UI_RED}====== 彻底卸载 NewAPI ======${UI_PLAIN}\n"
echo -e "${UI_YELLOW}以下项目将被永久删除：${UI_PLAIN}"
echo "  [容器]  new-api, mysql, redis, npm"
echo "  [网络]  newapi_default (docker network)"
echo "  [数据]  $NEWAPI_HOME (含数据库、上传文件、日志)"
echo "  [备份]  ${NEWAPI_HOME}/backups/*"
echo "  [定时]  相关 crontab 任务"
echo "  [配置]  ${TOOLKIT_ROOT}/config/* (可选)"
echo "  [日志]  ${LOG_DIR}/toolkit.log (可选)"
echo ""

# ---------- 显示将要删除的详细信息 ----------
if_novice && {
    echo -e "${UI_CYAN}详细清单：${UI_PLAIN}"

    # 检查容器
    echo "  📦 容器："
    for container in new-api mysql redis npm; do
        if docker ps -a --format '{{.Names}}' 2>/dev/null | grep -qw "$container"; then
            echo "     - $container (存在)"
        else
            echo "     - $container (不存在，跳过)"
        fi
    done

    # 检查数据目录
    echo "  📁 数据目录："
    if [[ -d "$NEWAPI_HOME" ]]; then
        _size=$(du -sh "$NEWAPI_HOME" 2>/dev/null | awk '{print $1}' || echo "未知")
        echo "     - $NEWAPI_HOME ($_size)"
    else
        echo "     - $NEWAPI_HOME (不存在，跳过)"
    fi

    # 检查备份
    echo "  💾 备份文件："
    _backup_count=$(ls -1 "${NEWAPI_HOME}/backups"/*.sql 2>/dev/null | wc -l)
    if [[ $_backup_count -gt 0 ]]; then
        echo "     - ${NEWAPI_HOME}/backups/* ($_backup_count 个文件)"
    else
        echo "     - 无备份文件"
    fi

    # 检查定时任务
    echo "  ⏰ 定时任务："
    if crontab -l 2>/dev/null | grep -q 'newapi-tools'; then
        echo "     - 存在 newapi-tools 相关任务"
    else
        echo "     - 无相关任务"
    fi

    echo ""
}

# ---------- 二次确认（增强版）----------
if ! ask_confirm "确认要彻底卸载 NewAPI 吗？此操作不可逆！（输入 yes 确认）"; then
    ui_info "已取消卸载"
    exit 0
fi

# ---------- 更新状态 ----------
update_install_status "uninstalling" "start"
mark_step_completed "uninstall_$(date +%Y%m%d_%H%M%S)"

# ---------- 步骤 1：停止并删除容器 ----------
ui_info "步骤 1/5：停止并删除容器..."
show_progress 1 5 "卸载进度"

cd "$NEWAPI_HOME" 2>/dev/null || true

if ! $DOCKER_COMPOSE_CMD down -v --remove-orphans 2>/dev/null; then
    ui_warn "Docker Compose 停止失败（可能已停止）"
fi

# 强制删除容器（如果存在）
for container in new-api mysql redis npm; do
    if docker ps -a --format '{{.Names}}' 2>/dev/null | grep -qw "$container"; then
        ui_info "删除容器: $container"
        docker rm -f "$container" 2>/dev/null || true
    fi
done

# 删除网络
if docker network ls --format '{{.Name}}' 2>/dev/null | grep -qw "newapi-network"; then
    ui_info "删除网络: newapi-network"
    docker network rm newapi-network 2>/dev/null || true
fi

ui_success "容器和网络已清理"
show_progress 2 5 "卸载进度"

# ---------- 步骤 2：删除数据目录 ----------
ui_info "步骤 2/5：删除数据目录..."
show_progress 2 5 "卸载进度"

if [[ -d "$NEWAPI_HOME" ]] && [[ -n "$NEWAPI_HOME" ]]; then
    _data_size=$(du -sh "$NEWAPI_HOME" 2>/dev/null | awk '{print $1}' || echo "未知")
    ui_info "删除数据目录: $NEWAPI_HOME ($_data_size)"
    rm -rf "${NEWAPI_HOME:?}/"
    ui_success "数据目录已删除"
else
    ui_warn "数据目录不存在: $NEWAPI_HOME"
fi

show_progress 3 5 "卸载进度"

# ---------- 步骤 3：清理定时任务 ----------
ui_info "步骤 3/5：清理定时任务..."
show_progress 3 5 "卸载进度"

if crontab -l 2>/dev/null | grep -q 'newapi-tools'; then
    crontab -l 2>/dev/null | grep -v 'newapi-tools' | crontab -
    ui_success "已删除 newapi-tools 相关定时任务"
else
    ui_debug "未发现相关定时任务"
fi

show_progress 4 5 "卸载进度"

# ---------- 步骤 4：删除配置（可选）----------
ui_info "步骤 4/5：处理配置文件..."
show_progress 4 5 "卸载进度"

if ask_yn "是否删除配置文件（config.yaml 等）？" "n"; then
    if [[ -n "${TOOLKIT_ROOT}" ]] && [[ -d "${TOOLKIT_ROOT}/config" ]]; then
        rm -rf "${TOOLKIT_ROOT:?}/config"
        ui_success "配置文件已删除"
    else
        ui_warn "配置文件目录不存在"
    fi
else
    ui_info "保留配置文件"
fi

show_progress 5 5 "卸载进度"

# ---------- 步骤 5：删除日志（可选）----------
ui_info "步骤 5/5：处理日志文件..."
show_progress 5 5 "卸载进度"

read -r -p "是否删除操作日志？[y/N]: " del_log
if [[ "$del_log" =~ ^[Yy]$ ]]; then
    [[ -n "${LOG_DIR}" ]] && [[ -d "${LOG_DIR}" ]] && rm -f "${LOG_DIR}/toolkit.log" "${LOG_DIR}/audit.log" 2>/dev/null || true
    ui_success "操作日志已删除"
else
    ui_info "保留操作日志"
fi

# ---------- 完成 ----------
update_install_status "uninstalled" "all_done"
set_state "newapi.installed" "false"
set_state "newapi.version" ""
set_state "newapi.compose_file" ""
set_state "newapi.env_file" ""

# 清除状态文件
clear_state

# 显示卸载摘要
show_summary "卸载完成" \
    "✓ 所有容器已删除" \
    "✓ 数据目录已清理" \
    "✓ 定时任务已清理" \
    "✓ 状态已重置" \
    "" \
    "提示：newapi-tools 脚本本身未被删除" \
    "位置: $TOOLKIT_ROOT"

# 发送通知
send_webhook "NewAPI 已卸载" \
    "服务器 $(hostname) 的 NewAPI 已被彻底卸载"

echo ""
ui_success "=== NewAPI 已彻底卸载 ==="
ui_info "如需删除工具集，请手动执行："
echo "  rm -rf $TOOLKIT_ROOT"
echo "  rm -f /usr/local/bin/newapi-tools"
echo ""

# 询问是否删除工具集（新手模式）
if_novice && {
    if ask_yn "是否立即删除 newapi-tools 工具集？" "n"; then
        [[ -n "${TOOLKIT_ROOT}" ]] && rm -rf "${TOOLKIT_ROOT:?}/" || true
        [[ -f "/usr/local/bin/newapi-tools" ]] && rm -f /usr/local/bin/newapi-tools || true
        ui_success "工具集已删除"
        echo ""
        ui_info "卸载完成，感谢使用 NewAPI Tools！"
        exit 0
    fi
}

ui_info "提示：工具集仍保留在: $TOOLKIT_ROOT"
ui_info "如需删除，请手动执行: rm -rf $TOOLKIT_ROOT && rm -f /usr/local/bin/newapi-tools"
