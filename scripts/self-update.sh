#!/bin/bash
# newapi-tools 自更新脚本（资深开发优化版）
# 增强：Git 状态检查、更新后权限修复、更新日志
set -eo pipefail

# shellcheck source=lib/_init.sh
source "${TOOLKIT_ROOT}/lib/_init.sh"

cd "$TOOLKIT_ROOT" || { log_error "无法进入工具目录"; exit 1; }

# ---------- 检查 Git 仓库状态 ----------
if [[ ! -d ".git" ]]; then
    log_error "当前目录不是 Git 仓库"
    log_info "提示：如果是通过压缩包安装的，请手动下载最新版本"
    exit 1
fi

# ---------- 检查是否有未提交的本地修改 ----------
if ! git diff --quiet 2>/dev/null || ! git diff --cached --quiet 2>/dev/null; then
    log_warn "检测到本地修改："
    git status -s
    read -r -p "更新将覆盖本地修改，确定继续吗？(yes/N): " confirm
    if [[ "$confirm" != "yes" ]]; then
        log_info "已取消更新"
        exit 0
    fi
    log_warn "将丢弃本地修改..."
    git reset --hard HEAD
fi

# ---------- 获取当前版本信息 ----------
OLD_VERSION=$(git log --oneline -1 2>/dev/null | cut -d' ' -f1 || echo "unknown")
log_info "当前版本: $OLD_VERSION"

# ---------- 拉取最新代码 ----------
log_info "正在从 GitHub 更新工具集..."
if ! git pull origin main 2>&1 | tee -a "$LOG_FILE"; then
    log_error "Git 拉取失败，请检查网络连接或仓库权限"
    exit 1
fi

# ---------- 获取新版本信息 ----------
NEW_VERSION=$(git log --oneline -1 2>/dev/null | cut -d' ' -f1 || echo "unknown")
log_info "更新后版本: $NEW_VERSION"

# ---------- 修复脚本执行权限 ----------
log_info "修复脚本执行权限..."
chmod +x "${TOOLKIT_ROOT}/newapi-tools.sh"
find "${TOOLKIT_ROOT}/modules" -name "*.sh" -exec chmod +x {} \; 2>/dev/null || true
find "${TOOLKIT_ROOT}/scripts" -name "*.sh" -exec chmod +x {} \; 2>/dev/null || true
log_success "脚本权限已修复"

# ---------- 更新日志 ----------
log_success "工具集已更新: $OLD_VERSION → $NEW_VERSION"
log_info "变更记录:"
git log --oneline "$OLD_VERSION..HEAD" 2>/dev/null | head -10 || echo "  (无变更记录或首次更新)"

# ---------- 重启主菜单 ----------
log_info "即将重启主菜单..."
sleep 1
exec "${TOOLKIT_ROOT}/newapi-tools.sh"
