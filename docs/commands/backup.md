# backup

备份 new-api 的数据、配置文件和数据库。

## 用法

```bash
newapi-tools backup [flags]
```

## 说明

backup 命令打包以下内容到一个 `.tar.gz` 归档文件：

- `docker-compose.yml` 和 `.env` 配置文件
- MySQL 数据库（`mysqldump` 全量导出）
- `data/` 目录（new-api 的文件数据）

## 标志

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--output` | 安装目录/backups | 备份文件保存位置 |

## 示例

```bash
# 默认备份到安装目录下的 backups/ 子目录
newapi-tools backup

# 指定备份位置
newapi-tools backup --output /var/backups/newapi
```

## 备份文件命名

```
newapi-backup-20240115-143022.tar.gz
```

格式：`newapi-backup-YYYYMMDD-HHmmss.tar.gz`

## 注意事项

- mysqldump 需要容器内的 MySQL 正在运行
- 备份前建议先检查磁盘空间
- 备份包含数据库密码信息，注意妥善保存
