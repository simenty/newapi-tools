#!/bin/bash
# NewAPI 部署脚本 v2.2
# 支持 NewAPI (calciumion/new-api)
set -eo pipefail

# 加载依赖库
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
export TOOLKIT_ROOT="$BASE_DIR"

# shellcheck source=lib/_init.sh
source "${TOOLKIT_ROOT}/lib/_init.sh"

# 仅在直接执行时进行权限检查
if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    check_root
    require_docker
fi

# ---------- 回滚机制 ----------
INSTALL_STATUS=1
INSTALL_BACKUP_DIR=""

_create_install_backup() {
    if [[ -d "$NEWAPI_HOME" ]]; then
        INSTALL_BACKUP_DIR="${BACKUP_DIR:-${NEWAPI_HOME}/backups}/.install_backup_$(date +%Y%m%d_%H%M%S)"
        mkdir -p "$INSTALL_BACKUP_DIR"
        cp -a "$NEWAPI_HOME"/{.env,docker-compose.yml} "$INSTALL_BACKUP_DIR/" 2>/dev/null || true
        log_info "已创建安装前备份: $INSTALL_BACKUP_DIR"
    fi
}

_rollback_installation() {
    log_error "部署失败，开始回滚..."
    if [[ -n "$INSTALL_BACKUP_DIR" ]] && [[ -d "$INSTALL_BACKUP_DIR" ]]; then
        if [[ -d "$NEWAPI_HOME" ]]; then
            mv "$NEWAPI_HOME" "${NEWAPI_HOME}.failed.$(date +%s)" 2>/dev/null || true
        fi
        mkdir -p "$NEWAPI_HOME"
        cp -a "$INSTALL_BACKUP_DIR"/* "$NEWAPI_HOME/" 2>/dev/null || true
        log_info "已回滚到安装前状态"
    fi
    update_install_status "failed" "installation_rollback"
}

_on_install_exit() {
    local exit_code=$?
    if [[ "$exit_code" -ne 0 ]]; then
        _rollback_installation
    fi
    unset MYSQL_ROOT_PASSWORD MYSQL_PASSWORD REDIS_PASSWORD SESSION_SECRET DB_ROOT_PASSWORD DB_PASSWORD 2>/dev/null || true
}

trap _on_install_exit EXIT

_create_install_backup

log_info "=== 开始部署 NewAPI 全家桶 v2.2 ==="

# ---------- 显示新手提示 ----------
novice_prompt "即将部署 NewAPI 及其依赖服务（MySQL + Redis + NPM）。整个过程将自动生成安全密码，无需手动输入。"

# ---------- 检查部署状态 ----------
if is_step_completed "newapi_install"; then
    ui_warn "检测到已完成部署"
    if ! ask_yn "是否重新部署？（将覆盖现有配置）" "n"; then
        ui_info "取消部署"
        exit 0
    fi
    update_install_status "reinstall" "backup"
    ui_info "正在备份现有数据..."
    bash "${MODULES_DIR}/manage/newapi/backup.sh" --manual
else
    update_install_status "installing" "prepare"
fi

# ---------- 步骤 1：准备配置 ----------
novice_step 1 5 "准备配置文件"

# 使用智能默认值（如果未配置）
if [[ -z "$(get_config 'deploy.mysql.root_password')" ]]; then
    ui_info "正在生成智能配置..."
    generate_all_passwords
fi

# 读取配置
MYSQL_ROOT_PASSWORD=$(get_config "deploy.mysql.root_password")
MYSQL_PASSWORD=$(get_config "deploy.mysql.password")
REDIS_PASSWORD=$(get_config "deploy.redis.password")
SESSION_SECRET=$(get_config "deploy.newapi.session_secret")
NEWAPI_PORT=$(get_config "deploy.newapi.port" "3000")
NPM_PORT=$(get_config "deploy.npm.port" "81")

# 专家模式：允许修改配置
if_expert && {
    echo ""
    ui_info "当前配置："
    echo "  MySQL Root 密码: ${MYSQL_ROOT_PASSWORD:0:8}..."
    echo "  MySQL 用户密码: ${MYSQL_PASSWORD:0:8}..."
    echo "  Redis 密码: ${REDIS_PASSWORD:0:8}..."
    echo "  NewAPI 端口: $NEWAPI_PORT"
    echo "  NPM 端口: $NPM_PORT"
    echo ""
    
    if ask_yn "是否修改配置？" "n"; then
        read -r -p "NewAPI 端口 [${NEWAPI_PORT}]: " input_port
        NEWAPI_PORT=${input_port:-$NEWAPI_PORT}
        set_config "deploy.newapi.port" "$NEWAPI_PORT"
    fi
}

show_progress 1 5 "准备配置"
update_install_status "installing" "config_ready"

# ---------- 步骤 2：创建目录和 .env ----------
novice_step 2 5 "创建目录和配置文件"

mkdir -p "$NEWAPI_HOME"
cd "$NEWAPI_HOME" || { log_error "无法进入安装目录"; exit 1; }

# 生成 .env（权限 600）
cat > .env << EOF
# NewAPI 环境配置 - 自动生成
# 生成时间: $(date '+%Y-%m-%d %H:%M:%S')

DB_ROOT_PASSWORD=${MYSQL_ROOT_PASSWORD}
MYSQL_PASSWORD=${MYSQL_PASSWORD}
REDIS_PASSWORD=${REDIS_PASSWORD}
SESSION_SECRET=${SESSION_SECRET}

# 数据库连接
SQL_DSN=root:${MYSQL_ROOT_PASSWORD}@tcp(db:3306)/newapi?charset=utf8mb4&parseTime=True&loc=Local
REDIS_CONN_STRING=redis://:${REDIS_PASSWORD}@redis:6379/0
EOF
chmod 600 .env

# 验证 .env 权限
ENV_PERM=$(stat -c %a .env 2>/dev/null || stat -f %Lp .env 2>/dev/null || echo "未知")
if [[ "$ENV_PERM" != "600" && "$ENV_PERM" != "Unknown" ]]; then
    log_warn ".env 文件权限异常: $ENV_PERM，正在重新设置..."
    chmod 600 .env
fi
log_success ".env 文件已生成（权限 600）"

# 清除密码变量（安全）
unset MYSQL_ROOT_PASSWORD MYSQL_PASSWORD REDIS_PASSWORD SESSION_SECRET

show_progress 2 5 "创建配置文件"
update_install_status "installing" "env_created"
mark_step_completed "env_create"

# ---------- 步骤 3：生成 docker-compose.yml ----------
novice_step 3 5 "生成 Docker Compose 配置"

log_info "生成 docker-compose.yml..."

# 从配置读取端口
NEWAPI_PORT=$(get_config "deploy.newapi.port" "3000")
NPM_HTTP_PORT=$(get_config "deploy.npm.http_port" "80")
NPM_HTTPS_PORT=$(get_config "deploy.npm.https_port" "443")
NPM_ADMIN_PORT=$(get_config "deploy.npm.port" "81")

cat > docker-compose.yml << 'EOF'
# 注意：version 属性已过时，新版 docker compose 会自动忽略

services:
  new-api:
    image: calciumion/new-api:latest
    container_name: new-api
    restart: always
    command: --log-dir /app/logs
    ports:
      - "127.0.0.1:${NEWAPI_PORT}:3000"
    volumes:
      - ./data:/data
      - ./logs:/app/logs
    environment:
      - SQL_DSN=\${SQL_DSN}
      - REDIS_CONN_STRING=\${REDIS_CONN_STRING}
      - SESSION_SECRET=\${SESSION_SECRET}
      - TZ=Asia/Shanghai
    depends_on:
      - db
      - redis
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:3000/api/status"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 30s

  db:
    image: mysql:8.0
    container_name: mysql
    restart: always
    environment:
      - MYSQL_ROOT_PASSWORD=\${DB_ROOT_PASSWORD}
      - MYSQL_DATABASE=newapi
      - MYSQL_USER=newapi
      - MYSQL_PASSWORD=\${MYSQL_PASSWORD}
    volumes:
      - ./mysql:/var/lib/mysql
    command: --default-authentication-plugin=mysql_native_password

  redis:
    image: redis:7.2
    container_name: redis
    restart: always
    command: redis-server --appendonly yes --requirepass \${REDIS_PASSWORD}
    volumes:
      - ./redis:/data

  npm:
    image: jc21/nginx-proxy-manager:latest
    container_name: npm
    restart: always
    ports:
      - "${NPM_HTTP_PORT}:80"
      - "${NPM_HTTPS_PORT}:443"
      - "${NPM_ADMIN_PORT}:81"
    volumes:
      - ./npm/data:/data
      - ./npm/letsencrypt:/etc/letsencrypt
    depends_on:
      - new-api

networks:
  default:
    name: newapi-network
EOF

log_success "docker-compose.yml 已生成"

# 设置安全权限
chmod 600 docker-compose.yml
log_info "已设置 docker-compose.yml 权限: 600"

show_progress 3 5 "生成 Docker Compose 配置"
update_install_status "installing" "compose_ready"
mark_step_completed "compose_generate"

# ---------- 步骤 4：拉取镜像并启动容器 ----------
novice_step 4 5 "拉取镜像并启动容器"

# 镜像预检：避免重复拉取
log_info "检查本地镜像缓存..."
images_need_pull=0
missing_images=()

# 从 docker-compose.yml 提取镜像名并检查
if [[ -f docker-compose.yml ]]; then
    while IFS= read -r line; do
        if [[ "$line" =~ ^[[:space:]]*image:[[:space:]]*(.+)$ ]]; then
            img="${BASH_REMATCH[1]}"
            if ! docker image inspect "$img" &>/dev/null; then
                images_need_pull=1
                missing_images+=("$img")
            fi
        fi
    done < docker-compose.yml
else
    # 兜底：检查默认镜像
    for img in calciumion/new-api:latest mysql:8.0 redis:7.2 jc21/nginx-proxy-manager:latest; do
        if ! docker image inspect "$img" &>/dev/null; then
            images_need_pull=1
            missing_images+=("$img")
        fi
    done
fi

if [[ $images_need_pull -eq 0 ]]; then
    log_info "所有镜像已存在，跳过拉取"
    if ask_yn "是否重新拉取镜像（获取更新）？" "n"; then
        log_info "拉取 Docker 镜像（获取更新）..."
        $DOCKER_COMPOSE_CMD pull --quiet 2>&1 | while read -r line; do
            echo "  $line"
        done
    else
        log_info "使用本地缓存镜像"
    fi
else
    log_info "缺失镜像（${missing_images[*]}），正在拉取..."
    $DOCKER_COMPOSE_CMD pull --quiet 2>&1 | while read -r line; do
        echo "  $line"
    done
fi

# 启动容器
log_info "启动容器..."
$DOCKER_COMPOSE_CMD up -d

show_progress 4 5 "启动容器"
update_install_status "installing" "containers_started"

# ---------- 步骤 5：等待服务初始化 ----------
novice_step 5 5 "等待服务初始化"

log_info "等待服务初始化（约 20 秒）..."
if_novice && {
    for i in {1..20}; do
        sleep 1
        show_progress "$i" 20 "初始化中"
    done
    echo ""
} || {
    sleep 20
}

# 检查服务健康状态
log_info "检查服务健康状态..."
FAILED_SERVICES=()
while IFS= read -r line; do
    SERVICE=$(echo "$line" | awk '{print $1}')
    STATUS=$(echo "$line" | awk '{print $NF}')
    
    if [[ "$STATUS" != "Up" && "$STATUS" != "healthy" ]]; then
        FAILED_SERVICES+=("$SERVICE")
    fi
done < <($DOCKER_COMPOSE_CMD ps --format "{{.Name}} {{.Status}}" 2>/dev/null || echo "")

if [[ ${#FAILED_SERVICES[@]} -gt 0 ]]; then
    ui_error "以下服务启动失败: ${FAILED_SERVICES[*]}"
    ui_info "查看日志: docker compose -f ${NEWAPI_HOME}/docker-compose.yml logs [服务名]"
    update_install_status "failed" "container_error"
    exit 1
fi

log_success "所有服务已启动"

show_progress 5 5 "部署完成"
update_install_status "completed" "all_done"
mark_step_completed "newapi_install"
set_state "newapi.installed" "true"
set_state "newapi.version" "latest"
set_state "newapi.compose_file" "$NEWAPI_HOME/docker-compose.yml"
set_state "newapi.env_file" "$NEWAPI_HOME/.env"

# ---------- 输出结果 ----------
echo ""
ui_success "=== NewAPI 部署完成 ==="
echo ""
echo -e "${UI_BOLD}访问地址：${UI_PLAIN}"
echo "  NewAPI 地址   : http://127.0.0.1:${NEWAPI_PORT} （仅本机访问，需通过 NPM 反代暴露）"
echo "  NPM 管理地址  : http://$(hostname -I | awk '{print $1}'):${NPM_ADMIN_PORT}"
echo ""
echo -e "${UI_BOLD}NPM 默认账号：${UI_PLAIN}"
echo "  邮箱: admin@example.com"
echo "  密码: changeme"
echo ""
ui_warn "安全提醒：请立即登录 NPM 修改默认密码！"
echo ""
echo -e "${UI_BOLD}下一步：${UI_PLAIN}"
echo "  1. 登录 NPM 修改默认密码"
echo "  2. 在菜单中选择『3) 配置 SSL 与反向代理』来配置域名访问"
echo "  3. 访问 NewAPI 完成初始化配置"
echo ""

# 显示部署摘要
show_summary "部署摘要" \
    "✓ MySQL 已部署并初始化" \
    "✓ Redis 已部署并配置密码" \
    "✓ NewAPI 已启动（端口 ${NEWAPI_PORT}）" \
    "✓ NPM 已部署（管理端口 ${NPM_ADMIN_PORT}）" \
    "✓ 所有服务健康状态正常"

# 发送 Webhook 通知
send_webhook "NewAPI 部署成功" "服务器 $(hostname) 的 NewAPI 已成功部署并启动"

# ---------- 部署成功：清理资源 ----------
INSTALL_STATUS=0  # 标记成功，避免 trap 触发回滚
trap - EXIT

# 安全清理敏感变量
unset MYSQL_ROOT_PASSWORD MYSQL_PASSWORD REDIS_PASSWORD SESSION_SECRET DB_ROOT_PASSWORD DB_PASSWORD 2>/dev/null || true

# 清理安装前备份
[[ -n "$INSTALL_BACKUP_DIR" ]] && [[ -d "$INSTALL_BACKUP_DIR" ]] && rm -rf "$INSTALL_BACKUP_DIR" 2>/dev/null || true
