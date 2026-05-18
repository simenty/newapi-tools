# Changelog - v1.2

## 📅 发布日期: 2025-05-12

---

## 🔴 严重 Bug 修复

### 1. 命令行路由完全失效
**文件**: `newapi-tools.sh`  
**问题**: `exec bash "${MODULES_DIR}/manage/${1}.sh"` 调用错误路径，导致 `backup`、`update` 等命令完全无法使用  
**修复**: 重写 `route_command()` 函数，建立正确的路由表映射

### 2. Webhook JSON 转义错误
**文件**: `lib/common.sh`  
**问题**: `"${title}\n${content}"` 直接拼接，特殊字符（换行、引号）必破坏 JSON 结构  
**修复**: 使用 `printf` 构建 payload，对特殊字符进行转义

### 3. 健康检查显示报错
**文件**: `modules/monitor/health.sh`  
**问题**: `$3"/"$2` awk 语法引号未闭合，导致内存/磁盘使用率显示失败  
**修复**: 修正为 `$3 " / " $2`

### 4. 更新脚本循环失效
**文件**: `modules/manage/update.sh`  
**问题**: `for i in {seq 1 30}` 语法错误，循环体不执行  
**修复**: 改为 `for i in $(seq 1 30)`

---

## 🟡 安全加固

### 1. 统一严格模式
**文件**: 所有 `.sh` 文件  
**修复**: 所有脚本添加 `set -euo pipefail`，确保遇错即停

### 2. 敏感变量白名单过滤
**文件**: `lib/common.sh`  
**修复**: `load_env_sensitive()` 只加载白名单变量，防止 .env 污染环境

### 3. .env 权限验证
**文件**: `modules/deploy/install.sh`  
**修复**: 设置权限后验证，不一致则重试

### 4. 密码变量安全清理
**修复**: 所有脚本在密码使用后立即 `unset`

---

## 🟢 体验优化

### 1. 幂等性检查
**文件**: `modules/deploy/install.sh`, `modules/init/docker.sh`  
**修复**: 已存在则提示确认，避免覆盖现有配置

### 2. ShellCheck 注释
**文件**: 所有 `.sh` 文件  
**修复**: 添加 `# shellcheck source=xxx` 注释，消除静态检查警告

### 3. 备份列表美化
**文件**: `modules/manage/restore.sh`  
**修复**: 表格形式展示备份文件，更直观

### 4. 日志查看增强
**文件**: `modules/monitor/logs.sh`  
**修复**: 支持 `-n` 行数参数和 `-f` 实时追踪

### 5. 自更新健壮性
**文件**: `scripts/self-update.sh`  
**修复**: 检测本地修改、权限修复、版本对比

### 6. 安装器优化
**文件**: `install.sh`  
**修复**: 幂等性、错误检查、版本显示

### 7. 加密脚本增强
**文件**: `scripts/encrypt-config.sh`  
**修复**: 密码确认、文件存在检查、安全清理

### 8. 卸载清单展示
**文件**: `modules/manage/uninstall.sh`  
**修复**: 明确列出将删除的所有项目

---

## 📊 文件变更统计

| 文件 | 变更类型 |
|------|----------|
| `newapi-tools.sh` | 重写路由、添加严格模式 |
| `lib/common.sh` | JSON修复、白名单过滤、错误捕获 |
| `lib/env.sh` | 变量声明修复 |
| `modules/deploy/install.sh` | 幂等性、权限验证 |
| `modules/deploy/ssl-proxy.sh` | JSON payload 重构 |
| `modules/manage/backup.sh` | 错误处理、--cron 模式 |
| `modules/manage/update.sh` | 循环修复、进度提示 |
| `modules/manage/restore.sh` | 边界检查、列表美化 |
| `modules/manage/reinstall.sh` | 确认清单 |
| `modules/manage/uninstall.sh` | 删除清单展示 |
| `modules/init/docker.sh` | 幂等性、镜像加速 |
| `modules/monitor/health.sh` | awk 修复 |
| `modules/monitor/logs.sh` | 参数支持 |
| `scripts/self-update.sh` | Git 状态检查 |
| `scripts/encrypt-config.sh` | 密码确认 |
| `install.sh` | 幂等性 |
| `docs/README.md` | v1.2 完整文档 |

---

## 🚀 如何应用此更新

### 方法 1: Git 提交（推荐）
```bash
cd /opt/newapi-tools
git pull origin main
# 将修复后的文件覆盖进去
git add -A
git commit -m "v1.2: 修复严重Bug + 全面安全加固"
git push origin main
```

### 方法 2: 使用一键脚本
```bash
bash /path/to/commit-and-push.sh
```

---

## ✅ 测试建议

提交后请在测试环境验证：
```bash
newapi-tools backup --manual      # 验证命令行路由
newapi-tools health              # 验证健康检查显示
newapi-tools logs -n 20         # 验证日志参数
```
