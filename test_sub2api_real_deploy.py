#!/usr/bin/env python3
"""Sub2API 真实部署测试（自动化）"""
import paramiko
import time

HOST = '10.100.10.36'
PORT = 22
USER = 'root'
PASS = '123qwe'
REMOTE_DIR = '/tmp/newapi-tools-verify'
NEWAPI_HOME = '/opt/newapi'

def run_cmd(ssh, cmd, timeout=60, get_exit_code=True):
    stdin, stdout, stderr = ssh.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode('utf-8', errors='replace')
    err = stderr.read().decode('utf-8', errors='replace')
    exit_code = stdout.channel.recv_exit_status() if get_exit_code else None
    return out, err, exit_code

def main():
    print("连接SSH...", flush=True)
    ssh = paramiko.SSHClient()
    ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    ssh.connect(HOST, port=PORT, username=USER, password=PASS, timeout=10)
    print("SSH OK\n", flush=True)

    # 0. 清理之前的状态
    print("=" * 50)
    print("[0] 清理测试环境", flush=True)
    cmds = [
        f"docker stop sub2api npm 2>/dev/null",
        f"docker rm sub2api npm 2>/dev/null",
        f"rm -rf {NEWAPI_HOME}/.env {NEWAPI_HOME}/docker-compose.yml",
        f"rm -f {REMOTE_DIR}/test-state.json",
    ]
    for cmd in cmds:
        run_cmd(ssh, cmd)
    print("  清理完成\n", flush=True)

    # 1. 检查 Docker 和镜像
    print("=" * 50)
    print("[1] 检查 Docker 和镜像", flush=True)
    out, _, code = run_cmd(ssh, "docker --version && docker compose version 2>&1 || docker-compose --version 2>&1")
    print(f"  版本: {out.strip()}", flush=True)

    # 拉取镜像（如果不存在）
    out, _, code = run_cmd(ssh, "docker images -q weishaw/sub2api 2>/dev/null")
    if not out.strip():
        print("  拉取 Sub2API 镜像（可能需要几分钟）...", flush=True)
        run_cmd(ssh, "docker pull weishaw/sub2api:latest", timeout=300)
        print("  镜像拉取完成", flush=True)
    else:
        print("  Sub2API 镜像已存在", flush=True)

    out, _, code = run_cmd(ssh, "docker images -q jc21/nginx-proxy-manager 2>/dev/null")
    if not out.strip():
        print("  拉取 NPM 镜像（可能需要几分钟）...", flush=True)
        run_cmd(ssh, "docker pull jc21/nginx-proxy-manager:latest", timeout=300)
        print("  镜像拉取完成", flush=True)
    else:
        print("  NPM 镜像已存在", flush=True)

    # 2. 准备配置（模拟 auto_config）
    print("=" * 50)
    print("[2] 准备配置", flush=True)
    session_secret = "test_session_secret_12345"
    run_cmd(ssh, f"mkdir -p {NEWAPI_HOME}")

    # 生成 .env
    env_content = f"""SESSION_SECRET={session_secret}
TZ=Asia/Shanghai
PORT=8080
"""
    # 用 echo 写入 .env
    run_cmd(ssh, f"cat > {NEWAPI_HOME}/.env << 'ENVEOF'\n{env_content}ENVEOF")
    run_cmd(ssh, f"chmod 600 {NEWAPI_HOME}/.env")
    print(f"  .env 已生成", flush=True)

    # 3. 生成 docker-compose.yml
    print("=" * 50)
    print("[3] 生成 docker-compose.yml", flush=True)
    compose_content = """# 注意：version 属性已过时，新版 docker compose 会自动忽略
services:
  sub2api:
    image: weishaw/sub2api:latest
    container_name: sub2api
    restart: always
    ports:
      - "127.0.0.1:8080:8080"
    volumes:
      - ./data:/app/data
    environment:
      - SESSION_SECRET=${SESSION_SECRET}
      - TZ=Asia/Shanghai
      - PORT=8080
    # sub2api 不依赖 npm，它可以独立启动
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080"]
      interval: 10s
      timeout: 5s
      retries: 3

  npm:
    image: jc21/nginx-proxy-manager:latest
    container_name: npm
    restart: always
    ports:
      - "80:80"
      - "443:443"
      - "81:81"
    volumes:
      - ./npm/data:/data
      - ./npm/letsencrypt:/etc/letsencrypt
    depends_on:
      sub2api:
        condition: service_healthy

networks:
  default:
    name: sub2api-network
"""
    run_cmd(ssh, f"cat > {NEWAPI_HOME}/docker-compose.yml << 'COMPOSEOF'\n{compose_content}COMPOSEOF")
    run_cmd(ssh, f"chmod 600 {NEWAPI_HOME}/docker-compose.yml")
    print(f"  docker-compose.yml 已生成", flush=True)

    # 4. 启动容器
    print("=" * 50)
    print("[4] 启动容器", flush=True)
    out, err, code = run_cmd(ssh, f"cd {NEWAPI_HOME} && docker compose up -d 2>&1")
    print(f"  输出: {out.strip()}", flush=True)
    if err.strip():
        print(f"  ERR: {err.strip()}", flush=True)

    # 5. 等待并检查健康状态
    print("=" * 50)
    print("[5] 检查服务健康状态", flush=True)
    print("  等待 20 秒让服务初始化...", flush=True)
    time.sleep(20)

    out, _, code = run_cmd(ssh, f"cd {NEWAPI_HOME} && docker compose ps 2>&1")
    print(f"  容器状态:\n{out}", flush=True)

    # 检查每个容器是否正常
    for svc in ['sub2api', 'npm']:
        out, _, code = run_cmd(ssh, f"docker inspect -f '{{.State.Status}}' {svc} 2>/dev/null")
        status = out.strip()
        print(f"  {svc} 状态: {status}", flush=True)

    # 6. 测试 API 可访问性
    print("=" * 50)
    print("[6] 测试 API 可访问性", flush=True)
    out, _, code = run_cmd(ssh, "curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:8080 2>/dev/null || echo 'connection refused'")
    print(f"  Sub2API HTTP 状态码: {out.strip()}", flush=True)

    # 7. 清理
    print("=" * 50)
    print("[7] 清理测试部署", flush=True)
    run_cmd(ssh, f"cd {NEWAPI_HOME} && docker compose down -v 2>&1")
    print("  容器已停止并删除", flush=True)

    ssh.close()
    print("\n" + "=" * 50)
    print("真实部署测试完成！", flush=True)

if __name__ == '__main__':
    main()
