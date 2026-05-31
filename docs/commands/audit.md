# audit

管理 newapi-tools 的审计日志，查看命令执行历史。

## 用法

```bash
newapi-tools audit <subcommand> [flags]
```

## 子命令

### list

列出审计日志条目，支持可选过滤。

```bash
newapi-tools audit list [flags]
```

#### 标志

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--last` | 0 | 显示最近 N 条记录（0 = 全部） |
| `--cmd` | "" | 按命令名称过滤（子串匹配） |
| `--since` | "" | 只显示此时间之后的记录（格式：2006-01-02 或 2006-01-02T15:04:05） |
| `--json` | false | 以 JSON 格式输出（方便脚本处理） |

#### 示例

```bash
# 列出所有审计记录
newapi-tools audit list

# 只显示最近 10 条
newapi-tools audit list --last 10

# 按命令名过滤
newapi-tools audit list --cmd install

# 只显示某天之后的记录
newapi-tools audit list --since 2026-01-01

# JSON 输出
newapi-tools audit list --last 5 --json
```

## 输出说明

表格输出示例：

```
TIME                    CMD       USER     RESULT  DURATION
2026-06-01 10:30:15     install   root     ok      120ms
2026-06-01 10:28:03     status    root     ok      45ms
2026-06-01 10:25:00     doctor    root     error   30ms
```

JSON 输出示例：

```json
[
  {
    "timestamp": "2026-06-01T10:30:15+08:00",
    "command": "install",
    "user": "root",
    "result": "ok",
    "duration_ms": 120
  }
]
```

## 注意事项

- 审计日志文件默认存储在 `~/.newapi-tools/audit.log`，自动轮转
- 使用 `--cmd` 过滤时执行子串匹配，例如 `--cmd status` 会匹配所有包含 "status" 的命令
- `--since` 支持日期（2006-01-02）和日期时间（2006-01-02T15:04:05）两种格式