# NewAPI Tools v2.0 安全审计报告

**审计日期**: 2026-05-15  
**审计版本**: v2.0  
**审计范围**: 所有 Shell 脚本（install.sh, newapi-tools.sh, lib/*.sh, scripts/*.sh, modules/**/*.sh）  
**审计工具**: 手动代码审查  

---

## 执行摘要

本次审计对 NewAPI Tools v2.0 项目进行了全面的安全审查，共发现 **17 个安全问题**，其中：

| 严重程度 | 数量 |
|---------|------|
| 🔴 严重 (Critical) | 4 |
| 🟠 高危 (High) | 6 |
| 🟡 中危 (Medium) | 4 |
| 🔵 低危 (Low) | 3 |

**总体评价**: 项目实现了多项安全最佳实践（如密码通过环境变量传递、回滚机制、权限检查等），但仍有若干关键安全问题需要立即修复。

---

## 审计发现（按严重程度排序）

### 🔴 严重 (Critical)

#### 1. 密码通过命令行参数传递（可被 ps 等工具查看）

**文件**: `scripts/encrypt-config.sh`, `modules/deploy/ssl-proxy.sh`  
**行号**: encrypt-config.sh:49, 79; ssl-proxy.sh:130-134

**问题描述**:
```bash
# encrypt-config.sh 第 49 行
openssl enc -aes-256-cbc -salt -in "$ENV_FILE" -out "$ENCRYPTED_FILE" -pass pass:"$PASS"
```

密码通过 `-pass pass:"$PASS"` 传递，其他用户可以通过 `ps aux` 查看命令行参数。

**修复建议**: 使用文件描述符或环境变量传递密码：
```bash
# 修复方案 1: 使用环境变量
MYSQL_PWD="$PASS" openssl enc -aes-256-cbc -salt -in "$ENV_FILE" -out "$ENCRYPTED_FILE" -pass env:MYSQL_PWD

# 修复方案 2: 使用文件描述符
echo "$PASS" | openssl enc -aes-256-cbc -salt -in "$ENV_FILE" -out "$ENCRYPTED_FILE" -pass stdin
```

---

#### 2. 直接执行从网络下载的脚本

**文件**: `modules/init/docker.sh`  
**行号**: 73, 234

**问题描述**:
```bash
curl -fsSL "${install_url}" | bash -s docker
```

直接执行从网络下载的脚本，如果 download.docker.com 被攻击或 DNS 劫持，会执行恶意代码。

**修复建议**:
1. 先下载脚本，验证 checksum，再执行
2. 或使用系统包管理器安装

---

#### 3. 显示默认密码（安全风险）

**文件**: `modules/deploy/install.sh`  
**行号**: 283-284

**问题描述**:
```bash
echo "  邮箱: admin@example.com"
echo "  密码: changeme"
```

在日志中显示默认密码，虽然是为了提示用户修改，但可能被日志记录。

**修复建议**: 在首次登录时强制修改密码，而不是显示默认密码。

---

#### 4. 下载文件未验证完整性

**文件**: `modules/init/repo-source.sh`  
**行号**: 178-179, 187-188, 193-194, 247-248, 267-268

**问题描述**:
使用 `curl -o` 直接下载 .repo 文件，但没有验证 GPG 签名或 checksum。

**修复建议**: 下载后验证 GPG 签名或 checksum。

---

### 🟠 高危 (High)

#### 5. 使用 MD5 进行校验（已被认为不安全）

**文件**: `lib/common.sh`, `modules/manage/backup.sh`  
**行号**: common.sh:99, 114; backup.sh:137

**问题描述**:
```bash
md5sum "$file" > "${file}.md5"
md5sum -c "$md5file" --status
```

MD5 已被证明不安全，容易被碰撞攻击。

**修复建议**: 使用 SHA-256:
```bash
sha256sum "$file" > "${file}.sha256"
sha256sum -c "${file}.sha256"
```

---

#### 6. 使用 sed 替换配置文件（存在注入风险）

**文件**: `lib/config.sh`, `lib/state.sh`  
**行号**: config.sh:249-254; state.sh:124-135

**问题描述**:
```bash
sed -i "s|^$key:.*|$key: $value|" "$MAIN_CONFIG"
```

如果 `$key` 或 `$value` 包含 sed 特殊字符（如 `/`, `&` 等），会导致命令注入或替换失败。

**修复建议**: 使用 `yq` 或 Python 进行安全的 YAML/JSON 操作。

---

#### 7. source 加载配置文件（可能执行任意代码）

**文件**: `lib/env.sh`  
**行号**: 38, 46

**问题描述**:
```bash
source "$CONFIG_FILE"
source "${PROFILES_DIR}/${env_profile}.conf"
```

如果配置文件被篡改，会执行任意代码。

**修复建议**:
1. 验证配置文件的完整性（GPG 签名）
2. 或使用安全的配置文件格式（如严格的 YAML 解析）

---

#### 8. NPM API 密码在命令行中可见

**文件**: `modules/deploy/ssl-proxy.sh`, `lib/npm-api.sh`  
**行号**: ssl-proxy.sh:83; npm-api.sh:11-14

**问题描述**:
```bash
TOKEN=$(npm_login "$NPM_EMAIL" "$NPM_PASS")
# npm-api.sh 中：
curl -s -X POST -H "Content-Type: application/json" \
    -d "{\"identity\":\"$email\",\"scope\":\"user\",\"secret\":\"$password\"}" \
    "$NPM_URL/api/tokens"
```

密码在命令行中通过 `-d` 参数传递，可被 `ps` 查看。

**修复建议**: 使用文件或环境变量传递密码和 token。

---

#### 9. Docker daemon.json 可能被注入

**文件**: `modules/init/docker.sh`  
**行号**: 303-309

**问题描述**:
```bash
cat > /etc/docker/daemon.json << EOF
{
  "registry-mirrors": ["$DOCKER_HUB_MIRROR"],
  ...
}
EOF
```

如果 `$DOCKER_HUB_MIRROR` 包含特殊字符（如 `"`, `\` 等），会导致 JSON 注入。

**修复建议**: 使用 `jq` 安全生成 JSON:
```bash
echo '{}' | jq --arg mirror "$DOCKER_HUB_MIRROR" '. + { "registry-mirrors": [$mirror] }' > /etc/docker/daemon.json
```

---

#### 10. 回滚时数据库密码可能被泄露

**文件**: `modules/manage/restore.sh`  
**行号**: 137-138, 344

**问题描述**:
```bash
docker exec -i mysql bash -c 'MYSQL_PWD="$1" mysql -uroot newapi' _ "$DB_ROOT_PASSWORD" \
    < "$ROLLBACK_DIR/pre_restore_db.sql" 2>>"$LOG_FILE"
```

虽然使用了环境变量，但 `$DB_ROOT_PASSWORD` 仍然在命令行参数中可见（`_ "$DB_ROOT_PASSWORD"`）。

**修复建议**: 使用文件传递密码，或使用 `docker exec -e MYSQL_PWD=...`。

---

### 🟡 中危 (Medium)

#### 11. 密码在内存中残留

**文件**: 多个文件  

**问题描述**: Bash 变量在内存中不会立即清除，可能导致密码泄露。

**修复建议**:
1. 使用 `unset` 立即清除密码变量
2. 考虑使用临时文件（权限 600）传递密码，用后立即删除

---

#### 12. 日志函数可能记录敏感信息

**文件**: `lib/common.sh`  
**行号**: 20-29

**问题描述**:
`_desensitize_log()` 使用 `sed` 进行脱敏，但可能无法处理所有敏感信息模式。

**修复建议**: 增强脱敏函数，覆盖更多敏感信息模式。

---

#### 13. 使用 eval 执行测试命令

**文件**: `test/unit-test.sh`  
**行号**: 28

**问题描述**:
```bash
if output=$(eval "$test_cmd" 2>&1); then
```

如果 `$test_cmd` 包含恶意代码，会被执行。

**修复建议**: 避免使用 `eval`，或使用严格的输入验证。

---

#### 14. JSON payload 可能注入

**文件**: `modules/deploy/ssl-proxy.sh`  
**行号**: 110-128

**问题描述**:
```bash
PAYLOAD=$(cat <<EOF
{
  "domain_names": ["$DOMAIN"],
  ...
}
EOF
)
```

如果 `$DOMAIN` 包含特殊字符（如 `"`, `\` 等），会导致 JSON 注入。

**修复建议**: 使用 `jq` 安全生成 JSON:
```bash
PAYLOAD=$(jq -n --arg domain "$DOMAIN" '{domain_names: [$domain]}')
```

---

### 🔵 低危 (Low)

#### 15. 代码一致性问题

**文件**: `lib/common.sh`, `lib/state.sh`  

**问题描述**: 部分函数使用 `jq`，部分使用 `sed` 或 `python3`，一致性较差。

**修复建议**: 统一使用 `jq` 或 `yq` 进行 JSON/YAML 操作。

---

#### 16. 缺少错误处理

**文件**: 多个文件  

**问题描述**: 部分命令没有检查返回值，或没有使用 `set -e`）。

**修复建议**: 为所有关键命令添加错误处理。

---

#### 17. 临时文件权限可能不安全

**文件**: 多个文件  

**问题描述**: 使用 `mktemp` 创建临时文件，但没有显式设置权限。

**修复建议**: 创建临时文件后立即设置权限（`chmod 600`）。

---

## 修复优先级建议

| 优先级 | 问题编号 | 修复工作量 |
|-------|---------|-----------|
| P0 (立即修复) | 1, 2, 3, 4 | 大 |
| P1 (本周内) | 5, 6, 7, 8, 9, 10 | 中 |
| P2 (本月内) | 11, 12, 13, 14 | 小 |
| P3 (持续优化) | 15, 16, 17 | 小 |

---

## 安全最佳实践建议

1. **密码管理**:
   - 永远不要在命令行参数中传递密码
   - 使用环境变量或文件（权限 600）传递密码
   - 使用 `unset` 立即清除密码变量

2. **输入验证**:
   - 对所有用户输入进行严格验证
   - 使用安全的 JSON/YAML 生成工具（`jq`, `yq`）

3. **文件完整性验证**:
   - 下载文件后验证 GPG 签名或 checksum
   - 使用 HTTPS 下载文件

4. **最小权限原则**:
   - 只授予必要的权限
   - 定期检查文件权限

5. **日志安全**:
   - 增强日志脱敏函数
   - 定期清理日志文件

---

## 总结

NewAPI Tools v2.0 项目在安全性方面做了一定的工作（如密码通过环境变量传递、回滚机制、权限检查等），但仍有若干关键安全问题需要修复，特别是：

1. **密码通过命令行参数传递**（问题 1, 8）
2. **直接执行网络下载的脚本**（问题 2）
3. **使用不安全的 MD5 校验**（问题 5）
4. **配置文件注入风险**（问题 6, 9, 14）

建议按照修复优先级逐步修复，并考虑引入自动化安全扫描工具（如 `shellcheck`）进行持续安全检查。

---

**报告生成时间**: 2026-05-15 01:48:10 GMT+8  
**审计人员**: Senior Developer (高级开发工程师)
