#!/bin/bash
# update-base.sh — 通用更新框架
# 只定义函数，不执行任何操作。被 newapi/update.sh source 后调用。
# 用法: source "${MODULES_DIR}/manage/base/update-base.sh"

set -eo pipefail

# shellcheck source=lib/_init.sh
source "${TOOLKIT_ROOT}/lib/_init.sh"

# ---------- 记录当前版本 ----------
# 参数: $1 = 容器名称
# 输出: 通过全局变量 OLD_IMAGE_ID, OLD_VERSION 保存
record_current_version() {
    local container_name="${1:-new-api}"

    OLD_IMAGE_ID=$(docker inspect --format='{{.Image}}' "$container_name" 2>/dev/null || true)
    OLD_VERSION=$(docker inspect --format='{{.Config.Image}}' "$container_name" 2>/dev/null || echo "unknown")

    if [[ -z "$OLD_IMAGE_ID" ]]; then
        log_warn "无法获取当前镜像 ID，回滚功能将不可用"
    fi

    log_info "当前版本: $OLD_VERSION (Image ID: ${OLD_IMAGE_ID:-unknown})"
    export OLD_IMAGE_ID OLD_VERSION
}

# ---------- 健康检查 ----------
# 参数: $1 = 容器名称, $2 = 最大等待秒数（默认 90）
# 返回: 0=健康, 1=不健康
verify_health_check() {
    local container_name="${1:-new-api}"
    local max_wait="${2:-90}"
    local interval=3
    local max_iterations=$(( max_wait / interval ))
    local health_passed=false

    ui_info "等待容器健康检查通过（最多 ${max_wait} 秒）..."

    for i in $(seq 1 "$max_iterations"); do
        sleep "$interval"
        show_progress "$i" "$max_iterations" "健康检查"

        local container_status
        container_status=$(docker inspect --format='{{.State.Health.Status}}' "$container_name" 2>/dev/null || echo "unhealthy")

        if [[ "$container_status" == "healthy" ]]; then
            health_passed=true
            echo ""  # 换行
            break
        fi

        if [[ $i -eq "$max_iterations" ]]; then
            echo ""  # 换行
        fi
    done

    if $health_passed; then
        ui_success "容器健康检查通过"
        return 0
    else
        ui_error "容器健康检查超时（${max_wait}秒）"
        return 1
    fi
}

# ---------- 执行回滚 ----------
# 参数: $1 = 旧镜像 ID, $2 = 容器名称, $3 = 镜像标签
# 返回: 0=回滚成功, 1=回滚失败
perform_rollback() {
    local old_image_id="$1"
    local container_name="${2:-new-api}"
    local image_tag="${3:-calciumion/new-api:latest}"

    log_error "=== 开始自动回滚 ==="
    ui_error "更新失败！正在自动回滚到更新前版本..."

    if [[ -z "$old_image_id" ]]; then
        ui_error "无法回滚：旧镜像 ID 未知，请手动处理！"
        send_webhook "更新失败（无法回滚）" \
            "服务器 $(hostname) 的更新失败且无法回滚，请立即人工介入！"
        return 1
    fi

    # 回滚到旧镜像
    docker tag "$old_image_id" "$image_tag"
    cd "$NEWAPI_HOME" 2>/dev/null && $DOCKER_COMPOSE_CMD up -d "$container_name"

    ui_warn "已回滚至更新前版本"
    ui_info "建议：手动排查新版本问题"

    send_webhook "更新回滚" \
        "服务器 $(hostname) 的自动更新失败，已自动回滚至旧版本。请手动排查新版本问题。"

    log_info "=== 回滚完成 ==="
    return 0
}
