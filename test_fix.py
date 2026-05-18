#!/usr/bin/env python3
"""快速测试：验证 common.sh 加载不再报 log_debug 错误"""
import paramiko
import os

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect('10.100.10.36', port=22, username='root', password='123qwe', timeout=15)
print('✓ SSH 连接成功')

# 上传 common.sh
sftp = client.open_sftp()
local_file = 'lib/common.sh'
remote_file = '/tmp/newapi-tools-verify/lib/common.sh'
client.exec_command(f'mkdir -p /tmp/newapi-tools-verify/lib')

with open(local_file, 'rb') as f:
    content = f.read()
content = content.replace(b'\r\n', b'\n').replace(b'\r', b'\n')
import tempfile
with tempfile.NamedTemporaryFile(delete=False, suffix='.tmp') as tmp:
    tmp.write(content)
    tmp_path = tmp.name
sftp.put(tmp_path, remote_file)
os.unlink(tmp_path)
sftp.close()
print(f'✓ 已上传 {local_file}')

# 测试：source common.sh 并捕获错误
print('\n>>> 测试加载 common.sh...')
stdin, stdout, stderr = client.exec_command(
    f'bash -c "source {remote_file} 2>&1"; echo "EXIT_CODE:$?"', timeout=30
)
out = stdout.read().decode('utf-8', errors='replace').strip()
err_out = stderr.read().decode('utf-8', errors='replace').strip()
rc = stdout.channel.recv_exit_status()

print(f'退出码: {rc}')
if out:
    print(f'输出:\n{out[:1000]}')
if err_out:
    print(f'stderr:\n{err_out[:500]}')

if 'command not found' in out or 'command not found' in err_out:
    print('\n✗ 仍有 command not found 错误！')
else:
    print('\n✓ 没有 command not found 错误，修复成功！')

# 测试：实际运行 newapi-tools.sh --help
print('\n>>> 测试运行 newapi-tools.sh --help...')
stdin2, stdout2, stderr2 = client.exec_command(
    f'cd /tmp/newapi-tools-verify && bash newapi-tools.sh --help 2>&1 | head -30', timeout=30
)
out2 = stdout2.read().decode('utf-8', errors='replace').strip()
rc2 = stdout2.channel.recv_exit_status()
print(f'退出码: {rc2}')
if out2:
    print(f'输出:\n{out2[:2000]}')

if 'log_debug: command not found' in out2:
    print('\n✗ 仍有 log_debug 错误！')
else:
    print('\n✓ log_debug 错误已修复！')

client.close()
print('\n完成')
