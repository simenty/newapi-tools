#!/bin/bash
# backup-base.sh — 通用备份框架
# 只定义函数，不执行任何操作。被 newapi/backup.sh source 后调用。
# 用法: source "${MODULES_DIR}/manage/base/backup-base.sh"

set -eo pipefail

# shellcheck source=lib/_init.sh
source "${TOOLKIT_ROOT}/lib/_init.sh"

# ---------- 创建备份目录 ----------
# 参数: $1 = 备份目录路径（可选，默认从配置读取）
# 返回: 通过 echo 输出备份目录路径
create_backup_dir() {
    local backup_dir="${1:-}"
    if [[ -z "$backup_dir" ]]; then
        backup_dir=$(get_config "backup.dir" "${NEWAPI_HOME}/backups")
    fi
    mkdir -p "$backup_dir"
    echo "$backup_dir"
}

# ---------- 计算备份大小 ----------
# 参数: $1 = 文件路径
# 返回: 人类可读的大小字符串
calculate_backup_size() {
    local file="$1"
    if [[ -f "$file" ]]; then
        du -h "$file" | awk '{print $1}'
    else
        echo "N/A"
    fi
}

# ---------- 清理过期备份 ----------
# 参数: $1 = 备份目录, $2 = 保留天数（可选，默认从配置读取）
cleanup_old_backups() {
    local backup_dir="$1"
    local retention_days="${2:-}"
    if [[ -z "$retention_days" ]]; then
        retention_days=$(get_config "backup.retention_days" "7")
    fi

    if [[ ! -d "$backup_dir" ]]; then
        log_warn "备份目录不存在，跳过清理: $backup_dir"
        return 0
    fi

    log_info "清理超过 ${retention_days} 天的旧备份..."
    find "$backup_dir" -type f \( -name "*.sql" -o -name "*.tar.gz" -o -name "*.md5" -o -name "*.sha256" \) -mtime +"${retention_days}" -delete -print 2>/dev/null | while read -r f; do
        log_info "已删除过期文件: $f"
    done
}

# ---------- 校验备份完整性 ----------
# 参数: $1 = 备份文件路径
# 返回: 0=通过, 1=失败
verify_backup_integrity() {
    local file="$1"
    if [[ ! -f "$file" ]]; then
        log_error "校验失败，文件不存在: $file"
        return 1
    fi

    local checksum_file="${file}.sha256"
    if [[ ! -f "$checksum_file" ]]; then
        # 回退到 .md5 格式（兼容旧版）
        checksum_file="${file}.md5"
    fi

    if [[ -f "$checksum_file" ]]; then
        if [[ "$checksum_file" == *.sha256 ]]; then
            if sha256sum -c "$checksum_file" --status 2>/dev/null; then
                log_success "SHA-256 校验通过: $file"
                return 0
            else
                log_error "SHA-256 校验失败: $file"
                return 1
            fi
        else
            if md5sum -c "$checksum_file" --status 2>/dev/null; then
                log_success "MD5 校验通过: $file"
                return 0
            else
                log_error "MD5 校验失败: $file"
                return 1
            fi
        fi
    else
        log_warn "未找到校验文件，跳过完整性校验: $file"
        return 0
    fi
}

# ---------- 创建压缩归档 ----------
# 参数: $1 = 输出文件路径, $2 = 源目录, $3... = 目标列表
# 返回: 0=成功, 1=失败
create_backup_archive() {
    local output_file="$1"
    local source_dir="$2"
    shift 2
    local targets=("$@")

    # 检测压缩工具
    local compress_tool="gzip"
    if command -v pigz &>/dev/null; then
        compress_tool="pigz"
    fi

    # 计算 CPU 核心数
    local cpu_count
    cpu_count=$(nproc 2>/dev/null || grep -c ^processor /proc/cpuinfo 2>/dev/null || echo 1)
    cpu_count=$(( cpu_count > 4 ? 4 : cpu_count ))

    # 计算源数据大小
    local total_size=0
    for t in "${targets[@]}"; do
        if [[ -e "$source_dir/$t" ]]; then
            local sz
            sz=$(du -sb "$source_dir/$t" 2>/dev/null | awk '{print $1}')
            sz=${sz:-0}
            total_size=$(( total_size + sz ))
        fi
    done

    log_info "压缩工具: ${compress_tool}（${cpu_count} 线程），源数据: $(( total_size / 1024 / 1024 ))MB"

    if [[ "$compress_tool" == "pigz" ]]; then
        if command -v pv &>/dev/null && [[ $total_size -gt 0 ]]; then
            tar -C "$source_dir" -cf - "${targets[@]}" 2>>"$LOG_FILE" \
                | pv -s "$total_size" -N "压缩中" \
                | pigz -p "$cpu_count" -c > "$output_file" 2>>"$LOG_FILE"
        else
            tar -C "$source_dir" -cf - "${targets[@]}" 2>>"$LOG_FILE" \
                | pigz -p "$cpu_count" -c > "$output_file" 2>>"$LOG_FILE"
        fi
    else
        if command -v pv &>/dev/null && [[ $total_size -gt 0 ]]; then
            tar -C "$source_dir" -cf - "${targets[@]}" 2>>"$LOG_FILE" \
                | pv -s "$total_size" -N "压缩中" \
                | gzip -c > "$output_file" 2>>"$LOG_FILE"
        else
            tar -C "$source_dir" -zcf "$output_file" "${targets[@]}" 2>>"$LOG_FILE"
        fi
    fi
    return $?
}
