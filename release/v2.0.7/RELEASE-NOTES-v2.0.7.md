# NewAPI Tools v2.0.7 发布说明

**发布日期**: 2026-05-14  
**版本类型**: 审计修复完善版（Bug Fix Release）  
**上一个版本**: v2.0.6

---

## 📝 发布概要

本版本完成了 v2.0.6 审计报告中遗留的 4 项修复，进一步提升代码质量和安全性。

**代码质量评分**: ⭐⭐⭐⭐⭐ (4.7/5.0) ↑（从 4.4 提升）

---

## 🔧 主要变更

### 1. 严格模式（set -euo pipefail）
- **影响脚本**: 16 个核心脚本
- **效果**: 命令失败立即退出，防止错误传播导致数据损坏

### 2. 函数定义规范化
- **文件**: `modules/manage/restore.sh`
- **修改**: 将 `_verify_file_bg()` 和 `_extract_with_progress()` 从嵌套定义改为顶层定义
- **原因**: 符合 bash 最佳实践，提高代码可读性

### 3. Trap 处理优化
- **文件**: `modules/manage/backup.sh`
- **修改**: 移除 trap 中的 `exit`，改为让 trap 函数自然返回
- **原因**: 避免递归触发 EXIT trap

### 4. 密码安全（日志脱敏）
- **新增函数**: `_desensitize_log()`（在 `lib/common.sh`）
- **脱敏规则**:
  - 密码: `password=***`, `pwd=***`, `passwd=***`
  - Token: `token=***`, `secret=***`, `key=***`
  - 数据库连接: `MYSQL_PWD=***`, `SQL_DSN=***`, `REDIS_CONN=***`
- **覆盖函数**: 所有 `log_*` 函数自动脱敏

---

## 📊 测试情况

| 测试类型 | 结果 |
|---------|------|
| 单元测试 | ✅ 16/16 通过 |
| 语法检查 | ✅ ShellCheck 通过 |
| 功能测试 | ✅ 备份/恢复/更新正常 |

---

## 📦 安装/升级方法

### 新安装
```bash
curl -fsSL https://raw.githubusercontent.com/simently/newapi-tools/main/install.sh | bash
```

### 从 v2.0.x 升级
```bash
cd /opt/newapi-tools
git pull origin main
```

---

## 🔍 完整变更列表

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `newapi-tools.sh` | 修改 | 添加 `set -euo pipefail` |
| `install.sh` | 修改 | 添加 `set -euo pipefail` |
| `lib/common.sh` | 修改 | 添加日志脱敏函数 |
| `lib/env.sh` | 修改 | 添加 `set -euo pipefail` |
| `lib/state.sh` | 修改 | 添加 `set -euo pipefail` |
| `lib/config.sh` | 修改 | 添加 `set -euo pipefail` |
| `modules/manage/backup.sh` | 修改 | 修复 trap、添加严格模式 |
| `modules/manage/restore.sh` | 修改 | 函数定义规范化、添加严格模式 |
| `modules/manage/update.sh` | 修改 | 添加 `set -euo pipefail` |
| `modules/manage/reinstall.sh` | 修改 | 添加 `set -euo pipefail` |
| `modules/manage/uninstall.sh` | 修改 | 添加 `set -euo pipefail` |
| `modules/monitor/health.sh` | 修改 | 添加 `set -euo pipefail` |
| `modules/monitor/logs.sh` | 修改 | 添加 `set -euo pipefail` |
| `CHANGELOG-v2.0.md` | 新增 | v2.0.7 更新记录 |
| `docs/audit-report-v2.0.7.md` | 新增 | 审计报告 |

---

## ⚠️ 注意事项

1. **严格模式可能影响现有脚本**:
   - 如果您的自定义脚本依赖 `set +e` 来忽略错误，请在升级前测试
   - 建议: 在测试环境先验证

2. **日志格式变化**:
   - 密码/Token 现在显示为 `***`
   - 这是**安全增强**，不是 bug

---

## 🎯 下一步计划

- **迭代 6**: 状态管理系统增强（文件锁 `flock`）
- **迭代 7**: 配置向导 + 环境检查工具

---

## 📝 附录：审计评分对比

| 维度 | v2.0.6 | v2.0.7 | 变化 |
|------|---------|---------|------|
| 代码质量 | 4.4/5.0 | 4.7/5.0 | ↑ +0.3 |
| 安全性 | 4.2/5.0 | 4.8/5.0 | ↑ +0.6 |
| 可维护性 | 4.5/5.0 | 4.6/5.0 | ↑ +0.1 |
| 测试覆盖 | 4.8/5.0 | 4.8/5.0 | - |

---

**发布负责人**: Vincent  
**审核**: 代码审计工具 + 单元测试  
**批准发布**: ✅
