#!/usr/bin/env python3
"""上传修复后的文件到远程并运行单元测试"""
import paramiko, os, tempfile

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect('10.100.10.36', port=22, username='root', password='123qwe', timeout=15)
print('SSH 连接成功')

sftp = client.open_sftp()

files_to_upload = ['lib/state.sh', 'lib/common.sh', 'test/unit-test.sh']
for f in files_to_upload:
    with open(f, 'rb') as fp:
        content = fp.read()
    content = content.replace(b'\r\n', b'\n').replace(b'\r', b'\n')
    remote = '/tmp/newapi-tools-verify/' + f.replace('\\', '/')
    parts = remote.rsplit('/', 1)
    if len(parts) == 2:
        client.exec_command(f'mkdir -p {parts[0]}')
    with tempfile.NamedTemporaryFile(delete=False, suffix='.tmp') as tmp:
        tmp.write(content)
        tmp_path = tmp.name
    sftp.put(tmp_path, remote)
    os.unlink(tmp_path)
    print(f'  已上传 {f}')

sftp.close()

# 运行单元测试（用 timeout 限制时长）
print('\n>>> 运行单元测试（最多 120 秒）...')
stdin, stdout, stderr = client.exec_command(
    'cd /tmp/newapi-tools-verify && timeout 120 bash test/unit-test.sh 2>&1',
    timeout=130
)
out = stdout.read().decode('utf-8', errors='replace')
rc = stdout.channel.recv_exit_status()
print(f'退出码: {rc}')
print('最后 3000 字符输出:')
print(out[-3000:] if len(out) > 3000 else out)
if rc == 124:
    print('\n⚠ 测试超时（timeout），仍有卡死问题！')
elif rc == 0:
    print('\n✓ 测试全部通过（退出码 0）')
else:
    print(f'\n✗ 测试失败（退出码 {rc}）')

client.close()
print('\n完成')
