# TRAE 接手指引

> 当你在 TRAE 中打开此项目时，先读这个文件。

## 快速入口

**项目全景** → 见 `docs/handoff.md`（完整交接手册）  
**项目概览** → 见 `.workbuddy/project-overview.md`（精简版）  

## 第一件事：跑通环境

```bash
make build     # 编译
make test      # 全量测试（16包必须全 PASS）
make vet       # go vet
make lint      # golangci-lint
./dist/newapi-tools version   # 确认可运行
```

如果 test 失败，不要改代码——先排查环境问题。

## 工作分支

当前在 `handoff` 分支。所有改动请基于此分支：

```bash
git checkout handoff
git pull origin handoff
# 改代码 → 测试 → 提交
git push origin handoff
```

合入 main 前需通过 PR Review。

## 包责任矩阵

| 包 | 谁改 | 注意 |
|----|------|------|
| `internal/apperr/` | 谨慎 | 错误码变更影响全局，需要同步更新 docs/errors.md |
| `internal/cli/` | 随便改 | 命令层，只做参数解析和 UI 输出，业务逻辑下放到 internal 包 |
| `internal/docker/` | 测试机验证 | Docker 操作需 Debian 12 实机验证（10.100.10.33） |
| `internal/selfupdate/` | 发布时注意 | 修改后需验证 GitHub Release 产物名匹配 resolveAssetName |
| `internal/audit/` | 注意兼容 | 审计日志格式变更会破坏历史查询 |

## 沟通

- 有问题 → 提交 Issue 到 GitHub
- 需要规划 → 先在 WorkBuddy 中拆任务
- 小改动直接改，大改动先拆任务

## 已就绪

- ✅ `handoff` 分支已创建并推送
- ✅ `docs/handoff.md` — 完整交接手册
- ✅ `.workbuddy/` — 项目上下文文件
- ✅ CI/CD 全部通过
- ✅ Linux 实机已验证
- ✅ GitHub Release v3.4.0 自动构建中
