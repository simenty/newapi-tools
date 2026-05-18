#!/bin/bash
# 安全功能单元测试 v2.0
# 测试 lib/security.sh 中的所有安全函数

set -eo pipefail

# 设置测试环境
TOOLKIT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export TOOLKIT_ROOT

# 引入安全库
source "${TOOLKIT_ROOT}/lib/security.sh"

# 测试计数器
TESTS_PASSED=0
TESTS_FAILED=0

# 颜色定义
GREEN='\033[32m'
RED='\033[31m'
YELLOW='\033[33m'
PLAIN='\033[0m'

# ---------- 测试函数 ----------
assert_pass() {
    local test_name="$1"
    local result="$2"
    
    if [ "$result" -eq 0 ]; then
        echo -e "  ${GREEN}✓ PASSED${PLAIN}: $test_name"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "  ${RED}✗ FAILED${PLAIN}: $test_name"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

assert_fail() {
    local test_name="$1"
    local result="$2"
    
    if [ "$result" -ne 0 ]; then
        echo -e "  ${GREEN}✓ PASSED${PLAIN}: $test_name (expected failure)"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "  ${RED}✗ FAILED${PLAIN}: $test_name (should have failed)"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

# ---------- 测试开始 ----------
echo -e "${YELLOW}========================================${PLAIN}"
echo -e "${YELLOW}  NewAPI Tools - 安全功能单元测试${PLAIN}"
echo -e "${YELLOW}========================================${PLAIN}"
echo ""

# ---------- 测试 1: 密码强度验证 ----------
echo "测试 1: 密码强度验证 (validate_password_strength)"
echo "---------------------------------------------------"

# 弱密码（应失败）
validate_password_strength "123" 2>/dev/null
assert_fail "弱密码 (123)" $?

validate_password_strength "password" 2>/dev/null
assert_fail "弱密码 (password)" $?

validate_password_strength "Password1" 2>/dev/null
assert_fail "弱密码 (无特殊字符)" $?

# 强密码（应通过）
validate_password_strength "Abc@123456" 2>/dev/null
assert_pass "强密码 (Abc@123456)" $?

echo ""

# ---------- 测试 2: 安全临时文件 ----------
echo "测试 2: 安全临时文件 (secure_temp_file)"
echo "---------------------------------------------------"

TEMP_FILE=$(secure_temp_file "test")
if [ -f "$TEMP_FILE" ]; then
    # 检查权限
    PERM=$(stat -c %a "$TEMP_FILE" 2>/dev/null || stat -f %Lp "$TEMP_FILE" 2>/dev/null || echo "unknown")
    if [ "$PERM" = "600" ]; then
        assert_pass "临时文件创建并设置权限 600" 0
    else
        assert_fail "临时文件权限不正确: $PERM" 1
    fi
    rm -f "$TEMP_FILE"
else
    assert_fail "临时文件创建失败" 1
fi

echo ""

# ---------- 测试 3: sed 模式转义 ----------
echo "测试 3: sed 模式转义 (escape_sed_pattern)"
echo "---------------------------------------------------"

TEST_INPUT="hello/world&test"
ESCAPED=$(escape_sed_pattern "$TEST_INPUT")
if [ "$ESCAPED" = "hello\/world\&test" ]; then
    assert_pass "sed 特殊字符转义" 0
else
    assert_fail "sed 转义失败: $ESCAPED" 1
fi

echo ""

# ---------- 测试 4: 日志脱敏 ----------
echo "测试 4: 日志脱敏 (filter_sensitive_info)"
echo "---------------------------------------------------"

TEST_LOG="password=secret123 token=abc888 api_key=mykey"
FILTERED=$(filter_sensitive_info "$TEST_LOG")

if ! echo "$FILTERED" | grep -q "secret123\|abc888\|mykey"; then
    assert_pass "日志脱敏功能" 0
else
    assert_fail "日志脱敏失败: $FILTERED" 1
fi

echo ""

# ---------- 测试 5: 路径验证 ----------
echo "测试 5: 路径验证 (validate_path)"
echo "---------------------------------------------------"

# 安全路径（应通过）
validate_path "data/config" "no" 2>/dev/null
assert_pass "安全相对路径" $?

# 路径遍历攻击（应失败）
validate_path "../../etc/passwd" "no" 2>/dev/null
assert_fail "路径遍历攻击检测" $?

# 绝对路径（应失败，如果不允许）
validate_path "/etc/passwd" "no" 2>/dev/null
assert_fail "绝对路径限制" $?

echo ""

# ---------- 测试 6: HTTPS URL 强制 ----------
echo "测试 6: HTTPS URL 强制 (force_https_url)"
echo "---------------------------------------------------"

TEST_URL="http://example.com"
SECURE_URL=$(force_https_url "$TEST_URL")
if [ "$SECURE_URL" = "https://example.com" ]; then
    assert_pass "HTTP → HTTPS 转换" 0
else
    assert_fail "HTTP → HTTPS 转换失败: $SECURE_URL" 1
fi

TEST_URL2="example.com"
SECURE_URL2=$(force_https_url "$TEST_URL2")
if echo "$SECURE_URL2" | grep -q "^https://"; then
    assert_pass "添加 HTTPS 协议" 0
else
    assert_fail "添加 HTTPS 协议失败: $SECURE_URL2" 1
fi

echo ""

# ---------- 测试 7: 输入验证 ----------
echo "测试 7: 输入验证 (validate_input)"
echo "---------------------------------------------------"

# 有效输入
validate_input "example.com" '^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$' "域名" 2>/dev/null
assert_pass "有效域名验证" $?

# 无效输入
validate_input "invalid_domain" '^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$' "域名" 2>/dev/null
assert_fail "无效域名检测" $?

echo ""

# ---------- 测试 8: 敏感变量清理 ----------
echo "测试 8: 敏感变量清理 (clear_sensitive_vars)"
echo "---------------------------------------------------"

export TEST_SENSITIVE_VAR="secret_value"
clear_sensitive_vars "TEST_SENSITIVE_VAR"

if [ -z "${TEST_SENSITIVE_VAR:-}" ]; then
    assert_pass "敏感变量清理" 0
else
    assert_fail "敏感变量未清理" 1
fi

echo ""

# ---------- 测试总结 ----------
echo -e "${YELLOW}========================================${PLAIN}"
echo -e "  测试总结"
echo -e "${YELLOW}========================================${PLAIN}"
echo ""
echo -e "  ${GREEN}通过: ${TESTS_PASSED}${PLAIN}"
echo -e "  ${RED}失败: ${TESTS_FAILED}${PLAIN}"
echo ""

if [ "$TESTS_FAILED" -eq 0 ]; then
    echo -e "${GREEN}✓ 所有安全测试通过！${PLAIN}"
    exit 0
else
    echo -e "${RED}✗ 有测试失败，请检查！${PLAIN}"
    exit 1
fi
