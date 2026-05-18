#!/usr/bin/env python3
"""上传 newapi-tools 到 Linux 测试机"""
import paramiko
import os
import tempfile

HOST = '10.100.10.36'
PORT = 22
USER = 'root'
PASS = '123qwe'
REMOTE_DIR = '/tmp/newapi-tools'
LOCAL_DIR = os.path.dirname(os.path.abspath(__file__))

SKIP_DIRS = {'node_modules', '__pycache__', '.git', '.workbuddy', 'release', 'logs'}
SKIP_FILES = {'.DS_Store', 'Thumbs.db'}

def collect_files():
    files = []
    for root, dirs, filenames in os.walk(LOCAL_DIR):
        dirs[:] = [d for d in dirs if d not in SKIP_DIRS]
        for f in filenames:
            if f in SKIP_FILES:
                continue
            if any(part in SKIP_DIRS for part in os.path.relpath(root, LOCAL_DIR).split(os.sep)):
                continue
            files.append(os.path.join(root, f))
    return files

def main():
    print("连接SSH...", flush=True)
    ssh = paramiko.SSHClient()
    ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    ssh.connect(HOST, port=PORT, username=USER, password=PASS, timeout=10)
    print("SSH OK", flush=True)

    sftp = ssh.open_sftp()

    # 清理远程目录并重新创建
    print("清理远程目录...", flush=True)
    ssh.exec_command(f'rm -rf {REMOTE_DIR}')
    ssh.exec_command(f'mkdir -p {REMOTE_DIR}')

    # 收集文件
    files = collect_files()
    print(f"找到 {len(files)} 个文件", flush=True)

    # 上传文件
    count = 0
    failed = 0
    for local_path in sorted(files):
        rel_path = os.path.relpath(local_path, LOCAL_DIR).replace('\\', '/')
        remote_path = f"{REMOTE_DIR}/{rel_path}"
        remote_dir = os.path.dirname(remote_path).replace('\\', '/')

        # 创建远程目录
        try:
            sftp.stat(remote_dir)
        except FileNotFoundError:
            ssh.exec_command(f'mkdir -p "{remote_dir}"')

        # 读取并转换换行符
        with open(local_path, 'rb') as f:
            content = f.read().replace(b'\r\n', b'\n')

        # 上传
        try:
            with tempfile.NamedTemporaryFile(mode='wb', delete=False) as tmp:
                tmp.write(content)
                tmp_path = tmp.name

            sftp.put(tmp_path, remote_path)
            ssh.exec_command(f'chmod +x "{remote_path}"')
            os.unlink(tmp_path)
            count += 1
            if count % 30 == 0:
                print(f"  已上传 {count}/{len(files)}", flush=True)
        except Exception as e:
            print(f"  上传失败: {rel_path} - {e}", flush=True)
            failed += 1

    print(f"上传完成: {count}/{len(files)} (失败 {failed})", flush=True)

    # 运行单元测试
    print("\n运行单元测试...", flush=True)
    stdin2, stdout2, stderr2 = ssh.exec_command(f'cd {REMOTE_DIR} && bash test/unit-test.sh 2>&1')
    output = stdout2.read().decode('utf-8', errors='ignore')
    print(output)
    
    return count, failed

if __name__ == "__main__":
    main()
