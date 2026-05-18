#!/usr/bin/env python3
"""用正确 Windows 路径运行单元测试"""
import subprocess, os

# 直接拼 Windows 全路径，绕开所有 shell 转义
script = r"D:\Users\Vincent\Desktop\newapi-tools\newapi-tools\run_test.py"
print(f"运行: {script}")

result = subprocess.run(
    ['python.exe', script],
    capture_output=True, text=True, timeout=180
)
print(f"退出码: {result.returncode}")
# 只显示摘要行
for line in result.stdout.split('\n'):
    if any(k in line for k in ['测试摘要', '总测试数', '失败:', '通过率', '✗', '✓']):
        print(line.strip())

if result.returncode == 0:
    print('\n✓ 全部测试通过！')
elif result.returncode == 124:
    print('\n⚠ 超时')
else:
    print(f'\n✗ 失败，退出码: {result.returncode}')
    # 显示末尾 500 字符
    print('末尾输出:')
    print(result.stdout[-500:])
