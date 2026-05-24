#!/bin/bash
# NewAPI 备份恢复模块 v2.2
# V2.2: source base 框架，调用通用函数
set -eo pipefail

# shellcheck source=lib/_init.sh
source "${TOOLKIT_ROOT}/lib/_init.sh"

# shellcheck source=modules/manage/base/restore-base.sh
source "${MODULES_DIR}/manage/base/restore-base.sh"

# 仅在直接执行时进行权限检查（source 时跳过，便于单元测试）
if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    check_root
    require_docker
fi

# ---------- 全局变量 ----------
BACKUP_DIR=$(get_config "backup.dir" "${NEWAPI_HOME}/backups")

# ---------- 回滚机制 ----------
# 回滚状态：0=不需要回滚（成功），1=需要回滚（失败）
ROLLBACK_NEEDED=1
ROLLBACK_DIR=""
ROLLBACK_DATA_BAK=""
ROLLBACK_NPM_BAK=""

# 创建回滚点：将当前数据目录 mv 到安全位置
_create_rollback_point() {
    local ts="$1"
    ROLLBACK_DIR="${BACKUP_DIR}/.rollback_${ts}"
    ROLLBACK_DATA_BAK="${NEWAPI_HOME}/data.bak.${ts}"
    ROLLBACK_NPM_BAK="${NEWAPI_HOME}/npm.bak.${ts}"

    log_info "创建回滚点..."
    mkdir -p "$ROLLBACK_DIR" 2>/dev/null || true

    # 保存当前数据库状态（用于回滚）
    ui_info "备份当前数据库状态（用于回滚）..."
    # 使用环境变量传递密码（避免命令行暴露）
    if docker exec -i mysql bash -c 'MYSQL_PWD="$1" mysqldump -uroot \
        --single-transaction --databases newapi' _ "$DB_ROOT_PASSWORD" > "$ROLLBACK_DIR/pre_restore_db.sql" 2>>"$LOG_FILE"; then
        log_info "数据库回滚点已创建"
    else
        log_warn "无法创建数据库回滚点，将继续（回滚能力受限）"
    fi

    # 用 mv 将当前数据目录移到安全位置（高效，不复制数据）
    ui_info "移动当前数据目录到回滚位置..."
    if [[ -d "$NEWAPI_HOME/data" ]]; then
        mv "$NEWAPI_HOME/data" "$ROLLBACK_DATA_BAK" 2>/dev/null && \
            log_info "data/ 已移至: $ROLLBACK_DATA_BAK" || \
            log_warn "移动 data/ 失败，跳过数据回滚"
    fi
    if [[ -d "$NEWAPI_HOME/npm" ]]; then
        mv "$NEWAPI_HOME/npm" "$ROLLBACK_NPM_BAK" 2>/dev/null && \
            log_info "npm/ 已移至: $ROLLBACK_NPM_BAK" || \
            log_warn "移动 npm/ 失败，跳过 npm 回滚"
    fi

    # 写入回滚标记
    touch "$ROLLBACK_DIR/.rollback_active"
    log_info "回滚点创建完成: $ROLLBACK_DIR"
}

# 执行回滚：恢复原始数据
_do_rollback() {
    log_error "=== 开始自动回滚 ==="
    ui_error "恢复失败！正在自动回滚到恢复前状态..."

    # 1. 恢复数据目录（用 mv 移回）
    if [[ -d "$ROLLBACK_DATA_BAK" ]]; then
        rm -rf "$NEWAPI_HOME/data" 2>/dev/null || true
        mv "$ROLLBACK_DATA_BAK" "$NEWAPI_HOME/data" 2>/dev/null && \
            log_info "data/ 已回滚" || \
            log_error "data/ 回滚失败！"
    fi
    if [[ -d "$ROLLBACK_NPM_BAK" ]]; then
        rm -rf "$NEWAPI_HOME/npm" 2>/dev/null || true
        mv "$ROLLBACK_NPM_BAK" "$NEWAPI_HOME/npm" 2>/dev/null && \
            log_info "npm/ 已回滚" || \
            log_error "npm/ 回滚失败！"
    fi

    # 2. 恢复数据库（不在日志中记录密码）
    if [[ -f "$ROLLBACK_DIR/pre_restore_db.sql" ]]; then
        ui_info "回滚数据库..."
        # 使用环境变量传递密码（避免命令行暴露）
        docker exec -i mysql bash -c 'MYSQL_PWD="$1" mysql -uroot newapi' _ "$DB_ROOT_PASSWORD" \
            < "$ROLLBACK_DIR/pre_restore_db.sql" 2>>"$LOG_FILE" && \
            log_info "数据库已回滚" || \
            log_error "数据库回滚失败！"
    fi

    # 3. 重启容器
    ui_info "重启容器..."
    cd "$NEWAPI_HOME" 2>/dev/null && $DOCKER_COMPOSE_CMD start 2>/dev/null || true

    # 4. 清理回滚文件
    rm -rf "$ROLLBACK_DIR" 2>/dev/null || true

    ui_success "回滚完成！系统已恢复到恢复前的状态。"
    send_webhook "NewAPI 恢复失败-已自动回滚" "服务器 $(hostname) 的恢复操作失败，已自动回滚到原状态"
    log_info "=== 回滚完成 ==="
}

# 清除回滚（恢复成功时调用）
_clear_rollback() {
    ROLLBACK_NEEDED=0
    rm -rf "$ROLLBACK_DIR" "$ROLLBACK_DATA_BAK" "$ROLLBACK_NPM_BAK" 2>/dev/null || true
    log_info "回滚点已清除（恢复成功，无需回滚）"
}

# EXIT trap：若 ROLLBACK_NEEDED=1 则执行回滚
_on_exit() {
    local exit_code=$?
    if [[ "$ROLLBACK_NEEDED" -eq 1 ]]; then
        _do_rollback
    fi
    # 确保容器最终是运行的
    if [[ "$exit_code" -ne 0 ]]; then
        cd "$NEWAPI_HOME" 2>/dev/null && $DOCKER_COMPOSE_CMD start 2>/dev/null || true
    fi
}
trap _on_exit EXIT

# ---------- 以下为主执行逻辑，仅在直接运行时执行（source 时跳过）----------
if [[ "${BASH_SOURCE[0]}" != "$0" ]]; then
    trap - EXIT
    return 0 2>/dev/null || true
fi

# ---------- 显示新手提示 ----------
novice_prompt "恢复操作将覆盖当前数据。请确保已备份当前数据，或确认要恢复到指定备份。"

# ---------- 检查备份是否存在 ----------
if [[ ! -d "$BACKUP_DIR" ]] || [[ -z "$(ls -A "$BACKUP_DIR"/newapi_data_*.tar.gz 2>/dev/null)" ]]; then
    ui_error "备份目录为空或不存在: $BACKUP_DIR"
    ui_info "提示：请先执行备份（菜单选项 5）"
    ROLLBACK_NEEDED=0
    exit 1
fi

# ---------- 显示备份列表 ----------
ui_info "正在扫描备份文件..."
echo ""
echo -e "${UI_BOLD}╔══════╦══════════════════════════════════════╦════════╦══════════════════╗${UI_PLAIN}"
echo -e "${UI_BOLD}║ 序号 ║ 文件名                               ║  大小  ║ 修改时间           ║${UI_PLAIN}"
echo -e "${UI_BOLD}╠══════╬══════════════════════════════════════╬════════╬══════════════════╣${UI_PLAIN}"

ls -1ht "$BACKUP_DIR"/newapi_data_*.tar.gz 2>/dev/null | nl -w 4 -s ' ' | while read -r line; do
    num=$(echo "$line" | awk '{print $1}')
    file=$(echo "$line" | cut -d' ' -f2-)
    size=$(du -h "$file" 2>/dev/null | awk '{print $1}')
    mtime=$(stat -c '%y' "$file" 2>/dev/null | cut -d'.' -f1 || stat -f '%Sm' "$file" 2>/dev/null | head -1)
    printf "${UI_CYAN}║ %-4s ║ ${UI_PLAIN}%-36s${UI_CYAN} ║ %6s ║ %-18s ║${UI_PLAIN}\n" \
        "$num" "$(basename "$file")" "$size" "$(echo "$mtime" | cut -d' ' -f1,2)"
done

echo -e "${UI_BOLD}╚══════╩══════════════════════════════════════╩════════╩══════════════════╝${UI_PLAIN}"
echo ""

# ---------- 用户选择 ----------
read -r -p "请输入要恢复的备份序号 (或输入 q 退出): " num

if [[ "$num" == "q" || "$num" == "Q" ]]; then
    ui_info "已取消恢复操作"
    ROLLBACK_NEEDED=0
    exit 0
fi

if ! [[ "$num" =~ ^[0-9]+$ ]]; then
    ui_error "无效输入: 请输入数字序号"
    ROLLBACK_NEEDED=0
    exit 1
fi

selected=$(select_backup_file "$BACKUP_DIR" "$num" "newapi_data_*.tar.gz") || {
    ROLLBACK_NEEDED=0
    exit 1
}

ui_success "已选择备份: $(basename "$selected")"

# ---------- 提取时间戳 ----------
TIMESTAMP=$(basename "$selected" | sed 's/newapi_data_\(.*\)\.tar\.gz/\1/')
DB_FILE="${BACKUP_DIR}/newapi_db_${TIMESTAMP}.sql"

if [[ ! -f "$DB_FILE" ]]; then
    ui_error "找不到对应的数据库备份: $DB_FILE"
    ui_info "提示：备份可能不完整，请检查备份目录"
    ROLLBACK_NEEDED=0
    exit 1
fi

ui_info "找到对应的数据库备份: $(basename "$DB_FILE")"

# ---------- 步骤 1：并行校验 ----------
ui_info "步骤 1/4：校验备份文件完整性（并行验证）..."

if ! verify_backup_file "$selected" "$DB_FILE"; then
    ui_error "备份校验失败！拒绝恢复"
    ROLLBACK_NEEDED=0
    exit 1
fi

ui_success "所有备份文件校验通过"

# ---------- 二次确认 ----------
if_novice && {
    echo ""
    ui_warn "⚠ 警告：此操作将覆盖当前数据！"
    echo ""
    echo "即将恢复："
    echo "  - 数据卷: $(basename "$selected")"
    echo "  - 数据库: $(basename "$DB_FILE")"
    echo ""
}

if ! ask_confirm "确定要继续恢复吗？（输入 yes 确认）"; then
    ui_info "已取消恢复操作"
    ROLLBACK_NEEDED=0
    exit 0
fi

# ---------- 读取数据库密码 ----------
DB_ROOT_PASSWORD=$(get_config "deploy.mysql.root_password")

if [[ -z "${DB_ROOT_PASSWORD:-}" ]]; then
    ui_error "未找到数据库密码，请检查配置文件 config/config.yaml"
    ROLLBACK_NEEDED=0
    exit 1
fi

# ---------- 步骤 2：停止容器 ----------
ui_info "步骤 2/4：停止 NewAPI 容器..."
show_progress 2 4 "恢复进度"

cd "$NEWAPI_HOME" || { ui_error "无法进入目录: $NEWAPI_HOME"; ROLLBACK_NEEDED=0; exit 1; }

$DOCKER_COMPOSE_CMD stop new-api 2>/dev/null || true

# ---------- 步骤 3：创建回滚点 ----------
ui_info "步骤 3/4：创建回滚点..."
show_progress 3 4 "恢复进度"
_create_rollback_point "$TIMESTAMP"

# ---------- 步骤 4：恢复数据 ----------
ui_info "步骤 4/4：恢复数据..."
show_progress 4 4 "恢复进度"

# 恢复数据卷
ui_info "恢复数据卷..."
if ! extract_backup_archive "$selected" "$NEWAPI_HOME"; then
    ui_error "数据卷恢复失败！将自动回滚..."
    update_install_status "failed" "data_restore_failed"
    exit 1
fi
ui_success "数据卷恢复完成"

# 恢复数据库（使用环境变量传递密码，避免命令行暴露）
ui_info "恢复 MySQL 数据库..."
if ! docker exec -i mysql bash -c 'MYSQL_PWD="$1" mysql -uroot newapi' _ "$DB_ROOT_PASSWORD" < "$DB_FILE" 2>>"$LOG_FILE"; then
    ui_error "数据库恢复失败！将自动回滚..."
    update_install_status "failed" "db_restore_failed"
    exit 1
fi
ui_success "数据库恢复完成"

# ---------- 重启容器 ----------
ui_info "重启 NewAPI 容器..."
if ! $DOCKER_COMPOSE_CMD restart new-api; then
    ui_error "容器重启失败！将自动回滚..."
    update_install_status "failed" "container_restart_failed"
    exit 1
fi

# ---------- 等待健康检查（最多 60 秒）----------
if ! verify_health_check "new-api" 60; then
    ui_error "容器健康检查超时，将自动回滚..."
    update_install_status "failed" "health_check_timeout"
    exit 1
fi

# ---------- 恢复成功：清除回滚 ----------
_clear_rollback

update_install_status "completed" "restore_done"

DECOMPRESS_TOOL="gzip"
command -v pigz &>/dev/null && DECOMPRESS_TOOL="pigz"

show_summary "恢复完成" \
    "✓ 数据卷已恢复: $(basename "$selected")" \
    "✓ 数据库已恢复: $(basename "$DB_FILE")" \
    "✓ 解压工具: ${DECOMPRESS_TOOL}" \
    "✓ NewAPI 容器已重启" \
    "✓ 恢复时间: $(date '+%Y-%m-%d %H:%M:%S')"

send_webhook "NewAPI 恢复完成" "服务器 $(hostname) 已从备份 ${TIMESTAMP} 恢复数据，解压工具：${DECOMPRESS_TOOL}"
ui_success "=== 数据恢复完成！==="
ui_info "建议执行: newapi-tools health 检查服务状态"

if_novice && {
    if ask_yn "是否立即检查服务健康状态？" "y"; then
        bash "${TOOLKIT_ROOT}/modules/monitor/health.sh"
    fi
}

# 安全清理敏感变量（防止泄露到环境）
unset DB_ROOT_PASSWORD DB_PASSWORD REDIS_PASSWORD SESSION_SECRET 2>/dev/null || true
