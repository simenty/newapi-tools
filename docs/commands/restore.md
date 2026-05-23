# restore

从备份文件恢复 new-api。

## 用法

```bash
newapi-tools restore [flags]
```

## 标志

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--file` | | 备份文件路径，或 `latest` 自动选择最新备份 |
| `--force` | false | 跳过确认提示直接恢复 |

## 示例

```bash
# 恢复指定备份文件
newapi-tools restore --file /var/backups/newapi-backup-20240115-143022.tar.gz

# 自动选择最新备份
newapi-tools restore --file latest

# 跳过确认直接恢复
newapi-tools restore --file latest --force
```

## 恢复流程

1. 解压备份归档到临时目录
2. 停止正在运行的容器
3. 恢复配置文件（`docker-compose.yml`、`.env`）
4. 恢复 MySQL 数据库（`mysql` 命令导入）
5. 恢复 `data/` 目录
6. 重启容器

!!! warning "注意"
    恢复操作会覆盖当前的数据库和数据目录。恢复前建议先备份当前数据：`newapi-tools backup`
