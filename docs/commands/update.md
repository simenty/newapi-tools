# update

更新 new-api 到最新版本。

## 用法

```bash
newapi-tools update [flags]
```

## 说明

update 命令执行以下步骤：

1. 自动检测并应用最快镜像源（可用 `--no-auto-mirror` 跳过）
2. 创建更新前备份（可用 `--backup=false` 跳过）
3. 拉取新镜像
4. 使用 `docker compose up -d --force-recreate` 重建容器

## 标志

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--image` | | 指定目标镜像版本（默认拉取 latest） |
| `--backup` | true | 更新前自动备份 |
| `--force` | false | 强制更新（跳过备份） |
| `--mirror` | | 指定镜像源加速拉取 |
| `--no-auto-mirror` | false | 跳过自动检测最快镜像源 |

## 示例

```bash
# 更新到最新版本
newapi-tools update

# 更新到指定版本
newapi-tools update --image calciumion/new-api:v0.3.5

# 使用指定镜像源更新
newapi-tools update --mirror tuna

# 跳过备份强制更新
newapi-tools update --force
```

## 回滚

如果更新后出现问题，可以从备份恢复：

```bash
newapi-tools restore --file latest
```
