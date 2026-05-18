# NewAPI Tools v2.0 多平台支持方案

> **文档版本**: v1.0  
> **日期**: 2026-05-13  
> **作者**: Vincent  
> **目标**: 从"NewAPI 专用工具"升级为"通用 API 中转站运维平台"

---

## 一、背景与动机

### 1.1 当前局限

当前 v1.2 工具**仅支持 NewAPI**，但市面上存在多个 API 中转站平台：

| 平台 | Docker 镜像 | 特点 | 用户群 |
|------|------------|------|--------|
| **NewAPI** | `calciumion/new-api` | 功能最全、更新活跃 | 主力推荐 |
| **One API** | `justsong/one-api` | 老牌稳定、生态成熟 | 老用户多 |
| **New API Horizon** | `calcium-ion/new-api-horizon` | NewAPI 分支、特定功能 | 特定需求 |
| **Sub2API** | 未知 | 订阅管理强化 | 特定场景 |

**问题：**
- 用户想从 One API 迁移到 NewAPI，但工具不支持 One API
- 用户想同时管理多个平台（测试用 NewAPI，生产用 One API）
- 工具代码里硬编码了 NewAPI 特定逻辑（镜像名、环境变量等）

### 1.2 多平台支持的价值

✅ **更大的用户群**：覆盖所有 API 中转站用户  
✅ **迁移工具**：帮助用户从 One API 迁移到 NewAPI  
✅ **统一管理**：一个工具管理多个平台  
✅ **生态整合**：成为"API 中转站运维标准工具"  

---

## 二、架构设计：平台抽象层

### 2.1 核心思路

引入**平台抽象层（Platform Abstraction Layer）**，把平台特定逻辑从主代码中抽离。

#### 当前架构（硬编码）
```bash
# modules/deploy/install.sh (v1.2)
# 硬编码了 NewAPI 的镜像名、环境变量等
docker run -d \
  --name new-api \
  -p 3000:3000 \
  -e SQL_DSN=... \
  calciumion/new-api:latest  # ← 硬编码
```

#### 目标架构（可插拔）
```bash
# modules/deploy/install.sh (v2.0)
# 从平台定义文件读取配置
source "${PLATFORMS_DIR}/${PLATFORM}/platform.def"

docker run -d \
  --name ${CONTAINER_NAME} \
  -p ${PORT}:${PORT} \
  -e ${ENV_SQL_DSN}=... \
  ${DOCKER_IMAGE}:${VERSION}
```

---

### 2.2 平台定义文件（Platform Definition）

每个平台有一个 `platform.def` 文件，定义该平台的所有特性。

#### 文件结构
```
platforms/
├── newapi/
│   ├── platform.def       # 平台定义
│   ├── docker-compose.yml # 默认编排文件
│   ├── install.sh        # 平台特定安装逻辑（可选）
│   └── migrate-from.sh  # 从其他平台迁移的脚本（可选）
├── one-api/
│   ├── platform.def
│   ├── docker-compose.yml
│   └── migrate-to-newapi.sh  # 迁移到 NewAPI
├── new-api-horizon/
│   ├── platform.def
│   └── docker-compose.yml
└── sub2api/
    ├── platform.def
    └── docker-compose.yml
```

#### `platform.def` 示例（NewAPI）
```bash
# platforms/newapi/platform.def

# 平台基本信息
PLATFORM_NAME="NewAPI"
PLATFORM_ID="newapi"
PLATFORM_DESC="NewAPI - 功能最全的 API 中转站"
PLATFORM_REPO="https://github.com/QuantumNous/new-api"

# Docker 镜像
DOCKER_IMAGE="calciumion/new-api"
DEFAULT_VERSION="latest"
SUPPORTED_ARCHS=("amd64" "arm64")

# 容器配置
CONTAINER_NAME="new-api"
DEFAULT_PORT=3000
PORT_BINDING="127.0.0.1:3000:3000"

# 环境变量（必需）
REQUIRED_ENVS=(
  "SQL_DSN"
  "REDIS_CONN_STRING"
  "SESSION_SECRET"
)

# 环境变量（可选）
OPTIONAL_ENVS=(
  "TZ"
  "DEBUG"
)

# 依赖服务
DEPENDENCIES=("mysql" "redis")

# 健康检查和初始化的特定逻辑
HEALTH_CHECK_CMD="curl -f http://localhost:3000/api/status"
INIT_SETUP_REQUIRED=true  # 首次安装需要初始化账号

# 数据库配置
DB_NAME="newapi"
DB_USER="root"

# 备份特定逻辑
BACKUP_DATABASE=true
BACKUP_DATA_DIRS=("data" "logs")

# 平台特定安装函数（可选，覆盖默认行为）
# platform_pre_install() { ... }
# platform_post_install() { ... }
```

#### `platform.def` 示例（One API）
```bash
# platforms/one-api/platform.def

PLATFORM_NAME="One API"
PLATFORM_ID="one-api"
PLATFORM_DESC="One API - 老牌稳定的 API 中转站"
PLATFORM_REPO="https://github.com/songquanpeng/one-api"

DOCKER_IMAGE="justsong/one-api"
DEFAULT_VERSION="latest"

CONTAINER_NAME="one-api"
DEFAULT_PORT=3000

REQUIRED_ENVS=(
  "SQL_DSN"
  "SESSION_SECRET"
)

DEPENDENCIES=("mysql" "redis")

HEALTH_CHECK_CMD="curl -f http://localhost:3000/api/status"
INIT_SETUP_REQUIRED=true

DB_NAME="oneapi"  # ← 注意：数据库名不同
DB_USER="root"

BACKUP_DATABASE=true
BACKUP_DATA_DIRS=("data")

# One API 特定：从 NewAPI 迁移
platform_migrate_from_newapi() {
  log_info "正在从 NewAPI 迁移到 One API..."
  # 转换数据库结构
  # ...
}
```

---

### 2.3 核心实现

#### **步骤 1：创建平台管理库 `lib/platform.sh`**

```bash
#!/bin/bash
# 平台管理库 —— 负责加载和管理平台定义

PLATFORMS_DIR="${TOOLKIT_ROOT}/platforms"

# 列出所有支持的平台
list_platforms() {
  local platforms=()
  if [[ -d "$PLATFORMS_DIR" ]]; then
    for dir in "$PLATFORMS_DIR"/*/; do
      if [[ -f "${dir}platform.def" ]]; then
        local platform_id=$(basename "$dir")
        local platform_name=$(grep "^PLATFORM_NAME=" "${dir}platform.def" | cut -d'"' -f2)
        platforms+=("$platform_id" "$platform_name")
      fi
    done
  fi
  
  # 如果没有安装任何平台定义，显示内置支持
  if [[ ${#platforms[@]} -eq 0 ]]; then
    platforms=(
      "newapi" "NewAPI (内置)"
      "one-api" "One API (内置)"
    )
  fi
  
  printf '%s\n' "${platforms[@]}"
}

# 加载平台定义
load_platform() {
  local platform_id="$1"
  local platform_def="${PLATFORMS_DIR}/${platform_id}/platform.def"
  
  # 内置平台（向后兼容）
  if [[ "$platform_id" == "newapi" && ! -f "$platform_def" ]]; then
    log_info "使用内置 NewAPI 定义"
    load_builtin_newapi_def
    return 0
  fi
  
  if [[ ! -f "$platform_def" ]]; then
    log_error "不支持的平台: $platform_id"
    log_info "支持的平台: $(list_platforms | tr '\n' ' ')"
    return 1
  fi
  
  source "$platform_def"
  log_info "已加载平台定义: $PLATFORM_NAME"
  
  # 导出变量（供其他脚本使用）
  export PLATFORM_NAME PLATFORM_ID PLATFORM_DESC
  export DOCKER_IMAGE DEFAULT_VERSION CONTAINER_NAME DEFAULT_PORT
  export REQUIRED_ENVS DEPENDENCIES
}

# 内置 NewAPI 定义（向后兼容）
load_builtin_newapi_def() {
  PLATFORM_NAME="NewAPI"
  PLATFORM_ID="newapi"
  PLATFORM_DESC="NewAPI - 功能最全的 API 中转站"
  
  DOCKER_IMAGE="calciumion/new-api"
  DEFAULT_VERSION="latest"
  CONTAINER_NAME="new-api"
  DEFAULT_PORT=3000
  
  REQUIRED_ENVS=("SQL_DSN" "REDIS_CONN_STRING" "SESSION_SECRET")
  DEPENDENCIES=("mysql" "redis")
  
  export PLATFORM_NAME PLATFORM_ID PLATFORM_DESC
  export DOCKER_IMAGE DEFAULT_VERSION CONTAINER_NAME DEFAULT_PORT
  export REQUIRED_ENVS DEPENDENCIES
}

# 检测已安装的平台
detect_installed_platform() {
  if docker ps -a --format '{{.Names}}' | grep -q "new-api"; then
    echo "newapi"
  elif docker ps -a --format '{{.Names}}' | grep -q "one-api"; then
    echo "one-api"
  else
    echo "none"
  fi
}

# 生成 docker-compose.yml（通用函数）
generate_docker_compose() {
  local platform_id="${1:-$PLATFORM_ID}"
  load_platform "$platform_id"
  
  cat > docker-compose.yml << EOF
services:
  ${CONTAINER_NAME}:
    image: ${DOCKER_IMAGE}:${VERSION:-$DEFAULT_VERSION}
    container_name: ${CONTAINER_NAME}
    restart: always
    ports:
      - "${PORT_BINDING:-127.0.0.1:${DEFAULT_PORT}:${DEFAULT_PORT}}"
    environment:
$(for env in "${REQUIRED_ENVS[@]}"; do
  echo "      - ${env}=\${${env}}"
done)
    depends_on:
$(for dep in "${DEPENDENCIES[@]}"; do
  echo "      - ${dep}"
done)
EOF
  
  log_success "docker-compose.yml 已生成（平台: $PLATFORM_NAME）"
}
```

---

#### **步骤 2：修改主入口支持平台选择**

```bash
# newapi-tools.sh (v2.0)

# 在菜单前增加平台选择
select_platform() {
  show_banner
  
  local installed=$(detect_installed_platform)
  
  if [[ "$installed" != "none" ]]; then
    log_info "检测到已安装平台: $installed"
    load_platform "$installed"
    return 0
  fi
  
  # 未安装，让用户选择平台
  echo "请选择要部署的平台："
  echo ""
  
  local platforms=($(list_platforms))
  local i=1
  while [[ $i -lt ${#platforms[@]} ]]; do
    echo "  ${YELLOW}$((i/2+1))${PLAIN}) ${platforms[$i-1]} - ${platforms[$i]}"
    i=$((i+2))
  done
  
  echo ""
  read -r -p "请选择 [1-$((${#platforms[@]}/2))]: " choice
  
  local idx=$((choice*2-2))
  if [[ $idx -ge 0 && $idx -lt ${#platforms[@]} ]]; then
    PLATFORM_ID="${platforms[$idx]}"
    load_platform "$PLATFORM_ID"
    
    # 保存到状态文件
    set_state "platform" "$PLATFORM_ID"
  else
    log_error "无效选择"
    select_platform
  fi
}
```

---

#### **步骤 3：修改部署模块支持多平台**

```bash
# modules/deploy/install.sh (v2.0)

source "${TOOLKIT_ROOT}/lib/platform.sh"

check_root
require_docker

# 确保已选择平台
if [[ -z "${PLATFORM_ID:-}" ]]; then
  select_platform
fi

log_info "=== 开始部署 ${PLATFORM_NAME} ==="

# 收集平台特定的配置
collect_platform_config() {
  case "$PLATFORM_ID" in
    newapi)
      # NewAPI 特定配置
      read -s -r -p "Session Secret: " SESSION_SECRET
      ;;
    one-api)
      # One API 特定配置
      # ...
      ;;
  esac
}

# 生成 docker-compose.yml（调用通用函数）
generate_docker_compose "$PLATFORM_ID"

# 启动容器
$DOCKER_COMPOSE_CMD up -d

log_success "=== ${PLATFORM_NAME} 部署完成 ==="
```

---

## 三、迁移工具设计

### 3.1 场景：One API → NewAPI

很多用户想从 One API 迁移到 NewAPI（功能更多、更新更活跃）。

#### 实现思路

```bash
# modules/migrate/one-api-to-new-api.sh

migrate_one_api_to_new_api() {
  show_banner
  echo "One API → NewAPI 迁移工具"
  echo "=========================="
  echo ""
  
  ui_warn "迁移前会自动备份 One API 数据"
  ui_info "迁移过程："
  echo "  1. 备份 One API 数据库"
  echo "  2. 转换数据库结构（One API → NewAPI）"
  echo "  3. 部署 NewAPI"
  echo "  4. 导入转换后的数据"
  echo "  5. 验证数据完整性"
  echo ""
  
  if ! ask_yn "确定要迁移吗？"; then
    return 0
  fi
  
  # 步骤 1：备份 One API
  show_step 1 5 "备份 One API 数据"
  if ! docker ps -a --format '{{.Names}}' | grep -q "one-api"; then
    ui_error "未检测到 One API 容器，无法迁移"
    return 1
  fi
  
  local backup_file="${NEWAPI_HOME}/backups/one-api-backup-$(date +%s).sql"
  docker exec one-api mysqldump -uroot -p"$ONE_API_MYSQL_PASS" oneapi > "$backup_file"
  generate_checksum "$backup_file"
  log_success "One API 备份完成: $backup_file"
  
  # 步骤 2：转换数据库结构
  show_step 2 5 "转换数据库结构"
  log_info "NewAPI 和 One API 的数据库结构略有不同"
  log_info "正在生成转换脚本..."
  
  # 这里需要编写转换逻辑（简化示例）
  cat > /tmp/convert.sql << 'EOF'
-- 转换 One API 数据库到 NewAPI 格式
-- 示例：渠道表结构变化
ALTER TABLE channel ADD COLUMN IF NOT EXISTS `new_field` VARCHAR(255);
-- ...
EOF
  
  log_success "转换脚本已生成"
  
  # 步骤 3：部署 NewAPI
  show_step 3 5 "部署 NewAPI"
  PLATFORM_ID="newapi"
  source "${MODULES_DIR}/deploy/install.sh"
  
  # 步骤 4：导入数据
  show_step 4 5 "导入数据到 NewAPI"
  # 应用转换脚本
  docker exec -i new-api mysql -uroot -p"$MYSQL_PASS" newapi < /tmp/convert.sql
  
  # 导入备份数据
  docker exec -i new-api mysql -uroot -p"$MYSQL_PASS" newapi < "$backup_file"
  
  log_success "数据导入完成"
  
  # 步骤 5：验证
  show_step 5 5 "验证数据完整性"
  local count_one=$(docker exec one-api mysql -uroot -p"$ONE_API_MYSQL_PASS" -e "SELECT COUNT(*) FROM oneapi.channel" | tail -1)
  local count_new=$(docker exec new-api mysql -uroot -p"$MYSQL_PASS" -e "SELECT COUNT(*) FROM newapi.channel" | tail -1)
  
  if [[ "$count_one" == "$count_new" ]]; then
    ui_success "数据验证通过！渠道数: $count_new"
  else
    ui_warn "数据可能不完整，请手动检查"
    echo "  One API 渠道数: $count_one"
    echo "  NewAPI 渠道数: $count_new"
  fi
  
  echo ""
  ui_success "迁移完成！"
  ui_info "One API 容器仍在运行，可以对比验证"
  ui_info "确认无误后，可手动停止 One API: docker stop one-api"
}
```

---

### 3.2 通用迁移框架

```bash
# lib/migration.sh

# 注册迁移路径
declare -A MIGRATION_PATHS
MIGRATION_PATHS["one-api→newapi"]="modules/migrate/one-api-to-new-api.sh"
MIGRATION_PATHS["newapi→one-api"]="modules/migrate/new-api-to-one-api.sh"

# 显示可用的迁移路径
show_migration_paths() {
  echo "可用的迁移路径："
  for path in "${!MIGRATION_PATHS[@]}"; do
    echo "  - $path"
  done
}

# 执行迁移
run_migration() {
  local from="$1"
  local to="$2"
  local key="${from}→${to}"
  
  if [[ -z "${MIGRATION_PATHS[$key]:-}" ]]; then
    ui_error "不支持的迁移路径: $key"
    show_migration_paths
    return 1
  fi
  
  source "${MIGRATION_PATHS[$key]}"
}
```

---

## 四、用户体验优化

### 4.1 平台选择向导

```bash
# 首次运行时显示
show_platform_wizard() {
  show_banner
  echo "欢迎使用 NewAPI Tools v2.0"
  echo "=========================="
  echo ""
  echo "请选择你要部署的平台："
  echo ""
  echo "  1) NewAPI    - 功能最全，推荐 ⭐"
  echo "  2) One API   - 老牌稳定，生态成熟"
  echo "  3) Custom    - 自定义平台（高级用户）"
  echo ""
  
  read -r -p "请选择 [1-3] (默认: 1): " choice
  choice=${choice:-1}
  
  case "$choice" in
    1)
      PLATFORM_ID="newapi"
      ;;
    2)
      PLATFORM_ID="one-api"
      ;;
    3)
      read -r -p "输入平台 ID: " PLATFORM_ID
      ;;
    *)
      log_error "无效选择"
      show_platform_wizard
      return
      ;;
  esac
  
  load_platform "$PLATFORM_ID"
  
  echo ""
  ui_success "已选择平台: $PLATFORM_NAME"
  echo "  描述: $PLATFORM_DESC"
  echo "  镜像: $DOCKER_IMAGE"
  echo "  默认端口: $DEFAULT_PORT"
  echo ""
  
  if ask_yn "确认选择？"; then
    set_state "platform" "$PLATFORM_ID"
    return 0
  else
    show_platform_wizard
  fi
}
```

---

### 4.2 多平台管理（高级功能）

```bash
# 管理多个平台实例
manage_multiple_platforms() {
  show_banner
  echo "多平台管理"
  echo "============"
  echo ""
  
  echo "当前安装的平台："
  docker ps -a --format '{{.Names}}' | while read -r container; do
    if [[ "$container" =~ ^(new-api|one-api|new-api-horizon)$ ]]; then
      local status=$(docker inspect --format='{{.State.Status}}' "$container" 2>/dev/null)
      echo "  - $container: $status"
    fi
  done
  
  echo ""
  echo "操作："
  echo "  1) 启动平台"
  echo "  2) 停止平台"
  echo "  3) 查看日志"
  echo "  4) 迁移数据（从一个平台到另一个）"
  echo ""
}
```

---

## 五、实施计划

### 5.1 分阶段实施

| 阶段 | 任务 | 工作量 | 优先级 |
|------|------|--------|--------|
| **Phase 1** | 创建 `platform.def` 格式定义 | 1 天 | 🔥 高 |
|  | 内置 NewAPI 和 One API 定义 | 1 天 | 🔥 高 |
|  | 修改主入口支持平台选择 | 1 天 | 🔥 高 |
| **Phase 2** | 实现 `lib/platform.sh` | 2 天 | 🔥 高 |
|  | 修改部署模块支持多平台 | 2 天 | 🔥 高 |
| **Phase 3** | 开发迁移工具（One API → NewAPI） | 3 天 | 中 |
|  | 开发迁移工具（NewAPI → One API） | 2 天 | 低 |
| **Phase 4** | 支持更多平台（New API Horizon、Sub2API） | 3 天 | 低 |
|  | 多平台实例管理 | 3 天 | 中 |

---

### 5.2 具体实施：Phase 1

**任务 1.1：定义 `platform.def` 格式**

文件：`docs/platform-def-spec.md`

```markdown
# Platform Definition File Specification

## 文件位置
`platforms/<platform-id>/platform.def`

## 必需字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `PLATFORM_NAME` | string | 平台显示名称 |
| `PLATFORM_ID` | string | 平台唯一标识（小写，连字符分隔） |
| `PLATFORM_DESC` | string | 平台描述 |
| `DOCKER_IMAGE` | string | Docker 镜像名 |
| `DEFAULT_VERSION` | string | 默认版本标签 |
| `CONTAINER_NAME` | string | 容器名 |
| `DEFAULT_PORT` | int | 默认端口 |
| `REQUIRED_ENVS` | array | 必需的环境变量 |
| `DEPENDENCIES` | array | 依赖的服务（mysql/redis） |

## 可选字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `PORT_BINDING` | string | 端口绑定（默认 `127.0.0.1:port:port`） |
| `HEALTH_CHECK_CMD` | string | 健康检查命令 |
| `INIT_SETUP_REQUIRED` | bool | 是否需要初始化设置 |
| `BACKUP_DATABASE` | bool | 是否备份数据库 |
| `BACKUP_DATA_DIRS` | array | 需要备份的数据目录 |

## 可选函数

| 函数 | 说明 |
|------|------|
| `platform_pre_install()` | 安装前执行（覆盖默认行为） |
| `platform_post_install()` | 安装后执行（覆盖默认行为） |
| `platform_migrate_to_<other>()` | 迁移到其它平台 |
| `platform_migrate_from_<other>()` | 从其它平台迁移 |
```

---

**任务 1.2：创建 NewAPI 的 `platform.def`**

文件：`platforms/newapi/platform.def`

```bash
#!/bin/bash
# NewAPI 平台定义

PLATFORM_NAME="NewAPI"
PLATFORM_ID="newapi"
PLATFORM_DESC="NewAPI - 功能最全的 API 中转站"
PLATFORM_REPO="https://github.com/QuantumNous/new-api"
PLATFORM_DOCS="https://docs.newapi.pro"

DOCKER_IMAGE="calciumion/new-api"
DEFAULT_VERSION="latest"
SUPPORTED_ARCHS=("amd64" "arm64")

CONTAINER_NAME="new-api"
DEFAULT_PORT=3000
PORT_BINDING="127.0.0.1:3000:3000"

REQUIRED_ENVS=(
  "SQL_DSN"
  "REDIS_CONN_STRING"
  "SESSION_SECRET"
)

OPTIONAL_ENVS=(
  "TZ"
  "DEBUG"
  "LOG_DIR"
)

DEPENDENCIES=("mysql" "redis")

HEALTH_CHECK_CMD="curl -f http://localhost:3000/api/status"
INIT_SETUP_REQUIRED=true

DB_NAME="newapi"
DB_USER="root"

BACKUP_DATABASE=true
BACKUP_DATA_DIRS=("data" "logs" "npm")

# 平台特定安装后处理
platform_post_install() {
  log_info "NewAPI 特定后处理..."
  
  # 等待容器完全启动
  sleep 10
  
  # 检查是否需要初始化
  if [[ $(docker exec new-api mysql -uroot -p"$MYSQL_PASS" -e "SELECT COUNT(*) FROM newapi.user" 2>/dev/null | tail -1) -eq 0 ]]; then
    ui_warn "检测到全新安装，请访问 http://localhost:3000 完成初始化设置"
  fi
}
```

---

**任务 1.3：创建 One API 的 `platform.def`**

文件：`platforms/one-api/platform.def`

```bash
#!/bin/bash
# One API 平台定义

PLATFORM_NAME="One API"
PLATFORM_ID="one-api"
PLATFORM_DESC="One API - 老牌稳定的 API 中转站"
PLATFORM_REPO="https://github.com/songquanpeng/one-api"

DOCKER_IMAGE="justsong/one-api"
DEFAULT_VERSION="latest"

CONTAINER_NAME="one-api"
DEFAULT_PORT=3000

REQUIRED_ENVS=(
  "SQL_DSN"
  "SESSION_SECRET"
)

DEPENDENCIES=("mysql" "redis")

HEALTH_CHECK_CMD="curl -f http://localhost:3000/api/status"
INIT_SETUP_REQUIRED=true

DB_NAME="oneapi"
DB_USER="root"

BACKUP_DATABASE=true
BACKUP_DATA_DIRS=("data")

# 从 NewAPI 迁移
platform_migrate_from_newapi() {
  log_info "正在从 NewAPI 迁移到 One API..."
  # 调用迁移脚本
  source "${TOOLKIT_ROOT}/modules/migrate/new-api-to-one-api.sh"
}
```

---

## 六、总结与展望

### 6.1 核心改进

| 改进点 | 当前状态 | 目标状态 |
|--------|---------|---------|
| 平台支持 | 仅 NewAPI | 多平台（NewAPI、One API、等） |
| 架构 | 硬编码 | 平台抽象层（可插拔） |
| 迁移工具 | 无 | 支持平台间迁移 |
| 用户群 | NewAPI 用户 | 所有 API 中转站用户 |

### 6.2 价值主张

**对用户：**
- 一个工具管理所有 API 中转站平台
- 轻松在不同平台间迁移
- 统一的使用体验

**对社区：**
- 成为"API 中转站运维标准工具"
- 社区可以贡献新平台定义
- 推动平台间的互操作性

### 6.3 下一步

**立即可做：**
1. 定义 `platform.def` 规范文档
2. 实现 `lib/platform.sh`（平台管理库）
3. 创建 NewAPI 和 One API 的 `platform.def`

**需要你确认：**
1. 是否要实现多平台支持？（工作量增加约 30%）
2. 优先支持哪些平台？（NewAPI + One API？）
3. 是否需要迁移工具？（One API → NewAPI）

---

**文档结束**

> 下一步：确认方案后，我开始写代码实现 Phase 1！
