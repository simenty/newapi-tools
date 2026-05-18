#!/usr/bin/env python3
"""在远程服务器上手动跑失败的测试项，拿到真实报错"""
import paramiko

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect('10.100.10.36', port=22, username='root', password='123qwe', timeout=15)

# 手动 source modules/deploy/install.sh，看报什么错
cmd = """bash -c "
set -x
cd /tmp/newapi-tools-verify
export TOOLKIT_ROOT=/tmp/newapi-tools-verify
source modules/deploy/install.sh 2>&1
echo EXIT_CODE:\$?
" 2>&1"""
print('>>> 手动 source modules/deploy/install.sh...')
stdin, stdout, stderr = client.exec_command(cmd, timeout=30)
out = stdout.read().decode('utf-8', errors='replace')
rc = stdout.channel.recv_exit_status()
print('退出码:', rc)
# 只显示最后 2000 字符
lines = out.split('\n')
for line in lines[-40:]:
    print(line)
client.close()
print('完成')
