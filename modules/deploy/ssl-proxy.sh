#!/bin/bash
# 自动配置 SSL 与 Nginx 反向代理 v2.0
# 新增：状态管理、UI 增强、智能默认值、进度显示
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
novice_prompt "SSL 配置将自动申请 Lets Encrypt 证书并配置 Nginx 反向代理。需要先部署 NPM（菜单选项 2 会自动部署）。"

# ---------- 检查前置条件 ----------
if ! is_step_completed "newapi_install"; then
    ui_error "未检测到 NewAPI 部署，请先执行安装（菜单选项 2）"
    exit 1
fi

if ! docker ps --format '{{.Names}}' 2>/dev/null | grep -qw "npm"; then
    ui_error "NPM 容器未运行"
    ui_info "修复建议："
    echo "  1. 检查 NPM 是否部署: cd ${NEWAPI_HOME} && docker compose ps"
    echo "  2. 启动 NPM: cd ${NEWAPI_HOME} && docker compose up -d npm"
    echo "  3. 重新部署: 菜单选项 2"
    exit 1
fi

# ---------- 步骤 1：收集信息 ----------
ui_info "步骤 1/3：收集配置信息..."
show_progress 1 3 "配置进度"

# 智能默认值：从配置读取或自动检测
DEFAULT_Domain=$(get_config "deploy.ssl.domain" "")
DEFAULT_EMAIL=$(get_config "deploy.ssl.email" "admin@example.com")

# 用户输入（带默认值）
read -r -p "请输入域名 (如 api.example.com) [${DEFAULT_Domain}]: " DOMAIN
DOMAIN=${DOMAIN:-$DEFAULT_Domain}

read -r -p "请输入 NPM 管理员邮箱 [${DEFAULT_EMAIL}]: " NPM_EMAIL
NPM_EMAIL=${NPM_EMAIL:-$DEFAULT_EMAIL}

read -s -p "请输入 NPM 管理员密码: " NPM_PASS; echo

if [[ -z "$DOMAIN" || -z "$NPM_EMAIL" || -z "$NPM_PASS" ]]; then
    ui_error "所有字段均为必填项"
    unset NPM_PASS
    exit 1
fi

# 保存配置（可选）
if ask_yn "是否保存域名和邮箱到配置（密码不保存）？" "y"; then
    set_config "deploy.ssl.domain" "$DOMAIN"
    set_config "deploy.ssl.email" "$NPM_EMAIL"
    ui_success "配置已保存"
fi

show_progress 2 3 "配置进度"

# ---------- 步骤 2：登录 NPM 获取 Token ----------
ui_info "步骤 2/3：登录 NPM 获取 Token..."
show_progress 2 3 "配置进度"

ui_info "正在登录 NPM..."
# 使用环境变量传递密码（避免命令行暴露）
export NPM_PASS
TOKEN=$(npm_login "$NPM_EMAIL")
unset NPM_PASS  # 立即清除密码

if [[ -z "$TOKEN" ]]; then
    ui_error "NPM 登录失败，请检查："
    echo "  1. NPM 是否已启动（等待 30 秒后再试）"
    echo "  2. 邮箱/密码是否正确（默认: admin@example.com / changeme）"
    echo "  3. 是否已修改默认密码"
    echo ""
    ui_info "修复建议："
    echo "  - 查看 NPM 日志: docker logs npm"
    echo "  - 重置 NPM: docker rm -f npm && cd ${NEWAPI_HOME} && docker compose up -d npm"
    exit 1
fi

ui_success "NPM 登录成功"
show_progress 3 3 "配置进度"

# ---------- 步骤 3：创建反向代理并申请 SSL 证书 ----------
ui_info "步骤 3/3：创建反向代理并申请 Let's Encrypt SSL 证书..."
show_progress 3 3 "配置进度"

ui_info "正在配置..."
ui_info "域名: $DOMAIN"
ui_info "转发: http://127.0.0.1:3000"

# 构建 JSON payload
PAYLOAD=$(cat <<EOF
{
  "domain_names": ["$DOMAIN"],
  "forward_scheme": "http",
  "forward_host": "127.0.0.1",
  "forward_port": 3000,
  "access_list_id": "0",
  "certificate_id": "new",
  "meta": {
    "letsencrypt_agree": true,
    "dns_challenge": false
  },
  "ssl_forced": true,
  "hsts_enabled": true,
  "http2_support": true,
  "block_exploits": true
}
EOF
)

RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d "$PAYLOAD" \
    "http://127.0.0.1:81/api/nginx/proxy-hosts")

# 分离响应体和状态码
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
RESP_BODY=$(echo "$RESPONSE" | head -n -1)

if [[ "$HTTP_CODE" == "200" || "$HTTP_CODE" == "201" ]]; then
    update_install_status "completed" "ssl_configured"
    mark_step_completed "ssl_$(echo "$DOMAIN" | sed 's/[^a-zA-Z0-9]/-/g')"
    
    # 显示配置摘要
    show_summary "SSL 配置完成" \
        "✓ 域名: https://$DOMAIN" \
        "✓ SSL 证书: Let's Encrypt（自动续期）" \
        "✓ 反向代理: http://127.0.0.1:3000 → https://$DOMAIN" \
        "✓ 安全选项: HSTS、HTTP/2、漏洞防护（已启用）" \
        "" \
        "提示：证书签发可能需要 1-2 分钟，请稍候"
    
    ui_success "SSL 证书申请与反向代理配置成功！"
    ui_info "请稍候 1-2 分钟等待 Let's Encrypt 证书签发生效。"
    ui_info "访问地址: https://$(mask_domain "$DOMAIN")"
    
    # 发送通知
    send_webhook "NewAPI SSL 配置完成" \
        "服务器 $(hostname) 的域名 https://${DOMAIN} 已配置 SSL 证书"
else
    ui_error "NPM API 请求失败，HTTP 状态码: $HTTP_CODE"
    ui_error "返回信息: $RESP_BODY"
    
    # 给出修复建议
    ui_info "可能的原因："
    echo "  1. 域名 DNS 未解析到本机 IP"
    echo "  2. 端口 80/443 被占用"
    echo "  3. Lets Encrypt 请求频率限制"
    echo ""
    ui_info "修复建议："
    echo "  - 检查域名 DNS: dig $DOMAIN"
    # echo "  - 手动配置: 访问 http://$(hostname -I | awk '{print $1}'):81"
    echo "  - 手动配置: 访问 http://IP:81（将 IP 替换为实际服务器 IP）"
    echo "  - 查看 NPM 日志: docker logs npm"
    
    update_install_status "failed" "ssl_failed"
    exit 1
fi

# 询问是否配置更多域名（新手模式）
if_novice && {
    if ask_yn "是否配置更多域名？" "n"; then
        bash "${TOOLKIT_ROOT}/modules/deploy/ssl-proxy.sh"
    fi
}
