#!/usr/bin/env python3
"""newapi-tools 安全审计脚本 v2.2"""
import os
import re
from pathlib import Path

TOOLKIT_ROOT = Path(__file__).parent.resolve()
SKIP_DIRS = {'test', '__pycache__', '.git', 'release', 'logs', '.workbuddy'}

print("=" * 50)
print("  newapi-tools 安全审计 v2.2")
print("=" * 50)
print()

pass_count = 0
fail_count = 0
warn_count = 0

def log_pass(msg):
    global pass_count
    print(f"  [PASS] {msg}")
    pass_count += 1

def log_fail(msg):
    global fail_count
    print(f"  [FAIL] {msg}")
    fail_count += 1

def log_warn(msg):
    global warn_count
    print(f"  [WARN] {msg}")
    warn_count += 1

# 收集所有 Shell 脚本
sh_files = []
for root, dirs, files in os.walk(TOOLKIT_ROOT):
    dirs[:] = [d for d in dirs if d not in SKIP_DIRS]
    for f in files:
        if f.endswith('.sh'):
            sh_files.append(Path(root) / f)

print(f"--- 审计范围: {len(sh_files)} 个 Shell 脚本 ---")
print()

# 1. set -eo pipefail 检查
print("--- 1. set -eo pipefail 检查 ---")
for sh in sh_files:
    content = sh.read_text(errors='ignore')
    if re.search(r'^set\s+-eo\s+pipefail', content, re.MULTILINE):
        log_pass(f"{sh.name}: 包含 set -eo pipefail")
    elif content.startswith('#!/'):
        log_warn(f"{sh.name}: 缺少 set -eo pipefail")

print()

# 2. BASH_SOURCE 守卫检查
print("--- 2. BASH_SOURCE 守卫检查 ---")
for sh in sh_files:
    content = sh.read_text(errors='ignore')
    if 'BASH_SOURCE[0]' in content:
        log_pass(f"{sh.name}: 包含 BASH_SOURCE 守卫")
    elif 'source "' in content or "source '" in content:
        log_warn(f"{sh.name}: 有 source 但可能缺少守卫")

print()

# 3. 敏感信息处理
print("--- 3. 敏感信息处理 ---")
for sh in sh_files:
    content = sh.read_text(errors='ignore')

    # 检查日志脱敏
    if 'desensitize' in content.lower():
        log_pass(f"{sh.name}: 包含日志脱敏")
        break

for sh in sh_files:
    content = sh.read_text(errors='ignore')
    # 检查 unset 敏感变量
    if re.search(r'unset\s+\w*(PASSWORD|PASS|SECRET|KEY)\w*', content, re.IGNORECASE):
        log_pass(f"{sh.name}: 包含敏感变量清理")
        break

print()

# 4. 权限设置检查
print("--- 4. 权限设置检查 ---")
for sh in sh_files:
    content = sh.read_text(errors='ignore')

    # 检查 chmod 600 或 400
    if re.search(r'chmod\s+[0-6][0-7][0-7]\s+.*\.(env|yml|yaml)', content):
        log_pass(f"{sh.name}: 设置安全文件权限")
        break

print()

# 5. HTTPS 检查
print("--- 5. 网络请求安全 ---")
for sh in sh_files:
    content = sh.read_text(errors='ignore')

    # 检查 -k (跳过证书验证)
    if re.search(r'curl\s+.*-k\b', content):
        log_fail(f"{sh.name}: 使用 curl -k 跳过证书验证")
        continue

    # 检查 --no-check-certificate
    if re.search(r'wget\s+.*--no-check-certificate', content):
        log_fail(f"{sh.name}: wget 跳过证书验证")
        continue

    # 检查 HTTPS 使用
    if 'https://' in content and 'curl' in content:
        log_pass(f"{sh.name}: 使用 HTTPS")
        break

print()

# 6. 命令注入风险检查
print("--- 6. 命令注入风险检查 ---")
for sh in sh_files:
    content = sh.read_text(errors='ignore')
    lines = content.split('\n')

    for i, line in enumerate(lines, 1):
        # 检查 eval 后跟变量
        if re.search(r'eval\s+\$\w+', line):
            log_warn(f"{sh.name}:{i}: eval + 未引用变量")
            continue

        # 检查 read 后直接 eval
        if 'read ' in line and 'eval' in content[content.find(line):content.find(line)+200]:
            log_warn(f"{sh.name}:{i}: read 后可能有 eval")

        # 检查未引用的命令替换
        if re.search(r'\$\(.*\)\s*[|;]', line) and '"' not in line[:line.find('$(')]:
            log_warn(f"{sh.name}:{i}: 未引用的命令替换")

print()

# 7. trap 错误处理
print("--- 7. trap 错误处理 ---")
for sh in sh_files:
    content = sh.read_text(errors='ignore')
    if 'trap ' in content:
        log_pass(f"{sh.name}: 包含 trap 处理")
        break

print()

# 8. Docker 相关安全
print("--- 8. Docker 安全检查 ---")
for sh in sh_files:
    content = sh.read_text(errors='ignore')

    # 检查容器端口绑定
    if re.search(r'"[^"]*:\d+:\d+"', content) and 'ports:' in content:
        # 检查是否绑定到 127.0.0.1
        if re.search(r'"127\.0\.0\.1:\d+:\d+"', content):
            log_pass(f"{sh.name}: 容器端口绑定到本地 (安全)")
            break
        elif re.search(r'"\d+:\d+"', content):
            log_warn(f"{sh.name}: 容器端口可能公开暴露")

print()

# 总结
print("=" * 50)
print("  安全审计结果汇总")
print("=" * 50)
print(f"  PASS: {pass_count}")
print(f"  FAIL: {fail_count}")
print(f"  WARN: {warn_count}")
print()

if fail_count > 0:
    print(f"  状态: 未通过 - 需要修复 {fail_count} 个失败项")
elif warn_count > 0:
    print(f"  状态: 警告 - 有 {warn_count} 个警告项需关注")
else:
    print("  状态: 通过 - 所有安全检查项均正常")
