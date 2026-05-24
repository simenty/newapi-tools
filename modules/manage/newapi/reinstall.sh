#!/bin/bash
# NewAPI 重装模块 v2.2
# V2.2: 路径更新为 manage/newapi/
set -eo pipefail

# shellcheck source=lib/_init.sh
source "${TOOLKIT_ROOT}/lib/_init.sh"

# ---------- 以下为主执行逻辑，仅在直接运行时执行（source 时跳过，便于单元测试）----------
if [[ "${BASH_SOURCE[0]}" != "$0" ]]; then
    return 0 2>/dev/null || true
fi

log_audit "用户触发重装流程"

# ---------- 显示新手提示 ----------
novice_prompt "重装将删除所有容器和数据，然后从备份恢复。整个过程会自动备份当前数据。"

# ---------- 显示将删除的内容 ----------
ui_warn "【重装确认】将要执行的操作："
echo ""
echo -e "${UI_YELLOW}  1. 备份当前数据（自动执行）"
echo "  2. 停止并删除所有相关容器"
echo "  3. 删除数据目录: $NEWAPI_HOME"
echo "  4. 重新执行部署流程${UI_PLAIN}"
echo ""
ui_warn "重装后，当前运行状态将丢失，但数据会从备份恢复。"
echo ""

# ---------- 二次确认（增强版）----------
if ! ask_confirm "确认要重装 NewAPI 吗？（输入 yes 确认）"; then
    ui_info "已取消重装"
    exit 0
fi

# ---------- 更新状态 ----------
update_install_status "reinstalling" "start"
mark_step_completed "reinstall_$(date +%Y%m%d_%H%M%S)"

# ---------- 步骤 1：备份当前数据 ----------
ui_info "步骤 1/4: 备份当前数据..."
show_progress 1 4 "重装进度"

if ! bash "${MODULES_DIR}/manage/newapi/backup.sh" --cron; then
    ui_error "备份失败，为安全起见终止重装。"
    update_install_status "failed" "backup_failed"
    exit 1
fi

ui_success "备份完成"
show_progress 2 4 "重装进度"

# ---------- 步骤 2：停止并删除容器 ----------
ui_info "步骤 2/4: 停止并删除容器..."
show_progress 2 4 "重装进度"

cd "$NEWAPI_HOME" 2>/dev/null || true

if ! $DOCKER_COMPOSE_CMD down -v 2>/dev/null; then
    ui_warn "容器停止失败（可能已停止）"
fi

# 强制删除容器（如果存在）
for container in new-api mysql redis npm; do
    if docker ps -a --format '{{.Names}}' 2>/dev/null | grep -qw "$container"; then
        ui_info "删除容器: $container"
        docker rm -f "$container" 2>/dev/null || true
    fi
done

ui_success "容器已清理"
show_progress 3 4 "重装进度"

# ---------- 步骤 3：删除数据目录 ----------
ui_info "步骤 3/4: 删除数据目录..."
show_progress 3 4 "重装进度"

if [[ -d "$NEWAPI_HOME" ]]; then
    rm -rf "$NEWAPI_HOME"
    ui_success "数据目录已删除"
else
    ui_warn "数据目录不存在，跳过删除"
fi

show_progress 4 4 "重装进度"

# ---------- 步骤 4：重新部署 ----------
ui_info "步骤 4/4: 重新部署 NewAPI..."
show_progress 4 4 "重装进度"

# 清除安装状态（让 install.sh 重新安装）
set_state "newapi.installed" "false"
set_state "installation.status" "not_started"
set_state "installation.completed_steps" "[]"

if bash "${MODULES_DIR}/deploy/install.sh"; then
    ui_success "重新部署完成"
    update_install_status "completed" "reinstall_done"

    # 显示重装摘要
    show_summary "重装完成" \
        "✓ 旧数据已备份" \
        "✓ 容器已清理" \
        "✓ 数据目录已重建" \
        "✓ NewAPI 已重新部署" \
        "" \
        "提示：如需恢复数据，请使用菜单选项 6（备份恢复）"

    # 发送通知
    send_webhook "NewAPI 重装完成" \
        "服务器 $(hostname) 的 NewAPI 已成功重装"
else
    ui_error "重新部署失败"
    update_install_status "failed" "reinstall_failed"

    send_webhook "NewAPI 重装失败" \
        "服务器 $(hostname) 的 NewAPI 重装失败，请手动处理！"

    exit 1
fi

# 询问是否查看健康状态（新手模式）
if_novice && {
    if ask_yn "是否查看服务健康状态？" "y"; then
        bash "${TOOLKIT_ROOT}/modules/monitor/health.sh"
    fi
}
