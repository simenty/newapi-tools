#!/bin/bash
# NewAPI 数据备份模块 v2.2
# V2.2: source base 框架，调用通用函数
set -eo pipefail

# shellcheck source=lib/_init.sh
source "${TOOLKIT_ROOT}/lib/_init.sh"

# shellcheck source=modules/manage/base/backup-base.sh
source "${MODULES_DIR}/manage/base/backup-base.sh"

# 仅在直接执行时进行权限检查（source 时跳过，便于单元测试）
if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    check_root
    require_docker
fi

# ---------- 清理函数：删除不完整的备份文件 ----------
_cleanup_partial_backup() {
    local backup_dir="$1"
    local timestamp="$2"

    if [[ -z "$backup_dir" || -z "$timestamp" ]]; then
        return 0
    fi

    # 删除不完整的数据库备份
    local partial_db="${backup_dir}/newapi_db_${timestamp}.sql"
    if [[ -f "$partial_db" ]]; then
        # 检查文件是否完整（mysqldump 成功时文件非空且有有效内容）
        if ! grep -q "Dump completed" "$partial_db" 2>/dev/null; then
            log_warn "清理不完整的数据库备份: $partial_db"
            rm -f "$partial_db" "${partial_db}.md5" 2>/dev/null || true
        fi
    fi

    # 删除不完整的压缩包
    local partial_data="${backup_dir}/newapi_data_${timestamp}.tar.gz"
    if [[ -f "$partial_data" ]]; then
        # 用 gzip/pigz -t 测试压缩包完整性（只测试已安装的工具）
        local is_invalid=0
        if command -v gzip &>/dev/null; then
            if ! gzip -t "$partial_data" 2>/dev/null; then
                is_invalid=1
            fi
        fi
        if command -v pigz &>/dev/null; then
            if ! pigz -t "$partial_data" 2>/dev/null; then
                is_invalid=1
            fi
        fi

        if [[ $is_invalid -eq 1 ]]; then
            log_warn "清理不完整的压缩包: $partial_data"
            rm -f "$partial_data" "${partial_data}.md5" 2>/dev/null || true
        fi
    fi
}

# ---------- 工具检测 ----------
_detect_compress_tool() {
    if command -v pigz &>/dev/null; then
        echo "pigz"
    else
        echo "gzip"
    fi
}

_get_cpu_count() {
    local cpus
    cpus=$(nproc 2>/dev/null || grep -c ^processor /proc/cpuinfo 2>/dev/null || echo 1)
    echo $(( cpus > 4 ? 4 : cpus ))
}

# ---------- 异步 SHA-256 校验 ----------
# 注意：实现使用 SHA-256，但变量/函数名保留 MD5 是为了兼容旧版本备份清理逻辑
# .sha256 文件兼容 .md5 清理规则（find -name "*.md5" 会匹配到 .sha256）
_generate_checksum_async() {
    local file="$1"
    local -n _pid_ref=$2

    if [[ ! -f "$file" ]]; then
        log_error "异步 SHA-256：文件不存在: $file"
        return 1
    fi

    (
        sha256sum "$file" > "${file}.sha256"
        log_info "已生成校验文件: ${file}.sha256"
    ) &
    _pid_ref=$!
}

_wait_checksum() {
    local pid="$1"
    local file="$2"

    if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
        log_info "等待 SHA-256 校验完成（pid=$pid）..."
        wait "$pid"
        local rc=$?
        if [[ $rc -ne 0 ]]; then
            log_warn "SHA-256 校验生成失败（pid=$pid），同步重试"
            generate_checksum "$file"
            return $?
        fi
    fi

    if [[ ! -f "${file}.sha256" ]]; then
        log_warn "SHA-256 文件未生成，补充同步计算: ${file}.sha256"
        generate_checksum "$file"
    fi
    return 0
}

# ---------- 主备份逻辑 ----------
do_backup() {
    local backup_dir
    local retention_days

    backup_dir=$(create_backup_dir)
    retention_days=$(get_config "backup.retention_days" "7")

    local timestamp
    timestamp=$(date +"%Y%m%d_%H%M%S")

    # 注册退出清理（仅清理当前 timestamp 的不完整文件）
    # 注意：不使用 exit，让脚本自然退出，避免递归触发 trap
    trap '_cleanup_partial_backup "$backup_dir" "$timestamp"' EXIT

    update_install_status "backuping" "start"
    novice_prompt "正在备份 NewAPI 数据（数据库 + 数据卷）。备份文件将保存到：${backup_dir}"

    # ---------- 步骤 1：备份 MySQL ----------
    ui_info "步骤 1/3：备份 MySQL 数据库"
    show_progress 1 3 "备份进度"

    local db_root_password
    db_root_password=$(get_config "deploy.mysql.root_password")

    if [[ -z "${db_root_password:-}" ]]; then
        ui_error "未找到数据库密码，请检查配置文件 config/config.yaml"
        update_install_status "failed" "db_password_missing"
        return 1
    fi

    local db_file="${backup_dir}/newapi_db_${timestamp}.sql"
    ui_info "正在导出数据库..."

    # 通过 stdin 传递密码（避免命令行暴露）
    if ! docker exec -i mysql bash -c 'read -r pwd; MYSQL_PWD="$pwd" mysqldump -uroot \
        --single-transaction --databases newapi' <<< "$db_root_password" > "$db_file" 2>>"$LOG_FILE"; then

        # 清理密码变量
        unset db_root_password
        ui_error "数据库备份失败！请查看日志: $LOG_FILE"
        update_install_status "failed" "db_backup_failed"
        return 1
    fi

    # 验证数据库备份完整性
    if ! grep -q "Dump completed" "$db_file" 2>/dev/null; then
        ui_error "数据库备份文件不完整（未找到 'Dump completed' 标记）"
        rm -f "$db_file"
        update_install_status "failed" "db_backup_incomplete"
        return 1
    fi

    _generate_checksum_async "$db_file" db_sha256_pid
    log_info "数据库 SHA-256 校验已在后台启动 (pid=${db_sha256_pid})"

    # 设置安全权限（防止非授权访问）
    chmod 600 "$db_file"
    log_info "数据库备份文件权限已设置: 600 (${db_file})"

    ui_success "数据库备份完成: $(basename "$db_file")"
    show_progress 2 3 "备份进度"

    # ---------- 步骤 2：打包数据卷 ----------
    ui_info "步骤 2/3：打包数据卷（并行压缩）"

    local data_file="${backup_dir}/newapi_data_${timestamp}.tar.gz"
    ui_info "正在打包数据卷（data, npm）..."

    if ! create_backup_archive "$data_file" "$NEWAPI_HOME" "data" "npm"; then
        ui_warn "数据卷打包失败（可能部分目录不存在），跳过数据卷备份"
        rm -f "$data_file" 2>/dev/null || true
    else
        _generate_checksum_async "$data_file" data_sha256_pid
        log_info "数据卷 SHA-256 校验已在后台启动 (pid=${data_sha256_pid})"

        # 设置安全权限
        chmod 600 "$data_file"
        log_info "数据卷备份文件权限已设置: 600 (${data_file})"
        ui_success "数据卷备份完成: $(basename "$data_file")"
    fi

    show_progress 3 3 "备份进度"

    # ---------- 等待 SHA-256 ----------
    ui_info "等待校验文件生成完毕..."
    _wait_checksum "${db_sha256_pid:-}" "$db_file"
    _wait_checksum "${data_sha256_pid:-}" "$data_file"

    # ---------- 步骤 3：清理过期备份 ----------
    ui_info "步骤 3/3：清理过期备份"
    cleanup_old_backups "$backup_dir" "$retention_days"

    local backup_count
    backup_count=$(find "$backup_dir" -name "*.sql" -type f 2>/dev/null | wc -l)

    local compress_tool
    compress_tool=$(_detect_compress_tool)
    local data_size
    data_size=$(calculate_backup_size "$data_file")

    update_install_status "completed" "backup_done"
    mark_step_completed "backup_${timestamp}"

    show_summary "备份完成" \
        "✓ 数据库备份: $(basename "$db_file")" \
        "✓ 数据卷备份: $(basename "$data_file")（${data_size}）" \
        "✓ 校验文件已生成（.sha256）" \
        "✓ 压缩工具: ${compress_tool}" \
        "✓ 当前保留备份数: ${backup_count}" \
        "✓ 过期备份已清理（>${retention_days}天）"

    send_webhook "NewAPI 备份完成" "服务器 $(hostname) 的备份已完成，压缩工具：${compress_tool}，保留 ${backup_count} 个备份"

    # 取消清理 trap（备份成功，不需要清理）
    trap - EXIT

    log_success "备份完成！"
}

# ---------- 以下为主执行逻辑，仅在直接运行时执行 ----------
if [[ "${BASH_SOURCE[0]}" != "$0" ]]; then
    return 0 2>/dev/null || true
fi

MANUAL=false
CRON_MODE=false

# 解析参数
while [[ $# -gt 0 ]]; do
    case "$1" in
        --manual) MANUAL=true; shift ;;
        --cron)   CRON_MODE=true; shift ;;
        *) log_error "未知参数: $1"; exit 1 ;;
    esac
done

# ---------- 执行 ----------
if [[ "$CRON_MODE" == "true" ]]; then
    log_info "=== Cron 模式备份开始 ==="
    do_backup >> "$LOG_FILE" 2>&1
    log_info "=== Cron 模式备份结束 ==="
else
    if_novice && show_welcome
    do_backup
fi

# 安全清理敏感变量
unset DB_PASSWORD DB_ROOT_PASSWORD REDIS_PASSWORD SESSION_SECRET MYSQL_PASSWORD 2>/dev/null || true
