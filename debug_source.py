#!/usr/bin/env python3
"""调试 install.sh source 行为"""
import paramiko

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect('10.100.10.36', port=22, username='root', password='123qwe', timeout=15)

# 把 install.sh 上传到远程 /tmp/install_test.sh
sftp = client.open_sftp()
with open('install.sh', 'rb') as f:
    content = f.read()
content = content.replace(b'\r\n', b'\n').replace(b'\r', b'\n')
import tempfile
tmp = tempfile.NamedTemporaryFile(delete=False, suffix='.tmp')
tmp.write(content)
tmp.close()
sftp.put(tmp.name, '/tmp/install_test.sh')
import os
os.unlink(tmp.name)
sftp.close()
print('已上传 install.sh -> /tmp/install_test.sh')

# 测试1：直接 source（看报错）
print('\n>>> 测试1：source /tmp/install_test.sh 2>&1')
stdin, stdout, stderr = client.exec_command(
    'bash -c "source /tmp/install_test.sh 2>&1; echo EXIT_CODE:$?"',
    timeout=15
)
out = stdout.read().decode('utf-8', errors='replace')
print('输出:', out[:1000])

# 测试2：在 subshell 里 source
print('\n>>> 测试2：( source /tmp/install_test.sh 2>/dev/null; echo OK ); echo $?')
stdin, stdout, stderr = client.exec_command(
    'bash -c "( source /tmp/install_test.sh 2>/dev/null; echo OK ); echo $?"',
    timeout=15
)
out = stdout.read().decode('utf-8', errors='replace')
print('输出:', out[:1000])

client.close()
print('\n完成')
