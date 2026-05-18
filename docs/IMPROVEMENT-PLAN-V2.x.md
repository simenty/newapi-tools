# NewAPI Tools V2.x 改进方案

> 版本：v2.0 | 日期：2026-05-16 | 范围：V2.x 系列

---

## 核心策略

V2.x 的改进资源集中投入在两个方向：

1. **生态扩展** — 支持更多 AI 网关，让 newapi-tools 不只能管 NewAPI
2. **性能优化** — 让脚本跑得更快、占用更少资源

其他改进方向（文档、用户体验、社区建设）继续推进，但不是本阶段重点。

---

## 方向一：生态扩展

### 为什么重要？

现在 newapi-tools 只支持 NewAPI。但用户可能在用其他 AI 网关（One API、Sub2API 等）。如果我们能支持它们，用户群会大很多。

### 目标网关列表

| 网关 | 类型 | 优先级 | 说明 |
|------|------|--------|------|
| NewAPI | 主流 | P0 | 已有支持，需维护 |
| One API | 主流 | P0 | 最老牌的开源 AI 网关 |
| new-api | 主流 | P0 | NewAPI 的前身/分支 |
| Sub2API | 主流 | P1 | 订阅管理增强版 |

### 实施方案

**阶段 1（V2.2）：支持 One API**

```
目标：让 newapi-tools 能安装和管理 One API
工作量：~5 人日

改动点：
1. 新增 FLAVOR 概念（newapi / one-api）
2. 目录结构调整：
   modules/deploy/newapi/   # NewAPI 专用
   modules/deploy/one-api/  # One API 专用
   modules/manage/           # 通用（不区分 FLAVOR）
3. 命令行接口：
   newapi-tools install --flavor newapi
   newapi-tools install --flavor one-api
```

**阶段 2（V2.3）：支持 new-api 和 Sub2API**

```
目标：扩展支持 2 个更多网关
工作量：~8 人日

改动点：
1. 新增 modules/deploy/new-api/ 和 modules/deploy/sub2api/
2. 统一配置文件格式（不同网关的配置适配层）
3. 更新 doctor.sh 支持检测多个网关
```

**阶段 3（V2.4）：插件式架构 + 网关间迁移**

```
目标：支持三个网关的插件式管理，并提供迁移工具
工作量：~5 人日

改动点：
1. 插件式架构（每个网关一个插件目录）
   modules/plugins/one-api/
   modules/plugins/new-api/
   modules/plugins/sub2api/
2. 网关间迁移工具
   newapi-tools migrate --from one-api --to newapi
3. 统一命令接口（所有网关共用同一套命令）
   newapi-tools install --flavor one-api
   newapi-tools backup --flavor new-api
   newapi-tools restore --flavor sub2api
```

**完成标志**：三个主流网关（One API、new-api、Sub2API）全部支持，插件式架构成型，迁移工具可用。

### 技术难点

| 难点 | 解决方案 |
|------|---------|
| 不同网关的配置格式不同 | 抽象配置适配层，统一接口 |
| Docker 镜像名称不同 | FLAVOR 映射到对应镜像 |
| 数据库连接方式不同 | 抽象数据库操作，按 FLAVOR 分发 |
| 版本兼容性 | 每个 FLAVOR 维护版本矩阵 |

---

## 方向二：性能优化

### 为什么重要？

随着功能增多，脚本执行时间变长。用户反馈 `doctor.sh` 要跑 15 秒，太慢了。优化后能让用户体验更流畅。

### 优化点

#### 优化 1：诊断并行执行

**当前问题**：`doctor.sh` 串行执行所有诊断，总耗时 ~15 秒。

**优化方案**：把独立的诊断项并行执行。

```bash
# 优化前（串行）
diagnose_network     # 3-5 秒
diagnose_docker     # 2-3 秒
diagnose_disk       # 1 秒
diagnose_memory     # 1 秒
# 总耗时：~15 秒

# 优化后（并行）
diagnose_network &  # 后台执行
pid_network=$!
diagnose_docker &   # 后台执行
pid_docker=$!
wait $pid_network $pid_docker
# 总耗时：~5 秒（提升 3x）
```

**注意事项**：
- 有依赖关系的诊断项不能并行（如 `diagnose_newapi` 依赖 `diagnose_docker`）
- 需要捕获每个并行任务的输出，最后统一显示

**工作量**：~2 人日

---

#### 优化 2：系统信息缓存

**当前问题**：每次调用 `detect_system_info()` 都重新读取 `/etc/os-release`，浪费时间。

**优化方案**：把系统信息缓存到文件，1 小时内不重复检测。

```bash
detect_system_info_cached() {
    local cache_file="$STATE_DIR/system-info.cache"
    
    # 缓存有效期：1 小时
    if [[ -f "$cache_file" ]] && [[ $(find "$cache_file" -mmin -60) ]]; then
        source "$cache_file"
        return 0
    fi
    
    # 重新检测
    OS_NAME=$(cat /etc/os-release | grep "^NAME=" | cut -d'"' -f2)
    OS_VERSION=$(cat /etc/os-release | grep "^VERSION_ID=" | cut -d'"' -f2)
    # ...
    
    # 写入缓存
    cat > "$cache_file" <<EOF
OS_NAME="$OS_NAME"
OS_VERSION="$OS_VERSION"
# ...
EOF
}
```

**工作量**：~1 人日

---

#### 优化 3：日志轮转

**当前问题**：`toolkit.log` 无限增长，可能占满磁盘。

**优化方案**：配置 logrotate，自动轮转日志。

```bash
setup_logrotate() {
    local logrotate_conf="/etc/logrotate.d/newapi-tools"
    
    cat > "$logrotate_conf" <<'EOF'
/opt/newapi-tools/logs/*.log {
    daily
    rotate 7
    compress
    missingok
    notifempty
    create 0644 root root
}
EOF
    
    log_info "已配置日志轮转（保留 7 天）"
}
```

**工作量**：~0.5 人日

---

#### 优化 4：Docker 镜像预检

**当前问题**：`install.sh` 每次都 `docker pull`，即使用户的镜像已经是最新的。

**优化方案**：先检查本地是否已有最新镜像，避免重复拉取。

```bash
pull_image_if_needed() {
    local image="$1"
    local tag="${2:-latest}"
    
    # 检查本地是否已有此镜像
    if docker image inspect "${image}:${tag}" &>/dev/null; then
        # 检查远程是否有更新（比较 digest）
        local local_digest=$(docker image inspect "${image}:${tag}" --format '{{.Id}}')
        local remote_digest=$(docker manifest inspect "${image}:${tag}" 2>/dev/null | jq -r '.config.digest' || echo "unknown")
        
        if [[ "$local_digest" == "$remote_digest" ]]; then
            log_info "镜像 ${image}:${tag} 已是最新，跳过拉取"
            return 0
        fi
    fi
    
    # 需要拉取
    docker pull "${image}:${tag}"
}
```

**工作量**：~1 人日

---

### 优化效果预估

| 指标 | 优化前 | 优化后 | 提升 |
|------|--------|--------|------|
| `doctor.sh` 执行时间 | ~15s | < 5s | 3x |
| `install.sh` 执行时间 | ~5min | ~3min | 1.7x |
| 系统信息检测耗时 | ~1s | < 0.1s（缓存命中） | 10x |
| 日志占用磁盘 | 无限制 | ≤ 100MB | - |

---

## 实施时间表

```
2026-05  |  2026-06  |  2026-07  |  2026-08  |  2026-09
────────┼───────────┼───────────┼───────────┼──────────
V2.0.8  |   V2.1     |   V2.2     |   V2.3     |   V2.4
(Bug修复)|  (文档+体验) | (One API) | (性能优化)  | (更多网关)
```

**里程碑**：
- 2026-05-20：V2.0.8 发布（Bug 修复完成）
- 2026-06-01：V2.1 发布（文档完善 + 用户体验优化）
- 2026-07-01：V2.2 发布（One API 支持）
- 2026-08-01：V2.3 发布（性能优化）
- 2026-09-01：V2.4 发布（更多网关支持）

---

## 资源需求

| 改进方向 | 所需资源 | 优先级 |
|---------|---------|--------|
| 生态扩展 | 开发工程师 1 人 | 🔴 高 |
| 性能优化 | 开发工程师 0.5 人 | 🟡 中 |
| 文档完善 | 文档工程师 0.5 人 | 🟡 中 |
| 用户体验优化 | 开发工程师 0.5 人 | 🟡 中 |

**总计**：~2.5 人月（V2.1 ~ V2.4）

---

## 风险评估

| 风险 | 影响 | 应对措施 |
|------|------|---------|
| One API 兼容性差 | V2.2 无法实现 | 提前调研，确认可行性 |
| 性能优化效果不明显 | 用户无感知 | 优化前先做基准测试，量化效果 |
| 三个网关的维护成本 | 代码重复 | 插件式架构，统一接口层 |
| 开发资源不足 | 路线图延期 | 优先高优先级项目，低优先级可延期 |

---

## 成功标准

- V2.4 结束时，支持的网关数量 ≥ 3（One API、new-api、Sub2API）
- `doctor.sh` 执行时间 < 5 秒
- 插件式架构成型，支持社区贡献新网关插件
- 网关间迁移工具可用（`migrate` 命令）
- 用户满意度（性能）≥ 4.0/5.0

---

*本文档随产品迭代持续更新。*

---

## 实施进度（截至 2026-05-16）

| 优化项 | 状态 | 说明 |
|--------|------|------|
| 优化1：诊断并行执行 | ✅ 已完成 | doctor.sh 支持并行诊断，效果~3x提升 |
| 优化2：系统信息缓存 | ✅ 已完成 | common.sh 添加 detect_system_info_cached() |
| 优化3：日志轮转 | ✅ 已完成 | common.sh 添加 setup_logrotate() |
| 优化4：Docker 镜像预检 | ✅ 已完成 | common.sh 添加 pull_image_if_needed()，install.sh 已集成 |
| 全面审计 #1-#4 | ✅ 已完成 | 每完成一阶段立即审计，全部通过 |

**下一步**：V2.2 进行中（FLAVOR 概念已实现，待审计和 Linux 环境验证）

---

## V2.2 实施详情（截至 2026-05-16）

### 已完成

1. **`lib/config.sh` 添加 FLAVOR 配置**
   - 新增 `deploy.flavor: "newapi"`（默认 FLAVOR）
   - 新增 `deploy.one_api.*` 配置段（Image、端口、Session Secret、SQLite 开关）
   - 保留 `deploy.newapi.*` 配置段（兼容旧配置）

2. **目录结构调整**
   - 创建 `modules/deploy/newapi/install.sh`（NewAPI 专用安装脚本）
   - 创建 `modules/deploy/one-api/install.sh`（One API 专用安装脚本）
   - 重写 `modules/deploy/install.sh` 作为路由脚本（支持 `--flavor` 参数）

3. **命令行接口**
   - `newapi-tools install`（默认 `--flavor newapi`）
   - `newapi-tools install --flavor newapi`
   - `newapi-tools install --flavor one-api`

4. **One API 安装脚本特性**
   - 支持 SQLite（默认）/ MySQL 两种数据库模式
   - 自动生成 Redis 密码和 Session Secret
   - 生成 `docker-compose.yml`（One API + Redis + NPM）
   - 镜像预检（避免重复拉取）
   - 回滚机制

### 待完成

- 审计（语法检查、安全检查、功能完整性）
- Linux 环境验证（10.100.10.36）
- 更新 `docs/FLAVOR-GUIDE.md`（FLAVOR 使用指南）

### 修改的文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `lib/config.sh` | 修改 | 添加 `deploy.flavor` 和 `deploy.one_api.*` |
| `modules/deploy/install.sh` | 重写 | 改为路由脚本，支持 `--flavor` 参数 |
| `modules/deploy/newapi/install.sh` | 创建 | NewAPI 专用安装脚本 |
| `modules/deploy/one-api/install.sh` | 创建 | One API 专用安装脚本 |
| `newapi-tools.sh` | 修改 | 更新 install 帮助文本 |