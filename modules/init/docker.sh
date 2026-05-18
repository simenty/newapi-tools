#!/bin/bash
# Docker 安装脚本 v2.0.2
# 支持 Debian/Ubuntu 和 CentOS/Rocky/AlmaLinux/RHEL
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

# ---------- 常量 ----------
readonly DOCKER_CE_PKGS="docker-ce docker-ce-cli containerd.io docker-compose-plugin"
readonly DOCKER_OFFICIAL_REPO_DEB="https://download.docker.com/linux"
readonly DOCKER_ALIYUN_REPO_DEB="https://mirrors.aliyun.com/docker-ce/linux"
readonly DOCKER_OFFICIAL_REPO_RPM="https://download.docker.com/linux/centos/docker-ce.repo"
readonly DOCKER_ALIYUN_REPO_RPM="https://mirrors.aliyun.com/docker-ce/linux/centos/docker-ce.repo"

# ---------- 幂等检查：已安装且运行则跳过 ----------
if command -v docker &>/dev/null && docker info &>/dev/null 2>&1; then
    log_info "Docker 已安装且正在运行，跳过安装。"
    docker --version
    # 确认 docker compose 插件可用
    if docker compose version &>/dev/null 2>&1; then
        log_info "Docker Compose Plugin: $(docker compose version)"
    else
        log_warn "Docker Compose 插件未安装，正在补充安装..."
        _install_compose_plugin
    fi
    exit 0
fi

# ---------- 若已安装但未运行，则启动 ----------
if command -v docker &>/dev/null; then
    log_info "Docker 已安装但未运行，正在启动..."
    if systemctl enable --now docker; then
        log_success "Docker 已启动: $(docker --version)"
    else
        log_error "Docker 启动失败，请查看: journalctl -xe -u docker"
        exit 1
    fi
    exit 0
fi

# ---------- 检测 OS 系列（使用 os_adapter.sh）----------
os_detect_full
log_info "检测到系统: OS_FAMILY=${OS_FAMILY}, ID=${OS_ID}, VERSION=${OS_VERSION}"

# ================================================================
# Debian/Ubuntu 安装
# ================================================================
_install_docker_debian() {
    log_info "使用 Debian/Ubuntu 安装方式..."

    # 策略1：官方一键脚本（支持国内镜像前缀）
    local install_url="https://get.docker.com"
    if [[ -n "${DOCKER_MIRROR_PREFIX:-}" ]]; then
        log_info "使用国内镜像加速: ${DOCKER_MIRROR_PREFIX}"
        export DOWNLOAD_URL="${DOCKER_MIRROR_PREFIX}"
    fi

    log_info "正在执行官方安装脚本（启用 SSL 验证）..."
    if curl --proto '=https' --tlsv1.2 -fsSL "${install_url}" | bash -s docker; then
        log_success "Docker 安装成功（官方脚本）"
        return 0
    fi

    # 策略2：手动添加 apt 仓库（降级方案）
    log_warn "官方脚本失败，尝试手动添加 apt 仓库..."

    apt-get update -qq
    apt-get install -y -qq ca-certificates curl gnupg lsb-release

    local codename
    codename=$(lsb_release -cs 2>/dev/null || grep VERSION_CODENAME /etc/os-release | cut -d= -f2 | tr -d '"')
    local distro
    distro=$(grep "^ID=" /etc/os-release | cut -d= -f2 | tr -d '"')

    # 尝试阿里云 → 官方，两路 GPG key
    local gpg_urls=(
        "https://mirrors.aliyun.com/docker-ce/linux/${distro}/gpg"
        "${DOCKER_OFFICIAL_REPO_DEB}/${distro}/gpg"
    )
    local repo_urls=(
        "${DOCKER_ALIYUN_REPO_DEB}/${distro}"
        "${DOCKER_OFFICIAL_REPO_DEB}/${distro}"
    )

    local i
    for i in 0 1; do
        log_info "尝试使用仓库: ${repo_urls[$i]}"
        install -m 0755 -d /etc/apt/keyrings
        if curl --proto '=https' --tlsv1.2 -fsSL "${gpg_urls[$i]}" | gpg --dearmor -o /etc/apt/keyrings/docker.gpg 2>/dev/null; then
            chmod a+r /etc/apt/keyrings/docker.gpg
            echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
                ${repo_urls[$i]} ${codename} stable" \
                > /etc/apt/sources.list.d/docker.list
            apt-get update -qq
            if apt-get install -y -qq ${DOCKER_CE_PKGS}; then
                log_success "Docker 安装成功（手动 apt 仓库）"
                return 0
            fi
        fi
        log_warn "仓库 ${repo_urls[$i]} 不可用，尝试下一个..."
    done

    log_error "所有 Debian/Ubuntu 安装策略均失败"
    return 1
}

# ================================================================
# CentOS/Rocky/AlmaLinux/RHEL 安装
# ================================================================
_install_compose_plugin() {
    # 补充安装 docker compose plugin（单独调用）
    if command -v dnf &>/dev/null; then
        dnf install -y docker-compose-plugin &>/dev/null || true
    elif command -v yum &>/dev/null; then
        yum install -y docker-compose-plugin &>/dev/null || true
    fi
}

_add_rhel_repo() {
    local pkg_mgr="$1"   # dnf 或 yum

    # 尝试顺序：官方 → 阿里云
    local repo_urls=(
        "${DOCKER_OFFICIAL_REPO_RPM}"
        "${DOCKER_ALIYUN_REPO_RPM}"
    )

    for repo_url in "${repo_urls[@]}"; do
        log_info "尝试添加 Docker 仓库: ${repo_url}"
        if [[ "$pkg_mgr" == "dnf" ]]; then
            dnf config-manager --add-repo "${repo_url}" &>/dev/null && {
                log_success "仓库已添加: ${repo_url}"
                return 0
            }
        else
            yum-config-manager --add-repo "${repo_url}" &>/dev/null && {
                log_success "仓库已添加: ${repo_url}"
                return 0
            }
        fi
        log_warn "仓库不可用: ${repo_url}"
    done

    # 最后降级方案：直接下载 repo 文件
    log_warn "config-manager 失败，尝试直接下载 repo 文件..."
    if curl -fsSL "${DOCKER_ALIYUN_REPO_RPM}" -o /etc/yum.repos.d/docker-ce.repo; then
        log_success "已直接下载阿里云 Docker repo 文件"
        return 0
    fi

    log_error "所有 RHEL 仓库添加策略均失败"
    return 1
}

_install_docker_rhel() {
    log_info "使用 RHEL 系 (CentOS/Rocky/AlmaLinux) 安装方式..."

    # CentOS 7 需要先启用 extras 仓库（containerd.io 依赖）
    if [[ "$OS_ID" == "centos" && "$OS_VERSION" == "7" ]]; then
        log_info "CentOS 7: 启用 extras 仓库..."
        yum install -y yum-utils
        yum-config-manager --enable extras &>/dev/null || true
    fi

    if command -v dnf &>/dev/null; then
        log_info "使用 dnf 安装..."
        dnf install -y dnf-plugins-core

        _add_rhel_repo "dnf" || return 1

        # 处理包冲突：--allowerasing 覆盖冲突包（如 podman-docker）
        if ! dnf install -y --allowerasing ${DOCKER_CE_PKGS}; then
            # 尝试移除冲突包后重新安装
            log_warn "安装遇到冲突，尝试移除 podman 后重新安装..."
            dnf remove -y podman buildah 2>/dev/null || true
            dnf install -y ${DOCKER_CE_PKGS} || {
                log_error "dnf 安装 Docker 失败"
                return 1
            }
        fi

    elif command -v yum &>/dev/null; then
        log_info "使用 yum 安装..."
        yum install -y yum-utils

        _add_rhel_repo "yum" || return 1

        yum install -y ${DOCKER_CE_PKGS} || {
            # 尝试移除冲突包后重新安装
            log_warn "安装遇到冲突，尝试移除 podman 后重新安装..."
            yum remove -y podman buildah 2>/dev/null || true
            yum install -y ${DOCKER_CE_PKGS} || {
                log_error "yum 安装 Docker 失败"
                return 1
            }
        }
    else
        log_error "未找到包管理器（dnf/yum），无法安装 Docker"
        return 1
    fi

    log_success "Docker 安装成功（RHEL 系）"
    return 0
}

# ================================================================
# 主安装流程
# ================================================================
log_info "开始安装 Docker..."

case "$OS_FAMILY" in
    debian)
        _install_docker_debian || { log_error "Debian/Ubuntu Docker 安装失败"; exit 1; }
        ;;
    rhel)
        _install_docker_rhel || { log_error "RHEL 系 Docker 安装失败"; exit 1; }
        ;;
    *)
        log_warn "未知系统类型 (${OS_FAMILY})，尝试使用官方安装脚本..."
        curl -fsSL https://get.docker.com | bash -s docker || {
            log_error "Docker 安装失败，请手动安装"
            exit 1
        }
        ;;
esac

# ---------- 启动并设置开机自启 ----------
log_info "启动 Docker 服务..."
if ! systemctl enable --now docker; then
    log_error "Docker 服务启动失败，请检查: journalctl -xe -u docker"
    exit 1
fi

# ---------- 处理 firewalld（RHEL 系）----------
if [[ "$OS_FAMILY" == "rhel" ]] && systemctl is-active --quiet firewalld 2>/dev/null; then
    log_info "检测到 firewalld，重载规则以应用 Docker 网络配置..."
    firewall-cmd --reload &>/dev/null || true
    log_success "firewalld 已重载"
fi

# ---------- 验证安装 ----------
if docker --version &>/dev/null; then
    log_success "Docker 安装完成: $(docker --version)"
else
    log_error "Docker 安装后验证失败，请手动检查。"
    exit 1
fi

# 验证 docker compose 插件
if docker compose version &>/dev/null 2>&1; then
    log_success "Docker Compose Plugin: $(docker compose version)"
else
    log_warn "Docker Compose 插件未安装，尝试单独安装..."
    _install_compose_plugin
    if docker compose version &>/dev/null 2>&1; then
        log_success "Docker Compose Plugin 已安装: $(docker compose version)"
    else
        log_warn "Docker Compose 插件安装失败，可使用 'docker-compose' 代替"
    fi
fi

# ---------- 配置 Docker 镜像加速（从 YAML 配置读取）----------
# 支持环境变量 DOCKER_HUB_MIRROR 或命令行传参
DOCKER_HUB_MIRROR="${DOCKER_HUB_MIRROR:-}"
if [[ -z "$DOCKER_HUB_MIRROR" ]] && command -v get_config &>/dev/null; then
    DOCKER_HUB_MIRROR=$(get_config "deploy.docker.registry_mirror" "")
fi

if [[ -n "$DOCKER_HUB_MIRROR" ]]; then
    log_info "配置 Docker Hub 镜像加速: $DOCKER_HUB_MIRROR"

    # 校验镜像地址格式（必须是 http:// 或 https:// 开头的 URL）
    if [[ ! "$DOCKER_HUB_MIRROR" =~ ^https?://[a-zA-Z0-9] ]]; then
        log_error "Docker 镜像地址格式无效: $DOCKER_HUB_MIRROR（必须以 http:// 或 https:// 开头）"
        log_error "跳过 daemon.json 配置"
        DOCKER_HUB_MIRROR=""
    fi
fi

if [[ -n "$DOCKER_HUB_MIRROR" ]]; then
    mkdir -p /etc/docker

    # 备份原 daemon.json（仅首次备份）
    if [[ -f /etc/docker/daemon.json && ! -f /etc/docker/daemon.json.bak ]]; then
        cp -a /etc/docker/daemon.json /etc/docker/daemon.json.bak
        log_info "已备份 /etc/docker/daemon.json → /etc/docker/daemon.json.bak"
    fi

    _daemon_result=1

    if [[ -f /etc/docker/daemon.json ]]; then
        # 已有 daemon.json：安全合并 registry-mirrors
        if command -v jq &>/dev/null; then
            # 使用 jq 合并（保留用户已有的其他配置项）
            _tmp_file=$(mktemp /tmp/daemon.json.XXXXXX) || { log_error "无法创建临时文件"; _daemon_result=1; }
            if jq --arg mirror "$DOCKER_HUB_MIRROR" \
                '.["registry-mirrors"] = ((.["registry-mirrors"] // []) | if index($mirror) then . else . + [$mirror] end)' \
                /etc/docker/daemon.json > "$_tmp_file" 2>/dev/null; then
                # 校验合并后的 JSON 是否合法
                if jq empty "$_tmp_file" 2>/dev/null; then
                    mv "$_tmp_file" /etc/docker/daemon.json
                    _daemon_result=0
                else
                    rm -f "$_tmp_file"
                    log_error "jq 合并后 JSON 校验失败，回滚到备份"
                    if [[ -f /etc/docker/daemon.json.bak ]]; then
                        cp -a /etc/docker/daemon.json.bak /etc/docker/daemon.json
                    fi
                fi
            else
                rm -f "$_tmp_file"
                log_error "jq 合并 daemon.json 失败"
            fi
        elif command -v python3 &>/dev/null; then
            # 降级方案：使用 python3 安全合并
            python3 - "$DOCKER_HUB_MIRROR" /etc/docker/daemon.json << 'PYEOF'
import json, sys
mirror, path = sys.argv[1], sys.argv[2]
try:
    with open(path) as f:
        cfg = json.load(f)
    cfg.setdefault("registry-mirrors", [])
    if mirror not in cfg["registry-mirrors"]:
        cfg["registry-mirrors"].append(mirror)
    with open(path, "w") as f:
        json.dump(cfg, f, indent=2)
except Exception as e:
    print(f"python3 合并失败: {e}", file=sys.stderr)
    sys.exit(1)
PYEOF
            _daemon_result=$?
        else
            log_error "无法安全合并 daemon.json：需要 jq 或 python3（均未安装）"
            log_error "请安装 jq: apt install jq / yum install jq"
            _daemon_result=1
        fi
    else
        # 不存在 daemon.json：安全创建
        if command -v jq &>/dev/null; then
            jq -n --arg mirror "$DOCKER_HUB_MIRROR" '{
                "registry-mirrors": [$mirror],
                "log-driver": "json-file",
                "log-opts": {"max-size": "100m", "max-file": "3"}
            }' > /etc/docker/daemon.json 2>/dev/null
            _daemon_result=$?
        elif command -v python3 &>/dev/null; then
            # 降级方案：使用 python3 创建
            python3 - "$DOCKER_HUB_MIRROR" /etc/docker/daemon.json << 'PYEOF'
import json, sys
mirror = sys.argv[1]
path = sys.argv[2]
cfg = {
    "registry-mirrors": [mirror],
    "log-driver": "json-file",
    "log-opts": {"max-size": "100m", "max-file": "3"}
}
with open(path, "w") as f:
    json.dump(cfg, f, indent=2)
PYEOF
            _daemon_result=$?
        else
            log_error "无法安全创建 daemon.json：需要 jq 或 python3（均未安装）"
            log_error "请安装 jq: apt install jq / yum install jq"
            _daemon_result=1
        fi
    fi

    # 合并/创建失败时回滚
    if [[ $_daemon_result -ne 0 ]]; then
        log_error "daemon.json 配置失败"
        if [[ -f /etc/docker/daemon.json.bak ]]; then
            cp -a /etc/docker/daemon.json.bak /etc/docker/daemon.json
            log_info "已从备份恢复 /etc/docker/daemon.json"
        fi
    else
        systemctl restart docker
        log_success "Docker 镜像加速已配置"
    fi
fi

log_success "=== Docker 安装和配置完成 ==="
