#!/bin/bash
# 仓库源配置脚本 v2.0.2
# 支持 Debian/Ubuntu 和 CentOS/Rocky/AlmaLinux/RHEL
set -eo pipefail

# shellcheck source=lib/_init.sh
source "${TOOLKIT_ROOT}/lib/_init.sh"
# 仅在直接执行时进行权限检查（source 时跳过，便于单元测试）
if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    check_root
fi

# ================================================================
# 常量：镜像 URL 表
# ================================================================

# --- Debian/Ubuntu 镜像基础 URL（不含 /ubuntu/ 后缀）---
declare -A APT_MIRRORS=(
    [aliyun]="https://mirrors.aliyun.com"
    [tsinghua]="https://mirrors.tuna.tsinghua.edu.cn"
    [ustc]="https://mirrors.ustc.edu.cn"
    [tencent]="https://mirrors.tencent.com"
    [huawei]="https://mirrors.huaweicloud.com"
    [official]="http://archive.ubuntu.com"
    [official_deb]="http://deb.debian.org"
)

# --- RHEL 系镜像 .repo 文件 URL（CentOS 7）---
declare -A YUM7_MIRRORS=(
    [aliyun]="https://mirrors.aliyun.com/repo/Centos-7.repo"
    [tencent]="https://mirrors.tencent.com/repo/Centos-7.repo"
    [huawei]="https://mirrors.huaweicloud.com/repository/conf/CentOS-7-anon.repo"
    [tsinghua]="https://mirrors.tuna.tsinghua.edu.cn/centos/7/os/x86_64/"
    [official]="RESTORE_BACKUP"
)

# --- RHEL 系镜像 .repo 文件 URL（CentOS 8 / Rocky / AlmaLinux）---
declare -A YUM8_MIRRORS=(
    [aliyun]="https://mirrors.aliyun.com/repo/Centos-vault-8.5.2111.repo"
    [tencent]="https://mirrors.tencent.com/repo/Centos-8.repo"
    [rocky_aliyun]="https://mirrors.aliyun.com/rockylinux/RPM-GPG-KEY-Rocky-9"
    [official]="RESTORE_BACKUP"
)

# ================================================================
# 工具函数
# ================================================================

# 检测镜像是否可达（最多 5 秒超时）
_check_mirror_available() {
    local url="$1"
    curl --proto '=https' --tlsv1.2 -fsSL --max-time 5 --head "$url" &>/dev/null
}

# 获取操作系统发行版详情
_get_os_details() {
    OS_ID=$(grep "^ID=" /etc/os-release 2>/dev/null | cut -d= -f2 | tr -d '"')
    OS_VERSION_ID=$(grep "^VERSION_ID=" /etc/os-release 2>/dev/null | cut -d= -f2 | tr -d '"')
    OS_VERSION_MAJOR="${OS_VERSION_ID%%.*}"   # 主版本号
    OS_CODENAME=$(grep "^VERSION_CODENAME=" /etc/os-release 2>/dev/null | cut -d= -f2 | tr -d '"')
    # 如果 /etc/os-release 没有 codename，用 lsb_release 补充
    if [[ -z "$OS_CODENAME" ]]; then
        OS_CODENAME=$(lsb_release -cs 2>/dev/null || echo "")
    fi
}

# ================================================================
# Debian/Ubuntu 仓库配置
# ================================================================
configure_apt_source() {
    local mirror="$1"
    _get_os_details

    log_info "配置 Apt 软件源: $mirror (OS: ${OS_ID} ${OS_CODENAME})"

    if [[ -z "$OS_CODENAME" ]]; then
        log_error "无法获取系统代号（codename），请手动配置"
        return 1
    fi

    # 备份（只备份一次）
    if [[ -f /etc/apt/sources.list && ! -f /etc/apt/sources.list.bak ]]; then
        cp /etc/apt/sources.list /etc/apt/sources.list.bak
        log_info "已备份 sources.list → sources.list.bak"
    fi

    local base_url=""
    local content=""

    case "$mirror" in
        official)
            if [[ "$OS_ID" == "debian" ]]; then
                base_url="${APT_MIRRORS[official_deb]}"
                content="deb ${base_url}/debian/ ${OS_CODENAME} main contrib non-free non-free-firmware
deb ${base_url}/debian/ ${OS_CODENAME}-updates main contrib non-free non-free-firmware
deb ${base_url}/debian-security/ ${OS_CODENAME}-security main contrib non-free non-free-firmware"
            else
                base_url="${APT_MIRRORS[official]}/ubuntu"
                content="deb ${base_url}/ ${OS_CODENAME} main restricted universe multiverse
deb ${base_url}/ ${OS_CODENAME}-security main restricted universe multiverse
deb ${base_url}/ ${OS_CODENAME}-updates main restricted universe multiverse
deb ${base_url}/ ${OS_CODENAME}-backports main restricted universe multiverse"
            fi
            ;;
        aliyun|tsinghua|ustc|tencent|huawei)
            local key="$mirror"
            base_url="${APT_MIRRORS[$key]}"
            if [[ "$OS_ID" == "debian" ]]; then
                content="deb ${base_url}/debian/ ${OS_CODENAME} main contrib non-free non-free-firmware
deb ${base_url}/debian/ ${OS_CODENAME}-updates main contrib non-free non-free-firmware
deb ${base_url}/debian-security/ ${OS_CODENAME}-security main contrib non-free non-free-firmware"
            else
                content="deb ${base_url}/ubuntu/ ${OS_CODENAME} main restricted universe multiverse
deb ${base_url}/ubuntu/ ${OS_CODENAME}-security main restricted universe multiverse
deb ${base_url}/ubuntu/ ${OS_CODENAME}-updates main restricted universe multiverse
deb ${base_url}/ubuntu/ ${OS_CODENAME}-backports main restricted universe multiverse"
            fi
            ;;
        *)
            log_error "未知的 apt 镜像: $mirror"
            return 1
            ;;
    esac

    # 写入配置
    printf '%s\n' "$content" > /etc/apt/sources.list
    log_success "已写入新的 sources.list"

    # 验证：apt update
    log_info "验证：正在更新软件列表..."
    if apt-get update -qq; then
        log_success "软件源验证通过，apt update 成功"
    else
        log_warn "apt update 失败，尝试自动恢复备份..."
        if [[ -f /etc/apt/sources.list.bak ]]; then
            cp /etc/apt/sources.list.bak /etc/apt/sources.list
            apt-get update -qq || true
            log_warn "已恢复原始软件源，请手动检查"
        fi
        return 1
    fi
}

# ================================================================
# CentOS/Rocky/AlmaLinux 仓库配置
# ================================================================
configure_yum_source() {
    local mirror="$1"
    _get_os_details

    log_info "配置 Yum/DNF 软件源: $mirror (OS: ${OS_ID} ${OS_VERSION_MAJOR})"

    # 备份现有仓库配置
    local backup_dir="/etc/yum.repos.d/backup-$(date +%Y%m%d%H%M%S)"
    if [[ ! -d "$backup_dir" ]]; then
        mkdir -p "$backup_dir"
        cp /etc/yum.repos.d/*.repo "$backup_dir/" 2>/dev/null || true
        log_info "已备份仓库配置到: $backup_dir"
    fi

    if [[ "$OS_ID" == "centos" && "$OS_VERSION_MAJOR" == "7" ]]; then
        _configure_yum7 "$mirror"
    else
        # CentOS 8+, Rocky, AlmaLinux, RHEL 8+
        _configure_yum8_dnf "$mirror"
    fi
}

# CentOS 7 使用 yum
_configure_yum7() {
    local mirror="$1"

    case "$mirror" in
        aliyun)
            curl --proto '=https' --tlsv1.2 -fsSL "${YUM7_MIRRORS[aliyun]}" -o /etc/yum.repos.d/CentOS-Base.repo || {
                log_error "下载阿里云 repo 失败"
                return 1
            }
            # 安装 EPEL（添加 SSL 验证和错误检查）
            if curl --proto '=https' --tlsv1.2 -fsSL "https://mirrors.aliyun.com/repo/epel-7.repo" \
                -o /etc/yum.repos.d/epel.repo; then
                # 验证 repo 文件格式
                if ! grep -q '^\[.*\]' /etc/yum.repos.d/epel.repo 2>/dev/null; then
                    log_warn "EPEL repo 文件格式可能无效"
                fi
            else
                log_warn "下载 EPEL repo 失败，跳过"
            fi
            ;;
        tencent)
            curl --proto '=https' --tlsv1.2 -fsSL "${YUM7_MIRRORS[tencent]}" -o /etc/yum.repos.d/CentOS-Base.repo || {
                log_error "下载腾讯云 repo 失败"
                return 1
            }
            ;;
        huawei)
            curl --proto '=https' --tlsv1.2 -fsSL "${YUM7_MIRRORS[huawei]}" -o /etc/yum.repos.d/CentOS-Base.repo || {
                log_error "下载华为云 repo 失败"
                return 1
            }
            ;;
        official)
            local backup_dir
            backup_dir=$(ls -td /etc/yum.repos.d/backup-* 2>/dev/null | head -1)
            if [[ -d "$backup_dir" ]] && ls "$backup_dir"/*.repo &>/dev/null; then
                cp "$backup_dir"/*.repo /etc/yum.repos.d/
                log_success "已恢复官方原始源"
            else
                log_warn "未找到备份，无法恢复官方源"
                return 1
            fi
            ;;
        *)
            log_error "CentOS 7 不支持的镜像: $mirror"
            return 1
            ;;
    esac

    log_info "更新 YUM 缓存..."
    if yum makecache fast; then
        log_success "软件源验证通过"
    else
        log_warn "yum makecache 失败，请手动检查"
        return 1
    fi
}

# CentOS 8+ / Rocky / AlmaLinux 使用 dnf
_configure_yum8_dnf() {
    local mirror="$1"

    # 判断具体发行版，设置对应 repo URL
    local base_url=""
    case "$mirror" in
        aliyun)
            case "$OS_ID" in
                rocky)
                    base_url="https://mirrors.aliyun.com/rockylinux"
                    # 使用 sed 替换 mirrorlist 为 baseurl
                    sed -e "s|^mirrorlist=|#mirrorlist=|g" \
                        -e "s|^#baseurl=https://dl.rockylinux.org|baseurl=${base_url}|g" \
                        -i.bak /etc/yum.repos.d/Rocky-*.repo 2>/dev/null || true
                    ;;
                almalinux)
                    base_url="https://mirrors.aliyun.com/almalinux"
                    sed -e "s|^mirrorlist=|#mirrorlist=|g" \
                        -e "s|^# baseurl=https://repo.almalinux.org|baseurl=${base_url}|g" \
                        -i.bak /etc/yum.repos.d/almalinux*.repo 2>/dev/null || true
                    ;;
                centos|rhel)
                    curl --proto '=https' --tlsv1.2 -fsSL "${YUM8_MIRRORS[aliyun]}" -o /etc/yum.repos.d/CentOS-Base.repo || {
                        log_error "下载阿里云 CentOS 8 repo 失败"
                        return 1
                    }
                    ;;
            esac
            ;;
        tencent)
            case "$OS_ID" in
                rocky)
                    sed -e "s|^mirrorlist=|#mirrorlist=|g" \
                        -e "s|^#baseurl=https://dl.rockylinux.org|baseurl=https://mirrors.tencent.com/rockylinux|g" \
                        -i.bak /etc/yum.repos.d/Rocky-*.repo 2>/dev/null || true
                    ;;
                almalinux)
                    sed -e "s|^mirrorlist=|#mirrorlist=|g" \
                        -e "s|^# baseurl=https://repo.almalinux.org|baseurl=https://mirrors.tencent.com/almalinux|g" \
                        -i.bak /etc/yum.repos.d/almalinux*.repo 2>/dev/null || true
                    ;;
                centos|rhel)
                    curl --proto '=https' --tlsv1.2 -fsSL "${YUM8_MIRRORS[tencent]}" -o /etc/yum.repos.d/CentOS-Base.repo || {
                        log_error "下载腾讯云 repo 失败"
                        return 1
                    }
                    # 验证 repo 文件格式
                    if ! grep -q '^\[.*\]' /etc/yum.repos.d/CentOS-Base.repo 2>/dev/null; then
                        log_error "Repo 文件格式无效: /etc/yum.repos.d/CentOS-Base.repo"
                        return 1
                    fi
                    ;;
            esac
            ;;
        huawei)
            case "$OS_ID" in
                rocky)
                    sed -e "s|^mirrorlist=|#mirrorlist=|g" \
                        -e "s|^#baseurl=https://dl.rockylinux.org|baseurl=https://mirrors.huaweicloud.com/rocky|g" \
                        -i.bak /etc/yum.repos.d/Rocky-*.repo 2>/dev/null || true
                    ;;
                almalinux)
                    sed -e "s|^mirrorlist=|#mirrorlist=|g" \
                        -e "s|^# baseurl=https://repo.almalinux.org|baseurl=https://mirrors.huaweicloud.com/almalinux|g" \
                        -i.bak /etc/yum.repos.d/almalinux*.repo 2>/dev/null || true
                    ;;
                *)
                    log_warn "华为云镜像暂不支持 ${OS_ID}，使用阿里云"
                    _configure_yum8_dnf "aliyun"
                    return $?
                    ;;
            esac
            ;;
        official)
            local backup_dir
            backup_dir=$(ls -td /etc/yum.repos.d/backup-* 2>/dev/null | head -1)
            if [[ -d "$backup_dir" ]] && ls "$backup_dir"/*.repo &>/dev/null; then
                cp "$backup_dir"/*.repo /etc/yum.repos.d/
                # 清除 sed 生成的 .bak 文件
                rm -f /etc/yum.repos.d/*.bak 2>/dev/null || true
                log_success "已恢复官方原始源"
            else
                log_warn "未找到备份，无法恢复官方源"
                return 1
            fi
            ;;
        *)
            log_error "不支持的镜像: $mirror"
            return 1
            ;;
    esac

    # 清除 .bak 残留
    rm -f /etc/yum.repos.d/*.bak 2>/dev/null || true

    log_info "更新 DNF 缓存..."
    if dnf makecache; then
        log_success "软件源验证通过"
    else
        log_warn "dnf makecache 失败，尝试恢复..."
        local backup_dir
        backup_dir=$(ls -td /etc/yum.repos.d/backup-* 2>/dev/null | head -1)
        [[ -d "$backup_dir" ]] && cp "$backup_dir"/*.repo /etc/yum.repos.d/ && log_warn "已恢复备份"
        return 1
    fi
}

# ================================================================
# 主函数
# ================================================================
main() {
    # 检测 OS 系列
    OS_FAMILY=$(get_os_family)
    log_info "检测到系统类型: $OS_FAMILY"

    # 根据 OS 类型确定支持的镜像列表
    local mirror_options=()
    if [[ "$OS_FAMILY" == "debian" ]]; then
        mirror_options=("1) 阿里云（推荐国内）" "2) 腾讯云（腾讯服务器推荐）" "3) 华为云" "4) 清华镜像（教育网推荐）" "5) 中科大镜像" "6) 官方源（国外服务器）")
    else
        mirror_options=("1) 阿里云（推荐国内）" "2) 腾讯云（腾讯服务器推荐）" "3) 华为云" "4) 官方源（国外服务器）")
    fi

    show_banner "仓库源配置"
    novice_prompt "配置软件源可以加速软件下载。国内服务器建议使用阿里云或腾讯云镜像。"

    # 解析 MIRROR 参数
    local MIRROR="${1:-}"

    if [[ -z "$MIRROR" ]]; then
        echo ""
        echo "请选择软件源镜像："
        printf '  %s\n' "${mirror_options[@]}"
        echo ""
        read -r -p "请输入选项 [默认: 1]: " mirror_choice
        mirror_choice="${mirror_choice:-1}"

        if [[ "$OS_FAMILY" == "debian" ]]; then
            case "$mirror_choice" in
                1) MIRROR="aliyun"   ;;
                2) MIRROR="tencent"  ;;
                3) MIRROR="huawei"   ;;
                4) MIRROR="tsinghua" ;;
                5) MIRROR="ustc"     ;;
                6) MIRROR="official" ;;
                *) log_warn "无效选项，使用默认（阿里云）"; MIRROR="aliyun" ;;
            esac
        else
            case "$mirror_choice" in
                1) MIRROR="aliyun"   ;;
                2) MIRROR="tencent"  ;;
                3) MIRROR="huawei"   ;;
                4) MIRROR="official" ;;
                *) log_warn "无效选项，使用默认（阿里云）"; MIRROR="aliyun" ;;
            esac
        fi
    fi

    log_info "选择的镜像: $MIRROR"

    # 执行配置
    if [[ "$OS_FAMILY" == "debian" ]]; then
        configure_apt_source "$MIRROR"
    elif [[ "$OS_FAMILY" == "rhel" ]]; then
        configure_yum_source "$MIRROR"
    else
        log_error "不支持的系统类型: $OS_FAMILY"
        exit 1
    fi

    log_success "=== 软件源配置完成 ==="
    log_info "提示：Docker 源配置由 modules/init/docker.sh 单独管理"
}

# ---------- 执行 ----------
main "$@"
