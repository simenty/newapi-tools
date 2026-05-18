#!/bin/bash
# 智能换源并更新系统 v2.0
# 新增：状态管理、UI 增强、多源选择、智能检测系统版本
set -eo pipefail

# shellcheck source=lib/_init.sh
source "${TOOLKIT_ROOT}/lib/_init.sh"

# 仅在直接执行时进行权限检查（source 时跳过，便于单元测试）
if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    check_root
fi

# ---------- 以下为主执行逻辑，仅在直接运行时执行（source 时跳过，便于单元测试）----------
if [[ "${BASH_SOURCE[0]}" != "$0" ]]; then
    return 0 2>/dev/null || true
fi

# ---------- 显示新手提示 ----------
novice_prompt "换源可以加速软件包下载。将备份当前源并更换为国内镜像源（清华/阿里/腾讯）。"

# ---------- 检测系统和版本（使用 os_adapter.sh）----------
ui_info "检测系统信息..."
os_detect_full

# apt-source.sh 只支持 Debian 系列
if [[ "$OS_FAMILY" != "debian" ]]; then
    ui_error "换源脚本仅支持 Debian/Ubuntu 系统（当前: $OS_FAMILY）"
    ui_info "RHEL 系请使用: dnf/yum 换源"
    ui_info "Arch 系请使用: pacman-mirrors"
    exit 1
fi

OS_NAME=$(grep "^PRETTY_NAME=" /etc/os-release 2>/dev/null | cut -d'"' -f2 || echo "Unknown")
ui_info "检测到系统: $OS_NAME $OS_VERSION ($OS_CODENAME)"
echo ""

# ---------- 检查当前源 ----------
if [[ -f /etc/apt/sources.list ]]; then
    ui_info "当前软件源:"
    grep -E "^deb " /etc/apt/sources.list | head -2 | while read -r line; do
        echo "  $line"
    done
    echo ""
    
    # 如果已经配置了我们推荐的源，询问是否跳过
    if grep -q "mirrors.tuna.tsinghua" /etc/apt/sources.list; then
        ui_success "已配置清华源"
        if ! ask_yn "是否重新配置软件源？" "n"; then
            ui_info "跳过换源"
            mark_step_completed "apt_source"
            set_state "system.apt_sourced" "true"
            exit 0
        fi
    fi
fi

# ---------- 选择镜像源 ----------
echo -e "${UI_BOLD}请选择镜像源：${UI_PLAIN}"
echo "  1) 清华大学源（推荐） - mirrors.tuna.tsinghua.edu.cn"
echo "  2) 阿里云源 - mirrors.aliyun.com"
echo "  3) 腾讯云源 - mirrors.cloud.tencent.com"
echo "  4) 华为云源 - repo.huaweicloud.com"
echo "  5) 保持当前源不变"
echo ""

read -r -p "请选择 [1-5]: " source_choice

case "$source_choice" in
    1)
        MIRROR="mirrors.tuna.tsinghua.edu.cn"
        MIRROR_DESC="清华大学源"
        ;;
    2)
        MIRROR="mirrors.aliyun.com"
        MIRROR_DESC="阿里云源"
        ;;
    3)
        MIRROR="mirrors.cloud.tencent.com"
        MIRROR_DESC="腾讯云源"
        ;;
    4)
        MIRROR="repo.huaweicloud.com"
        MIRROR_DESC="华为云源"
        ;;
    5)
        ui_info "保持当前源不变"
        exit 0
        ;;
    *)
        ui_error "无效选择"
        exit 1
        ;;
esac

# ---------- 备份当前配置 ----------
ui_info "步骤 1/3：备份当前配置..."
show_progress 1 3 "换源进度"

if [[ -f /etc/apt/sources.list ]]; then
    backup_file /etc/apt/sources.list
    ui_success "已备份: /etc/apt/sources.list"
else
    ui_warn "源文件不存在，将创建新文件"
fi

# ---------- 更换为国内镜像源 ----------
ui_info "步骤 2/3：更换为 $MIRROR_DESC..."
show_progress 2 3 "换源进度"

# 根据系统版本生成源
cat > /etc/apt/sources.list << EOF
# NewAPI Tools 自动配置 ($MIRROR_DESC)
# 配置时间: $(date '+%Y-%m-%d %H:%M:%S')

deb https://${MIRROR}/ubuntu/ ${OS_CODENAME} main restricted universe multiverse
deb https://${MIRROR}/ubuntu/ ${OS_CODENAME}-updates main restricted universe multiverse
deb https://${MIRROR}/ubuntu/ ${OS_CODENAME}-backports main restricted universe multiverse
deb http://security.ubuntu.com/ubuntu/ ${OS_CODENAME}-security main restricted universe multiverse
EOF

ui_success "源已更换为 $MIRROR_DESC"
show_progress 3 3 "换源进度"

# ---------- 更新 apt 缓存并升级系统 ----------
ui_info "步骤 3/3：更新 apt 缓存并升级系统..."
show_progress 3 3 "换源进度"

ui_info "更新 apt 缓存..."
if ! apt update; then
    ui_error "apt 更新失败"
    ui_info "可能的原因："
    echo "  1. 网络不通"
    echo "  2. 镜像源暂时不可用"
    echo "  3. GPG 密钥问题"
    echo ""
    ui_info "修复建议："
    echo "  - 检查网络: ping -c 1 baidu.com"
    echo "  - 更换其他镜像源"
    echo "  - 手动执行: apt update  查看详细错误"
    
    update_install_status "failed" "apt_update_failed"
    exit 1
fi

ui_info "升级系统（自动确认）..."
apt upgrade -y

ui_success "系统更新完成"

# ---------- 更新状态 ----------
update_install_status "completed" "apt_source_done"
mark_step_completed "apt_source"
set_state "system.apt_sourced" "true"

# 显示配置摘要
show_summary "换源完成" \
    "✓ 镜像源: $MIRROR_DESC" \
    "✓ 系统版本: $OS_NAME $OS_VERSION" \
    "✓ apt 缓存已更新" \
    "✓ 系统已升级到最新" \
    "" \
    "配置文件: /etc/apt/sources.list" \
    "备份文件: /etc/apt/sources.list.bak.*"

# 发送通知
send_webhook "换源完成" \
    "服务器 $(hostname) 的软件源已更换为 $MIRROR_DESC"

# 询问是否继续安装 Docker（新手模式）
if_novice && {
    echo ""
    ui_info "建议下一步：安装 Docker（菜单选项 1 的子步骤）"
    
    if ask_yn "是否立即安装 Docker？" "y"; then
        bash "${TOOLKIT_ROOT}/modules/init/docker.sh"
    fi
}
