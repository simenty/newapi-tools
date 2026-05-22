# newapi-tools V3.0.0 发布完成

## TL;DR

newapi-tools V3.0.0 已正式发布到 GitHub，5 平台产物全部上传。

## 交付概览

| 指标 | 数值 |
|------|------|
| 版本 | v3.0.0 |
| 测试 | 83 PASS / 0 FAIL |
| CLI 命令 | 9 个 |
| 平台产物 | 5 个（linux amd64/arm64, darwin amd64/arm64, windows amd64） |
| GitHub | https://github.com/simenty/newapi-tools |

## 本轮完成的任务

| 任务 | 状态 |
|------|------|
| T3 更新 .gitignore | ✅ |
| T8 清理根目录临时文件 | ✅ |
| T1+T9 README 更新（含 mirror 命令） | ✅ |
| T2 添加 MIT LICENSE | ✅ |
| T4+T5 关联 simenty remote + push main/v2 | ✅ |
| T7 goreleaser v2 snapshot 验证 | ✅ |
| T6 打 v3.0.0 tag + GitHub Release | ✅ |

## 关键变更

- GitHub 账号从 Bonus520 改为 simenty
- .goreleaser.yml 升级到 v2 格式
- Fine-grained PAT 权限不足，改用 Classic PAT
- force push main 覆盖旧 V2 Shell 历史

## 用户下一步建议

1. 访问 https://github.com/simenty/newapi-tools/releases/tag/v3.0.0 下载对应平台产物
2. Linux 一键安装：`curl -sL https://github.com/simenty/newapi-tools/releases/download/v3.0.0/newapi-tools_3.0.0_linux_amd64.tar.gz | tar xz && sudo mv newapi /usr/local/bin/`
3. 运行 `newapi install` 部署 new-api
4. 如遇拉取慢，先 `newapi mirror add tuna` 切换国内镜像源
5. 定期 `newapi update` 保持更新
