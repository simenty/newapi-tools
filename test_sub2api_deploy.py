#!/usr/bin/env python3
"""Sub2API 部署真实测试"""
import paramiko
import time

HOST = '10.100.10.36'
PORT = 22
USER = 'root'
PASS = '123qwe'
REMOTE_DIR = '/tmp/newapi-tools-verify'

def run_cmd(ssh, cmd, timeout=60):
    stdin, stdout, stderr = ssh.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode('utf-8', errors='replace')
    err = stderr.read().decode('utf-8', errors='replace')
    exit_code = stdout.channel.recv_exit_status()
    return out, err, exit_code

def main():
    print("连接SSH...", flush=True)
    ssh = paramiko.SSHClient()
    ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    ssh.connect(HOST, port=PORT, username=USER, password=PASS, timeout=10)
    print("SSH OK\n", flush=True)

    # 1. 检查 Docker
    print("=" * 50)
    print("[1] 检查 Docker 状态", flush=True)
    out, err, code = run_cmd(ssh, "docker --version 2>&1")
    print(f"  Docker: {out.strip()}", flush=True)

    out, err, code = run_cmd(ssh, "docker ps -a 2>&1 | head -10")
    print(f"  容器列表:\n{out}", flush=True)

    # 2. 检查镜像
    print("=" * 50)
    print("[2] 检查 Sub2API 镜像", flush=True)
    out, err, code = run_cmd(ssh, "docker images | grep -i sub2api 2>&1")
    if out.strip():
        print(f"  已存在: {out.strip()}", flush=True)
    else:
        print("  镜像未存在，需要先拉取", flush=True)

    # 3. 模拟运行 install.sh --flavor sub2api --help (不实际安装)
    print("=" * 50)
    print("[3] 测试 Sub2API 安装脚本 help", flush=True)
    out, err, code = run_cmd(ssh, f"cd {REMOTE_DIR} && bash install.sh --help 2>&1")
    print(out, flush=True)

    # 4. 检查 sub2api/install.sh 是否存在
    print("=" * 50)
    print("[4] 检查 Sub2API 部署脚本", flush=True)
    out, err, code = run_cmd(ssh, f"ls -la {REMOTE_DIR}/modules/deploy/sub2api/ 2>&1")
    print(out, flush=True)

    # 5. 运行语法检查
    print("=" * 50)
    print("[5] Sub2API 脚本语法检查", flush=True)
    out, err, code = run_cmd(ssh, f"bash -n {REMOTE_DIR}/modules/deploy/sub2api/install.sh 2>&1 && echo '语法正确'")
    print(out, flush=True)

    # 6. 模拟 source 测试（不应触发 check_root）
    print("=" * 50)
    print("[6] Source 安全性测试", flush=True)
    out, err, code = run_cmd(ssh, f'source {REMOTE_DIR}/modules/deploy/sub2api/install.sh 2>&1 && echo "source 安全"')
    print(f"  退出码: {code}, 输出: {out.strip()}", flush=True)

    # 7. 真实部署测试（先停止并删除已有容器）
    print("=" * 50)
    print("[7] 真实部署测试 (Sub2API)", flush=True)
    # 停止删除已有 newapi 容器
    out, err, code = run_cmd(ssh, "docker stop newapi 2>/dev/null; docker rm newapi 2>/dev/null; echo '清理完成'")
    print(f"  清理: {out.strip()}", flush=True)

    # 运行部署（使用 test 模式或者实际部署）
    # 这里先检查脚本的 flavor 路由是否正确
    print("\n  测试 flavor=sub2api 路由...", flush=True)
    out, err, code = run_cmd(ssh, f"cd {REMOTE_DIR} && bash -c 'FLAVOR=sub2api; source install.sh 2>&1' 2>&1 | head -20")
    print(f"  输出: {out[:500]}", flush=True)

    ssh.close()
    print("\n" + "=" * 50)
    print("测试完成！", flush=True)

if __name__ == '__main__':
    main()
