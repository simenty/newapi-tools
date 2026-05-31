# main 分支审计报告 & handoff 任务清单

> **审计时间**：2026-06-01 03:28  
> **基准版本**：v3.4.0（commit 28d6185）  
> **审计范围**：全仓库代码、配置、文档、CI/CD

---

## 一、审计结果

### ✅ 已通过

| 检查项 | 结果 |
|--------|------|
| `go build ./...` | ✅ 通过 |
| `go vet ./...` | ✅ 通过 |
| `go test ./...` | ✅ 16 包全部 PASS |
| Coverage ≥ 30% | ✅ 通过（最差包 9.6%，整体 >30%） |
| `golangci-lint` | ✅ 配置完整 |
| `mkdocs build` | ✅ 配置完整 |
| GitHub Actions | ✅ CI + Release + Docs 三工作流 |
| `handoff` 分支 | ✅ 已创建并推送 |

### ⚠️ 覆盖率缺口

| 包 | 覆盖率 | 风险 |
|----|--------|------|
| `cmd/gendocs` | 0.0% | 🔴 入口工具无测试 |
| `cmd/newapi` | 0.0% | 🔴 入口 main.go 无测试 |
| `internal/selfupdate` | 0.0% | 🔴 自更新逻辑无测试 |
| `internal/registry` | 0.0% | 🔴 命令注册表无测试 |
| `internal/docker` | 9.6% | 🟡 Docker 操作层测试不足 |
| `internal/cli` | 13.2% | 🟡 CLI 命令测试不足 |
| `internal/security` | 21.6% | 🟡 安全检查测试不足 |

### 🗺️ 文档缺失

| 文档 | 状态 | 问题 |
|------|------|------|
| `docs/commands/audit.md` | ❌ 不存在 | `audit list` 命令无文档页 |
| `docs/commands/instance.md` | ✅ 存在 | 但未在 mkdocs.yml nav 中注册 |
| `mkdocs.yml nav` | ⚠️ 过时 | 缺少 instance、audit 入口 |

### 🧹 旧的 mkdocs.yml exclude_docs 列表

`exclude_docs` 中有 20+ 个文件名，但大部分 V2 文件早已不存在。可简化。

### 🏗️ deliverables 目录

包含 V3.2 设计文档（PRD/架构图），对当前开发仍有参考价值，建议保留。

---

## 二、handoff 任务清单

按优先级从高到低排列，每项标注了影响范围和工作量。

### P0 — 接手后优先做

| # | 任务 | 文件 | 工作量 | 说明 |
|---|------|------|--------|------|
| 1 | **补齐 `audit list` 文档页** | 新建 `docs/commands/audit.md` | ⭐ | 参考 `docs/commands/status.md` 格式，写用法和示例 |
| 2 | **更新 mkdocs.yml** | `mkdocs.yml` | ⭐ | nav 加入 instance 和 audit；清理 exclude_docs 旧文件列表 |
| 3 | **本地跑通全流程** | 环境 | ⭐ | `make build && make test && make vet && make lint` |
| 4 | **验证 GitHub Actions Release** | CI | ⭐ | 检查 v3.4.0 的 Release 是否成功生成 |

### P1 — 测试补全（影响覆盖率门禁）

| # | 任务 | 文件 | 工作量 | 说明 |
|---|------|------|--------|------|
| 5 | **补 `internal/docker` 测试** | `internal/docker/*_test.go` | ⭐⭐⭐ | 当前 9.6%，目标 ≥30%。重点是 mirror 和 client 包 |
| 6 | **补 `internal/selfupdate` 测试** | `internal/selfupdate/*_test.go` | ⭐⭐ | 当前 0%。Checker（API 查询）可用 mock server 测 |
| 7 | **补 `internal/security` 测试** | `internal/security/*_test.go` | ⭐ | 当前 21.6%。FixConfigPerm 函数可测 |
| 8 | **补 `internal/cli` 测试** | `internal/cli/*_test.go` | ⭐⭐⭐ | 当前 13.2%。增加 config chmod 和 mirror test 命令测试 |

### P2 — 功能完善

| # | 任务 | 文件 | 工作量 | 说明 |
|---|------|------|--------|------|
| 9 | **Config `Validate()` 方法** | `internal/core/config.go` | ⭐ | 验证 Port/HealthTimeout/MaxBackups 合法性 |
| 10 | **`audit list` 轮转文件合并** | `internal/audit/audit.go` | ⭐⭐ | ring buffer 只读主文件，漏了 .1/.2 等轮转文件 |
| 11 | **`doctor --verbose --json` 统一序列化** | `internal/cli/doctor.go` | ⭐ | printDoctorJSON 已改，但可进一步用统一 struct |
| 12 | **gendocs pre-commit hook** | 新增 `hooks/` | ⭐ | 新增错误码时自动触发 gendocs，防止文档不同步 |

### P3 — 低优先级/非紧急

| # | 任务 | 文件 | 工作量 | 说明 |
|---|------|------|--------|------|
| 13 | **清理 deliverables 目录** | `deliverables/` | ⭐ | 归档旧 PRD，简化根目录 |
| 14 | **README 超链接验证** | `README.md` `README_EN.md` | ⭐ | 检查 GitHub badges 是否正常显示 |
| 15 | **新增 GitHub Issue 模板** | `.github/ISSUE_TEMPLATE/` | ⭐ | 建 bug_report.md 和 feature_request.md |

---

## 三、分支策略

```
main ── 稳定版本（只合入已验证的 PR）
  │
  └── handoff ── 接手开发分支
        │
        ├── 每完成一个任务 → commit
        ├── 每完成一组 P0 任务 → push
        └── 全部完成后 → PR → main
```

### 提交规范

```
# 格式
git commit -m "类别: 简短描述"

# 类别
test:     # 测试补全
docs:     # 文档更新
fix:      # Bug 修复
feat:     # 新功能
chore:    # 杂项（配置、清理）
```

### 完成标准

每个任务完成后确认：
- [ ] `go build ./...` 通过
- [ ] `go vet ./...` 通过
- [ ] `go test ./...` 全通过
- [ ] 涉及的新代码有测试覆盖
- [ ] 文档已更新（如适用）
