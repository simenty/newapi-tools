#!/bin/bash
# newapi-tools 安全审计脚本 v2.2
# 审计范围：命令注入、变量展开、权限问题、敏感信息处理

set -eo pipefail
TOOLKIT_ROOT="${1:-/tmp/newapi-tools-verify}"

echo "=========================================="
echo "  newapi-tools 安全审计 v2.2"
echo "=========================================="
echo ""

PASS_COUNT=0
FAIL_COUNT=0
WARN_COUNT=0

log_pass() { echo "  [PASS] $1"; ((PASS_COUNT++)); }
log_fail() { echo "  [FAIL] $1"; ((FAIL_COUNT++)); }
log_warn() { echo "  [WARN] $1"; ((WARN_COUNT++)); }

# 1. 检查未引用的变量展开（命令注入风险）
echo "--- 1. 命令注入风险检测 ---"
UNSAFE_PATTERNS=(
    'eval\s+\$'
    '\$\(.*\)'
    '`.*`'
)

for sh in $(find "$TOOLKIT_ROOT" -name "*.sh" -type f 2>/dev/null); do
    # 跳过测试脚本
    [[ "$sh" == */test/* ]] && continue

    # 检查 eval 后跟未引用的变量
    if grep -qP 'eval\s+\$\w+' "$sh" 2>/dev/null; then
        log_warn "$sh: 使用 eval + 未引用变量"
    fi

    # 检查双括号算术展开是否在双引号内
    # 允许: [[ "$var" =~ pattern ]], echo "$((a+b))"
    # 警告: eval "$cmd" 或 echo $((a+b)) 不带引号
    if grep -qP 'echo\s+\$\(\(' "$sh" 2>/dev/null; then
        # 检查是否在双引号内
        line=$(grep -nP 'echo\s+\$\(\(' "$sh" 2>/dev/null | head -1)
        if [[ "$line" =~ \"\$\(\( ]]; then
            log_pass "$sh: echo \$((...)) 在引号内"
        else
            log_warn "$sh: echo \$((...)) 可能缺少引号"
        fi
    fi
done

echo ""

# 2. 检查文件权限
echo "--- 2. 敏感文件权限检查 ---"
SENSITIVE_FILES=(
    ".env"
    ".env.example"
    "docker-compose.yml"
    "docker-compose.yaml"
)

for dir in "$TOOLKIT_ROOT" "$TOOLKIT_ROOT/.."; do
    for pattern in "${SENSITIVE_FILES[@]}"; do
        while IFS= read -r -d '' file; do
            # 获取文件权限（数字形式）
            perm=$(stat -c %a "$file" 2>/dev/null || stat -f %Lp "$file" 2>/dev/null || echo "???")
            if [[ "$perm" == "600" || "$perm" == "400" ]]; then
                log_pass "$file: 权限 $perm (安全)"
            elif [[ "$perm" == "6"* || "$perm" == "7"* ]]; then
                log_fail "$file: 权限 $perm (不安全，应为 600)"
            else
                log_warn "$file: 权限 $perm (需确认)"
            fi
        done < <(find "$dir" -name "$pattern" -type f -print0 2>/dev/null)
    done
done

echo ""

# 3. 检查密码/密钥生成安全性
echo "--- 3. 敏感信息处理检查 ---"
for sh in $(find "$TOOLKIT_ROOT" -name "*.sh" -type f 2>/dev/null); do
    [[ "$sh" == */test/* ]] && continue

    # 检查是否生成安全随机密码
    if grep -qP 'tr.*-dc.*/dev/urandom' "$sh" 2>/dev/null; then
        log_pass "$sh: 使用 /dev/urandom 生成随机数 (安全)"
    elif grep -qP 'tr.*-dc.*/dev/random' "$sh" 2>/dev/null; then
        log_warn "$sh: 使用 /dev/random 可能阻塞"
    fi

    # 检查是否有硬编码密码
    if grep -qPi 'password\s*=\s*["'"'"']*[a-zA-Z0-9]\{8,\}' "$sh" 2>/dev/null; then
        if grep -qP '(PASSWORD|PASS|API_KEY|SECRET)\s*=\s*["'"'"']*[A-Za-z0-9+/=]{20,}' "$sh" 2>/dev/null; then
            : # 跳过，这个可能是 API key 示例
        else
            grep -nPi 'password\s*=\s*["'"'"']*[a-zA-Z0-9]\{8,\}' "$sh" 2>/dev/null | head -1 | while read -r line; do
                log_warn "$sh:$line: 可能存在硬编码密码"
            done
        fi
    fi

    # 检查日志脱敏
    if grep -q '_desensitize_log\|desensitize' "$sh" 2>/dev/null; then
        log_pass "$sh: 包含日志脱敏函数"
    fi

    # 检查敏感变量是否 unset
    if grep -qP 'unset.*PASSWORD|unset.*SECRET|unset.*API_KEY' "$sh" 2>/dev/null; then
        log_pass "$sh: 包含敏感变量清理"
    fi
done

echo ""

# 4. 检查 set -eo pipefail 使用
echo "--- 4. 严格错误处理检查 ---"
for sh in $(find "$TOOLKIT_ROOT" -name "*.sh" -type f 2>/dev/null); do
    [[ "$sh" == */test/* ]] && continue

    if grep -qP '^set\s+-eo\s+pipefail' "$sh" 2>/dev/null; then
        log_pass "$sh: 包含 set -eo pipefail"
    elif grep -qP '^#!/bin/bash' "$sh" 2>/dev/null; then
        log_warn "$sh: 缺少 set -eo pipefail"
    fi
done

echo ""

# 5. 检查 BASH_SOURCE 守卫
echo "--- 5. Source 守卫检查 ---"
for sh in $(find "$TOOLKIT_ROOT" -name "*.sh" -type f 2>/dev/null); do
    [[ "$sh" == */test/* ]] && continue

    if grep -qP 'BASH_SOURCE\[0\]' "$sh" 2>/dev/null; then
        log_pass "$sh: 包含 BASH_SOURCE 守卫"
    fi
done

echo ""

# 6. 检查外部输入验证
echo "--- 6. 输入验证检查 ---"
for sh in $(find "$TOOLKIT_ROOT" -name "*.sh" -type f 2>/dev/null); do
    [[ "$sh" == */test/* ]] && continue

    # 检查 read 输入是否验证
    if grep -qP 'read\s+.*;.*eval|\$\(\s*read' "$sh" 2>/dev/null; then
        log_warn "$sh: read 后直接 eval 可能不安全"
    fi

    # 检查端口范围验证
    if grep -qP 'PORT.*\[.*[0-9]{5}' "$sh" 2>/dev/null; then
        log_warn "$sh: 可能有无效端口范围"
    fi
done

echo ""

# 7. 检查网络请求安全性
echo "--- 7. 网络请求安全检查 ---"
for sh in $(find "$TOOLKIT_ROOT" -name "*.sh" -type f 2>/dev/null); do
    [[ "$sh" == */test/* ]] && continue

    if grep -qP 'curl.*-k\b' "$sh" 2>/dev/null; then
        log_warn "$sh: 使用 curl -k 跳过证书验证"
    fi

    if grep -qP 'wget.*--no-check-certificate' "$sh" 2>/dev/null; then
        log_warn "$sh: wget 跳过证书验证"
    fi

    if grep -qP 'curl.*https' "$sh" 2>/dev/null; then
        log_pass "$sh: 使用 HTTPS"
    fi
done

echo ""

# 总结
echo "=========================================="
echo "  安全审计结果汇总"
echo "=========================================="
echo "  PASS: $PASS_COUNT"
echo "  FAIL: $FAIL_COUNT"
echo "  WARN: $WARN_COUNT"
echo ""

if [[ $FAIL_COUNT -gt 0 ]]; then
    echo "  状态: 未通过 - 需要修复 $FAIL_COUNT 个失败项"
    exit 1
elif [[ $WARN_COUNT -gt 0 ]]; then
    echo "  状态: 警告 - 有 $WARN_COUNT 个警告项需关注"
    exit 0
else
    echo "  状态: 通过 - 所有安全检查项均正常"
    exit 0
fi
