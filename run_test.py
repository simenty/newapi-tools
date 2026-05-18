#!/usr/bin/env python3
"""上传所有必要的 .sh 文件到远程并运行单元测试"""
import paramiko, os, glob, tempfile

LOCAL_DIR = r"D:\Users\Vincent\Desktop\newapi-tools\newapi-tools"

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect('10.100.10.36', port=22, username='root', password='123qwe', timeout=15)
print('SSH 连接成功')

sftp = client.open_sftp()

# 收集所有需要上传的 .sh 文件
patterns = [
    'install.sh',
    'newapi-tools.sh',
    'lib/*.sh',
    'modules/**/*.sh',
    'scripts/*.sh',
    'test/unit-test.sh',
]
remote_base = '/tmp/newapi-tools-verify'
uploaded = []

for pattern in patterns:
    full_pattern = os.path.join(LOCAL_DIR, pattern)
    for local_path in glob.glob(full_pattern, recursive=True):
        rel_path = os.path.relpath(local_path, LOCAL_DIR)
        remote_path = remote_base + '/' + rel_path.replace('\\', '/')
        remote_dir = os.path.dirname(remote_path)
        # 创建远程目录
        cmd = f'mkdir -p "{remote_dir}"'
        client.exec_command(cmd)
        # 读取并转换换行符
        with open(local_path, 'rb') as f:
            content = f.read()
        content = content.replace(b'\r\n', b'\n').replace(b'\r', b'\n')
        tmp = tempfile.NamedTemporaryFile(delete=False, suffix='.tmp')
        tmp.write(content)
        tmp.close()
        sftp.put(tmp.name, remote_path)
        os.unlink(tmp.name)
        uploaded.append(rel_path)

sftp.close()
print(f'已上传 {len(uploaded)} 个文件')

# 运行单元测试
print('\n>>> 运行单元测试（120 秒超时）...')
stdin, stdout, stderr = client.exec_command(
    f'cd {remote_base} && timeout 120 bash test/unit-test.sh 2>&1',
    timeout=130
)
out = stdout.read().decode('utf-8', errors='replace')
rc = stdout.channel.recv_exit_status()
print('退出码:', rc)

for line in out.split('\n'):
    if any(k in line for k in ['测试摘要', '总测试数', '失败:', '通过率', '✗ ', '━━━━']):
        print(line.strip())

if rc == 0:
    print('\n\033[32m✓ 全部测试通过！\033[0m')
elif rc == 124:
    print('\n\033[33m⚠ 测试超时（timeout）\033[0m')
else:
    print(f'\n\033[31m✗ 有测试失败，退出码: {rc}\033[0m')
    print('末尾 500 字符:')
    print(out[-500:])

client.close()
print('\n完成')
