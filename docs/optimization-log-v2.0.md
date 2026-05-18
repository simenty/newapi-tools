# NewAPI Tools V2.0 优化记录

本文档记录从 v2.0 到 v3.0 的10次迭代优化过程。

---

## 第1次优化 (2026-05-13) - 修复残留bug和代码清理

### 优化内容

#### 1. 修复 `restore.sh` 中的无效 unset 语句

**问题描述**：
- 位置：`modules/manage/restore.sh` 第152行
- 问题：`unset DB_ROOT_PASSWORD`，但该变量已是局部变量 `db_root_password`
- 影响：unset 无效且可能引起混淆

**修复方法**：
```diff
     ui_error "数据卷恢复失败！请查看日志: $LOG_FILE"
     update_install_status "failed" "data_restore_failed"
-    unset DB_ROOT_PASSWORD
     exit 1
```

**结果**：✅ 已修复

---

#### 2. 标记 `lib/env.sh` 为 v1.0 兼容层

**分析结果**：
- `lib/env.sh` 仍在设置关键环境变量：`NEWAPI_HOME`、`DOCKER_COMPOSE_CMD`、`BACKUP_RETENTION_DAYS` 等
- 这些变量在多个模块中广泛使用
- 暂时不能移除，但应标记为弃用

**处理方法**：
更新文件头注释：
```bash
#!/bin/bash
# NewAPI 运维工具集 — 环境变量与配置加载（资深开发修复版）
# 
# ⚠️ [V2.0 兼容性层 - 计划在 V3.0 移除]
# 此文件提供 v1.0 风格的环境变量配置，用于向后兼容。
# 新代码应使用 lib/config.sh 中的 YAML 配置系统。
#
# 迁移指南：
#   - 使用 get_config("deploy.xxx") 替代 ${DEPLOY_XXX}
#   - 使用 set_config("deploy.xxx" "value") 替代 export DEPLOY_XXX=value
#   - 在 V3.0 中，此文件将被完全移除
```

**结果**：✅ 已标记弃用

---

#### 3. 标记 `load_env_sensitive()` 为弃用

**分析结果**：
- `lib/common.sh` 中定义了 `load_env_sensitive()` 函数
- 通过 `grep -rn "load_env_sensitive"` 检查，发现**无人调用**
- 应标记为弃用，为未来移除做准备

**处理方法**：
更新函数注释：
```bash
# ---------- [V1.0 兼容层 - 计划在 V3.0 移除] ----------
# 加载 .env 敏感变量（安全版）
# 只导出白名单变量，防止 .env 中的随意变量污染环境
# ⚠️ 此函数已弃用，新代码应使用 get_config() 函数
load_env_sensitive() {
    log_warn "load_env_sensitive() 已弃用，请使用 get_config() 替代"
    ...
}
```

**结果**：✅ 已标记弃用

---

### 测试过程

#### 测试环境
- **操作系统**: Windows 11 + Git Bash (模拟 Linux 环境)
- **测试工具**: ShellCheck, grep
- **项目路径**: D:\Users\Vincent\Desktop\newapi-tools\newapi-tools

#### 测试用例

**测试1：检查 v1.0 遗留引用**
```bash
grep -rn "lib/env.sh" --include="*.sh"
```
- **结果**: ✅ 仅在 `newapi-tools.sh` 中引用（正常）
- **说明**: 这是主脚本，需要保持向后兼容

**测试2：检查无效的 unset 语句**
```bash
grep -rn "unset DB_ROOT_PASSWORD" --include="*.sh"
```
- **结果**: ✅ 未找到（已修复）

**测试3：检查 `load_env_sensitive()` 调用**
```bash
grep -rn "load_env_sensitive" --include="*.sh"
```
- **结果**: ✅ 仅在定义处出现（未被调用）

**测试4：检查环境变量使用**
```bash
grep -rn "NEWAPI_HOME\|DOCKER_COMPOSE_CMD" --include="*.sh"
```
- **结果**: ✅ 仍在多处使用，验证了保留 `lib/env.sh` 的必要性

---

### 修复的 Bug

| Bug ID | 描述 | 严重程度 | 状态 |
|--------|------|----------|------|
| BUG-001 | `restore.sh` 第152行无效 unset 语句 | 低 | ✅ 已修复 |
| LEGACY-001 | `lib/env.sh` 无 v1.0 兼容层标记 | 低 | ✅ 已标记弃用 |
| LEGACY-002 | `load_env_sensitive()` 无弃用标记 | 低 | ✅ 已标记弃用 |

---

### 性能对比

本次优化主要是代码清理，未涉及性能相关改动。

---

### 遗留问题

1. **`lib/env.sh` 仍需保留**
   - 原因：多个模块仍在使用传统环境变量
   - 影响：V3.0 前必须迁移到 YAML 配置
   - 优先级：高

2. **`load_env_sensitive()` 保留定义**
   - 原因：保持向后兼容，不影响功能
   - 影响：代码库中仍有未使用的函数
   - 优先级：低

---

### 下一步计划（第2次优化）

**目标**：完善 CentOS/Rocky Linux 的 Docker 安装

**具体任务**：
1. 检查 `docker.sh` 中 RHEL 分支的错误处理
2. 添加 Docker 仓库配置失败时的多重降级方案
3. 确保阿里云镜像源配置正确
4. 添加安装后的版本验证和功能测试

---

## 第2次优化 (2026-05-13) - 完善 CentOS/Rocky Linux Docker 安装

### 修复的 Bug

| # | 问题 | 严重程度 | 状态 |
|---|------|----------|------|
| 1 | `docker info` 更准确判断 Docker 运行状态（原 `systemctl is-active` 在容器内可能误判） | 中 | ✅ 修复 |
| 2 | RHEL 安装时若两个仓库都失败，脚本直接报错退出而无第三条路 | 高 | ✅ 修复 |
| 3 | CentOS 7 缺少 `extras` 仓库启用（containerd.io 依赖） | 高 | ✅ 修复 |
| 4 | 存在 podman/buildah 包冲突时安装失败，缺少自动处理 | 高 | ✅ 修复 |
| 5 | 安装后未验证 `docker compose` 插件可用性 | 中 | ✅ 修复 |
| 6 | firewalld 环境下 Docker 网络规则不生效 | 中 | ✅ 修复 |
| 7 | `daemon.json` 已存在时直接覆盖导致配置丢失 | 中 | ✅ 修复 |

### 优化亮点
- **三级降级安装**（Debian）：官方脚本 → 手动添加 apt 仓库（阿里云） → 官方仓库
- **双仓库降级**（RHEL）：官方 repo → 阿里云 repo → 直接下载 .repo 文件
- **智能冲突处理**：检测到 podman/buildah 冲突时自动移除后重试
- **daemon.json 安全合并**：使用 Python JSON 合并，不覆盖已有配置
- **compose 插件自动补装**：安装完成后验证，缺失时自动补装

### 测试结果
- 语法检查：✅ 通过
- 单元测试：✅ 16/16

---

## 第3次优化 (2026-05-13) - 完善仓库源配置脚本

### 修复的 Bug

| # | 问题 | 严重程度 | 状态 |
|---|------|----------|------|
| 1 | `$releasever` 变量未定义（第100/108/115行），导致 URL 错误 | 致命 | ✅ 修复 |
| 2 | Debian 分支 URL 写死 ubuntu，不支持 Debian 原生发行版 | 高 | ✅ 修复 |
| 3 | `apt update -y` 参数无效（update 不接受 -y） | 中 | ✅ 修复 |
| 4 | 缺少镜像可用性验证（配置后不测试） | 高 | ✅ 修复 |
| 5 | Rocky/AlmaLinux 用 add-repo 路径方式不正确 | 高 | ✅ 修复 |
| 6 | 缺少华为云镜像 | 低 | ✅ 修复 |
| 7 | 备份策略不完善（多次运行会覆盖备份） | 中 | ✅ 修复 |

### 优化亮点
- **完整镜像矩阵**：Debian/Ubuntu 支持 6 个镜像源；RHEL 系支持 4 个
- **发行版智能适配**：Rocky/AlmaLinux 用 sed 替换 mirrorlist；CentOS 7/8 分别处理
- **配置后自动验证**：`apt-get update` / `dnf makecache`，失败自动恢复备份
- **带时间戳备份**：避免多次运行互相覆盖，每次创建独立备份目录
- **Debian 与 Ubuntu 区分**：不同发行版使用不同 URL 格式和组件名

### 测试结果
- 语法检查：✅ 通过
- 单元测试：✅ 16/16

---

## 第4次优化 (2026-05-13) - 优化备份和恢复性能

### 修复的 Bug

| # | 问题 | 严重程度 | 状态 |
|---|------|----------|------|
| 1 | `restore.sh` 第126行在函数外部使用 `local` 关键字（bash warning，`set -e` 下报错） | 高 | ✅ 修复 |

### 新增功能

#### backup.sh

| 功能 | 实现方式 | 降级策略 |
|------|----------|----------|
| 并行压缩 | `pigz -p N`（N=CPU核数，上限4） | 无 pigz → 标准 gzip |
| 实时压缩进度 | `tar \| pv \| pigz` 管道 | 无 pv → 静默压缩 |
| 异步 MD5 校验 | 后台子进程 `md5sum &`，两文件并行 | 后台失败 → 同步重算 |
| 等待校验 | `wait $pid` + 补偿同步逻辑 | 始终确保 .md5 生成 |

#### restore.sh

| 功能 | 实现方式 | 降级策略 |
|------|----------|----------|
| 并行文件校验 | 两文件同时后台 `verify_checksum &` | N/A |
| 并行解压 | `pv \| pigz -d \| tar -xf` 管道 | 无 pigz/pv → 标准 tar -zxf |
| 实时解压进度 | `pv -s 文件大小` 显示百分比 | 无 pv → 静默解压 |

### 关键代码设计

#### 异步 MD5 设计原则
```bash
# 触发异步
_generate_checksum_async "$db_file" db_md5_pid   # db_md5_pid 接收 PID
_generate_checksum_async "$data_file" data_md5_pid

# ... 中间可以执行其他操作 ...

# 最终等待（带补偿）
_wait_checksum "$db_md5_pid" "$db_file"     # 失败则同步重算
_wait_checksum "$data_md5_pid" "$data_file"
```

#### 并行校验设计原则
```bash
# 并行启动
_verify_file_bg "$selected" "数据卷" "$DATA_CHECK_FILE" &
_verify_file_bg "$DB_FILE"  "数据库" "$DB_CHECK_FILE"  &

# 顺序等待（先等数据卷，再等数据库）
wait $DATA_CHECK_PID → 读取结果 → 判断
wait $DB_CHECK_PID   → 读取结果 → 判断
```

### 测试结果
- 语法检查：✅ backup.sh 通过，restore.sh 通过
- 单元测试：✅ 16/16

### 预估性能提升

| 指标 | 优化前 | 优化后（4核服务器）|
|------|--------|------------------|
| 压缩时间（1GB） | ~60s | ~20s（-67%）|
| 解压时间（1GB） | ~40s | ~15s（-63%）|
| MD5 生成（两文件串行）| ~10s | ~5s（-50%）|
| 文件校验（两文件串行）| ~10s | ~5s（-50%）|

---

## 第5次优化 (2026-05-13) - 增强错误处理和回滚机制

### 修复的 Bug

| # | 问题 | 严重程度 | 状态 |
|---|------|----------|------|
| 1 | `backup.sh` `total_size` 计算使用 `${targets[@]/#/$source_dir/}`（非有效 bash 语法，始终返回 0）| 高 | ✅ 修复 |
| 2 | `backup.sh` 数据库备份失败时未清理不完整的 .sql 文件 | 中 | ✅ 修复 |
| 3 | `restore.sh` 恢复失败时没有自动回滚机制 | 高 | ✅ 修复 |

### 新增功能

#### backup.sh - 错误处理增强

| 函数 | 功能 |
|------|------|
| `_cleanup_partial_backup()` | EXIT trap 回调，删除不完整的备份文件（无 "Dump completed" 的 .sql、未通过 gzip -t 的 .tar.gz）|
| `trap '_cleanup_partial_backup ...; exit' EXIT` | 异常退出时自动清理 |
| 数据库备份后 `grep "Dump completed"` | 确保 .sql 文件完整 |

#### restore.sh - 回滚机制（核心新增）

| 函数 | 功能 |
|------|------|
| `_create_rollback_point()` | 创建回滚点：mv data/npm 到 .bak.{timestamp}，导出当前 DB 快照 |
| `_do_rollback()` | 执行回滚：mv 恢复 data/npm，从快照恢复 DB，重启容器 |
| `_clear_rollback()` | 恢复成功后清除回滚点 |
| `trap _on_exit EXIT` | 脚本退出时根据 `ROLLBACK_NEEDED` 决定是否触发回滚 |

### 回滚设计亮点

**为什么用 mv 而不是 cp 保存回滚点？**
- data/ 可能几十 GB，cp 非常慢
- mv 是 O(1) 操作（仅修改目录项）
- 恢复时也是 mv 回去，同样高效

**回滚点包含的内容：**
```
.rollback_{timestamp}/
  ├── pre_restore_db.sql    # DB 快照（小文件，KB 级）
  └── .rollback_active      # 回滚标记（存在 = 需要回滚）

{NewAPI_HOME}/
  ├── data.bak.{timestamp}  # mv 出来的数据目录
  └── npm.bak.{timestamp}   # mv 出来的 npm 目录
```

**ROLLBACK_NEEDED 守卫变量设计：**
```
正常流程：                              失败流程：
1. ROLLBACK_NEEDED=1（默认）         1. ROLLBACK_NEEDED=1
2. 执行恢复...                        2. 某步失败 → exit 1
3. 成功 → _clear_rollback()          3. EXIT trap 触发
4.   → ROLLBACK_NEEDED=0            4. _on_exit() 检查 ROLLBACK_NEEDED=1
5.   → 删除回滚文件                   5. 调用 _do_rollback()
6. 脚本退出                          6. 恢复数据 + DB + 重启容器
7. EXIT trap 触发                    7. 脚本退出
8. _on_exit() 检查 ROLLBACK_NEEDED=0 → 跳过回滚
```

### 测试结果
- 语法检查：✅ backup.sh / restore.sh 通过
- 单元测试：✅ 16/16

---

## 后续优化计划

| 迭代 | 主题 | 状态 |
|------|------|------|
| 第1次 | 修复残留bug和代码清理 | ✅ 完成 |
| 第2次 | 完善CentOS/Rocky Linux的Docker安装 | ✅ 完成 |
| 第3次 | 完善仓库源配置脚本 | ✅ 完成 |
| 第4次 | 优化备份和恢复性能 | ✅ 完成 |
| 第5次 | 增强错误处理和回滚机制 | ✅ 完成 |
| 第6次 | 完善状态管理系统 | ⏳ 待开始 |
| 第7次 | 新增配置向导和诊断工具 | ⏳ 待开始 |
| 第8次 | 跨平台兼容性测试和优化 | ⏳ 待开始 |
| 第9次 | 安全性增强 | ⏳ 待开始 |
| 第10次 | V3.0架构准备 | ⏳ 待开始 |

---

**最后更新**: 2026-05-13 13:15 GMT+8
**当前版本**: v2.0.5
**下一步**: 第6次优化 - 完善状态管理系统
