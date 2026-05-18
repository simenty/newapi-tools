#!/bin/bash
# DNS 配置模块 v2.0
# 新增：状态管理、UI 增强、智能检测、多 DNS 选择
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
novice_prompt "配置 DNS 可以加速软件包下载和 API 访问。将配置国内公共 DNS（阿里 / 腾讯 / 114）。"

# ---------- 检查当前 DNS ----------
ui_info "检测当前 DNS 配置..."
current_dns=$(grep -E "^nameserver" /etc/resolv.conf 2>/dev/null | awk '{print $2}' | tr '\n' ' ' || echo "未配置")

if [[ -n "$current_dns" ]]; then
    ui_info "当前 DNS: $current_dns"
    
    # 如果已经配置了我们推荐的 DNS，跳过
    if echo "$current_dns" | grep -q "223.5.5.5"; then
        ui_success "DNS 已配置（阿里 DNS）"
        
        if ! ask_yn "是否重新配置 DNS？" "n"; then
            ui_info "跳过 DNS 配置"
            mark_step_completed "dns"
            set_state "system.dns_configured" "true"
            exit 0
        fi
    fi
fi

# ---------- 选择 DNS ----------
echo ""
echo -e "${UI_BOLD}请选择 DNS 服务器：${UI_PLAIN}"
echo "  1) 阿里 DNS（推荐国内） - 223.5.5.5, 223.6.6.6"
echo "  2) 腾讯 DNS - 119.29.29.29, 182.254.116.116"
echo "  3) 114 DNS - 114.114.114.114, 114.114.115.115"
echo "  4) Google DNS（推荐国外） - 8.8.8.8, 8.8.4.4"
echo "  5) 自定义"
echo "  q) 跳过"
echo ""

read -r -p "请选择 [1-5/q]: " dns_choice

case "$dns_choice" in
    1)
        DNS_SERVER="223.5.5.5 223.6.6.6"
        DNS_DESC="阿里 DNS"
        ;;
    2)
        DNS_SERVER="119.29.29.29 182.254.116.116"
        DNS_DESC="腾讯 DNS"
        ;;
    3)
        DNS_SERVER="114.114.114.114 114.114.115.115"
        DNS_DESC="114 DNS"
        ;;
    4)
        DNS_SERVER="8.8.8.8 8.8.4.4"
        DNS_DESC="Google DNS"
        ;;
    5)
        read -r -p "请输入主 DNS: " dns1
        read -r -p "请输入备 DNS: " dns2
        DNS_SERVER="$dns1 $dns2"
        DNS_DESC="自定义 DNS"
        ;;
    q|Q)
        ui_info "跳过 DNS 配置"
        exit 0
        ;;
    *)
        ui_error "无效选择"
        exit 1
        ;;
esac

# ---------- 备份当前配置 ----------
ui_info "备份当前 DNS 配置..."
backup_file "/etc/resolv.conf"

# ---------- 配置 DNS ----------
ui_info "配置 $DNS_DESC..."
show_progress 1 2 "配置 DNS"

cat > /etc/resolv.conf << EOF
# NewAPI Tools 自动配置 ($DNS_DESC)
# 配置时间: $(date '+%Y-%m-%d %H:%M:%S')

nameserver $(echo "$DNS_SERVER" | awk '{print $1}')
nameserver $(echo "$DNS_SERVER" | awk '{print $2}')
EOF

show_progress 2 2 "配置 DNS"

# ---------- 验证配置 ----------
ui_info "验证 DNS 配置..."
new_dns=$(grep -E "^nameserver" /etc/resolv.conf | awk '{print $2}' | tr '\n' ' ')

if [[ -n "$new_dns" ]]; then
    ui_success "DNS 配置完成"
    ui_info "当前 DNS: $new_dns"
    
    # 测试 DNS 解析
    ui_info "测试 DNS 解析（google.com）..."
    if ping -c 1 -W 2 google.com &>/dev/null; then
        ui_success "DNS 解析正常"
    else
        ui_warn "DNS 解析可能异常，请检查配置"
    fi
    
    # 更新状态
    mark_step_completed "dns"
    set_state "system.dns_configured" "true"
    
    # 显示配置摘要
    show_summary "DNS 配置完成" \
        "✓ DNS 服务器: $DNS_DESC" \
        "✓ 主 DNS: $(echo "$DNS_SERVER" | awk '{print $1}')" \
        "✓ 备 DNS: $(echo "$DNS_SERVER" | awk '{print $2}')" \
        "✓ 配置文件: /etc/resolv.conf"
    
    # 发送通知
    send_webhook "DNS 配置完成" "服务器 $(hostname) 的 DNS 已配置为 $DNS_DESC"
else
    ui_error "DNS 配置失败"
    update_install_status "failed" "dns_failed"
    exit 1
fi

# 询问是否继续配置其他初始化（新手模式）
if_novice && {
    echo ""
    ui_info "建议下一步："
    echo "  1. 配置软件源（菜单选项 1 的子步骤）"
    echo "  2. 安装 Docker（菜单选项 1 的子步骤）"
    echo ""
    
    if ask_yn "是否立即配置软件源？" "y"; then
        bash "${TOOLKIT_ROOT}/modules/init/apt-source.sh"
    fi
    
    if ask_yn "是否立即安装 Docker？" "y"; then
        bash "${TOOLKIT_ROOT}/modules/init/docker.sh"
    fi
}
