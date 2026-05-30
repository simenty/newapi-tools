# NewAPI Tools V3.2 — 产品需求文档（PRD）

## 项目信息

| 字段 | 值 |
|------|-----|
| 项目名称 | newapi-tools |
| 版本 | V3.2 |
| 编程语言 | Go 1.22.10 |
| 技术栈 | Cobra CLI + Viper 配置 + slog 日志 + Docker Compose |
| 项目路径 | `d:\Users\Vincent\Desktop\newapi-tools\newapi-tools` |
| 仓库 | https://github.com/simenty/newapi-tools |
| 基线版本 | V3.1.0（已通过 Debian 12 真机验证） |
| PRD 语言 | 中文 |

### 原始需求复述

V3.2 以 **CLI 体验增强 + 自动更新** 为核心，分四批迭代：
1. 第一批（A+D）：交互式安装向导完善、config chmod 自动修复、mirror test 并发测试、命令别名、update 自检与自更新
2. 第二批（B）：审计日志查询、错误码文档生成、doctor --verbose
3. 第三批（C）：多实例管理
4. 第四批（E）：文档与社区

---

## 产品目标

1. **降低零基础用户首次部署门槛**：通过完整的交互式安装向导，让不懂 Docker 的用户 3 分钟内完成 new-api 部署，无需手写任何配置文件
2. **消除工具自身运维负担**：通过自更新和自动修复机制，让用户永远运行最新版本、最健康状态的工具链，无需手动关注版本和安全告警
3. **提升日常运维效率**：通过命令别名、并发镜像测速、审计日志查询等增强，让已部署用户的日常操作更快捷、排查更精准

---

## 用户故事

1. **作为**零基础用户，**我希望**运行 `newapi install --interactive` 后通过问答式向导选择端口/数据库/Redis/镜像源，**以便**无需手动编写 .env 和 docker-compose.yml 就能完成部署

2. **作为**已部署用户，**我希望**运行 `newapi update --check` 自动对比 GitHub 最新 Release，**以便**及时知道是否有新版本可用；运行 `newapi update --self` 自动升级工具自身，**以便**无需手动下载替换二进制

3. **作为**国内用户，**我希望**运行 `newapi mirror test` 时并发测试所有镜像源并按延迟排序展示，**以便**快速选到最快的镜像源加速拉取

4. **作为**运维人员，**我希望** `newapi doctor --fix` 发现配置文件权限过宽时自动执行 chmod 600 修复，**以便**不再看到重复的 WARN 提示且配置安全性得到保障

5. **作为**日常使用者，**我希望**使用 `newapi ls` 快速查看所有容器状态（等同 `newapi status --all`），**以便**减少键盘输入提高操作效率

---

## 需求池

### P0 — 必须完成（第一批：V3.2 核心）

| 编号 | 需求 | 验收标准 | 涉及模块 |
|------|------|----------|----------|
| A-1 | `install --interactive` 交互式安装向导完善 | 向导包含 5 个步骤：① 端口选择 ② 域名输入（可选，为空则跳过） ③ 数据库类型 MySQL/SQLite ④ Redis 开关（独立容器/内嵌/跳过） ⑤ 镜像加速选择（从 6 内置源中选/自动选最快/跳过）。基于问卷结果生成 `.env` + `docker-compose.yml`，SQLite 模式不启动 MySQL 容器，Redis-off 模式从 compose 中移除 Redis 服务 | `internal/cli/install.go` |
| A-2 | `config chmod` 自动修复 | `doctor --fix` 检测到配置文件权限为 0644 时，自动执行 `os.Chmod(path, 0600)` 并输出 `[FIXED]`，不再仅打印 HINT。需 root 或文件属主权限 | `internal/cli/doctor.go` → `runAutoFix` |
| A-3 | `mirror test` 并发测试 + 延迟排序 | 对 6 个内置源并发执行 HTTP HEAD 测速（复用 `docker.AutoSelectMirror` 的并发逻辑），输出表格含：源名称、URL、状态（OK/FAIL）、延迟（ms），按延迟升序排列。无参数时测试所有内置源，有参数时测试指定源 | `internal/cli/mirror.go` → `runMirrorTest` |
| A-4 | 命令别名 `ls` → `status --all` | 在 rootCmd 上注册 `ls` 作为 `status --all` 的别名。用户运行 `newapi ls` 等效于 `newapi status --all` | `internal/cli/status.go` 或 `root.go` |
| D-1 | `update --check` 版本检查 | 从 `https://api.github.com/repos/simenty/newapi-tools/releases/latest` 获取最新 tag，与当前 `core.Version` 对比。有新版本时输出：当前版本、最新版本、下载 URL；无新版本时输出 "已是最新版本"。网络失败时友好提示而非 panic | `internal/cli/update.go`（新增 flag + 函数） |
| D-2 | `update --self` 自更新 | 下载最新 Release 中对应 OS/ARCH 的二进制文件到临时路径 → 备份当前二进制（`newapi-tools.bak.{timestamp}`）→ 原子替换（`os.Rename`）→ 验证新版本 `--version` → 输出升级成功信息。替换失败时从备份恢复。需处理：当前进程正在运行的二进制无法直接覆盖的问题（Linux 下可行） | `internal/cli/update.go`（新增 flag + 自更新逻辑） |
| D-3 | `update --self` 前自动触发 backup | 执行 `update --self` 时自动调用 `performBackup()`，与现有 `update` 的 `--backup=true` 行为一致 | `internal/cli/update.go` |

### P1 — 应当完成（第二批：日志诊断升级）

| 编号 | 需求 | 验收标准 | 涉及模块 |
|------|------|----------|----------|
| B-1 | `audit list` 命令 | 新增 `newapi audit list` 子命令，读取 `~/.config/newapi-tools/audit.log`（JSON Lines），支持 `--last N`（最近 N 条）、`--cmd install`（按命令过滤）、`--since "2026-05-01"`（按时间过滤）。输出表格含：时间、命令、用户、结果、耗时 | `internal/cli/audit.go`（新文件） |
| B-2 | 错误码文档自动生成 | 新增 `go generate` 指令或 Makefile target，从 `internal/apperr/apperr.go` 的常量定义和 `suggestions` map 中提取，生成 `docs/reference/error-codes.md`，含错误码、描述、建议修复。CI 或 `make docs` 时执行 | `internal/apperr/` + `Makefile` |
| B-3 | `doctor --verbose` | 每项检查显示检测详情（如 docker binary 路径、daemon 响应时间、容器运行时长、磁盘总量/可用量等），`--verbose` 关闭时保持现有精简输出 | `internal/cli/doctor.go` → `checkResult` 增加 `Detail` 字段 |

### P2 — 锦上添花（第三批/第四批）

| 编号 | 需求 | 验收标准 | 涉及模块 |
|------|------|----------|----------|
| C-1 | `instance add/list/switch/remove` 多实例管理 | 支持 `newapi instance add --name prod --home /opt/newapi-prod --port 3001`，实例配置存储在 `~/.config/newapi-tools/instances.yml`。`switch` 切换当前活跃实例，后续命令自动作用于活跃实例 | 新增 `internal/cli/instance.go` + `internal/core/instance.go` |
| C-2 | `status --instance <name>` | status 命令增加 `--instance` flag，指定查询非活跃实例的状态 | `internal/cli/status.go` |
| E-1 | `--interactive` 安装教程文档 | 更新 `docs/guide/getting-started.md`，包含交互式安装的完整步骤截图/示例 | `docs/` |
| E-2 | 错误码参考页 | 将 B-2 生成的错误码文档纳入 MkDocs 站点导航 | `docs/` + `mkdocs.yml` |
| E-3 | CHANGELOG.md 标准化 | 按 [Keep a Changelog](https://keepachangelog.com/) 格式补充 V3.0 ~ V3.2 变更记录 | 项目根目录 |

---

## 技术规范要点

### A-1 交互式安装向导 — 详细流程

```
[1/5] 端口选择        → 默认 3000，输入 1-65535 整数
[2/5] 域名配置        → 默认空（跳过），输入域名后写入 .env 的 DOMAIN 变量
[3/5] 数据库类型      → mysql(默认) / sqlite，选择后决定 compose 中是否包含 mysql 服务
[4/5] Redis 配置      → 独立容器(默认) / 内嵌(使用 new-api 内置) / 跳过，决定 compose 中是否包含 redis 服务
[5/5] 镜像加速        → 自动选最快 / 手动选择内置源 / 跳过
```

生成文件规则：
- **docker-compose.yml**：根据 DB/Redis 选择组装 services 块，SQLite 模式不含 mysql，Redis-off 模式不含 redis 且 .env 中 `REDIS_CONN_STRING=` 留空
- **.env**：根据选择写入 `SQL_DSN`、`REDIS_CONN_STRING`、`SESSION_SECRET`、`DOMAIN`（如有）、`MYSQL_ROOT_PASSWORD`（MySQL 模式）

### D-2 自更新 — 流程

```
1. GET /repos/simenty/newapi-tools/releases/latest → 获取 tag + assets
2. 匹配 asset: newapi-tools-{GOOS}-{GOARCH}.tar.gz
3. 下载到 /tmp/newapi-tools-update-xxxxx
4. 备份当前二进制 → newapi-tools.bak.{timestamp}
5. os.Rename(临时文件, 当前路径)
6. 执行 new-api-tools version 验证
7. 验证失败 → 从备份恢复
```

### A-3 mirror test 并发 — 实现思路

复用 `docker.AutoSelectMirror()` 的并发模型（goroutine + channel），但收集所有结果而非仅最快一个。输出按 `Latency` 升序排列的表格。

---

## 待确认问题

| # | 问题 | 影响范围 | 建议 |
|---|------|----------|------|
| 1 | `update --self` 在 Windows 上的原子替换可行性？Windows 运行中的 .exe 无法被覆盖 | D-2 | V3.2 仅支持 Linux 自更新，Windows 用户提示手动下载 |
| 2 | 交互式安装向导的 Redis "内嵌"模式，new-api 镜像是否内置 Redis？ | A-1 | 需确认 new-api 镜像是否支持 `REDIS_CONN_STRING=` 留空时自动使用内嵌 Redis，否则"跳过 Redis"选项可能导致功能异常 |
| 3 | `instance` 多实例的配置文件格式：独立文件 `instances.yml` 还是合并到现有 `newapi-tools.yml`？ | C-1 | 建议独立文件，避免与现有配置结构冲突 |
| 4 | `audit list` 读取大日志文件（>10MB 旋转后）的性能：是否需要索引？ | B-1 | 建议首版仅顺序扫描 + `--last N` 限制条数，不做索引 |
| 5 | GitHub API rate limit：`update --check` 使用匿名 API 每小时 60 次，是否足够？ | D-1 | 对 CLI 工具足够（用户不会高频检查），但需加 `GITHUB_TOKEN` 环境变量支持以备扩展 |
| 6 | 命令别名 `ls` 是否与潜在的未来子命令冲突？ | A-4 | Cobra 的 Alias 机制天然处理，后续新增 `ls` 子命令时自动覆盖别名 |
