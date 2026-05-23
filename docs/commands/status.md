# status

查看 new-api 及相关容器的运行状态。

## 用法

```bash
newapi-tools status [flags]
```

## 标志

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--json` | false | 以 JSON 格式输出（方便脚本处理） |

## 示例

```bash
# 表格输出
newapi-tools status

# JSON 输出
newapi-tools status --json
```

## 输出说明

表格输出示例：

```
CONTAINER   IMAGE                           STATUS     PORTS
new-api     calciumion/new-api:latest       running    0.0.0.0:3000->3000/tcp
mysql       mysql:8.0                       running
redis       redis:7                         running
```

JSON 输出示例：

```json
[
  {
    "name": "new-api",
    "image": "calciumion/new-api:latest",
    "state": "running",
    "status": "Up 2 hours",
    "ports": "0.0.0.0:3000->3000/tcp"
  }
]
```

## 注意事项

- 此命令读取 Docker 运行状态，不需要 root 权限（前提是当前用户在 docker 组）
- 如果容器不存在，会提示运行 `newapi-tools install`
