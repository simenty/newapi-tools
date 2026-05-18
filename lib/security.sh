#!/bin/bash
# security.sh - NewAPI Tools V2.0 安全工具库
# 提供密码管理、文件安全、输入验证、下载校验、日志脱敏等功能
# 版本: 2.0.0
# 创建时间: 2026-05-15

set -eo pipefail

# 避免重复加载
if [ "${_SECURITY_SH_LOADED:-0}" -eq 1 ]; then
    return 0 2>/dev/null || exit 0
fi
export _SECURITY_SH_LOADED=1

# ============================================================================
# 密码管理函数
# ============================================================================

# 安全密码输入（无回显）
# 用法: secure_password_input "提示信息"
# 返回: 密码（通过 stdout 输出）
secure_password_input() {
    local prompt="${1:-请输入密码: }"
    local password
    
    # 检测是否在交互式终端
    if [ -t 0 ]; then
        read -rsp "$prompt" password
        echo "" >&2  # 输出换行到 stderr
    else
        # 非交互式环境，从 stdin 读取
        IFS= read -r password
    fi
    
    echo "$password"
}

# 密码强度验证
# 用法: validate_password_strength "密码"
# 返回: 0=强密码, 1=弱密码
validate_password_strength() {
    local password="$1"
    
    # 长度至少 8 位
    if [ ${#password} -lt 8 ]; then
        log_warn "密码长度至少 8 位"
        return 1
    fi
    
    # 包含大写字母
    if ! echo "$password" | grep -q '[A-Z]'; then
        log_warn "密码必须包含大写字母"
        return 1
    fi
    
    # 包含小写字母
    if ! echo "$password" | grep -q '[a-z]'; then
        log_warn "密码必须包含小写字母"
        return 1
    fi
    
    # 包含数字
    if ! echo "$password" | grep -q '[0-9]'; then
        log_warn "密码必须包含数字"
        return 1
    fi
    
    # 包含特殊字符
    if ! echo "$password" | grep -q '[!@#$%^&*(),.?":{}|<>]'; then
        log_warn "密码必须包含特殊字符"
        return 1
    fi
    
    log_info "密码强度验证通过"
    return 0
}

# 清理敏感环境变量
# 用法: clear_sensitive_vars VAR1 VAR2 ...
clear_sensitive_vars() {
    for var in "$@"; do
        unset "$var" 2>/dev/null || true
        log_debug "已清理敏感变量: $var"
    done
}

# ============================================================================
# 文件安全函数
# ============================================================================

# 创建安全权限文件
# 用法: secure_file_create "文件路径" "权限模式" "所有者"
# 示例: secure_file_create "/path/to/file" "600" "app:app"
secure_file_create() {
    local file_path="$1"
    local mode="${2:-600}"
    local owner="${3:-}"
    
    # 创建目录（如果不存在）
    local dir_path
    dir_path=$(dirname "$file_path")
    if [ ! -d "$dir_path" ]; then
        mkdir -p "$dir_path" || {
            log_error "无法创建目录: $dir_path"
            return 1
        }
    fi
    
    # 创建文件
    touch "$file_path" || {
        log_error "无法创建文件: $file_path"
        return 1
    }
    
    # 设置权限
    chmod "$mode" "$file_path" || {
        log_error "无法设置文件权限: $file_path ($mode)"
        return 1
    }
    
    # 设置所有者（如果指定）
    if [ -n "$owner" ]; then
        chown "$owner" "$file_path" || {
            log_warn "无法设置文件所有者: $file_path ($owner)"
        }
    fi
    
    log_debug "安全文件创建成功: $file_path (mode=$mode)"
    return 0
}

# 安全删除文件（防止空变量）
# 用法: secure_file_delete "文件路径"
secure_file_delete() {
    local file_path="$1"
    
    # 检查变量是否为空
    if [ -z "$file_path" ]; then
        log_error "secure_file_delete: 文件路径为空"
        return 1
    fi
    
    # 检查文件是否存在
    if [ ! -f "$file_path" ] && [ ! -L "$file_path" ]; then
        log_warn "文件不存在: $file_path"
        return 0
    fi
    
    # 安全删除（覆盖后删除）
    if [ -f "$file_path" ]; then
        # 用随机数据覆盖文件内容
        local file_size
        file_size=$(stat -c%s "$file_path" 2>/dev/null || stat -f%z "$file_path" 2>/dev/null || echo "0")
        
        if [ "$file_size" -gt 0 ] 2>/dev/null; then
            # 用随机数据覆盖 3 次
            for i in 1 2 3; do
                dd if=/dev/urandom of="$file_path" bs=1k count=$((file_size / 1024 + 1)) 2>/dev/null || true
            done
        fi
    fi
    
    # 删除文件
    rm -f "$file_path" || {
        log_error "无法删除文件: $file_path"
        return 1
    }
    
    log_debug "安全删除文件: $file_path"
    return 0
}

# 创建安全临时文件
# 用法: secure_temp_file "前缀"
# 返回: 临时文件路径（通过 stdout 输出）
secure_temp_file() {
    local prefix="${1:-newapi}"
    local temp_file
    
    # 使用 mktemp 创建临时文件
    temp_file=$(mktemp "/tmp/${prefix}.XXXXXX") || {
        log_error "无法创建临时文件"
        return 1
    }
    
    # 设置安全权限
    chmod 600 "$temp_file" || {
        log_error "无法设置临时文件权限: $temp_file"
        rm -f "$temp_file"
        return 1
    }
    
    log_debug "创建安全临时文件: $temp_file"
    echo "$temp_file"
}

# 设置敏感文件权限（600）
# 用法: secure_chmod_sensitive "文件路径"
secure_chmod_sensitive() {
    local file_path="$1"
    
    if [ ! -f "$file_path" ]; then
        log_error "文件不存在: $file_path"
        return 1
    fi
    
    chmod 600 "$file_path" || {
        log_error "无法设置敏感文件权限: $file_path"
        return 1
    }
    
    log_debug "设置敏感文件权限: $file_path (600)"
    return 0
}

# ============================================================================
# 输入验证函数
# ============================================================================

# 通用输入验证
# 用法: validate_input "输入值" "正则表达式模式" "字段名称"
# 返回: 0=验证通过, 1=验证失败
validate_input() {
    local input="$1"
    local pattern="$2"
    local field_name="${3:-输入}"
    
    if [ -z "$input" ]; then
        log_error "$field_name 不能为空"
        return 1
    fi
    
    if ! echo "$input" | grep -qE "$pattern"; then
        log_error "$field_name 格式不正确"
        return 1
    fi
    
    log_debug "$field_name 验证通过: $input"
    return 0
}

# 域名验证
# 用法: validate_domain "域名"
validate_domain() {
    local domain="$1"
    local domain_pattern='^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$'
    
    validate_input "$domain" "$domain_pattern" "域名"
}

# 路径验证
# 用法: validate_path "路径" "是否允许绝对路径"
validate_path() {
    local path="$1"
    local allow_absolute="${2:-yes}"
    
    # 检查路径遍历攻击
    if echo "$path" | grep -q '\.\.'; then
        log_error "路径包含非法字符: $path"
        return 1
    fi
    
    # 检查是否允许绝对路径
    if [ "$allow_absolute" != "yes" ] && echo "$path" | grep -q '^/'; then
        log_error "不允许使用绝对路径: $path"
        return 1
    fi
    
    # 检查路径中的特殊字符
    if echo "$path" | grep -q '[;`$(){}|&<>]'; then
        log_error "路径包含非法字符: $path"
        return 1
    fi
    
    log_debug "路径验证通过: $path"
    return 0
}

# sed 特殊字符转义
# 用法: escape_sed_pattern "字符串"
# 返回: 转义后的字符串（通过 stdout 输出）
escape_sed_pattern() {
    local pattern="$1"
    
    # 转义 sed 特殊字符: / & \ 
    echo "$pattern" | sed 's/[\/&]/\\&/g'
}

# Shell 参数转义
# 用法: escape_shell_argument "字符串"
# 返回: 转义后的字符串（通过 stdout 输出）
escape_shell_argument() {
    local arg="$1"
    
    # 使用单引号包裹，内部的单引号用 '\'' 替换
    printf "'%s'" "$(echo "$arg" | sed "s/'/'\\\\''/g")"
}

# ============================================================================
# 下载和校验函数
# ============================================================================

# 带校验的下载
# 用法: download_with_verify "URL" "输出文件" "期望的SHA256"
download_with_verify() {
    local url="$1"
    local output="$2"
    local expected_sha256="$3"
    local max_retries="${4:-3}"
    local retry_count=0
    
    # 强制 HTTPS
    url=$(force_https_url "$url")
    
    while [ $retry_count -lt $max_retries ]; do
        log_info "下载文件 (尝试 $((retry_count + 1))/$max_retries): $url"
        
        # 使用 curl 下载（启用 SSL 验证）
        if curl --proto '=https' \
                --tlsv1.2 \
                --cacert /etc/ssl/certs/ca-certificates.crt \
                -fsSL "$url" -o "$output" 2>/dev/null; then
            
            # 如果指定了 SHA256，进行校验
            if [ -n "$expected_sha256" ]; then
                if verify_checksum "$output" "$expected_sha256"; then
                    log_info "下载并校验成功: $url"
                    return 0
                else
                    log_error "校验失败，删除文件: $output"
                    rm -f "$output"
                    return 1
                fi
            else
                log_info "下载成功（未校验）: $url"
                return 0
            fi
        else
            retry_count=$((retry_count + 1))
            log_warn "下载失败，重试 $retry_count/$max_retries: $url"
            sleep 2
        fi
    done
    
    log_error "下载失败（已达最大重试次数）: $url"
    rm -f "$output"
    return 1
}

# SHA-256 校验
# 用法: verify_checksum "文件" "期望的SHA256"
# 返回: 0=校验通过, 1=校验失败
verify_checksum() {
    local file="$1"
    local expected_sha256="$2"
    local actual_sha256
    
    if [ ! -f "$file" ]; then
        log_error "文件不存在: $file"
        return 1
    fi
    
    # 计算实际 SHA256
    actual_sha256=$(sha256sum "$file" 2>/dev/null | awk '{print $1}') || {
        log_error "无法计算文件校验和: $file"
        return 1
    }
    
    # 比对
    if [ "$expected_sha256" = "$actual_sha256" ]; then
        log_info "SHA-256 校验通过: $file"
        return 0
    else
        log_error "SHA-256 校验失败: $file"
        log_error "期望: $expected_sha256"
        log_error "实际: $actual_sha256"
        return 1
    fi
}

# GPG 签名验证
# 用法: verify_gpg_signature "文件" "签名URL"
# 返回: 0=验证通过, 1=验证失败
verify_gpg_signature() {
    local file="$1"
    local signature_url="$2"
    local temp_sig
    
    if [ ! -f "$file" ]; then
        log_error "文件不存在: $file"
        return 1
    fi
    
    # 创建临时文件存储签名
    temp_sig=$(secure_temp_file "sig") || return 1
    
    # 下载签名文件
    log_info "下载 GPG 签名: $signature_url"
    if ! curl -fsSL "$signature_url" -o "$temp_sig" 2>/dev/null; then
        log_error "无法下载 GPG 签名: $signature_url"
        rm -f "$temp_sig"
        return 1
    fi
    
    # 验证签名
    log_info "验证 GPG 签名: $file"
    if gpg --verify "$temp_sig" "$file" 2>/dev/null; then
        log_info "GPG 签名验证通过: $file"
        rm -f "$temp_sig"
        return 0
    else
        log_error "GPG 签名验证失败: $file"
        rm -f "$temp_sig"
        return 1
    fi
}

# 强制 HTTPS URL
# 用法: force_https_url "URL"
# 返回: HTTPS URL（通过 stdout 输出）
force_https_url() {
    local url="$1"
    
    # 如果是 HTTP，替换为 HTTPS
    if echo "$url" | grep -q '^http://'; then
        url=$(echo "$url" | sed 's/^http:/https:/')
        log_warn "已强制转换为 HTTPS: $url"
    fi
    
    # 如果不是 HTTP/HTTPS，添加 HTTPS
    if ! echo "$url" | grep -q '^https://'; then
        url="https://$url"
        log_warn "已添加 HTTPS 协议: $url"
    fi
    
    echo "$url"
}

# ============================================================================
# 日志脱敏函数
# ============================================================================

# 过滤敏感信息
# 用法: filter_sensitive_info "文本"
# 返回: 过滤后的文本（通过 stdout 输出）
filter_sensitive_info() {
    local text="$1"
    
    # 过滤密码
    text=$(echo "$text" | sed -E 's/password[=:][^[:space:]]+/password=***/gi')
    text=$(echo "$text" | sed -E 's/passwd[=:][^[:space:]]+/passwd=***/gi')
    
    # 过滤 token
    text=$(echo "$text" | sed -E 's/token[=:][^[:space:]]+/token=***/gi')
    text=$(echo "$text" | sed -E 's/access_token[=:][^[:space:]]+/access_token=***/gi')
    
    # 过滤 API key
    text=$(echo "$text" | sed -E 's/api_key[=:][^[:space:]]+/api_key=***/gi')
    text=$(echo "$text" | sed -E 's/apikey[=:][^[:space:]]+/apikey=***/gi')
    
    # 过滤 secret
    text=$(echo "$text" | sed -E 's/secret[=:][^[:space:]]+/secret=***/gi')
    text=$(echo "$text" | sed -E 's/client_secret[=:][^[:space:]]+/client_secret=***/gi')
    
    # 过滤数据库密码
    text=$(echo "$text" | sed -E 's/MYSQL_PWD[=:][^[:space:]]+/MYSQL_PWD=***/gi')
    text=$(echo "$text" | sed -E 's/SQL_DSN[=:][^[:space:]]+/SQL_DSN=***/gi')
    text=$(echo "$text" | sed -E 's/REDIS_CONN[=:][^[:space:]]+/REDIS_CONN=***/gi')
    
    # 过滤 authorization
    text=$(echo "$text" | sed -E 's/authorization[=:][^[:space:]]+/authorization=***/gi')
    text=$(echo "$text" | sed -E 's/Bearer[[:space:]]+[^[:space:]]+/Bearer ***/gi')
    
    # 过滤密钥（通用模式）
    text=$(echo "$text" | sed -E 's/(key|k)[=:][^[:space:]]{16,}/key=***/gi')
    
    # 过滤 API key 模式（如 sk-..., pk-...）
    text=$(echo "$text" | sed -E 's/(sk-|pk-)[a-zA-Z0-9_]{20,}/***/g')
    
    # 过滤 GitHub token 模式（如 ghp_..., github_pat_...）
    text=$(echo "$text" | sed -E 's/(ghp_|github_pat_)[a-zA-Z0-9_]{20,}/***/g')
    
    # 过滤私钥模式
    text=$(echo "$text" | sed -E '/-----BEGIN.*PRIVATE KEY-----/,/-----END.*PRIVATE KEY-----/d')
    
    # 过滤身份证号模式（18 位）
    text=$(echo "$text" | sed -E 's/[1-9][0-9]{5}(19|20)[0-9]{2}(0[1-9]|1[0-2])(0[1-9]|[12][0-9]|3[01])[0-9]{3}[0-9Xx]/***/g')
    
    # 过滤手机号模式（11 位）
    text=$(echo "$text" | sed -E 's/1[3-9][0-9]{9}/***/g')
    
    echo "$text"
}

# 安全日志记录（脱敏后）
# 用法: log_secure "级别" "消息"
log_secure() {
    local level="$1"
    local message="$2"
    local filtered_message
    
    # 脱敏处理
    filtered_message=$(filter_sensitive_info "$message")
    
    # 根据级别调用相应的日志函数
    case "$level" in
        DEBUG)
            log_debug "$filtered_message"
            ;;
        INFO)
            log_info "$filtered_message"
            ;;
        WARN)
            log_warn "$filtered_message"
            ;;
        ERROR)
            log_error "$filtered_message"
            ;;
        *)
            log_info "$filtered_message"
            ;;
    esac
}

# ============================================================================
# 初始化和自检
# ============================================================================

# 自检函数（用于测试）
security_self_test() {
    log_info "开始安全模块自检..."
    
    # 测试密码强度验证
    if validate_password_strength "Abc@123456"; then
        log_info "✅ 密码强度验证测试通过"
    else
        log_error "❌ 密码强度验证测试失败"
    fi
    
    # 测试临时文件创建
    local test_temp
    test_temp=$(secure_temp_file "selftest")
    if [ -f "$test_temp" ]; then
        log_info "✅ 临时文件创建测试通过"
        rm -f "$test_temp"
    else
        log_error "❌ 临时文件创建测试失败"
    fi
    
    # 测试 sed 转义
    local test_escape
    test_escape=$(escape_sed_pattern "hello/world&test")
    if [ "$test_escape" = "hello\/world\&test" ]; then
        log_info "✅ sed 转义测试通过"
    else
        log_error "❌ sed 转义测试失败: $test_escape"
    fi
    
    # 测试日志脱敏
    local test_filter
    test_filter=$(filter_sensitive_info "password=secret123 token=abc888")
    if ! echo "$test_filter" | grep -q "secret123\|abc888"; then
        log_info "✅ 日志脱敏测试通过"
    else
        log_error "❌ 日志脱敏测试失败"
    fi
    
    log_info "安全模块自检完成"
}

log_debug "安全工具库加载完成: lib/security.sh"
return 0 2>/dev/null || true
