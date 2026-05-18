# NewAPI Tools V2.0 归档清单

**版本**: V2.0  
**归档日期**: 2026-05-15  
**归档路径**: `./release/v2.0/`

---

## 📦 归档内容

### 核心脚本
```
├── newapi-tools.sh          # 主入口脚本
├── install.sh               # 一键安装脚本
└── scripts/
    ├── encrypt-config.sh    # 配置加密工具
    ├── self-update.sh       # 自更新工具
    └── verify-deployment.sh # 部署验证工具
```

### 核心库
```
├── lib/
│   ├── common.sh            # 通用函数库
│   ├── config.sh            # 配置管理库
│   ├── state.sh             # 状态管理库
│   ├── ui.sh                # UI 组件库
│   ├── mode.sh              # 模式切换库
│   ├── smart-defaults.sh    # 智能默认值库
│   ├── security.sh          # 安全工具库 (新增)
│   └── npm-api.sh           # NPM API 封装库
```

### 功能模块
```
├── modules/
│   ├── deploy/
│   │   ├── install.sh       # 部署模块
│   │   └── ssl-proxy.sh     # SSL 配置模块
│   ├── init/
│   │   ├── docker.sh        # Docker 初始化
│   │   └── repo-source.sh   # 仓库源配置
│   ├── manage/
│   │   ├── backup.sh        # 备份管理
│   │   ├── restore.sh       # 恢复管理
│   │   ├── update.sh        # 更新管理
│   │   ├── uninstall.sh     # 卸载管理
│   │   └── reinstall.sh     # 重新安装
│   └── monitor/
│       └── health.sh         # 健康检查
```

### 配置文件
```
├── config/
│   └── config.yaml.example  # 配置示例
```

### 测试文件
```
├── test/
│   ├── unit-test.sh         # 单元测试
│   ├── integration-test.sh  # 集成测试
│   └── test-common.sh       # 测试公共函数
```

### 文档
```
├── docs/
│   ├── USER-GUIDE.md        # 用户指南
│   ├── SECURITY-CODING-STANDARDS.md  # 安全编码标准
│   ├── ARCHITECTURE.md      # 架构设计
│   ├── MODULE-REFERENCE.md  # 模块参考
│   ├── FAQ.md               # 常见问题
│   └── TROUBLESHOOTING.md   # 故障排除
```

### 版本文档
```
├── CHANGELOG-v1.2.md        # V1.2 变更记录
├── CHANGELOG-v2.0.md        # V2.0 变更记录
├── SECURITY-FIXES-COMPLETED.md  # 安全修复报告
├── V2.0-IMPLEMENTATION-REPORT.md # 实施报告
├── RELEASE-NOTES-v2.0.md    # 发布说明
└── README.md                # 项目说明
```

### 其他文件
```
├── .system_info             # 系统信息
├── .gitignore               # Git 忽略配置
└── LICENSE                  # 许可证
```

---

## ✅ 完整性校验

### MD5/SHA256 校验

```bash
# 在归档目录执行
sha256sum newapi-tools-v2.0.tar.gz > newapi-tools-v2.0.tar.gz.sha256
cat newapi-tools-v2.0.tar.gz.sha256
# 将此值与发布页面上的值对比
```

### 归档命令

```bash
# 创建归档
tar -czvf newapi-tools-v2.0.tar.gz \
    --exclude='*.log' \
    --exclude='*.tmp' \
    --exclude='.git' \
    --exclude='backups/*' \
    --exclude='.env' \
    --exclude='docker-compose.yml' \
    -C .. newapi-tools/

# 验证归档
tar -tzvf newapi-tools-v2.0.tar.gz | head -50
```

---

## 🚀 快速部署命令

### 方式一：直接使用归档

```bash
# 解压归档
tar -xzf newapi-tools-v2.0.tar.gz
cd newapi-tools

# 执行安装
bash newapi-tools.sh
```

### 方式二：使用安装脚本

```bash
# 一键安装
curl -fsSL https://raw.githubusercontent.com/example/newapi-tools/v2.0/install.sh -o install.sh
bash install.sh
```

### 方式三：Git 克隆

```bash
# 克隆仓库
git clone https://github.com/example/newapi-tools.git
cd newapi-tools
git checkout v2.0.0

# 执行安装
bash newapi-tools.sh
```

---

## 📋 部署前检查清单

- [ ] 满足系统要求（Ubuntu 18.04+ / Debian 10+ / CentOS 7+）
- [ ] Docker 和 Docker Compose 已安装
- [ ] 有 root 或 sudo 权限
- [ ] 端口 3000, 3306, 6379, 80, 81 可用
- [ ] 网络连接正常（可访问 Docker Hub）

---

## 📞 支持信息

- **文档**: https://docs.example.com/newapi-tools
- **Issue**: https://github.com/example/newapi-tools/issues
- **讨论**: https://github.com/example/newapi-tools/discussions

---

## 📅 版本历史

| 版本 | 日期 | 类型 | 状态 |
|------|------|------|------|
| V2.0 | 2026-05-15 | Major | ✅ 发布 |
| V1.2 | 2025-xx-xx | Minor | 🔄 维护 |
| V1.1 | 2025-xx-xx | Minor | 🔄 维护 |
| V1.0 | 2025-xx-xx | Initial | 🔄 维护 |

---

**归档完成时间**: 2026-05-15  
**归档人员**: Senior Developer  
**归档状态**: ✅ 完成
