# 常见问题

## 安装问题

### Docker 拉取镜像超时或很慢

在中国大陆直连 Docker Hub 很慢，运行以下命令配置镜像加速：

```bash
newapi-tools mirror add tuna
```

或者在 install 时加 `--mirror` 参数：

```bash
newapi-tools install --mirror tuna
```

详细说明见[镜像加速](mirror.md)。

---

### install 提示 "new-api is already running"

new-api 已在运行，如需重装：

```bash
newapi-tools install --force
```

---

### 端口被占用

修改端口再安装：

```bash
newapi-tools config set newapi.port 8080
newapi-tools install
```

---

## 运行问题

### status 显示容器未运行

先运行 doctor 诊断原因：

```bash
newapi-tools doctor
```

如果有可修复的问题，使用 `--fix` 自动修复：

```bash
newapi-tools doctor --fix
```

---

### MySQL 容器启动失败

常见原因：MySQL 数据目录权限问题或端口冲突。

检查容器日志：

```bash
docker logs mysql
```

---

## 备份与恢复

### backup 报错 "mysqldump: command not found"

`mysqldump` 在容器内执行，需要 MySQL 容器正在运行。先确保 MySQL 容器运行：

```bash
newapi-tools status
docker start mysql  # 如果 mysql 未运行
newapi-tools backup
```

---

### 如何自动定时备份

结合 cron 使用：

```bash
# 每天凌晨 3 点备份
0 3 * * * /usr/local/bin/newapi-tools backup >> /var/log/newapi-backup.log 2>&1
```

---

## 权限问题

### mirror 命令报 "permission denied"

写入 `/etc/docker/daemon.json` 需要 root 权限：

```bash
sudo newapi-tools mirror add tuna
```

---

### 无法连接 Docker

当前用户不在 `docker` 组：

```bash
sudo usermod -aG docker $USER
newgrp docker
```

---

## 版本问题

### 如何降级

先恢复备份，再指定旧版本镜像：

```bash
newapi-tools restore --file latest
newapi-tools update --image calciumion/new-api:v0.3.x
```

---

## 其他

### 配置文件在哪里

```
~/.config/newapi-tools/newapi-tools.yml
```

查看当前配置：

```bash
newapi-tools config
```

---

### 如何卸载

```bash
# 停止并删除容器
docker compose -f /opt/newapi/docker-compose.yml down

# 删除数据（谨慎操作）
rm -rf /opt/newapi

# 删除二进制
sudo rm /usr/local/bin/newapi-tools
```
