#!/usr/bin/env python3
"""
上传文件到远程服务器并运行单元测试（修复版）
"""
import paramiko
import os
from pathlib import Path
import tempfile

# 连接配置
HOST = '10.100.10.36'
PORT = 22
USER = 'root'
PASS = '123qwe'
REMOTE_DIR = '/tmp/newapi-tools-verify'

# 本地项目目录
LOCAL_DIR = Path(__file__).parent

def connect_ssh():
    """建立SSH连接"""
    ssh = paramiko.SSHClient()
    ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    ssh.connect(HOST, port=PORT, username=USER, password=PASS, timeout=10)
    return ssh

def recursive_mkdir(sftp, remote_dir):
    """递归创建远程目录"""
    dirs = []
    d = remote_dir
    while d and d != '/':
        dirs.append(d)
        d = str(Path(d).parent)
    dirs.reverse()
    
    for d in dirs:
        try:
            sftp.stat(d)
        except FileNotFoundError:
            try:
                sftp.mkdir(d)
            except Exception:
                pass

def main():
    print("=== 上传文件到远程服务器 ===", flush=True)
    
    # 测试SSH连接
    print("正在连接SSH...", flush=True)
    try:
        ssh = connect_ssh()
        print("SSH连接成功!", flush=True)
    except Exception as e:
        print(f"SSH连接失败: {e}", flush=True)
        return
    
    try:
        print("打开SFTP...", flush=True)
        sftp = ssh.open_sftp()
        print("SFTP打开成功!", flush=True)
    except Exception as e:
        print(f"SFTP打开失败: {e}", flush=True)
        ssh.close()
        return
    ssh.exec_command(f'rm -rf {REMOTE_DIR}')
    ssh.exec_command(f'mkdir -p {REMOTE_DIR}')
    
    # 收集所有需要上传的文件
    files_to_upload = []
    for pattern in ['*.sh', '*.md', 'config/*', 'docker-compose*', 'logs/*', 'release/*', 'scripts/*', 'test/*', '.system_info']:
        if '/' in pattern:
            # 通配符模式
            for f in LOCAL_DIR.rglob(pattern.split('/')[-1]):
                if 'node_modules' not in str(f) and f.is_file():
                    if f not in files_to_upload:
                        files_to_upload.append(f)
        else:
            # 后缀模式
            for f in LOCAL_DIR.rglob(pattern):
                if 'node_modules' not in str(f) and f.is_file():
                    if f not in files_to_upload:
                        files_to_upload.append(f)
    
    print(f"找到 {len(files_to_upload)} 个文件待上传")
    
    # 先创建所有远程目录
    print("创建远程目录结构...")
    remote_dirs = set()
    for local_file in files_to_upload:
        rel_path = str(local_file.relative_to(LOCAL_DIR)).replace('\\', '/')
        remote_file = f"{REMOTE_DIR}/{rel_path}"
        remote_dir = str(Path(remote_file).parent)
        remote_dirs.add(remote_dir)
    
    for d in sorted(remote_dirs):
        recursive_mkdir(sftp, d)
    
    # 上传所有文件
    print("上传文件...")
    success_count = 0
    for local_file in files_to_upload:
        rel_path = str(local_file.relative_to(LOCAL_DIR)).replace('\\', '/')
        remote_file = f"{REMOTE_DIR}/{rel_path}"
        
        # 读取文件内容，转换换行符
        content = local_file.read_bytes()
        content = content.replace(b'\r\n', b'\n')
        
        # 写入临时文件（Windows）
        with tempfile.NamedTemporaryFile(mode='wb', delete=False, suffix=f'_{local_file.name}') as f:
            f.write(content)
            temp_file = f.name
        
        # 上传
        try:
            sftp.put(temp_file, remote_file)
            ssh.exec_command(f'chmod +x {remote_file} 2>/dev/null')
            success_count += 1
            if success_count % 10 == 0:
                print(f"  已上传 {success_count}/{len(files_to_upload)}")
        except Exception as e:
            print(f"  失败: {rel_path} - {e}")
        finally:
            os.unlink(temp_file)
    
    print(f"\n✅ 上传完成: {success_count}/{len(files_to_upload)} 个文件")
    
    # 验证目录结构
    print("\n验证远程目录结构:")
    stdin, stdout, stderr = ssh.exec_command(f'find {REMOTE_DIR} -name "*.sh" | head -20')
    print(stdout.read().decode('utf-8', errors='replace'))
    
    # 运行单元测试
    print("\n=== 运行单元测试 ===")
    cmd = f'cd {REMOTE_DIR}/test && bash unit-test.sh 2>&1'
    stdin, stdout, stderr = ssh.exec_command(cmd, timeout=120)
    output = stdout.read().decode('utf-8', errors='replace')
    
    print(output)
    
    # 检查失败项
    if '失败' in output or 'FAILED' in output:
        print("\n=== 失败的测试 ===")
        for line in output.split('\n'):
            if '✗' in line or 'FAILED' in line or '失败' in line:
                print(f"  {line.strip()}")
    
    sftp.close()
    ssh.close()

if __name__ == '__main__':
    try:
        main()
    except Exception as e:
        import traceback
        print(f"错误: {e}", flush=True)
        traceback.print_exc()
        sys.exit(1)
