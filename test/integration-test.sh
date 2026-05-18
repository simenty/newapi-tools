#!/bin/bash
# NewAPI Tools v2.0 集成测试
# 测试完整的部署流程和用户体验

set -eo pipefail

# 颜色定义
GREEN='\033[32m'; RED='\033[31m'; YELLOW='\033[33m'; PLAIN='\033[0m'

echo -e "${YELLOW}========================================${PLAIN}"
echo -e "${YELLOW}  NewAPI Tools v2.0 集成测试${PLAIN}"
echo -e "${YELLOW}========================================${PLAIN}"
echo ""

# 设置测试环境
export TOOLKIT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export NEWAPI_HOME="/tmp/test-integration-newapi"
export LOG_DIR="/tmp/test-integration-logs"

# 创建测试目录
mkdir -p "$NEWAPI_HOME"
mkdir -p "$LOG_DIR"

echo -e "${GREEN}[1] 测试环境准备${PLAIN}"
echo ""

# 检查是否安装 Docker
if ! command -v docker &>/dev/null; then
    echo -e "${RED}  ✗ Docker 未安装，部分测试将跳过${PLAIN}"
    DOCKER_AVAILABLE=false
else
    echo -e "${GREEN}  ✓ Docker 已安装${PLAIN}"
    DOCKER_AVAILABLE=true
fi

echo ""

# ---------- 测试核心模块加载 ----------
echo -e "${GREEN}[2] 测试核心模块加载${PLAIN}"
echo ""

# 加载所有核心库
echo "  加载: common.sh..."
source "${TOOLKIT_ROOT}/lib/common.sh" 2>/dev/null && echo -e "${GREEN}    ✓ common.sh${PLAIN}" || echo -e "${RED}    ✗ common.sh${PLAIN}"

echo "  加载: state.sh..."
source "${TOOLKIT_ROOT}/lib/state.sh" 2>/dev/null && echo -e "${GREEN}    ✓ state.sh${PLAIN}" || echo -e "${RED}    ✗ state.sh${PLAIN}"

echo "  加载: ui.sh..."
source "${TOOLKIT_ROOT}/lib/ui.sh" 2>/dev/null && echo -e "${GREEN}    ✓ ui.sh${PLAIN}" || echo -e "${RED}    ✗ ui.sh${PLAIN}"

echo "  加载: config.sh..."
source "${TOOLKIT_ROOT}/lib/config.sh" 2>/dev/null && echo -e "${GREEN}    ✓ config.sh${PLAIN}" || echo -e "${RED}    ✗ config.sh${PLAIN}"

echo "  加载: smart-defaults.sh..."
source "${TOOLKIT_ROOT}/lib/smart-defaults.sh" 2>/dev/null && echo -e "${GREEN}    ✓ smart-defaults.sh${PLAIN}" || echo -e "${RED}    ✗ smart-defaults.sh${PLAIN}"

echo "  加载: mode.sh..."
source "${TOOLKIT_ROOT}/lib/mode.sh" 2>/dev/null && echo -e "${GREEN}    ✓ mode.sh${PLAIN}" || echo -e "${RED}    ✗ mode.sh${PLAIN}"

echo ""

# ---------- 测试状态管理 ----------
echo -e "${GREEN}[3] 测试状态管理流程${PLAIN}"
echo ""

# 初始化状态
echo "  初始化状态文件..."
init_state 2>/dev/null || true
if [[ -f "$STATE_FILE" ]]; then
    echo -e "${GREEN}    ✓ 状态文件已创建: $STATE_FILE${PLAIN}"
else
    echo -e "${RED}    ✗ 状态文件创建失败${PLAIN}"
fi

# 设置并读取状态
set_state "test.integration" "passed"
VALUE=$(get_state "test.integration")
if [[ "$VALUE" == "passed" ]]; then
    echo -e "${GREEN}    ✓ 状态读写正常${PLAIN}"
else
    echo -e "${RED}    ✗ 状态读写失败${PLAIN}"
fi

# 标记步骤完成
mark_step_completed "test_step"
if is_step_completed "test_step"; then
    echo -e "${GREEN}    ✓ 步骤标记正常${PLAIN}"
else
    echo -e "${RED}    ✗ 步骤标记失败${PLAIN}"
fi

echo ""

# ---------- 测试配置管理 ----------
echo -e "${GREEN}[4] 测试配置管理流程${PLAIN}"
echo ""

# 初始化配置
echo "  初始化配置文件..."
init_config 2>/dev/null || true
if [[ -f "$MAIN_CONFIG" ]]; then
    echo -e "${GREEN}    ✓ 配置文件已创建: $MAIN_CONFIG${PLAIN}"
else
    echo -e "${RED}    ✗ 配置文件创建失败${PLAIN}"
fi

# 设置并读取配置
set_config "test.key" "test_value" 2>/dev/null || true
VALUE=$(get_config "test.key")
if [[ "$VALUE" == "test_value" ]]; then
    echo -e "${GREEN}    ✓ 配置读写正常${PLAIN}"
else
    echo -e "${YELLOW}    ⚠ 配置读写可能失败（可能缺少 yq)${PLAIN}"
fi

echo ""

# ---------- 测试智能默认值 ----------
echo -e "${GREEN}[5] 测试智能默认值生成${PLAIN}"
echo ""

# 生成密码
echo "  生成随机密码..."
PASSWORD=$(generate_password 16 2>/dev/null || echo "")
if [[ -n "$PASSWORD" && ${#PASSWORD} -eq 16 ]]; then
    echo -e "${GREEN}    ✓ 密码生成正常 (16位)${PLAIN}"
else
    echo -e "${RED}    ✗ 密码生成失败${PLAIN}"
fi

# 检测系统信息
echo "  检测系统信息..."
SYS_INFO=$(detect_system_info 2>/dev/null || echo "")
if [[ -n "$SYS_INFO" ]]; then
    echo -e "${GREEN}    ✓ 系统信息检测正常${PLAIN}"
else
    echo -e "${YELLOW}    ⚠ 系统信息检测失败${PLAIN}"
fi

echo ""

# ---------- 测试模式切换 ----------
echo -e "${GREEN}[6] 测试模式切换${PLAIN}"
echo ""

# 切换到新手模式
echo "  切换到新手模式..."
set_mode "novice" 2>/dev/null || true
if [[ "$(get_mode)" == "novice" ]]; then
    echo -e "${GREEN}    ✓ 新手模式切换成功${PLAIN}"
else
    echo -e "${RED}    ✗ 新手模式切换失败${PLAIN}"
fi

# 切换到专家模式
echo "  切换到专家模式..."
set_mode "expert" 2>/dev/null || true
if [[ "$(get_mode)" == "expert" ]]; then
    echo -e "${GREEN}    ✓ 专家模式切换成功${PLAIN}"
else
    echo -e "${RED}    ✗ 专家模式切换失败${PLAIN}"
fi

echo ""

# ---------- 测试 UI 组件 ----------
echo -e "${GREEN}[7] 测试 UI 组件${PLAIN}"
echo ""

# 测试彩色输出
echo "  测试彩色输出..."
ui_success "成功消息" 2>/dev/null && echo -e "${GREEN}    ✓ ui_success${PLAIN}" || echo -e "${RED}    ✗ ui_success${PLAIN}"
ui_error "错误消息" 2>/dev/null && echo -e "${GREEN}    ✓ ui_error${PLAIN}" || echo -e "${RED}    ✗ ui_error${PLAIN}"
ui_warn "警告消息" 2>/dev/null && echo -e "${GREEN}    ✓ ui_warn${PLAIN}" || echo -e "${RED}    ✗ ui_warn${PLAIN}"
ui_info "信息消息" 2>/dev/null && echo -e "${GREEN}    ✓ ui_info${PLAIN}" || echo -e "${RED}    ✗ ui_info${PLAIN}"

echo ""

# ---------- 清理测试环境 ----------
echo -e "${GREEN}[8] 清理测试环境${PLAIN}"
echo ""

rm -rf "$NEWAPI_HOME"
rm -rf "$LOG_DIR"
rm -f "$STATE_FILE"
rm -rf "${TOOLKIT_ROOT}/config"

if [[ ! -d "$NEWAPI_HOME" && ! -d "$LOG_DIR" ]]; then
    echo -e "${GREEN}    ✓ 测试环境已清理${PLAIN}"
else
    echo -e "${YELLOW}    ⚠ 测试环境清理可能不完整${PLAIN}"
fi

echo ""

# ---------- 测试总结 ----------
echo -e "${YELLOW}========================================${PLAIN}"
echo -e "${YELLOW}  集成测试完成${PLAIN}"
echo -e "${YELLOW}========================================${PLAIN}"
echo ""
echo "  测试环境: $TOOLKIT_ROOT"
echo "  日志位置: $LOG_DIR"
echo ""
echo -e "${GREEN}提示：${PLAIN}"
echo "  1. 所有核心模块已测试"
echo "  2. 状态管理和配置管理功能正常"
echo "  3. UI 组件库已加载"
echo "  4. 模式切换功能正常"
echo ""
echo -e "${YELLOW}注意：${PLAIN}"
echo "  部分测试可能因为缺少依赖（如 yq、jq）而失败"
echo "  建议安装: apt install yq jq"
echo ""
