#!/usr/bin/env python3
"""远程 Linux 服务器验证脚本 - 使用 paramiko SSH 连接"""

import paramiko
import sys
import os
import glob

HOST = "10.100.10.36"
PORT = 22
USER = "root"
PASS = "123qwe"

def ssh_connect():
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    try:
        client.connect(HOST, port=PORT, username=USER, password=PASS, timeout=15)
        print(f"✓ SSH 连接成功: {USER}@{HOST}:{PORT}")
        return client
    except Exception as e:
        print(f"✗ SSH 连接失败: {e}")
        sys.exit(1)

def run_cmd(client, cmd, title="", timeout=120):
    if title:
        print(f"\n{'='*60}")
        print(f"  {title}")
        print(f"{'='*60}")
    stdin, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode('utf-8', errors='replace').strip()
    err = stderr.read().decode('utf-8', errors='replace').strip()
    rc = stdout.channel.recv_exit_status()
    if out:
        lines = out.split('\n')
        if len(lines) > 30:
            print('\n'.join(lines[:15]))
            print(f"... ({len(lines)-30} 行省略) ...")
            print('\n'.join(lines[-15:]))
        else:
            print(out)
    if err and rc != 0:
        # 只显示错误的最后几行
        err_lines = err.split('\n')
        print(f"[stderr]: {' '.join(err_lines[-3:])}")
    print(f"[退出码: {rc}]")
    return rc, out, err

def fix_apt_sources(client):
    """修复 Debian apt 源（从 cdrom 改为网络源）"""
    print("\n>>> 修复 apt 源（cdrom -> 网络）...")
    cmd = """cat > /etc/apt/sources.list << 'EOF'
deb http://deb.debian.org/debian bookworm main contrib non-free non-free-firmware
deb http://deb.debian.org/debian bookworm-updates main contrib non-free non-free-firmware
deb http://security.debian.org/debian-security bookworm-security main contrib non-free non-free-firmware
EOF
"""
    run_cmd(client, cmd, "写入网络 apt 源")
    run_cmd(client, "apt-get update -qq 2>&1 | tail -5", "apt-get update", timeout=120)

def sftp_upload_dir(sftp, local_base, remote_base, client, patterns):
    """上传匹配的文件到远程服务器，自动转换换行符"""
    uploaded = []
    for pattern in patterns:
        files = glob.glob(os.path.join(local_base, pattern), recursive=True)
        for local_path in files:
            rel_path = os.path.relpath(local_path, local_base)
            remote_path = os.path.join(remote_base, rel_path).replace("\\", "/")
            remote_dir = os.path.dirname(remote_path)
            try:
                stdin, stdout, stderr = client.exec_command(f"mkdir -p {remote_dir}")
                stdin.channel.recv_exit_status()
                
                # 读取文件，转换换行符，然后上传
                with open(local_path, 'rb') as f:
                    content = f.read()
                # 转换 \r\n 为 \n
                content = content.replace(b'\r\n', b'\n').replace(b'\r', b'\n')
                # 写入临时文件
                import tempfile
                with tempfile.NamedTemporaryFile(delete=False, suffix='.tmp') as tmp:
                    tmp.write(content)
                    tmp_path = tmp.name
                sftp.put(tmp_path, remote_path)
                os.unlink(tmp_path)
                uploaded.append(rel_path)
                print(f"  ✓ {rel_path}")
            except Exception as e:
                print(f"  ✗ 上传失败 {rel_path}: {e}")
    return uploaded

def main():
    client = ssh_connect()

    # 1. 系统环境检查
    run_cmd(client, "cat /etc/os-release | head -3", "系统版本")
    run_cmd(client, "free -h", "内存信息")
    run_cmd(client, "df -h /", "磁盘空间")

    # 2. 修复 apt 源并安装必要工具
    fix_apt_sources(client)

    # 3. 安装 curl、wget（如果没有）
    run_cmd(client, "which curl || apt-get install -y -qq curl 2>&1 | tail -3", "安装 curl", timeout=60)
    run_cmd(client, "curl --version | head -1", "Curl 版本")

    # 4. 安装 Docker
    print("\n>>> 检查 Docker...")
    rc, _, _ = run_cmd(client, "which docker", "检查 Docker", timeout=10)
    if rc != 0:
        print("  Docker 未安装，开始安装...")
        run_cmd(client, "curl -fsSL https://get.docker.com | sh 2>&1 | tail -5", "安装 Docker", timeout=300)
        run_cmd(client, "systemctl enable docker && systemctl start docker 2>&1", "启动 Docker", timeout=30)
    else:
        print("  Docker 已安装，跳过安装")
    
    run_cmd(client, "docker --version", "Docker 版本")
    run_cmd(client, "systemctl status docker --no-pager | head -5", "Docker 服务状态")

    # 5. 安装 docker-compose
    print("\n>>> 检查 docker-compose...")
    rc, _, _ = run_cmd(client, "which docker-compose", "检查 docker-compose", timeout=10)
    if rc != 0:
        print("  docker-compose 未安装，开始安装...")
        run_cmd(client, 
                "curl -L \"https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)\" -o /usr/local/bin/docker-compose 2>&1 && chmod +x /usr/local/bin/docker-compose && echo '✓ docker-compose 安装成功'",
                "安装 docker-compose", timeout=120)
    run_cmd(client, "docker-compose --version 2>/dev/null || echo 'docker-compose 路径问题，尝试使用 docker compose v2'", "docker-compose 版本")

    # 6. 上传项目文件（使用 glob 模式）
    print("\n>>> 上传项目文件...")
    sftp = client.open_sftp()
    local_base = "."
    remote_base = "/tmp/newapi-tools-verify"

    run_cmd(client, f"rm -rf {remote_base} && mkdir -p {remote_base}", "创建临时目录")

    # 定义需要上传的文件模式
    upload_patterns = [
        "newapi-tools.sh",
        "lib/*.sh",
        "modules/**/*.sh",
        "scripts/*.sh",
        "test/*.sh",
    ]
    
    uploaded = sftp_upload_dir(sftp, local_base, remote_base, client, upload_patterns)
    print(f"\n  共上传 {len(uploaded)} 个文件")
    sftp.close()

    # 7. 远程语法检查
    print("\n>>> 远程语法检查...")
    syntax_ok = 0
    syntax_fail = 0
    for f in uploaded:
        if not f.endswith('.sh'):
            continue
        remote_path = os.path.join(remote_base, f).replace("\\", "/")
        rc, out, err = run_cmd(client, f"bash -n {remote_path} 2>&1 && echo 'OK' || echo 'FAIL'", f"语法检查: {f}", timeout=10)
        if 'OK' in out:
            syntax_ok += 1
        else:
            syntax_fail += 1
    
    print(f"\n  语法检查: {syntax_ok} 通过, {syntax_fail} 失败")

    # 8. 功能验证（dry-run）
    print("\n>>> 功能验证...")
    run_cmd(client, f"cd {remote_base} && bash newapi-tools.sh --help 2>&1 | head -30", "主程序 --help", timeout=30)

    # 9. 检查镜像（不实际拉取，只检查清单）
    print("\n>>> 检查 Docker 镜像（仅检查清单，不拉取）...")
    run_cmd(client, "docker manifest inspect justsong/one-api:latest >/dev/null 2>&1 && echo '✓ One API 镜像清单可访问' || echo '✗ 无法访问 One API 镜像清单（可能需要 docker login 或网络问题）'", "检查 One API 镜像", timeout=30)
    run_cmd(client, "docker manifest inspect calciumion/new-api:latest >/dev/null 2>&1 && echo '✓ NewAPI 镜像清单可访问' || echo '✗ 无法访问 NewAPI 镜像清单'", "检查 NewAPI 镜像", timeout=30)

    print("\n" + "="*60)
    print(f"  验证完成 | 上传 {len(uploaded)} 个文件 | 语法检查 {syntax_ok}✓ {syntax_fail}✗")
    print("="*60)

    client.close()

if __name__ == "__main__":
    main()
