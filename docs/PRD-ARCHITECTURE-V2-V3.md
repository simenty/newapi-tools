# NewAPI Tools 架构演进 PRD：V2 整合优化 → V3 架构升级

> 版本：v1.0
> 日期：2026-07
> 作者：产品经理
> 状态：草案

---

## 一、产品目标

### 1.1 V2 整合优化目标

1. **修掉该修的，稳住基本盘** — 清偿 V2.0 遗留的技术债（ssl-proxy.sh 语法错误、unit-test.sh 挂起、env.sh 弃用未清理），让当前功能"能用且靠谱"
2. **理顺模块边界，为扩展扫路** — 把 deploy 模块已验证的 FLAVOR 拆分模式推广到 manage/monitor 模块，消除路由硬编码和跨模块耦合
3. **统一基础设施** — 把散落在各处的配置读取、错误处理、日志输出收敛到 lib/ 层，给 V3 插件框架打地基

### 1.2 V3 架构升级目标

1. **插件式架构** — 每个网关一个插件目录，社区可以零修改核心代码地贡献新网关支持
2. **多平台兼容** — 从"只跑 x86_64 Ubuntu"扩展到 x86_64 + ARM64 + 7 个主流 Linux 发行版
3. **网关间迁移** — 用户从 One API 搬到 NewAPI（或反过来），一条命令搞定

### 1.3 V2→V3 渐进式演进策略

核心原则：**每个版本都是可发布的，不搞大爆炸重写。**

```
V2.0.8  修 Bug，止血
  ↓
V2.1    文档 + 体验优化（不动架构）
  ↓
V2.2    FLAVOR 推广到 manage 模块 + 路由表注册化
  ↓      ← 此时架构已接近 V3 的雏形
V2.3    高级运维功能（在新的模块边界上开发）
  ↓
V2.4    插件系统 MVP（lib/plugin.sh + metadata.yml 规范冻结）
  ↓      ← V2.4 = V3.0-alpha
V3.0    正式版：插件市场 + 多平台 + 迁移工具
```

判断标准：V2.4 发布后，如果新网关支持只需要"加一个目录 + 写 metadata.yml"而不改任何核心代码，则架构演进成功。

---

## 二、用户故事

### 2.1 核心用户场景

**US-1：一键部署 AI 网关**
> 作为个人 AI 玩家，我想在刚买的云主机上一键部署 NewAPI，这样我不用学 Docker 也能用上自己的 AI 接口。
> 验收标准：运行 `newapi-tools install`，10 分钟内完成部署，浏览器能打开管理页面。

**US-2：换个网关继续用**
> 作为 One API 老用户，我想迁移到 NewAPI 而不丢数据，这样我能用上更丰富的功能。
> 验收标准：`newapi-tools migrate --from one-api --to newapi` 完成后，令牌和渠道数据完整迁移。

**US-3：出了问题能自诊**
> 作为不熟悉 Linux 的用户，我想一键检查系统哪里有问题，这样我不用自己翻日志。
> 验收标准：`newapi-tools doctor` 输出诊断报告，红色标记具体问题和修复建议。

**US-4：自动备份睡得着**
> 作为小团队技术负责人，我想让系统每天自动备份数据，这样即使服务器挂了也不慌。
> 验收标准：配置 cron 后，每天自动备份到指定目录，超期备份自动清理。

**US-5：更新出问题能回滚**
> 作为正在升级 NewAPI 的用户，我想如果新版本有问题就自动回退，这样我不怕升级翻车。
> 验收标准：`newapi-tools update` 后健康检查失败时，自动回滚到旧版本并通知。

**US-6：一台服务器管多个实例**
> 作为管理多个 AI 网关实例的用户，我想在一个入口看到所有实例状态，这样不用逐个登录。
> 验收标准：`newapi-tools instances` 列出所有已安装的网关实例及其运行状态。

**US-7：按需安装网关插件**
> 作为进阶用户，我想选择安装哪些网关支持，这样工具不会臃肿。
> 验收标准：`newapi-tools plugin list` 查看可用插件，`plugin install <name>` 按需安装。

**US-8：在 ARM 服务器上也能用**
> 作为树莓派/ARM 服务器用户，我想用同一套工具管理我的 AI 网关，这样不用找替代方案。
> 验收标准：在 aarch64 Ubuntu 上正常运行所有核心功能。

### 2.2 针对架构改进的用户故事

**US-A1：模块独立更新**
> 作为 newapi-tools 维护者，我想更新 backup 模块而不影响 install 模块，这样改一个功能不用全量回归测试。
> 关联架构需求：模块解耦（P0）

**US-A2：新增网关零改动核心**
> 作为社区贡献者，我想通过添加一个目录来支持新网关，这样不用 fork 整个项目。
> 关联架构需求：插件框架（P0）

**US-A3：配置一处定义处处生效**
> 作为用户，我想通过一个 config.yaml 管理所有配置，而不是到处找 .env 和 toolkit.conf。
> 关联架构需求：配置统一（P0）

---

## 三、需求池

### P0 — 必须做（不做就卡住）

| 编号 | 需求 | 说明 | 目标版本 |
|------|------|------|---------|
| P0-1 | 模块解耦 | manage/ 和 monitor/ 模块按 FLAVOR 拆分，与 deploy/ 对齐；消除模块间的 source 链式依赖 | V2.2 |
| P0-2 | 路由表注册化 | newapi-tools.sh 的 `route_command()` 从硬编码 case 改为动态注册表，模块自声明路由 | V2.2 |
| P0-3 | 统一错误处理框架 | 每个模块统一 `trap_error` + 错误码体系（`ERR_模块_序号`），不再各写各的 | V2.1 |
| P0-4 | 配置体系统一 | 废弃 `lib/env.sh`，全部走 `lib/config.sh` 的 `get_config()/set_config()` | V2.2 |
| P0-5 | 修复 ssl-proxy.sh 语法错误 | 第 158/162 行未闭合引号，导致 SSL 配置命令直接报错 | V2.0.8 |
| P0-6 | 修复 unit-test.sh 挂起 | state.sh 测试后进程不退出，阻塞 CI | V2.0.8 |
| P0-7 | 插件框架核心 | `lib/plugin.sh`（加载/卸载/校验）+ `metadata.yml` 规范 + 示例插件 | V2.4 |

### P1 — 应该做（显著提升产品价值）

| 编号 | 需求 | 说明 | 目标版本 |
|------|------|------|---------|
| P1-1 | manage 模块 FLAVOR 拆分 | backup/restore/update/reinstall 按 FLAVOR 拆分子脚本，与 deploy/ 对齐 | V2.2 |
| P1-2 | doctor.sh 多网关诊断 | 诊断结果区分 FLAVOR，不再硬编码 `new-api` 容器名 | V2.2 |
| P1-3 | 多实例管理 | 支持同一台服务器上安装和切换多个网关实例 | V2.3 |
| P1-4 | 迁移工具 | `newapi-tools migrate --from <flavor> --to <flavor>`，支持数据导出→格式转换→导入 | V3.0 |
| P1-5 | 监控增强 | 实时资源监控 + 告警规则（CPU/内存/磁盘阈值 + Webhook 通知） | V2.3 |
| P1-6 | OS 适配层 | `lib/os_adapter.sh`，统一包管理器/Docker 安装/服务管理的跨发行版差异 | V3.0 |
| P1-7 | ARM64 支持 | 在 aarch64 上验证全功能，修复架构相关的兼容问题 | V3.0 |

### P2 — 可以做（锦上添花）

| 编号 | 需求 | 说明 | 目标版本 |
|------|------|------|---------|
| P2-1 | i18n 框架 | 脚本输出支持中/英双语，通过 `$LANG` 自动检测 | V2.4+ |
| P2-2 | 错误码 + 文档映射 | 所有错误分配 `ERR_xxx_yyy` 编号，文档中可查修复方法 | V2.1 |
| P2-3 | 脚本执行性能优化 | 减少 `command -v` 重复调用、缓存检测结果、并行执行独立任务 | V2.3 |
| P2-4 | 备份后端扩展 | 支持 S3/OSS 远程备份、增量备份、GPG 加密 | V2.3 |
| P2-5 | 插件市场（MVP） | `newapi-tools plugin search/install/remove`，从 GitHub 仓库拉取 | V3.1 |
| P2-6 | Web 管理面板（可选） | 轻量 Web UI，可选安装，不替代命令行 | V3.1+ |

---

## 四、V2.0 整合优化方案

### 4.1 当前架构问题诊断

#### 问题 1：模块耦合 — "manage/ 是个大杂烩"

**现状**：deploy/ 已经按 FLAVOR 拆分了子目录（newapi/、one-api/、sub2api/），但 manage/ 下 7 个脚本全是"NewAPI 专用但没标明"的：

```
manage/
  backup.sh      ← 硬编码 new-api 容器名和 MySQL 数据库
  restore.sh     ← 同上
  update.sh      ← 硬编码 calciumion/new-api:latest 镜像
  reinstall.sh   ← 硬编码 NewAPI 逻辑
  uninstall.sh   ← 同上
  config.sh      ← 假设 NewAPI 配置结构
  doctor.sh      ← 硬编码 new-api、mysql、redis、npm 容器名
```

**问题**：One API 用 SQLite 不用 MySQL，Sub2API 端口是 8080 不是 3000——manage/ 现有的脚本跑在别的 FLAVOR 上全废。

#### 问题 2：路由硬编码 — "加个命令得改主入口"

**现状**：`newapi-tools.sh` 第 48-79 行的 `route_command()` 是硬编码的 case 语句：

```bash
route_command() {
    case "$cmd" in
        backup|restore|update) echo "${MODULES_DIR}/manage/${cmd}.sh" ;;
        install)               echo "${MODULES_DIR}/deploy/install.sh" ;;
        ssl)                   echo "${MODULES_DIR}/deploy/ssl-proxy.sh" ;;
        ...
    esac
}
```

**问题**：每加一个命令（比如 `migrate`、`instances`）就得改这个文件，模块无法"自注册"。

#### 问题 3：依赖加载混乱 — "source 链不知道会拉进来什么"

**现状**：
- `newapi-tools.sh` 一次性 source 全部 6 个 lib
- 子脚本（backup.sh 等）又自己 source 一遍（有的还用 `$TOOLKIT_ROOT`，有的用 `$SCRIPT_DIR/../..`）
- `lib/state.sh` 内部 source `common.sh` + `security.sh`
- `lib/common.sh` 内部 source `security.sh`
- 结果：security.sh 被加载了 2-3 次，靠 `_SECURITY_SH_LOADED` 守卫防重入

**问题**：依赖关系是隐式的，改一个 lib 的 source 顺序可能炸掉另一个模块。

#### 问题 4：配置双轨制 — "env.sh 和 config.sh 到底听谁的"

**现状**：
- `lib/env.sh`（V1.0 兼容层）从 `toolkit.conf` 读 key=value 配置
- `lib/config.sh`（V2.0 新增）从 `config.yaml` 读 YAML 配置
- 两者同时被 `newapi-tools.sh` source，优先级不明确
- `env.sh` 头部注释说"计划 V3.0 移除"但没人敢动它

**问题**：新功能不知道该用 `get_config()` 还是 `$NEWAPI_HOME`，两套配置可能互相覆盖。

#### 问题 5：已知 Bug 阻塞

| Bug | 位置 | 影响 |
|-----|------|------|
| ssl-proxy.sh 第 158/162 行引号未闭合 | `modules/deploy/ssl-proxy.sh` | SSL 配置命令直接语法报错，无法使用 |
| unit-test.sh 在 state.sh 测试后挂起 | `test/unit-test.sh` | CI 跑不完，测试覆盖率无法持续监控 |
| doctor.sh 重复声明并行诊断函数 | `modules/manage/doctor.sh` 第 28-134 行 | `_run_diag_bg` 等函数被定义了两遍，功能正常但代码冗余 |

### 4.2 整合优化策略

#### 策略 1：模块边界重新划分

**原则**：deploy/ 的 FLAVOR 拆分模式已验证可行，推广到 manage/ 和 monitor/。

```
modules/
  deploy/
    install.sh              # 路由脚本（已有）
    ssl-proxy.sh            # 通用 SSL（不区分 FLAVOR）
    newapi/install.sh       # NewAPI 部署（已有）
    one-api/install.sh      # One API 部署（已有）
    sub2api/install.sh      # Sub2API 部署（已有）
  manage/
    backup.sh               # 路由脚本（新增）
    restore.sh              # 路由脚本（新增）
    update.sh               # 路由脚本（新增）
    reinstall.sh            # 路由脚本（新增）
    uninstall.sh            # 路由脚本（新增）
    config.sh               # 通用配置向导（保留）
    doctor.sh               # 通用诊断框架（重构）
    newapi/                 # NewAPI 专用管理
      backup.sh
      restore.sh
      update.sh
      reinstall.sh
      uninstall.sh
    one-api/                # One API 专用管理
      backup.sh
      restore.sh
      ...
  monitor/
    health.sh               # 路由脚本（新增）
    logs.sh                 # 路由脚本（新增）
    newapi/
      health.sh
      logs.sh
    one-api/
      health.sh
      logs.sh
```

**迁移方式**：先把现有 backup.sh 的逻辑提取到 `manage/newapi/backup.sh`，原文件改为路由脚本。增量改造，不改功能。

#### 策略 2：路由注册表

用一个简单的注册表文件替代硬编码 case：

```bash
# lib/registry.sh — 命令注册表
declare -A CMD_REGISTRY

register_cmd() {
    local cmd="$1"
    local script="$2"
    local help_text="${3:-}"
    CMD_REGISTRY["$cmd"]="$script"
}

# 模块自注册（在模块脚本头部调用）
# deploy/install.sh 里：
register_cmd "install" "${MODULES_DIR}/deploy/install.sh" "部署 AI 网关 [--flavor <flavor>]"

# 管理模块里：
register_cmd "backup"  "${MODULES_DIR}/manage/backup.sh"  "手动备份数据"
register_cmd "migrate" "${MODULES_DIR}/manage/migrate.sh"  "网关间迁移"

# 路由查找
route_command() {
    local cmd="$1"
    echo "${CMD_REGISTRY[$cmd]:-}"
}
```

**好处**：加新命令只需在新模块里 `register_cmd`，不用改 `newapi-tools.sh`。

#### 策略 3：依赖加载收敛

**目标**：只有 `newapi-tools.sh` 负责加载 lib/，子脚本不再自行 source。

```
newapi-tools.sh（加载所有 lib）
  └─ exec bash "$SCRIPT_PATH" "$@"（子脚本继承环境）
```

子脚本通过 `export -f` 继承函数，不再自己 `source lib/xxx.sh`。

**问题**：`exec bash` 启动新进程会丢掉 export -f 的函数。需要改为：

```bash
# 方案 A：子脚本用 #!/bin/bash + source init.sh（单文件初始化）
# 方案 B：用 BASH_ENV 环境变量传递函数
# 方案 C：生成一个 init 脚本让子脚本 source
```

推荐方案 C：生成 `${TOOLKIT_ROOT}/lib/_init.sh`，子脚本头部统一 `source "${TOOLKIT_ROOT}/lib/_init.sh"`，内部只 source 一次。

#### 策略 4：配置统一

- V2.0.8：`lib/env.sh` 加 deprecation warning（运行时打 WARN 日志）
- V2.1：`lib/env.sh` 内部改为调用 `get_config()` 读取，保持兼容
- V2.2：`lib/env.sh` 正式移除，所有代码走 `get_config()/set_config()`

### 4.3 关键决策点

| 决策 | 选项 A | 选项 B | 建议 |
|------|--------|--------|------|
| manage/ 是按 FLAVOR 拆子目录还是用函数分发？ | 子目录（与 deploy/ 对齐） | 函数内 if/else 分发 | **选 A**：目录结构一致，插件化时直接迁移 |
| 路由注册用 Shell 关联数组还是外部文件？ | `declare -A` 关联数组 | `registry.conf` 文件 | **选 A**：Shell 内置，不增加文件 I/O；Bash 4+ 已是主流 |
| lib/ 依赖加载是一次性全加载还是按需加载？ | 全加载（当前模式） | 按需加载（只 source 用到的） | **选 A**：全加载简单可靠，Shell 脚本体量不大，性能差异可忽略 |
| env.sh 何时移除？ | V2.1（早移早轻松） | V2.2（留一个版本缓冲） | **选 B**：V2.1 先改内部实现为代理模式，V2.2 确认无下游依赖后移除 |

---

## 五、V3.0 架构愿景

### 5.1 目标架构

```
newapi-tools.sh（主入口，极简：加载 core + 查注册表 + 路由）
│
├── lib/core/                    # 核心库（不可插拔）
│   ├── common.sh                # 日志、颜色、工具函数
│   ├── security.sh              # 安全函数
│   ├── state.sh                 # 状态管理
│   ├── config.sh                # 配置管理（YAML）
│   ├── ui.sh                    # UI 组件
│   ├── mode.sh                  # 新手/专家模式
│   ├── registry.sh              # 命令注册表
│   ├── plugin.sh                # 插件框架（V3.0 新增）
│   └── os_adapter.sh            # OS 适配层（V3.0 新增）
│
├── modules/core/                # 核心模块（不可插拔）
│   ├── init/                    # 环境初始化
│   │   ├── dns.sh
│   │   ├── apt-source.sh → os_adapter.sh 代理
│   │   └── docker.sh   → os_adapter.sh 代理
│   ├── deploy/ssl-proxy.sh      # SSL 配置（通用）
│   ├── manage/
│   │   ├── config.sh            # 配置向导（通用）
│   │   └── doctor.sh            # 诊断框架（通用）+ 插件扩展点
│   └── monitor/
│       └── health-base.sh       # 健康检查框架（通用）
│
├── modules/plugins/             # 网关插件（可插拔）
│   ├── newapi/
│   │   ├── metadata.yml         # 插件元数据
│   │   ├── install.sh           # 安装
│   │   ├── backup.sh            # 备份
│   │   ├── restore.sh           # 恢复
│   │   ├── update.sh            # 更新
│   │   ├── doctor.sh            # 诊断扩展
│   │   └── migrate-from/        # 迁移入支持
│   │       └── one-api.sh       # 从 One API 迁入
│   ├── one-api/
│   │   ├── metadata.yml
│   │   └── ...
│   └── sub2api/
│       ├── metadata.yml
│       └── ...
│
└── scripts/                     # 工具脚本
    ├── encrypt-config.sh
    ├── self-update.sh
    └── verify-deployment.sh
```

### 5.2 插件框架设计

**metadata.yml 规范**：

```yaml
name: "NewAPI"
flavor: "newapi"                    # 唯一标识，命令行用
version: "1.0.0"                    # 插件版本
min_tools_version: "3.0.0"          # 最低工具集版本
docker_image: "calciumion/new-api"  # Docker 镜像
default_port: 3000                  # 默认端口
database: "mysql"                   # mysql | sqlite | none
dependencies:                       # 依赖的服务
  - docker
  - mysql
  - redis
  - npm
hooks:                              # 可注册的钩子
  - install                         # 必须实现
  - backup                          # 必须实现
  - restore                         # 必须实现
  - update                          # 必须实现
  - doctor                          # 可选
  - health                          # 可选
  - logs                            # 可选
migrate_from:                       # 可迁移入的源网关
  - one-api
  - new-api
```

**插件加载流程**：

```bash
# lib/plugin.sh
load_plugin() {
    local flavor="$1"
    local plugin_dir="$MODULES_DIR/plugins/$flavor"

    # 1. 校验 metadata.yml
    validate_metadata "$plugin_dir/metadata.yml" || return 1

    # 2. 检查最低版本
    check_min_version "$plugin_dir/metadata.yml" || return 1

    # 3. 注册命令（自动）
    #   install  → newapi-tools install --flavor newapi
    #   backup   → newapi-tools backup --flavor newapi
    #   doctor   → newapi-tools doctor（自动扩展）
    register_plugin_commands "$flavor" "$plugin_dir"

    # 4. 注册钩子
    register_plugin_hooks "$flavor" "$plugin_dir"
}

list_plugins() {
    for dir in "$MODULES_DIR/plugins"/*/; do
        [[ -f "$dir/metadata.yml" ]] && parse_metadata "$dir/metadata.yml"
    done
}
```

### 5.3 技术约束

| 约束 | 说明 | 原因 |
|------|------|------|
| 纯 Shell | 不引入 Python/Node/Ruby 等运行时依赖 | 产品定位：零依赖、轻量 |
| Bash 4.0+ | 可以用 `declare -A` 关联数组 | Ubuntu 20.04+ 默认 Bash 5.x，CentOS 7 默认 4.2 |
| 可选依赖 | yq/jq/python3 用于 YAML 解析，缺失时降级 | 不强装，已有降级方案 |
| 向下兼容 | V2.x 的命令和配置在 V3.0 必须可用 | 不逼用户升级 |
| 单文件可执行 | 主入口仍是一个 `newapi-tools.sh` | 安装方式不变 |

### 5.4 关键里程碑

| 里程碑 | 日期 | 交付物 | 前置条件 |
|--------|------|--------|---------|
| V2.0.8 发布 | 2026-05 | Bug 清零 + 安全审计通过 | — |
| V2.1 发布 | 2026-06 | 错误码体系 + env.sh 代理模式 + 文档完善 | V2.0.8 |
| V2.2 发布 | 2026-07 | manage/ FLAVOR 拆分 + 路由注册表 + env.sh 移除 | V2.1 |
| V2.3 发布 | 2026-08 | 多实例管理 + 监控增强 + OS 适配层雏形 | V2.2 |
| V2.4 发布 (= V3.0-alpha) | 2026-09 | lib/plugin.sh + metadata.yml 规范冻结 + 3 个内置插件迁移到插件目录 | V2.3 |
| V3.0-beta | 2026-11 | 迁移工具 + ARM64 验证 + 多 OS 测试 | V2.4 |
| V3.0 正式 | 2026-12 | 全功能发布 | V3.0-beta |

---

## 六、已确认决策

> 以下 8 项关键问题经用户于 2026-05-16 确认，已关闭。

| # | 问题 | 决策 | 决策理由 |
|---|------|------|---------|
| 1 | **metadata.yml 解析方案** | 使用 yq | 项目已在 config.sh 中依赖 yq，非新增依赖；metadata.yml 有嵌套结构（commands/dependencies/backup），简化 Shell 解析无法胜任；缺 yq 时提示用户安装 |
| 2 | **manage/ 拆分粒度** | 粗粒度 | 每个 FLAVOR 按"模块大块"拆（backup.sh/restore.sh 等），不做细粒度单命令拆分；通用逻辑抽取到 base 脚本，FLAVOR 脚本 source 后只实现差异部分 |
| 3 | **env.sh 移除时机** | V2.1 开始移除 | 项目未正式发布，无向后兼容负担；V2.1 将 env.sh 改为代理模式（转发到 config.sh），V2.2 完全删除 |
| 4 | **旧路径向后兼容** | 直接断开，不保留兼容层 | 项目未正式发布，不存在用户自定义脚本调用旧路径的场景 |
| 5 | **插件加载方式** | 启动时全量扫描 | 3 个 FLAVOR 插件数量少，启动时扫描 metadata.yml 注册命令和钩子；不做运行时热加载（Shell 不适合） |
| 6 | **跨 FLAVOR 迁移工具** | V2.2 实现 | V2.2 完成 manage/ FLAVOR 拆分 + 路由注册化后，架构已支持多 FLAVOR 共存，此时实现迁移工具水到渠成 |
| 7 | **hook 执行顺序** | 简单优先级数字 | 项目仅 3 个 FLAVOR 插件，hook 注册量极少（每个插件 4-5 个），优先级数字足够；DAG 在 Shell 中实现成本高、收益低；如未来需要可升级 |
| 8 | **V3.0 目录重组方式** | 一次性全量迁移 | V2.4(=V3.0-alpha) 时一次性将 lib/ → lib/core/、manage/ → plugins/，所有 source 路径集中更新；增量迁移反而增加维护成本 |

---

## 附录 A：V2.0 代码依赖关系图（现状）

```
newapi-tools.sh
  ├─ source lib/common.sh
  │    └─ source lib/security.sh
  ├─ source lib/env.sh          ← V1.0 兼容层（弃用但未移除）
  ├─ source lib/state.sh
  │    └─ source lib/common.sh  ← 重复加载
  │    └─ source lib/security.sh ← 重复加载
  ├─ source lib/ui.sh
  ├─ source lib/config.sh
  │    └─ source lib/common.sh  ← 重复加载
  │    └─ source lib/security.sh ← 重复加载
  ├─ source lib/smart-defaults.sh
  └─ source lib/mode.sh

子脚本（backup.sh 等）：
  ├─ source lib/common.sh       ← 又加载一遍
  ├─ source lib/state.sh        ← 又加载一遍
  ├─ source lib/ui.sh           ← 又加载一遍
  ├─ source lib/config.sh       ← 又加载一遍
  └─ source lib/mode.sh         ← 又加载一遍
```

问题一目了然：security.sh 在一次完整执行中可能被加载 5-6 次。

## 附录 B：V2.2 目标依赖关系图

```
newapi-tools.sh
  ├─ source lib/_init.sh        ← 单入口初始化
  │    ├─ source lib/common.sh
  │    │    └─ source lib/security.sh
  │    ├─ source lib/state.sh   ← 不再自行 source common/security
  │    ├─ source lib/ui.sh      ← 不再自行 source common
  │    ├─ source lib/config.sh  ← 不再自行 source common/security
  │    ├─ source lib/smart-defaults.sh
  │    ├─ source lib/mode.sh
  │    └─ source lib/registry.sh ← 新增
  └─ source lib/env.sh          ← 代理模式（V2.2 后移除）

子脚本（backup.sh 等）：
  └─ source lib/_init.sh        ← 统一单入口
```

每个 lib 文件只加载一次。

## 附录 C：RICE 评分（架构需求）

| 需求 | Reach | Impact | Confidence | Effort | RICE | 优先级 |
|------|-------|--------|------------|--------|------|--------|
| P0-5 修复 ssl-proxy.sh | 100 | 5 | 100% | 0.5 | 1000 | P0 |
| P0-6 修复 unit-test.sh | 50 | 4 | 90% | 0.5 | 360 | P0 |
| P0-3 统一错误处理 | 100 | 3 | 90% | 1 | 270 | P0 |
| P0-4 配置体系统一 | 100 | 3 | 80% | 1.5 | 160 | P0 |
| P0-1 模块解耦 | 80 | 4 | 80% | 2 | 128 | P0 |
| P0-2 路由注册化 | 60 | 3 | 85% | 1.5 | 102 | P0 |
| P0-7 插件框架核心 | 40 | 5 | 70% | 3 | 47 | P0 |
| P1-1 manage FLAVOR 拆分 | 60 | 4 | 85% | 2 | 102 | P1 |
| P1-4 迁移工具 | 30 | 4 | 60% | 3 | 24 | P1 |

---

*本文档随产品迭代持续更新。*
