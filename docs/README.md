# NewAPI 自动化运维工具集 (newapi-tools)

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![ShellCheck](https://img.shields.io/badge/ShellCheck-passing-green.svg)](https://github.com/koalaman/shellcheck)
[![Version](https://img.shields.io/badge/version-2.0.1-blueviolet.svg)](CHANGELOG-v2.0.md)

你是不是受够了手动敲一堆命令部署 NewAPI？Docker 不会配？SSL 证书申请麻烦？备份不知道怎么搞？

这个工具就是来帮你偷懒的。

**v2.0 重大更新**：
- ✅ **断点续装**：安装过程可中断恢复，不用重新来
- ✅ **智能默认值**：自动生成安全密码，无需手动输入
- ✅ **新手/专家模式**：新手看详细提示，专家跳过啰嗦步骤
- ✅ **统一配置管理**：YAML 格式，支持分层配置
- ✅ **友好界面**：进度条、状态面板、彩色输出

它会帮你把环境配置、Docker 安装、NewAPI 部署、SSL 证书、备份恢复这些烦人的事儿全自动化了。你只需要对着菜单按数字，剩下的它帮你搞定。

---

## 这玩意儿能干啥？

### 🆕 v2.0 新增功能

#### 1. 断点续装（不用怕中断了）
- 安装过程自动保存进度到 `state.json`
- 中途断开？重新运行自动从断点继续
- 查看安装状态：`newapi-tools status`

#### 2. 智能默认值（不用手动输密码了）
- 自动生成 16 位安全密码（MySQL、Redis、Session）
- 自动检测系统信息（内存、CPU、磁盘）
- 根据服务器配置推荐最优参数
- 检测端口冲突，提前预警

#### 3. 新手/专家模式（按需显示提示）
- **新手模式**：详细提示、进度条、操作说明，一步一步教你
- **专家模式**：跳过非必要提示，显示所有高级选项
- 随时切换：菜单按 `m` 切换模式

#### 4. 统一配置管理（不用到处找配置了）
- YAML 格式配置文件（`config/config.yaml`）
- 分层配置：用户配置覆盖默认配置
- 支持环境变量优先级最高
- 配置验证：启动时检查必要配置

### 1. 帮你把环境配好（不用自己查文档了）
- 自动配置 DNS（国内用阿里的 223.5.5.5，速度快）
- 自动换国内源（清华源或者阿里源），下载软件快得多
- 帮你把 Docker 装上，还配好国内镜像加速（拉镜像不用等半天）

### 2. 一键把 NewAPI 跑起来
- 一条命令把 **NewAPI + MySQL 数据库 + Redis 缓存 + Nginx 管理面板** 全跑起来
- 容器有健康检查，挂了会自动重启（不用半夜爬起来重启）

### 3. 安全方面帮你考虑好了
- **不直接暴露到公网**：NewAPI 只监听本地端口，外面访问不到（安全）
- **密码藏好了**：所有密码都存在 `.env` 文件里，权限设成只有 root 能读
- **输密码时看不到**：别人站在你旁边也看不到密码
- **强制用 HTTPS**：自动帮你申请 Let's Encrypt 免费证书，网站访问都是加密的

### 4. 备份和恢复不用担心
- **备份时自动算校验值**：生成一个 `.md5` 文件，恢复时先检查备份是不是坏的
- **更新失败会自动回滚**：更新 NewAPI 后如果启动失败，自动回到旧版本（不会搞崩）
- **可以定时自动备份**：配好 Crontab 后每天自动备份，旧的自动删除
- **出问题会通知你**：可以配飞书或钉钉 Webhook，容器挂了会发消息提醒

### 5. 所有操作都有记录
- 执行的每个操作都记日志（`logs/toolkit.log`），出问题可以查
- 删除、重置这种危险操作单独记录（`logs/audit.log`）
- 危险操作必须手打 `yes` 确认（防止手滑 rm -rf 删库跑路）

---

## 菜单里都有啥？

运行 `newapi-tools` 后会出现一个菜单，长这样：

| 按这个数字 | 干啥用的 | 啥时候用 |
|-----------|---------|----------|
| 1 | 环境准备 | **第一次用必选**：配 DNS、换国内源、装 Docker |
| 2 | 安装部署 | 装 NewAPI 全家桶（数据库、Redis、Nginx 管理面板） |
| 3 | SSL 配置 | 申请 HTTPS 证书（让网站变成 https://） |
| 4 | 更新 | 更新 NewAPI 到最新版本（会自动备份+回滚） |
| 5 | 手动备份 | 立马备份一次数据库和数据（放心折腾前先备一份） |
| 6 | 备份恢复 | 选一个备份文件恢复（搞崩了可以救回来） |
| 7 | 健康检查 | 看看服务器资源够不够、容器是不是正常运行 |
| 8 | 重装/卸载 | 重装（会先备份）或者彻底删除 |
| 9 | 查看日志 | 实时看 NewAPI 的日志（排错用） |
| 0 | 自更新 | 更新这个工具本身（有新功能时跑一下） |
| m | 切换模式 | 在新手模式和专家模式之间切换 |
| s | 显示状态 | 查看系统状态、安装进度（v2.0 新增） |

**v2.0 新增功能**：
- **新手/专家模式**：按 `m` 切换，新手看到详细提示，专家跳过啰嗦步骤
- **状态显示**：按 `s` 查看系统状态、安装进度、断点续装信息
- **智能默认值**：安装时自动生成密码，无需手动输入（v2.0 优化）

**建议顺序**：第一次用就按 1 → 2 → 3 这个顺序来，基本就搞定了。

---

## 手把手教你用

### 第一步：确认你的服务器符合要求

你的服务器需要满足这几个条件：
- **系统**：Ubuntu 22.04 / 24.04 或者 Debian 11 / 12（其他系统我没测过，不保证能用）
- **权限**：必须用 root 用户登录（不要用 sudo，直接 root）
- **网络**：服务器能访问公网（要拉 Docker 镜像和下载脚本）

**怎么确认是不是 root？**
```bash
whoami   # 如果输出是 root，就没问题
```

如果不是 root，先切换：
```bash
sudo -i   # 输入密码后变成 root
```

### 第二步：安装这个工具

复制这段代码，粘贴到服务器终端里回车：
```bash
curl -fsSL https://raw.githubusercontent.com/simenty/newapi-tools/main/install.sh | bash
```

**这一步会干啥？**
- 把工具下载到 `/opt/newapi-tools` 目录
- 创建一个全局命令 `newapi-tools`（随便在哪个目录都能用）

等它跑完，看到成功提示就 OK 了。

### 第三步：开始用

**如果你是完全小白，用交互式菜单（推荐）**：
```bash
newapi-tools
```
然后会出现一个菜单，按数字选择你要的功能就行。

**如果你喜欢敲命令，也可以直接用命令行模式**：
```bash
newapi-tools backup --manual   # 手动备份一次
newapi-tools update           # 更新 NewAPI（失败会自动回滚）
newapi-tools health          # 检查健康状态（可以配监控）
newapi-tools ssl             # 配置 SSL 证书
newapi-tools logs -f        # 实时查看日志（排错用）
```

---

## 第一次部署完整流程（小白必看）

假设你有一台全新的服务器，想部署 NewAPI，按这个步骤来：

### 1. 先准备环境（菜单选 1）
```bash
newapi-tools
# 按 1，然后按照提示操作
```
这会帮你：
- 配置 DNS（选国内阿里 DNS 就行）
- 把软件源换成国内的（下载快）
- 安装 Docker 和 Docker Compose

**要多久？** 大概 5-10 分钟，看服务器网速。

### 2. 部署 NewAPI（菜单选 2）
```bash
# 还是在 newapi-tools 菜单里，按 2
```
**v2.0 改进**：
- ✅ **自动生成密码**：MySQL、Redis、Session Secret 自动生成，无需手动输入
- ✅ **进度显示**：5 步进度条，清楚看到安装到哪一步了
- ✅ **断点续装**：中断后重新运行，自动从断点继续

它会自动帮你：
- 创建 `docker-compose.yml`
- 生成 `config.yaml` 配置文件（密码自动生成并保存）
- 拉取 Docker 镜像并启动
- 保存安装进度到 `state.json`

**要多久？** 第一次拉镜像可能要 10-20 分钟。

**v2.0 提示**：新手模式会显示详细提示和进度条，专家模式会跳过非必要提示。

### 3. 配置 SSL 证书（菜单选 3）
```bash
# 菜单里按 3
```
会自动帮你申请 Let's Encrypt 免费证书（有效期 90 天，会自动续期）。

**注意**：这一步要求你的域名已经解析到服务器 IP，而且 80 和 443 端口没有被占用。

### 4. 验证是不是成功了
打开浏览器，访问 `https://你的域名`，能看到 NewAPI 的登录页面就成功了。

---

## 常见问题（FAQ）

### Q: 我域名还没解析，能先部署吗？
**A**: 可以先部署，但是访问只能用 IP 地址。建议先把域名解析配好再部署，这样 SSL 证书也能自动申请。

### Q: 万一部署搞崩了怎么办？
**A**: 菜单里选 6（备份恢复），可以恢复到之前的备份。如果连备份都没有，选 8（重装/卸载）重装一次。

### Q: 怎么备份？
**A**: 菜单里选 5（手动备份）可以立即备份。如果想定时自动备份，可以参考 `config/config.yaml` 里的配置。

### Q: 更新 NewAPI 会丢数据吗？
**A**: 不会。更新前会自动备份数据库和数据卷，如果更新失败会自动回滚。但是建议更新前还是手动备份一下（菜单选 5），双保险。

### Q: 我想彻底删除重来怎么办？
**A**: 菜单里选 8（重装/卸载），会先问你要不要备份，然后再卸载。选"彻底卸载"会删掉所有数据（包括数据库），小心操作。

---

## 安全相关（重要）

### 密码存在哪？
**v2.0**: 存在 `/opt/newapi-tools/config/config.yaml` 文件中，权限是 600（只有 root 能读）。

**v1.x 兼容**: 如果使用旧版本，密码存在 `/home/new-api/.env` 文件里。

**别手贱改权限**，不然密码可能被别的用户读到。

### 端口为啥不暴露公网？
NewAPI 默认只监听 `127.0.0.1:3000`，外面直接访问 IP:3000 是访问不到的。

外部访问必须通过 Nginx Proxy Manager（80/443 端口）反代进来，而且强制 HTTPS 加密。

**这样更安全**，防止有人直接扫描你的端口搞事情。

### 备份为啥要校验？
如果备份文件损坏了（比如磁盘坏了、下载中断），恢复时会出问题。

所以每次备份都会生成一个 `.md5` 校验文件，恢复时先检查，校验不通过就拒绝恢复（防止恢复出一堆坏数据）。

---

## 文件都装在哪？

### 工具自身在这：`/opt/newapi-tools`
```
/opt/newapi-tools/
├── newapi-tools.sh          # 主程序，所有的开始
├── install.sh              # 安装器（一般不用管）
├── lib/                   # 核心函数库（v2.0 重构）
│   ├── common.sh         # 日志、颜色、工具函数
│   ├── state.sh          # 状态管理（v2.0 新增，断点续装）
│   ├── config.sh         # 配置管理（v2.0 新增，YAML 格式）
│   ├── ui.sh             # UI 组件库（v2.0 新增，进度条、彩色输出）
│   ├── smart-defaults.sh # 智能默认值（v2.0 新增，自动生成密码）
│   └── mode.sh          # 模式管理（v2.0 新增，新手/专家模式）
├── config/                # 配置文件（v2.0 新增）
│   ├── config.yaml      # 用户配置（覆盖默认配置）
│   └── config.default.yaml  # 默认配置（不要直接改）
├── modules/               # 各个功能的代码
├── logs/                  # 日志文件
│   ├── toolkit.log      # 操作日志
│   └── audit.log        # 审计日志（危险操作）
├── state.json            # 状态文件（v2.0 新增，断点续装）
└── backups/              # 临时备份
```

### NewAPI 和数据在这：`/home/new-api`
```
/home/new-api/
├── .env                   # 环境变量（v1.x 兼容，v2.0 改用 config.yaml）
├── docker-compose.yml     # 容器配置
├── backups/              # 备份文件（含 .md5 校验）
├── data/                 # NewAPI 应用数据
├── mysql/                # MySQL 数据库文件
├── redis/                # Redis 数据
├── npm/                  # Nginx 管理面板配置和 SSL 证书
└── logs/                 # 运行日志
```

**v2.0 变化**：
- 配置文件从 `.env` 和 `toolkit.conf` 统一为 `config/config.yaml`
- 新增 `state.json` 跟踪安装进度和系统状态
- 日志目录统一到 `/opt/newapi-tools/logs/`

---

## 想改配置？看这 (v2.0)

### v2.0 配置系统（YAML 格式）

v2.0 使用 YAML 格式的配置文件，支持分层配置：

```
配置优先级：环境变量 > config.yaml > config.default.yaml
```

### 配置文件位置
- **用户配置**：`/opt/newapi-tools/config/config.yaml`（你的自定义配置）
- **默认配置**：`/opt/newapi-tools/config/config.default.yaml`（默认值和文档）

### 常用配置项（`config.yaml`）

```yaml
# 部署配置
deploy:
  platform: "newapi"
  work_dir: "/home/new-api"
  
  mysql:
    password: "你的密码"  # 如果不设，会自动生成
  
  redis:
    password: "你的密码"  # 如果不设，会自动生成
  
  newapi:
    port: 3000
    session_secret: "你的密钥"  # 如果不设，会自动生成

# 备份配置
backup:
  dir: "/opt/newapi-tools/backups"
  retention_days: 7
  compress: true

# 日志配置
log:
  dir: "/opt/newapi-tools/logs"
  level: "info"  # debug | info | warn | error
```

### 怎么改配置？

**方法 1：编辑配置文件**
```bash
nano /opt/newapi-tools/config/config.yaml
```
改完保存即可，下次运行自动生效。

**方法 2：用命令修改（推荐）**
```bash
# 查看配置
newapi-tools config get deploy.mysql.password

# 修改配置
newapi-tools config set deploy.port 8080
```

**方法 3：用环境变量（最高优先级）**
```bash
export DEPLOY_PORT=8080
newapi-tools install
```

### v1.x 迁移提示
如果你从 v1.x 升级：
- 旧的 `.env` 文件仍然兼容（会自动读取）
- 建议迁移到 `config.yaml` 统一管理
- 运行 `newapi-tools config migrate` 自动迁移

---

## 免责声明（必看）

这个工具里有**删除、重置、卸载**这种高危操作，用之前**务必先备份**。

虽然我已经尽量让工具安全（有二次确认、有备份、有回滚），但是：
- 如果你手滑输了个 `yes`，把生产环境卸了，我不背锅
- 如果服务器本身就有问题（磁盘坏了、内存不够），导致部署失败，我不背锅
- 如果你改了配置文件改坏了，导致出问题，我不背锅

**所以**：重要数据一定定期备份，别全靠自动备份。

---

## 遇到问题了？

### Bug 反馈
去这开 Issue：https://github.com/simenty/newapi-tools/issues

**怎么开 Issue？**
1. 点击上面的链接
2. 点 "New Issue"
3. 描述清楚：
   - 你的系统版本（Ubuntu 22.04 还是啥）
   - 你执行了啥操作（按了哪个菜单）
   - 报了啥错误（把错误信息复制上来）
   - 最好把日志也贴一下（`logs/toolkit.log`）

### 功能建议
也去开 Issue，描述清楚你想要啥功能、为啥需要。

### 直接改代码（如果你会的话）
1. Fork 这个仓库
2. 拉个分支：`git checkout -b feature/你想要的功能`
3. 改代码（记得跑 `shellcheck` 检查语法）
4. 推上去：`git push origin feature/你想要的功能`
5. 开 Pull Request

---

**最后**: 本工具纯AI手残结晶...
