#!/bin/bash
set -eo pipefail
# OS 适配层 —— 统一封装不同 Linux 发行版的差异
# 提供 OS 检测、包管理、服务管理、系统更新等抽象接口
# 所有函数使用 os_ 前缀，遵循 snake_case 命名规范

# 防重入守卫
[[ "${_OS_ADAPTER_LOADED:-0}" -eq 1 ]] && return 0
export _OS_ADAPTER_LOADED=1

# ---------- 缓存文件路径 ----------
_OS_CACHE_DIR="${STATE_DIR:-${TOOLKIT_ROOT}/state}"
_OS_CACHE_FILE="${_OS_CACHE_DIR}/os-info.cache"

# ==============================================================================
# os_get_id — 返回 OS ID
# 从 /etc/os-release 读取 ID 字段（如 debian, ubuntu, centos 等）
# 返回值: 小写 OS ID 字符串，检测失败返回 "unknown"
# ==============================================================================
os_get_id() {
    # 优先使用缓存的全局变量
    if [[ -n "${OS_ID:-}" ]]; then
        echo "$OS_ID"
        return 0
    fi

    if [[ -f /etc/os-release ]]; then
        local id
        id=$(grep "^ID=" /etc/os-release 2>/dev/null | cut -d= -f2 | tr -d '"' | tr '[:upper:]' '[:lower:]')
        if [[ -n "$id" ]]; then
            echo "$id"
            return 0
        fi
    fi

    # 降级：检查 /etc/redhat-release（RHEL 系列可能没有 ID 字段）
    if [[ -f /etc/redhat-release ]]; then
        echo "rhel"
        return 0
    fi

    echo "unknown"
}

# ==============================================================================
# os_get_family — 返回 OS 系列
# 返回值: "debian" | "rhel" | "arch" | "unknown"
# 映射规则:
#   debian  ← Debian/Ubuntu/Linux Mint
#   rhel    ← CentOS/Rocky/AlmaLinux/RHEL/Fedora/Oracle Linux
#   arch    ← Arch/Manjaro
# 降级检测: 检查包管理器（apt→debian, dnf/yum→rhel, pacman→arch）
# ==============================================================================
os_get_family() {
    # 优先使用缓存的全局变量
    if [[ -n "${OS_FAMILY:-}" && "${OS_FAMILY:-}" != "unknown" ]]; then
        echo "$OS_FAMILY"
        return 0
    fi

    if [[ -f /etc/os-release ]]; then
        # 先检查 ID_LIKE 字段
        local id_like
        id_like=$(grep "^ID_LIKE=" /etc/os-release 2>/dev/null | cut -d= -f2 | tr -d '"' | tr '[:upper:]' '[:lower:]')

        # ID_LIKE 可能包含多个值（如 "rhel centos fedora"），逐一匹配
        local like_word
        for like_word in $id_like; do
            case "$like_word" in
                debian)
                    echo "debian"
                    return 0
                    ;;
                rhel|fedora)
                    echo "rhel"
                    return 0
                    ;;
                arch)
                    echo "arch"
                    return 0
                    ;;
            esac
        done

        # ID_LIKE 未匹配，检查 ID 字段
        local id
        id=$(os_get_id)
        case "$id" in
            debian|ubuntu|linuxmint)
                echo "debian"
                return 0
                ;;
            centos|rocky|almalinux|rhel|fedora|ol)
                echo "rhel"
                return 0
                ;;
            arch|manjaro)
                echo "arch"
                return 0
                ;;
        esac
    fi

    # 降级检测：通过包管理器推断
    if command -v apt-get &>/dev/null || command -v apt &>/dev/null; then
        echo "debian"
    elif command -v dnf &>/dev/null || command -v yum &>/dev/null; then
        echo "rhel"
    elif command -v pacman &>/dev/null; then
        echo "arch"
    else
        echo "unknown"
    fi
}

# ==============================================================================
# os_get_version — 返回 OS 主版本号
# 从 /etc/os-release VERSION_ID 字段提取主版本号（如 "12" 从 "12.5"）
# 返回值: 主版本号字符串，检测失败返回 "unknown"
# ==============================================================================
os_get_version() {
    # 优先使用缓存的全局变量
    if [[ -n "${OS_VERSION:-}" && "${OS_VERSION:-}" != "unknown" ]]; then
        echo "$OS_VERSION"
        return 0
    fi

    if [[ -f /etc/os-release ]]; then
        local version_id
        version_id=$(grep "^VERSION_ID=" /etc/os-release 2>/dev/null | cut -d= -f2 | tr -d '"')
        if [[ -n "$version_id" ]]; then
            # 提取主版本号：取第一个点号前的部分
            local major_version
            major_version="${version_id%%.*}"
            echo "$major_version"
            return 0
        fi
    fi

    echo "unknown"
}

# ==============================================================================
# os_get_codename — 返回 OS 代号
# 从 /etc/os-release VERSION_CODENAME 字段读取（如 bookworm, jammy）
# 返回值: 代号字符串，不存在时返回空字符串
# ==============================================================================
os_get_codename() {
    # 优先使用缓存的全局变量
    if [[ -n "${OS_CODENAME:-}" ]]; then
        echo "$OS_CODENAME"
        return 0
    fi

    if [[ -f /etc/os-release ]]; then
        local codename
        codename=$(grep "^VERSION_CODENAME=" /etc/os-release 2>/dev/null | cut -d= -f2 | tr -d '"')
        if [[ -n "$codename" ]]; then
            echo "$codename"
            return 0
        fi
    fi

    # 无代号返回空字符串（RHEL 系列通常没有代号）
    echo ""
}

# ==============================================================================
# os_get_pkg_manager — 返回包管理器命令
# 选择规则:
#   apt-get — Debian/Ubuntu 系列
#   dnf     — Fedora / Rocky 9+ / AlmaLinux 9+（优先于 yum）
#   yum     — CentOS 7 / Rocky 8 / AlmaLinux 8
#   pacman  — Arch/Manjaro
# 返回值: 包管理器命令名（不含路径），未知时返回空字符串
# ==============================================================================
os_get_pkg_manager() {
    # 优先使用缓存的全局变量
    if [[ -n "${OS_PKG_MGR:-}" ]]; then
        echo "$OS_PKG_MGR"
        return 0
    fi

    local family
    family=$(os_get_family)

    case "$family" in
        debian)
            echo "apt-get"
            return 0
            ;;
        arch)
            echo "pacman"
            return 0
            ;;
        rhel)
            # RHEL 系列需要区分 dnf 和 yum
            local id version
            id=$(os_get_id)
            version=$(os_get_version)

            case "$id" in
                fedora)
                    # Fedora 默认 dnf
                    echo "dnf"
                    ;;
                rocky|almalinux)
                    # Rocky/AlmaLinux 9+ 使用 dnf，8 使用 yum
                    if [[ "$version" =~ ^[0-9]+$ && "$version" -ge 9 ]]; then
                        echo "dnf"
                    else
                        echo "yum"
                    fi
                    ;;
                centos)
                    # CentOS 7 使用 yum，8 使用 dnf
                    if [[ "$version" =~ ^[0-9]+$ && "$version" -ge 8 ]]; then
                        echo "dnf"
                    else
                        echo "yum"
                    fi
                    ;;
                rhel)
                    # RHEL 8+ 使用 dnf
                    if [[ "$version" =~ ^[0-9]+$ && "$version" -ge 8 ]]; then
                        echo "dnf"
                    else
                        echo "yum"
                    fi
                    ;;
                ol)
                    # Oracle Linux 8+ 使用 dnf
                    if [[ "$version" =~ ^[0-9]+$ && "$version" -ge 8 ]]; then
                        echo "dnf"
                    else
                        echo "yum"
                    fi
                    ;;
                *)
                    # 降级：检测可用命令
                    if command -v dnf &>/dev/null; then
                        echo "dnf"
                    elif command -v yum &>/dev/null; then
                        echo "yum"
                    else
                        echo ""
                    fi
                    ;;
            esac
            return 0
            ;;
        *)
            # 未知系列，尝试检测可用命令
            if command -v apt-get &>/dev/null; then
                echo "apt-get"
            elif command -v dnf &>/dev/null; then
                echo "dnf"
            elif command -v yum &>/dev/null; then
                echo "yum"
            elif command -v pacman &>/dev/null; then
                echo "pacman"
            else
                echo ""
            fi
            ;;
    esac
}

# ==============================================================================
# os_install_packages — 统一安装软件包
# 用法: os_install_packages pkg1 pkg2 ...
# 自动选择当前系统的包管理器执行安装
# 返回值: 包管理器的退出码
# ==============================================================================
os_install_packages() {
    local packages=("$@")

    if [[ ${#packages[@]} -eq 0 ]]; then
        log_warn "os_install_packages: 未指定要安装的软件包"
        return 1
    fi

    local pkg_mgr
    pkg_mgr=$(os_get_pkg_manager)

    if [[ -z "$pkg_mgr" ]]; then
        log_error "无法确定包管理器，安装失败: ${packages[*]}"
        return 1
    fi

    log_info "使用 $pkg_mgr 安装软件包: ${packages[*]}"

    case "$pkg_mgr" in
        apt-get)
            apt-get install -y "${packages[@]}"
            ;;
        dnf)
            dnf install -y "${packages[@]}"
            ;;
        yum)
            yum install -y "${packages[@]}"
            ;;
        pacman)
            pacman -S --noconfirm "${packages[@]}"
            ;;
        *)
            log_error "不支持的包管理器: $pkg_mgr"
            return 1
            ;;
    esac
}

# ==============================================================================
# os_service_action — 统一服务管理
# 用法: os_service_action <action> <service>
# 支持操作: start, stop, restart, enable, disable, status
# 使用 systemctl（所有现代 Linux 发行版均支持）
# 返回值: systemctl 的退出码
# ==============================================================================
os_service_action() {
    local action="$1"
    local service="$2"

    if [[ -z "$action" ]]; then
        log_error "os_service_action: 未指定操作"
        return 1
    fi

    if [[ -z "$service" ]]; then
        log_error "os_service_action: 未指定服务名"
        return 1
    fi

    # 验证操作类型
    case "$action" in
        start|stop|restart|enable|disable|status)
            # 合法操作
            ;;
        *)
            log_error "不支持的服务操作: $action（支持: start, stop, restart, enable, disable, status）"
            return 1
            ;;
    esac

    if ! command -v systemctl &>/dev/null; then
        log_error "systemctl 不可用，无法管理服务: $service"
        return 1
    fi

    log_debug "服务操作: systemctl $action $service"
    systemctl "$action" "$service"
}

# ==============================================================================
# os_update_system — 统一系统更新
# 根据不同发行版执行对应的系统更新命令
# 返回值: 更新命令的退出码
# ==============================================================================
os_update_system() {
    local pkg_mgr
    pkg_mgr=$(os_get_pkg_manager)

    if [[ -z "$pkg_mgr" ]]; then
        log_error "无法确定包管理器，系统更新失败"
        return 1
    fi

    log_info "正在更新系统（使用 $pkg_mgr）..."

    case "$pkg_mgr" in
        apt-get)
            apt-get update && apt-get upgrade -y
            ;;
        dnf)
            dnf upgrade -y
            ;;
        yum)
            yum update -y
            ;;
        pacman)
            pacman -Syu --noconfirm
            ;;
        *)
            log_error "不支持的包管理器: $pkg_mgr"
            return 1
            ;;
    esac
}

# ==============================================================================
# os_detect_full — 一次性检测全部 OS 信息并设置全局变量
# 设置变量: OS_ID, OS_FAMILY, OS_VERSION, OS_CODENAME, OS_PKG_MGR, OS_ARCH
# 带缓存机制（1 小时有效期），缓存文件: ${STATE_DIR}/os-info.cache
# 调用此函数后，其他 os_get_* 函数会自动使用缓存的全局变量
# ==============================================================================
os_detect_full() {
    # 检查缓存是否有效（1 小时有效期）
    if [[ -f "$_OS_CACHE_FILE" ]]; then
        local now cache_mtime age
        now=$(date +%s)
        cache_mtime=$(stat -c %Y "$_OS_CACHE_FILE" 2>/dev/null || echo 0)
        age=$(( now - cache_mtime ))

        # 检查 /etc/os-release 是否在缓存之后被修改（确保缓存与系统一致）
        local os_mtime=0
        if [[ -f /etc/os-release ]]; then
            os_mtime=$(stat -c %Y /etc/os-release 2>/dev/null || echo 0)
        fi

        if [[ $age -lt 3600 && $cache_mtime -ge $os_mtime ]]; then
            # 加载缓存
            source "$_OS_CACHE_FILE"
            log_debug "使用缓存的 OS 信息（缓存年龄: ${age}s）"
            return 0
        fi
    fi

    # 重新检测所有 OS 信息
    OS_ID=$(os_get_id)
    OS_FAMILY=$(os_get_family)
    OS_VERSION=$(os_get_version)
    OS_CODENAME=$(os_get_codename)
    OS_PKG_MGR=$(os_get_pkg_manager)
    OS_ARCH=$(uname -m 2>/dev/null || echo "unknown")

    # 导出全局变量，供其他 os_get_* 函数直接使用
    export OS_ID OS_FAMILY OS_VERSION OS_CODENAME OS_PKG_MGR OS_ARCH

    # 写入缓存
    mkdir -p "$_OS_CACHE_DIR"
    cat > "$_OS_CACHE_FILE" << EOF
#!/bin/bash
# OS 信息缓存 — 由 os_detect_full() 自动生成
# 缓存时间: $(date '+%Y-%m-%d %H:%M:%S')
OS_ID="$OS_ID"
OS_FAMILY="$OS_FAMILY"
OS_VERSION="$OS_VERSION"
OS_CODENAME="$OS_CODENAME"
OS_PKG_MGR="$OS_PKG_MGR"
OS_ARCH="$OS_ARCH"
EOF

    log_debug "OS 信息已检测并缓存: id=$OS_ID family=$OS_FAMILY version=$OS_VERSION codename=$OS_CODENAME pkg_mgr=$OS_PKG_MGR arch=$OS_ARCH"
}

# ---------- 导出函数 ----------
export -f os_get_id os_get_family os_get_version os_get_codename
export -f os_get_pkg_manager os_install_packages os_service_action
export -f os_update_system os_detect_full
