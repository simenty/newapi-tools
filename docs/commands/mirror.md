# mirror

管理 Docker Hub 镜像加速源。

## 用法

```bash
newapi-tools mirror <subcommand> [args] [flags]
```

## 子命令

| 子命令 | 说明 |
|--------|------|
| `add <镜像源>...` | 添加一个或多个镜像源 |
| `remove <镜像源>` | 删除一个镜像源 |
| `list` | 查看当前配置的镜像源 |
| `apply` | 将镜像源写入 daemon.json 并重载 Docker |
| `test [镜像源]` | 测试镜像源连通性 |
| `reset` | 清除所有镜像源 |
| `builtin` | 列出所有内置镜像源快捷名 |

## 内置镜像源

| 快捷名 | URL |
|--------|-----|
| `tuna` | `https://docker.mirrors.tuna.tsinghua.edu.cn` |
| `aliyun` | `https://registry.cn-hangzhou.aliyuncs.com` |
| `ustc` | `https://docker.mirrors.ustc.edu.cn` |
| `163` | `https://hub-mirror.c.163.com` |
| `azure` | `https://dockerhub.azk8s.cn` |
| `daocloud` | `https://f1361db2.m.daocloud.io` |

## 示例

```bash
# 添加清华源
newapi-tools mirror add tuna

# 添加多个源
newapi-tools mirror add tuna aliyun

# 查看当前源
newapi-tools mirror list

# 测试连通性
newapi-tools mirror test tuna

# 测试所有已配置的源
newapi-tools mirror test

# 删除一个源
newapi-tools mirror remove tuna

# 清除所有源
newapi-tools mirror reset

# 查看所有内置快捷名
newapi-tools mirror builtin
```

## 工作原理

mirror 命令读写 `/etc/docker/daemon.json` 中的 `registry-mirrors` 字段，并通过 `systemctl reload docker` 让配置生效，不需要重启 Docker（零停机）。

!!! warning "需要 root 权限"
    写入 `/etc/docker/daemon.json` 需要 root 或 sudo 权限。
