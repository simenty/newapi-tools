#!/bin/bash
# newapi-tools 一键安装器（资深开发优化版）
# 修复：错误检查、幂等性、版本检测

# 守卫：如果是被 source，直接 return（不执行安装逻辑，不设置 set -e）
if [[ "${BASH_SOURCE[0]}" != "$0" ]]; then
    return 0
fi

set -eo pipefail

# ---------- 颜色定义 ----------
GREEN='\033[32m'; RED='\033[31m'; YELLOW='\033[33m'; PLAIN='\033[0m'

TOOL_DIR="/opt/newapi-tools"
REPO_URL="${REPO_URL:-https://github.com/simenty/newapi-tools.git}"

# ---------- 权限检查 ----------
if [[ $EUID -ne 0 ]]; then
    echo -e "${RED}错误：请使用 root 用户执行本脚本${PLAIN}"
    echo "提示：使用 sudo bash $0 或切换到 root 用户"
    exit 1
fi

echo -e "${GREEN}>>> 正在安装 NewAPI 运维工具集 ...${PLAIN}"

# ---------- 幂等性：若已存在则更新 ----------
if [[ -d "$TOOL_DIR/.git" ]]; then
    echo -e "${YELLOW}目录已存在且为 Git 仓库，将执行 git pull 更新...${PLAIN}"
    cd "$TOOL_DIR" || { echo "无法进入目录: $TOOL_DIR"; exit 1; }
    if git pull origin main; then
        log_success "工具集已更新"
    else
        log_error "更新失败，请检查网络或手动处理"
        exit 1
    fi
elif [[ -d "$TOOL_DIR" ]]; then
    log_warn "目录已存在但不是 Git 仓库，将备份后重新克隆..."
    mv "$TOOL_DIR" "${TOOL_DIR}.bak.$(date +%s)"
    log_info "已备份原目录为: ${TOOL_DIR}.bak.*"
fi

# ---------- 首次安装：克隆仓库 ----------
if [[ ! -d "$TOOL_DIR" ]]; then
    echo -e "${GREEN}正在从 GitHub 克隆仓库...${PLAIN}"
    if command -v git &>/dev/null; then
        if ! git clone "$REPO_URL" "$TOOL_DIR" 2>&1 | tee -a /tmp/newapi-tools-install.log; then
            echo -e "${RED}克隆失败，请检查网络连接${PLAIN}"
            echo "如果 GitHub 访问受限，请设置环境变量: export REPO_URL=https://mirror.example.com/newapi-tools.git"
            exit 1
        fi
    else
        log_error "系统未安装 git，请先执行: apt install git"
        exit 1
    fi
fi

# ---------- 确保脚本可执行 ----------
chmod +x "$TOOL_DIR/newapi-tools.sh"
find "$TOOL_DIR/modules" -name "*.sh" -exec chmod +x {} \; 2>/dev/null || true
find "$TOOL_DIR/scripts" -name "*.sh" -exec chmod +x {} \; 2>/dev/null || true

# ---------- 创建全局软链接 ----------
if [[ -L /usr/local/bin/newapi-tools ]]; then
    rm -f /usr/local/bin/newapi-tools
fi
ln -sf "$TOOL_DIR/newapi-tools.sh" /usr/local/bin/newapi-tools

# ---------- 验证安装 ----------
if [[ -x /usr/local/bin/newapi-tools ]]; then
    echo -e "${GREEN}安装完成！${PLAIN}"
    echo ""
    echo "  运行命令:  newapi-tools"
    echo "  安装位置:  $TOOL_DIR"
    echo "  版本信息:  $(cd "$TOOL_DIR" && git log --oneline -1 2>/dev/null || echo '未知')"
    echo ""
    echo -e "${YELLOW}提示：首次使用请先执行菜单选项 1 进行环境初始化${PLAIN}"
else
    echo -e "${RED}安装验证失败，请检查: $TOOL_DIR${PLAIN}"
    exit 1
fi
