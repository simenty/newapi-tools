# NewAPI Tools V3.0 规划

> 版本：v1.0 | 日期：2026-05-16 | 目标发布：2026-Q4

---

## 核心目标

V3.0 是 NewAPI Tools 的**架构升级版本**，重点解决两个问题：

1. **强大的兼容性** — 支持更多 CPU 架构和操作系统
2. **生态集成** — 支持多个 AI 网关，插件式架构，社区可扩展

---

## 核心特性一：强大的兼容性（多平台架构）

### 1.1 CPU 架构支持

| 架构 | 状态 | 优先级 | 说明 |
|------|------|--------|------|
| x86_64 | ✅ 已支持 | P0 | **V3.0 重点，全力保障** |
| ARM64 / aarch64 | 🎯 V3.0 新增 | P1 | 树莓派、ARM 服务器 |
| ARMv7 | 🎯 V3.0 新增 | P2 | 老款 ARM 设备 |

**实施策略**：
- V3.0 开发阶段，**x86_64 是主要测试平台**，所有功能优先在 x86_64 上验证
- ARM64 和 ARMv7 在 V3.0 后期做兼容适配，不阻塞主功能开发
- 如果资源有限，V3.0 可以只发 x86_64 版本，ARM 支持延至 V3.1

### 1.2 操作系统支持

| 操作系统 | 状态 | 优先级 | 说明 |
|---------|------|--------|------|
| Ubuntu | ✅ 已支持 | P0 | 主要开发和测试平台 |
| Debian | ✅ 已支持 | P0 | 与 Ubuntu 高度兼容 |
| CentOS | ✅ 已支持 | P0 | 企业常用 |
| Rocky Linux | ✅ 已支持 | P0 | CentOS 替代品 |
| AlmaLinux | ✅ 已支持 | P1 | Rocky 的同源替代品 |
| Fedora | 🎯 V3.0 新增 | P1 | 较新内核，部分兼容 |
| Arch Linux | 🎯 V3.0 新增 | P2 | 滚动更新，社区需求 |

**已知难点**：
- Fedora 的包管理器是 `dnf`（不是 `yum`）
- Arch Linux 的包管理器是 `pacman`（语法完全不同）
- 不同系统的 Docker 安装方式略有差异

**解决方案**：抽象出 `os_adapter.sh`，统一操作系统接口：
```bash
# os_adapter.sh — 操作系统适配层

get_pkg_manager() {
    case "$OS_FAMILY" in
        debian) echo "apt-get" ;;
        rhel)   echo "yum" ;;
        fedora) echo "dnf" ;;
        arch)   echo "pacman" ;;
    esac
}

install_docker() {
    local pkg_manager=$(get_pkg_manager)
    case "$OS_FAMILY" in
        debian) apt-get install -y docker.io ;;
        rhel|fedora) $pkg_manager install -y docker ;;
        arch)   pacman -S --noconfirm docker ;;
    esac
}
```

---

## 核心特性二：生态集成（多网关支持）

### 2.1 支持的主流 AI 网关

V3.0 重点支持 **3 个主流 AI 网关**（不贪多，做深做透）：

| 网关 | 优先级 | 说明 | V3.0 目标 |
|------|--------|------|-----------|
| **One API** | P0 | 最老牌的开源 AI 网关，用户基数大 | ✅ 完整支持 |
| **new-api** | P0 | NewAPI 的前身/分支，仍有用户在使用 | ✅ 完整支持 |
| **Sub2API** | P1 | 订阅管理增强版，特色功能 | ✅ 完整支持 |

> **设计原则**：不追求支持所有网关，而是把这三个做深做透（安装、备份、还原、升级、监控全套）。
> 更多网关通过**插件系统**（见 2.4）由社区贡献。

### 2.2 插件式架构

每个网关是一个**插件**，放在 `modules/plugins/` 目录下：

```
modules/
  plugins/
    one-api/           # One API 插件
      install.sh       # 安装逻辑
      backup.sh        # 备份逻辑
      restore.sh       # 还原逻辑
      upgrade.sh       # 升级逻辑
      config.sh        # 配置管理
      doctor.sh        # 诊断逻辑
      metadata.yml     # 插件元数据（名称、版本、依赖）
    new-api/           # new-api 插件
      ...
    sub2api/          # Sub2API 插件
      ...
  core/               # 核心功能（所有插件共用）
    docker.sh
    backup.sh
    restore.sh
    doctor.sh
```

**插件元数据示例**（`modules/plugins/one-api/metadata.yml`）：
```yaml
name: "One API"
flavor: "one-api"
version: "0.1.0"
min_newapi_tools_version: "3.0.0"
docker_image: "justsong/one-api"
default_port: 3000
dependencies:
  - docker
  - mysql  # 或 sqlite
```

**插件加载逻辑**（`lib/plugin.sh`）：
```bash
load_plugin() {
    local flavor="$1"
    local plugin_dir="$MODULES_DIR/plugins/$flavor"
    
    if [[ ! -d "$plugin_dir" ]]; then
        log_error "未找到插件：$flavor"
        return 1
    fi
    
    # 加载插件元数据
    parse_yaml "$plugin_dir/metadata.yml"
    
    # 加载插件脚本
    for script in "$plugin_dir"/*.sh; do
        source "$script"
    done
    
    log_info "已加载插件：$flavor"
}
```

### 2.3 网关间迁移工具

**使用场景**：用户想从 One API 迁移到 NewAPI（功能更强大），或者从 new-api 迁移到 NewAPI。

**命令设计**：
```bash
# 从 One API 迁移到 NewAPI
newapi-tools migrate --from one-api --to newapi

# 从 new-api 迁移到 NewAPI
newapi-tools migrate --from new-api --to newapi

# 从 Sub2API 迁移到 NewAPI
newapi-tools migrate --from sub2api --to newapi
```

**迁移流程**（以 One API → NewAPI 为例）：
1. 读取 One API 的配置文件（`data/one-api.db` 或 MySQL）
2. 转换数据格式（令牌、渠道、用户）
3. 写入 NewAPI 的数据库
4. 迁移 Docker 卷数据（如果有）
5. 生成迁移报告

**核心代码**（`modules/plugins/migrate.sh`）：
```bash
migrate() {
    local from_flavor="$1"
    local to_flavor="$2"
    
    log_info "开始迁移：$from_flavor → $to_flavor"
    
    # 1. 验证源网关已安装
    if ! is_installed "$from_flavor"; then
        log_error "源网关未安装：$from_flavor"
        return 1
    fi
    
    # 2. 导出源数据
    export_data "$from_flavor" "$TEMP_DIR/migration/"
    
    # 3. 转换数据格式
    convert_data "$from_flavor" "$to_flavor" "$TEMP_DIR/migration/" "$TEMP_DIR/converted/"
    
    # 4. 导入到目标网关
    import_data "$to_flavor" "$TEMP_DIR/converted/"
    
    # 5. 生成迁移报告
    generate_migration_report "$TEMP_DIR/migration-report.txt"
    
    log_info "迁移完成！"
}
```

### 2.4 插件系统（社区扩展）

**目标**：让社区可以贡献新的网关插件，不需要修改核心代码。

**插件开发指南**（`docs/PLUGIN-GUIDE.md`）：
1. 在 `modules/plugins/` 下创建新目录（如 `my-gateway/`）
2. 编写 `metadata.yml` 和对应的 `.sh` 脚本
3. 提交 Pull Request
4. 核心团队 Review 后合并

**插件市场**（未来方向，V3.0 不实现）：
- 用户可以浏览和安装社区插件
- `newapi-tools plugin install <plugin-name>`

---

## 实施路线图

```
2026-Q3（7-9月）         2026-Q4（10-12月）
─────────────────      ────────────────────
V3.0 开发               V3.0 测试 & 发布
│                       │
├─ 7月：多平台架构      ├─ 10月：Alpha 测试
│  ├─ x86_64 全功能     │  ├─ 内部测试
│  ├─ ARM64 适配        │  ├─ 社区 Beta 测试
│  └─ 操作系统适配       │  └─ Bug 修复
│     (Ubuntu/Debian/   └─ 11月：RC 发布
│      CentOS/Rocky/    ｜
│      Fedora/Arch)      └─ 12月：V3.0 正式发布
│                       ｜
├─ 8月：插件式架构      V3.1（次年 Q1）
│  ├─ 核心插件框架      ├─ ARM64 正式支持
│  ├─ One API 插件      ├─ ARMv7 支持
│  ├─ new-api 插件      └─ 更多社区插件
│  └─ Sub2API 插件
│
└─ 9月：迁移工具
   ├─ migrate 命令
   ├─ 数据转换脚本
   └─ 迁移报告
```

**里程碑**：
- 2026-07-31：V3.0 Alpha（多平台架构 + 插件框架完成）
- 2026-09-30：V3.0 Beta（三个网关插件 + 迁移工具完成）
- 2026-11-30：V3.0 RC（测试通过，文档齐全）
- 2026-12-15：V3.0 正式发布

---

## 资源需求

| 任务 | 所需资源 | 优先级 |
|------|---------|--------|
| 多平台架构（x86_64 + ARM64 + ARMv7） | 开发工程师 1 人 | 🔴 高 |
| 多操作系统适配（7 个系统） | 开发工程师 0.5 人 | 🔴 高 |
| 插件式架构 | 开发工程师 1 人 | 🔴 高 |
| 三个网关插件（One API、new-api、Sub2API） | 开发工程师 1 人 | 🔴 高 |
| 迁移工具 | 开发工程师 0.5 人 | 🟡 中 |
| 插件系统（社区扩展） | 开发工程师 0.5 人 | 🟡 中 |
| 测试（多平台 + 多网关） | 测试工程师 1 人 | 🔴 高 |

**总计**：~5 人月（V3.0 开发周期）

---

## 风险评估

| 风险 | 影响 | 应对措施 |
|------|------|---------|
| ARM64 兼容性问题 | V3.0 无法发 ARM 版本 | 提前在 ARM 设备上测试，必要时延至 V3.1 |
| 插件式架构设计不合理 | 后续维护困难 | 参考现有成熟插件系统（如 Vim、Oh My Zsh） |
| 迁移工具数据丢失 | 用户数据丢失 | 迁移前自动备份，迁移后可回滚 |
| 三个网关的接口变化 | 插件需要频繁更新 | 每个插件维护版本矩阵，适配主流版本 |
| 开发资源不足 | 路线图延期 | 优先 x86_64 + One API，其他可延期 |

---

## 成功标准

- V3.0 发布时，支持 **3 个主流网关**（One API、new-api、Sub2API）
- 支持 **x86_64** 架构（主要），**ARM64** 和 **ARMv7** 可选安装
- 支持 **5 个操作系统**（Ubuntu、Debian、CentOS、Rocky、Fedora/Arch 至少支持一个）
- 插件式架构成型，**社区可以贡献新网关插件**
- 迁移工具可用（`migrate` 命令支持三个网关间互迁）
- 用户满意度 ≥ 4.5/5.0

---

## 附录：V2.x vs V3.0 对比

| 特性 | V2.x | V3.0 |
|------|------|------|
| 支持的网关 | 1 个（NewAPI） | 3 个（One API、new-api、Sub2API） |
| CPU 架构 | x86_64 | x86_64 + ARM64 + ARMv7 |
| 操作系统 | 5 个 | 7 个 |
| 架构设计 | 单一脚本 | 插件式架构 |
| 迁移工具 | 无 | 有（`migrate` 命令） |
| 社区扩展 | 无 | 插件系统 |

---

*本文档随产品迭代持续更新。*
