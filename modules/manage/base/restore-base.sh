#!/bin/bash
# restore-base.sh — 通用恢复框架
# 只定义函数，不执行任何操作。被 newapi/restore.sh source 后调用。
# 用法: source "${MODULES_DIR}/manage/base/restore-base.sh"

set -eo pipefail

# shellcheck source=lib/_init.sh
source "${TOOLKIT_ROOT}/lib/_init.sh"

# ---------- 列出可用备份 ----------
# 参数: $1 = 备份目录, $2 = 文件匹配模式（如 "newapi_data_*.tar.gz"）
# 输出: 按时间排序的备份文件列表（最新在前）
list_available_backups() {
    local backup_dir="$1"
    local pattern="${2:-*.tar.gz}"

    if [[ ! -d "$backup_dir" ]]; then
        log_error "备份目录不存在: $backup_dir"
        return 1
    fi

    local count
    count=$(ls -1ht "$backup_dir"/$pattern 2>/dev/null | wc -l)
    if [[ $count -eq 0 ]]; then
        log_warn "备份目录为空: $backup_dir"
        return 1
    fi

    ls -1ht "$backup_dir"/$pattern 2>/dev/null
}

# ---------- 选择备份文件 ----------
# 参数: $1 = 备份目录, $2 = 序号
# 返回: 通过 echo 输出选中的文件路径
select_backup_file() {
    local backup_dir="$1"
    local num="$2"
    local pattern="${3:-newapi_data_*.tar.gz}"

    local selected
    selected=$(ls -1t "$backup_dir"/$pattern 2>/dev/null | sed -n "${num}p")

    if [[ -z "$selected" || ! -f "$selected" ]]; then
        log_error "无效的序号: $num"
        return 1
    fi

    echo "$selected"
}

# ---------- 验证备份文件 ----------
# 参数: $1 = 数据备份文件, $2 = 数据库备份文件
# 返回: 0=全部通过, 1=校验失败
verify_backup_file() {
    local data_file="$1"
    local db_file="$2"
    local all_ok=true

    # 并行校验
    local data_result_file db_result_file
    data_result_file=$(mktemp)
    db_result_file=$(mktemp)
    chmod 600 "$data_result_file" "$db_result_file"

    # 后台校验函数
    (
        if verify_checksum "$data_file" >/dev/null 2>&1; then
            echo "ok" > "$data_result_file"
        else
            echo "fail" > "$data_result_file"
        fi
    ) &
    local data_pid=$!

    (
        if verify_checksum "$db_file" >/dev/null 2>&1; then
            echo "ok" > "$db_result_file"
        else
            echo "fail" > "$db_result_file"
        fi
    ) &
    local db_pid=$!

    # 等待校验完成
    wait "$data_pid"
    wait "$db_pid"

    local data_result db_result
    data_result=$(cat "$data_result_file")
    db_result=$(cat "$db_result_file")
    rm -f "$data_result_file" "$db_result_file"

    if [[ "$data_result" != "ok" ]]; then
        log_error "数据卷备份校验失败: $data_file"
        all_ok=false
    fi

    if [[ "$db_result" != "ok" ]]; then
        log_error "数据库备份校验失败: $db_file"
        all_ok=false
    fi

    if $all_ok; then
        log_success "所有备份文件校验通过"
        return 0
    else
        return 1
    fi
}

# ---------- 解压归档 ----------
# 参数: $1 = 归档文件路径, $2 = 目标目录
# 返回: 0=成功, 1=失败
extract_backup_archive() {
    local archive="$1"
    local dest_dir="$2"

    if [[ ! -f "$archive" ]]; then
        log_error "归档文件不存在: $archive"
        return 1
    fi

    mkdir -p "$dest_dir"

    local file_size
    file_size=$(stat -c '%s' "$archive" 2>/dev/null || stat -f '%z' "$archive" 2>/dev/null || echo 0)

    if command -v pigz &>/dev/null; then
        local cpu_count
        cpu_count=$(nproc 2>/dev/null || grep -c ^processor /proc/cpuinfo 2>/dev/null || echo 1)
        cpu_count=$(( cpu_count > 4 ? 4 : cpu_count ))
        if command -v pv &>/dev/null && [[ $file_size -gt 0 ]]; then
            pv -s "$file_size" -N "解压中" "$archive" \
                | pigz -d -p "$cpu_count" -c \
                | tar -xf - -C "$dest_dir" 2>>"$LOG_FILE"
        else
            pigz -d -p "$cpu_count" -c "$archive" \
                | tar -xf - -C "$dest_dir" 2>>"$LOG_FILE"
        fi
    else
        if command -v pv &>/dev/null && [[ $file_size -gt 0 ]]; then
            pv -s "$file_size" -N "解压中" "$archive" \
                | tar -zxf - -C "$dest_dir" 2>>"$LOG_FILE"
        else
            tar -zxf "$archive" -C "$dest_dir" 2>>"$LOG_FILE"
        fi
    fi
    return $?
}
