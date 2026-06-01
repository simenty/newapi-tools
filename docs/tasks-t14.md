# Phase 3 — 尾项（T14 Linux 验证）

> 仅剩 1 项，完成后 handoff 全部任务清仓。

---

## T14: Linux 实机验证

**测试机**：`root@10.100.10.33`（Debian 12）  
**SSH 密钥**：`~/.ssh/id_ed25519_auto`

### 步骤

```bash
# 1. 编译 Linux 二进制
cd newapi-tools
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
  -ldflags="-s -w -X github.com/simenty/newapi-tools/internal/core.Version=v3.4.1" \
  -o dist/newapi-tools-linux-amd64 ./cmd/newapi/

# 2. 上传到测试机
scp -i ~/.ssh/id_ed25519_auto dist/newapi-tools-linux-amd64 root@10.100.10.33:/tmp/newapi-tools

# 3. 运行验证
ssh -i ~/.ssh/id_ed25519_auto root@10.100.10.33 "
  chmod +x /tmp/newapi-tools
  echo '=== version ==='
  /tmp/newapi-tools version
  echo
  echo '=== doctor --json ==='
  /tmp/newapi-tools doctor --json
  echo
  echo '=== mirror builtin ==='
  /tmp/newapi-tools mirror builtin
  echo
  echo '=== mirror test ==='
  /tmp/newapi-tools mirror test tuna aliyun
  echo
  echo '=== audit list --last 3 ==='
  /tmp/newapi-tools audit list --last 3
  echo
  echo '=== status ==='
  /tmp/newapi-tools status
"
```

### 预期结果

| 命令 | 预期 |
|------|------|
| `version` | 显示 v3.4.1 |
| `doctor --json` | 12 项检查，0 失败，JSON 格式 |
| `mirror builtin` | 6 个内置源表格 |
| `mirror test` | 并发测速，延迟表格（tuna 可能 FAIL 是正常的） |
| `audit list --last 3` | 显示最近 3 条审计记录 |
| `status` | 显示容器状态（表格或 JSON） |

### 如果 doctor 有 Docker 连通性问题

```bash
# 确认 Docker 运行中
ssh root@10.100.10.33 "docker ps"
# 如果没有 docker，检查 PATH
ssh root@10.100.10.33 "which docker"
```

### 完成后

```bash
git add -A
git commit -m "test: Linux real environment verification (v3.4.1 PASS)"
git push origin handoff
```
