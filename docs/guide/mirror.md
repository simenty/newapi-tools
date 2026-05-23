# 镜像加速

在中国大陆，直接拉取 Docker Hub 镜像往往很慢甚至超时。newapi-tools 提供了完整的镜像加速方案。

## 自动检测（推荐）

`install` 和 `update` 命令在没有配置镜像源时，会**自动并发测试**所有内置镜像源并应用延迟最低的一个：

```bash
newapi-tools install
# 输出示例：
# No registry mirror configured. Auto-detecting fastest mirror...
#   Fastest mirror: tuna (https://docker.mirrors.tuna.tsinghua.edu.cn), latency 234ms
#   Applying mirror tuna...
#   Mirror applied. Image pull will use the mirror.
```

如果想跳过自动检测，加 `--no-auto-mirror`：

```bash
newapi-tools install --no-auto-mirror
```

## 手动配置镜像源

```bash
# 添加清华源（推荐）
newapi-tools mirror add tuna

# 添加多个源（自动按顺序尝试）
newapi-tools mirror add tuna aliyun

# 查看已配置的源
newapi-tools mirror list

# 测试连通性
newapi-tools mirror test
```

## 内置镜像源

| 快捷名 | 机构 | URL |
|--------|------|-----|
| `tuna` | 清华大学 | https://docker.mirrors.tuna.tsinghua.edu.cn |
| `aliyun` | 阿里云 | https://registry.cn-hangzhou.aliyuncs.com |
| `ustc` | 中科大 | https://docker.mirrors.ustc.edu.cn |
| `163` | 网易 | https://hub-mirror.c.163.com |
| `azure` | Azure CN | https://dockerhub.azk8s.cn |
| `daocloud` | DaoCloud | https://f1361db2.m.daocloud.io |

!!! tip "推荐"
    建议同时配置 2-3 个镜像源，Docker 会按顺序尝试，提高成功率。

## 一次性镜像源

如果只想在本次拉取时使用镜像源，可以通过 `--mirror` 标志临时指定：

```bash
newapi-tools install --mirror tuna
newapi-tools update --mirror aliyun
```

这会临时写入 daemon.json 并在拉取完成后保留（方便下次使用）。

## 清除镜像源

```bash
newapi-tools mirror reset
```

## 原理

镜像加速通过修改 `/etc/docker/daemon.json` 中的 `registry-mirrors` 字段实现。应用后通过 `systemctl reload docker` 热重载配置，**不需要重启 Docker 守护进程**（零停机）。

每次修改前，原 daemon.json 会自动备份为 `daemon.json.bak.YYYYMMDDHHMMSS`。
