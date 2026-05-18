# NewAPI Tools — 30 秒极速上手

> 目标：让任何人都能在 30 秒内完成 NewAPI 部署

---

## 🚀 极速安装（15 秒）

```bash
# 一行命令完成安装
curl -fsSL https://newapi-tools.ai/install.sh | bash
```

**15 秒内完成**：
- 自动检测系统（Ubuntu/Debian）
- 自动安装 Docker
- 自动配置镜像加速
- 自动部署 NewAPI

---

## 🔧 配置域名（10 秒）

```bash
# 运行配置向导
newapi-tools config
```

**按提示输入**：
- 域名：`api.example.com`
- 邮箱：`admin@example.com`
- 密码：`（自动生成，无需手动输入）`

---

## 🔒 SSL 证书（5 秒）

```bash
# 自动申请 Let's Encrypt 证书
newapi-tools ssl
```

**完成后访问**：`https://你的域名`

---

## 📊 状态检查

```bash
# 查看系统状态
newapi-tools status
```

---

## 📖 完整文档

- **新手教程**：`docs/beginner/quick-start.md`
- **视频教程**：B 站搜索「NewAPI Tools 零基础」
- **故障排查**：`newapi-tools doctor`

---

## 💡 提示

- 全程无需编辑配置文件
- 所有步骤都有默认值（直接回车即可）
- 出问题运行 `newapi-tools doctor` 自动诊断

---

*30 秒，从零到上线。*
