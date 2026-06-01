# Phase 3 — 收尾 & 增强

> 基于 handoff Phase 1-2 完成的交付  
> 接手人：TRAE | 分支：handoff | 基准：v3.4.0

---

## 已完成回顾

| 阶段 | 任务 | 状态 |
|------|------|------|
| P0 | audit 文档页、mkdocs nav 更新、lint 修复 | ✅ |
| P1 | docker 34.8% selfupdate 21.7% cli 17.4% security 27% | ✅ |
| P2 | Config Validate、audit 轮转文件、doctor JSON、gendocs hook | ✅ |
| Extra | Docker API 集成（net/http，零新增依赖） | ✅ |

**当前测试 16 包全 PASS，覆盖率整体达标。**

---

## P3 任务清单

---

### T13: 补 `internal/registry` 测试

**文件**：`internal/registry/*_test.go`  
**工作量**：⭐  
**目标**：0% → ≥50%

`internal/registry` 包很小（一个 registry.go），只有 `RegisterPlugin` 和 `Registry` 结构体。写基础测试覆盖正常注册和重复注册 panic。

---

### T14: Linux 实机验证

**测试机**：`root@10.100.10.33`  
**工作量**：⭐  
**工具**：SSH 密钥 `id_ed25519_auto`

#### 验证清单

```bash
# 编译 Linux 二进制
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
  -ldflags="-s -w -X github.com/simenty/newapi-tools/internal/core.Version=v3.4.0" \
  -o dist/newapi-tools-linux-amd64 ./cmd/newapi/

# 上传
scp dist/newapi-tools-linux-amd64 root@10.100.10.33:/tmp/newapi-tools

# 测试
ssh root@10.100.10.33 "
  chmod +x /tmp/newapi-tools
  /tmp/newapi-tools version
  /tmp/newapi-tools doctor
  /tmp/newapi-tools mirror builtin
  /tmp/newapi-tools mirror test tuna aliyun
  /tmp/newapi-tools audit list --last 3
  /tmp/newapi-tools config chmod
"
```

注意：Docker API 集成代码（http_client.go / sdk_client.go）需要额外验证是否能正常工作。先跑 `doctor` 看 Docker 连通性。

---

### T15: 检查 Docker API 集成是否有lint/编译警告

**文件**：`internal/docker/http_client.go`, `internal/docker/sdk_client.go`  
**工作量**：⭐

```bash
go vet ./internal/docker/...
golangci-lint run ./internal/docker/...
```

如果有 golangci-lint 警告，修复后再提交。

---

### T16: Config WriteConfig 补齐 instance 字段

**文件**：`internal/core/config.go`  
**工作量**：⭐

`WriteConfig` 函数在写入 YAML 时遗漏了 `instance.active` 字段。当前 `config init` 和 `instance switch` 都会调用 `WriteConfig`，如果缺失 instance 字段，切换实例后配置不会持久化。

**检查**：在 `WriteConfig` 的数据映射中加入：

```go
"instance": map[string]interface{}{
    "active": cfg.Instance.Active,
},
```

**测试**：`go test ./internal/core/... -v` 确认不破坏现有测试。

---

### T17: 版本发布 v3.4.1

**工作量**：⭐

```bash
# 更新版本号
echo 'var Version = "v3.4.1"' > internal/core/version.go

# 提交
git add -A
git commit -m "chore: bump to v3.4.1"

# 打 tag 发布
git tag v3.4.1
git push origin v3.4.1
# → 触发 GitHub Actions Release 自动构建
```

---

### T18: 检查 docs/commands/audit.md 在文档站中的渲染效果

**工作量**：⭐

```bash
# 本地预览文档站
pip install mkdocs-material
mkdocs serve
# → http://localhost:8000 查看 "命令参考 > audit"
```

确认：
- [ ] 导航显示 audit 条目
- [ ] 页面内容完整（示例、表格、参数说明）
- [ ] 链接正常跳转

---

## 执行顺序

```
T13 (registry test)  ← 最快，10 分钟
  ↓
T16 (WriteConfig fix) ← 简单修复
  ↓
T15 (lint check)    ← 验证 Docker API 代码质量
  ↓
T14 (Linux test)    ← 实机验证
  ↓
T18 (docs preview)  ← 文档验证
  ↓
T17 (v3.4.1 release) ← 全部完成后打 tag 发布
```

每完成一项：`git add -A && git commit -m "类别: 任务描述" && git push origin handoff`
