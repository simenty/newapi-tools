#!/bin/bash
set -eo pipefail
# 智能默认值模块 —— 自动生成配置，减少用户输入
# 自动生成安全密码、检测系统信息、推荐配置

# ---------- 生成安全随机密码 ----------
generate_password() {
    local length="${1:-16}"
    local password=""
    # 使用更安全的字符集（避免 shell 特殊字符导致问题）
    local charset='A-Za-z0-9!@#$%^&*()_+-='
    
    # 优先使用 openssl（更可靠的长度控制）
    if command -v openssl &>/dev/null; then
        # 使用 /dev/urandom 的完整字节生成，再映射到字符集
        local bytes_needed=$(( (length * 3 + 3) / 4 ))  # 确保足够随机性
        password=$(openssl rand -bytes "$bytes_needed" 2>/dev/null | od -An -tx1 | tr -d ' \n' | fold -w1 | while read -r c; do
            # 将每个字节映射到字符集索引
            printf '%s' "${charset:$(( 0x$c % ${#charset} )):1}"
        done | head -c "$length")
    # 降级到 /dev/urandom
    elif [[ -e /dev/urandom ]]; then
        password=$(head -c "$length" /dev/urandom | od -An -tx1 | tr -d ' \n' | fold -w1 | while read -r c; do
            printf '%s' "${charset:$(( 0x$c % ${#charset} )):1}"
        done | head -c "$length")
    # 最后降级到简单随机
    else
        # 使用多个熵源组合
        local entropy=""
        entropy+=$(date +%s%N)
        entropy+=$(cat /proc/sys/kernel/random/uuid 2>/dev/null || echo "$RANDOM$RANDOM$RANDOM")
        password=$(echo "$entropy" | sha512sum | base64 | tr -dc "$charset" | head -c "$length")
    fi
    
    # 确保输出不为空且长度正确（兜底）
    [[ -z "$password" ]] && password=$(openssl rand -base64 24 2>/dev/null | tr -dc "$charset" | head -c "$length")
    [[ ${#password} -lt "$length" ]] && password+=$(openssl rand -base64 8 2>/dev/null | tr -dc "$charset" | head -c $(( length - ${#password} )))
    
    echo "$password"
}

# ---------- 检测系统信息 ----------
detect_system_info() {
    local info_file="${TOOLKIT_ROOT}/.system_info"
    
    # 如果已检测过，直接返回缓存
    if [[ -f "$info_file" ]]; then
        cat "$info_file"
        return 0
    fi
    
    # 检测信息
    local os=""
    local version=""
    local arch=""
    local cpu_cores=""
    local memory_mb=""
    local disk_gb=""
    local public_ip=""
    local private_ip=""
    
    # 操作系统
    if [[ -f /etc/os-release ]]; then
        os=$(grep PRETTY_NAME /etc/os-release | cut -d'"' -f2)
        os_family=$(get_os_family)
    elif [[ -f /etc/issue ]]; then
        os=$(cat /etc/issue | head -1)
        os_family="unknown"
    else
        os="Unknown"
        os_family="unknown"
    fi
    
    # 架构
    arch=$(uname -m)
    
    # CPU 核心数
    cpu_cores=$(nproc 2>/dev/null || echo "1")
    
    # 内存（MB）
    memory_mb=$(free -m 2>/dev/null | awk '/^Mem:/ {print $2}' || echo "0")
    
    # 磁盘（GB）
    disk_gb=$(df -BG / 2>/dev/null | awk 'NR==2 {print $2}' | tr -d 'G' || echo "0")
    
    # IP 地址
    private_ip=$(ip route get 1 2>/dev/null | awk '{print $NF; exit}' || hostname -I | awk '{print $1}')
    public_ip=$(curl -fsS --max-time 5 ifconfig.me 2>/dev/null || echo "未知")
    
    # 缓存结果
    cat > "$info_file" << EOF
OS=$os
OS_FAMILY=$os_family
ARCH=$arch
CPU_CORES=$cpu_cores
MEMORY_MB=$memory_mb
DISK_GB=$disk_gb
PRIVATE_IP=$private_ip
PUBLIC_IP=$public_ip
EOF
    
    cat "$info_file"
}

# ---------- 获取 OS 系列 ----------
# 兼容包装：委托给 os_adapter.sh 的 os_get_family()
# 原实现已迁移至 lib/os_adapter.sh，此处保留函数签名以兼容已有调用方
get_os_family() {
    os_get_family
}

# ---------- 推荐配置 ----------
recommend_config() {
    local memory_mb
    local cpu_cores
    
    memory_mb=$(free -m | awk '/^Mem:/ {print $2}')
    cpu_cores=$(nproc)
    
    # 根据内存推荐配置
    if [[ $memory_mb -ge 8192 ]]; then
        # 8GB+ 内存：高性能配置
        echo "deploy.mysql.innodb_buffer_pool_size=2G"
        echo "deploy.newapi.workers=4"
        echo "deploy.newapi.memory_limit=2g"
    elif [[ $memory_mb -ge 4096 ]]; then
        # 4-8GB 内存：平衡配置
        echo "deploy.mysql.innodb_buffer_pool_size=1G"
        echo "deploy.newapi.workers=2"
        echo "deploy.newapi.memory_limit=1g"
    else
        # <4GB 内存：保守配置
        echo "deploy.mysql.innodb_buffer_pool_size=512M"
        echo "deploy.newapi.workers=1"
        echo "deploy.newapi.memory_limit=512m"
        ui_warn "系统内存较低（${memory_mb}MB），建议使用更高配置的服务器以获得更好性能"
    fi
}

# ---------- 检测已安装的服务 ----------
detect_installed_services() {
    local services=()
    
    # 检测 Docker
    if command -v docker &>/dev/null && docker info &>/dev/null; then
        services+=("docker")
    fi
    
    # 检测 MySQL
    if command -v mysql &>/dev/null || systemctl is-active mysql &>/dev/null; then
        services+=("mysql")
    fi
    
    # 检测 Redis
    if command -v redis-cli &>/dev/null || systemctl is-active redis &>/dev/null; then
        services+=("redis")
    fi
    
    # 检测 Nginx
    if command -v nginx &>/dev/null || systemctl is-active nginx &>/dev/null; then
        services+=("nginx")
    fi
    
    # 检测 NewAPI（通过 Docker Compose）
    if [[ -f /home/new-api/docker-compose.yml ]]; then
        services+=("newapi")
    fi
    
    echo "${services[@]}"
}

# ---------- 检查端口占用 ----------
check_port_available() {
    local port="$1"
    
    if command -v netstat &>/dev/null; then
        if netstat -tln | grep -q ":$port "; then
            return 1  # 端口已占用
        fi
    elif command -v ss &>/dev/null; then
        if ss -tln | grep -q ":$port "; then
            return 1  # 端口已占用
        fi
    fi
    
    return 0  # 端口可用
}

# ---------- 推荐可用端口 ----------
recommend_port() {
    local default_port="$1"
    local port="$default_port"
    
    while ! check_port_available "$port"; do
        ui_warn "端口 $port 已被占用，建议使用其他端口"
        port=$((port + 1))
        
        # 防止无限循环
        if [[ $port -gt $((default_port + 100)) ]]; then
            ui_error "无法找到可用端口（从 $default_port 开始）"
            return 1
        fi
    done
    
    echo "$port"
}

# ---------- 生成所有默认密码 ----------
generate_all_passwords() {
    local config_dir="${TOOLKIT_ROOT}/config"
    local passwords_file="${config_dir}/.passwords"
    
    # 如果已生成过，询问是否重新生成
    if [[ -f "$passwords_file" ]]; then
        if ! ask_yn "密码已生成，是否重新生成？" "n"; then
            return 0
        fi
    fi
    
    ui_info "正在生成安全密码..."
    
    # 生成密码
    local mysql_root_pwd=$(generate_password 20)
    local mysql_pwd=$(generate_password 16)
    local redis_pwd=$(generate_password 16)
    local session_secret=$(generate_password 32)
    
    # 保存到配置文件
    set_config "deploy.mysql.root_password" "$mysql_root_pwd"
    set_config "deploy.mysql.password" "$mysql_pwd"
    set_config "deploy.redis.password" "$redis_pwd"
    set_config "deploy.newapi.session_secret" "$session_secret"
    
    # 保存到密码文件（限制权限）
    cat > "$passwords_file" << EOF
# 自动生成的密码 - 请妥善保管
# 生成时间: $(date '+%Y-%m-%d %H:%M:%S')

MYSQL_ROOT_PASSWORD=$mysql_root_pwd
MYSQL_PASSWORD=$mysql_pwd
REDIS_PASSWORD=$redis_pwd
SESSION_SECRET=$session_secret
EOF
    chmod 600 "$passwords_file"
    
    ui_success "密码已生成并保存到: $passwords_file"
    ui_warn "请妥善保管密码文件，不要泄露！"
    
    # 显示密码（可选）
    if ask_yn "是否显示生成的密码？" "n"; then
        echo ""
        echo "MySQL Root 密码: $mysql_root_pwd"
        echo "MySQL 用户密码: $mysql_pwd"
        echo "Redis 密码: $redis_pwd"
        echo "Session Secret: $session_secret"
        echo ""
    fi
}

# ---------- 智能检测并填充配置 ----------
auto_config() {
    ui_info "正在智能检测系统并生成推荐配置..."
    
    # 1. 检测系统信息
    detect_system_info > /dev/null
    
    # 2. 生成密码
    generate_all_passwords
    
    # 3. 推荐配置
    local recommended
    recommended=$(recommend_config)
    
    # 4. 检测已安装服务
    local installed
    installed=$(detect_installed_services)
    
    if [[ -n "$installed" ]]; then
        ui_info "检测到已安装的服务: $installed"
        
        # 如果 Docker 已安装，更新状态
        if [[ "$installed" =~ docker ]]; then
            set_state "system.docker_installed" "true"
        fi
        
        # 如果 NewAPI 已部署，更新状态
        if [[ "$installed" =~ newapi ]]; then
            set_state "newapi.installed" "true"
        fi
    fi
    
    # 5. 推荐端口
    local newapi_port
    newapi_port=$(recommend_port 3000)
    set_config "deploy.newapi.port" "$newapi_port"
    
    local npm_port
    npm_port=$(recommend_port 81)
    set_config "deploy.npm.port" "$npm_port"
    
    ui_success "智能配置完成！"
    
    # 显示配置摘要
    if ask_yn "是否查看生成的配置？" "y"; then
        show_config
    fi
}

# ---------- 导出函数 ----------
export -f generate_password detect_system_info
export -f get_os_family
export -f recommend_config detect_installed_services
export -f check_port_available recommend_port
export -f generate_all_passwords auto_config
