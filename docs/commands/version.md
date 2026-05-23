# version

打印版本和构建信息。

## 用法

```bash
newapi-tools version
# 或
newapi-tools --version
```

## 输出示例

```
newapi-tools v3.0.0 (commit: 3b9a540, built: 2026-05-22T12:00:00Z)
```

| 字段 | 说明 |
|------|------|
| 版本号 | SemVer 格式，如 `v3.0.0` |
| commit | Git 短哈希 |
| built | 构建时间（UTC） |
