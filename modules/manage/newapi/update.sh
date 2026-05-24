#!/bin/bash
# NewAPI 更新模块 v2.2
# V2.2: source base 框架，调用通用函数
set -eo pipefail

# shellcheck source=lib/_init.sh
source "${TOOLKIT_ROOT}/lib/_init.sh"

# shellcheck source=modules/manage/base/update-base.sh
source "${MODULES_DIR}/manage/base/update-base.sh"

# 仅在直接执行时进行权限检查（source 时跳过，便于单元测试）
if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    check_root
    require_docker
fi

# ---------- 以下为主执行逻辑，仅在直接运行时执行 ----------
if [[ "${BASH_SOURCE[0]}" != "$0" ]]; then
    return 0 2>/dev/null || true
fi

# ---------- 设置错误捕获 ----------
trap 'trap_error $LINENO "$BASH_COMMAND"' ERR

cd "$NEWAPI_HOME" || { ui_error "目录不存在: $NEWAPI_HOME"; exit 1; }

# ---------- 显示新手提示 ----------
novice_prompt "更新前会自动备份数据，然后拉取最新 NewAPI 镜像并重启容器。如果健康检查失败，会自动回滚到旧版本。"

# ---------- 更新前状态检查 ----------
if ! is_step_completed "newapi_install"; then
    ui_warn "未检测到 NewAPI 安装，请先执行安装（菜单选项 2）"
    exit 1
fi

# ---------- 步骤 1：更新前自动备份 ----------
ui_info "步骤 1/4：更新前自动备份..."
show_progress 1 4 "更新进度"

ui_info "正在执行自动备份..."
if bash "${MODULES_DIR}/manage/newapi/backup.sh" --cron; then
    ui_success "备份完成"
else
    ui_error "备份失败，更新已取消"
    exit 1
fi

update_install_status "updating" "backup_done"
show_progress 2 4 "更新进度"

# ---------- 步骤 2：记录当前镜像 ID ----------
ui_info "步骤 2/4：记录当前版本信息..."
show_progress 2 4 "更新进度"

record_current_version "new-api"

update_install_status "updating" "version_recorded"
show_progress 3 4 "更新进度"

# ---------- 步骤 3：拉取最新镜像 ----------
ui_info "步骤 3/4：拉取最新 NewAPI 镜像..."
show_progress 3 4 "更新进度"

ui_info "正在拉取最新镜像（可能需要几分钟）..."
if ! $DOCKER_COMPOSE_CMD pull new-api; then
    ui_error "镜像拉取失败，更新中断"
    update_install_status "failed" "pull_failed"
    exit 1
fi

ui_success "镜像拉取完成"
update_install_status "updating" "image_pulled"
show_progress 4 4 "更新进度"

# ---------- 步骤 4：重新创建容器 ----------
ui_info "步骤 4/4：重启 NewAPI 容器（应用新镜像）..."
show_progress 4 4 "更新进度"

if ! $DOCKER_COMPOSE_CMD up -d new-api; then
    ui_error "容器重启失败"
    update_install_status "failed" "container_start_failed"
    exit 1
fi

ui_success "容器已重启，等待健康检查..."
update_install_status "updating" "waiting_health_check"

# ---------- 等待健康检查（最多 90 秒）----------
if ! verify_health_check "new-api" 90; then
    ui_error "新版本容器健康检查未通过，开始自动回滚..."
    update_install_status "rollback" "start"

    if perform_rollback "$OLD_IMAGE_ID" "new-api" "calciumion/new-api:latest"; then
        update_install_status "completed" "rollback_done"
    else
        update_install_status "failed" "rollback_failed"
    fi
    exit 1
fi

# ---------- 清理旧镜像 ----------
ui_info "清理旧镜像..."
$DOCKER_COMPOSE_CMD exec -T new-api true 2>/dev/null  # 确认容器可用
docker image prune -f &>/dev/null || true

# ---------- 更新成功 ----------
update_install_status "completed" "update_done"
mark_step_completed "update_$(date +%Y%m%d_%H%M%S)"

NEW_VERSION=$(docker inspect --format='{{.Config.Image}}' new-api 2>/dev/null || echo "latest")
set_state "newapi.version" "$NEW_VERSION"

# 显示更新摘要
show_summary "更新完成" \
    "✓ 旧版本已备份" \
    "✓ 已更新到最新版本" \
    "✓ 容器健康状态正常" \
    "✓ 旧镜像已清理" \
    "" \
    "更新前版本: $OLD_VERSION" \
    "当前版本:   $NEW_VERSION"

# 发送通知
send_webhook "NewAPI 更新成功" \
    "服务器 $(hostname) 的 NewAPI 已从 ${OLD_VERSION} 更新到 ${NEW_VERSION}"

ui_success "NewAPI 更新成功，容器健康状态正常！"

# 询问是否查看健康状态（新手模式）
if_novice && {
    if ask_yn "是否查看详细健康状态？" "y"; then
        bash "${TOOLKIT_ROOT}/modules/monitor/health.sh"
    fi
}
