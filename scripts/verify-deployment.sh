#!/bin/bash
# NewAPI Tools V2.0 部署验证脚本
# 用于验证安装的完整性和安全性
set -eo pipefail

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 计数器
TOTAL_CHECKS=0
PASSED_CHECKS=0
FAILED_CHECKS=0
WARNINGS=0

# 打印函数
print_header() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
    ((PASSED_CHECKS++))
    ((TOTAL_CHECKS++))
}

print_fail() {
    echo -e "${RED}❌ $1${NC}"
    ((FAILED_CHECKS++))
    ((TOTAL_CHECKS++))
}

print_warn() {
    echo -e "${YELLOW}⚠️  $1${NC}"
    ((WARNINGS++))
}

print_info() {
    echo -e "  ℹ️  $1"
}

# 检查 Docker
check_docker() {
    print_header "Docker 检查"
    
    if command -v docker &>/dev/null; then
        print_success "Docker 命令可用"
        if docker info &>/dev/null; then
            print_success "Docker 服务正在运行"
        else
            print_fail "Docker 服务未运行"
        fi
    else
        print_fail "Docker 未安装"
    fi
    
    if command -v docker-compose &>/dev/null || docker compose version &>/dev/null; then
        print_success "Docker Compose 可用"
    else
        print_fail "Docker Compose 未安装"
    fi
}

# 检查目录权限
check_permissions() {
    print_header "目录和权限检查"
    
    # 检查备份目录
    if [[ -d "./backups" ]]; then
        local perm
        perm=$(stat -c '%a' "./backups" 2>/dev/null || stat -f '%Lp' "./backups" 2>/dev/null)
        if [[ "$perm" == "700" ]] || [[ "$perm" == "600" ]]; then
            print_success "备份目录权限正确: $perm"
        else
            print_warn "备份目录权限应更严格: $perm (建议 700)"
        fi
    else
        print_info "备份目录不存在（正常，首次部署后创建）"
    fi
    
    # 检查配置文件
    if [[ -f "./.env" ]]; then
        perm=$(stat -c '%a' "./.env" 2>/dev/null || stat -f '%Lp' "./.env" 2>/dev/null)
        if [[ "$perm" == "600" ]]; then
            print_success ".env 文件权限正确: $perm"
        else
            print_fail ".env 文件权限应设置为 600，当前: $perm"
        fi
    else
        print_info ".env 文件不存在（正常，首次部署后创建）"
    fi
    
    # 检查 docker-compose.yml
    if [[ -f "./docker-compose.yml" ]]; then
        perm=$(stat -c '%a' "./docker-compose.yml" 2>/dev/null || stat -f '%Lp' "./docker-compose.yml" 2>/dev/null)
        if [[ "$perm" == "600" ]]; then
            print_success "docker-compose.yml 权限正确: $perm"
        else
            print_warn "docker-compose.yml 权限应更严格: $perm (建议 600)"
        fi
    else
        print_info "docker-compose.yml 不存在（正常，首次部署后创建）"
    fi
}

# 检查脚本完整性
check_scripts() {
    print_header "脚本完整性检查"
    
    local required_files=(
        "lib/common.sh"
        "lib/config.sh"
        "lib/state.sh"
        "lib/ui.sh"
        "lib/mode.sh"
        "lib/smart-defaults.sh"
        "lib/security.sh"
        "lib/npm-api.sh"
        "modules/deploy/install.sh"
        "modules/manage/newapi/backup.sh"
        "modules/manage/newapi/restore.sh"
        "modules/monitor/health.sh"
    )
    
    for file in "${required_files[@]}"; do
        if [[ -f "$file" ]]; then
            print_success "脚本存在: $file"
        else
            print_fail "脚本缺失: $file"
        fi
    done
}

# 检查 ShellCheck（如果可用）
check_shellcheck() {
    print_header "ShellCheck 静态分析"
    
    if command -v shellcheck &>/dev/null; then
        print_info "ShellCheck 可用，正在检查关键脚本..."
        
        # 检查关键脚本
        if shellcheck lib/security.sh 2>/dev/null | grep -q "SC"; then
            print_warn "lib/security.sh 存在 ShellCheck 警告"
        else
            print_success "lib/security.sh 通过 ShellCheck"
        fi
    else
        print_info "ShellCheck 未安装（可选）"
        print_info "安装命令: apt install shellcheck"
    fi
}

# 检查配置
check_config() {
    print_header "配置检查"
    
    if [[ -f "config/config.yaml" ]]; then
        print_success "配置文件存在"
        
        # 检查 YAML 语法
        if command -v python3 &>/dev/null; then
            if python3 -c "import yaml; yaml.safe_load(open('config/config.yaml'))" 2>/dev/null; then
                print_success "YAML 格式正确"
            else
                print_fail "YAML 格式错误"
            fi
        fi
    else
        print_warn "配置文件不存在（正常，首次部署后创建）"
    fi
}

# 安全检查
security_check() {
    print_header "安全检查"
    
    # 检查是否包含硬编码密码
    if grep -r "password.*=" lib/ modules/ 2>/dev/null | grep -v "generate_password\|get_config\|MYSQL_PWD" | grep -qv "^.*#"; then
        print_warn "可能存在硬编码密码，请检查"
    else
        print_success "未发现明显硬编码密码"
    fi
    
    # 检查敏感文件权限
    if [[ -f ".env" ]]; then
        local owner
        owner=$(stat -c '%U' ".env" 2>/dev/null || stat -f '%Su' ".env" 2>/dev/null)
        if [[ "$owner" == "root" ]]; then
            print_success ".env 文件由 root 拥有（安全）"
        else
            print_warn ".env 文件应由 root 拥有"
        fi
    fi
    
    # 检查备份文件权限
    if find ./backups -name "*.sql" -o -name "*.tar.gz" 2>/dev/null | head -1 | xargs stat -c '%a' 2>/dev/null | grep -q "600\|700"; then
        print_success "备份文件权限安全"
    elif [[ -d "./backups" ]] && [[ -n "$(ls -A ./backups/*.sql 2>/dev/null)" ]]; then
        print_warn "部分备份文件权限可能不够严格"
    fi
}

# Docker 容器检查
check_containers() {
    print_header "Docker 容器检查"
    
    if docker ps -a --format '{{.Names}}' 2>/dev/null | grep -qE "new-api|mysql|redis|nginx"; then
        print_success "检测到 NewAPI 相关容器"
        
        # 检查容器状态
        for container in new-api mysql redis nginx npm; do
            if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "^${container}$"; then
                print_success "容器 $container 正在运行"
            elif docker ps -a --format '{{.Names}}' 2>/dev/null | grep -q "^${container}$"; then
                print_warn "容器 $container 已停止"
            fi
        done
    else
        print_info "未检测到 NewAPI 容器（正常，未部署或已卸载）"
    fi
}

# 健康检查
health_check() {
    print_header "健康检查"
    
    if [[ -f "modules/monitor/health.sh" ]]; then
        if bash modules/monitor/health.sh 2>&1 | grep -q "正常\|healthy\|running"; then
            print_success "健康检查通过"
        else
            print_warn "健康检查发现问题，请查看详细输出"
        fi
    else
        print_info "健康检查脚本不存在"
    fi
}

# 总结
print_summary() {
    print_header "检查总结"
    
    echo -e "总检查项: ${TOTAL_CHECKS}"
    echo -e "${GREEN}通过: ${PASSED_CHECKS}${NC}"
    echo -e "${RED}失败: ${FAILED_CHECKS}${NC}"
    echo -e "${YELLOW}警告: ${WARNINGS}${NC}"
    echo ""
    
    if [[ $FAILED_CHECKS -eq 0 ]] && [[ $WARNINGS -eq 0 ]]; then
        echo -e "${GREEN}🎉 所有检查通过！${NC}"
        return 0
    elif [[ $FAILED_CHECKS -eq 0 ]]; then
        echo -e "${YELLOW}✅ 检查通过，但有警告信息${NC}"
        return 0
    else
        echo -e "${RED}❌ 检查失败，请修复上述问题${NC}"
        return 1
    fi
}

# 主函数
main() {
    echo ""
    echo -e "${BLUE}╔══════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║    NewAPI Tools V2.0 部署验证脚本          ║${NC}"
    echo -e "${BLUE}║    版本: 2.0 | 日期: $(date +%Y-%m-%d)               ║${NC}"
    echo -e "${BLUE}╚══════════════════════════════════════════════╝${NC}"
    
    # 执行检查
    check_docker
    check_permissions
    check_scripts
    check_config
    security_check
    check_containers
    health_check
    check_shellcheck
    
    # 输出总结
    print_summary
}

# 执行主函数
main "$@"
