#!/bin/bash
set -eo pipefail
# NPM API 交互函数

# common.sh 和 security.sh 已通过 lib/_init.sh 按依赖顺序加载

NPM_URL="http://127.0.0.1:81"

npm_login() {
    local email="$1"
    local password="${NPM_PASS:-$2}"
    local resp
    local tmp_file
    
    # 使用临时文件传递密码（避免命令行暴露）
    tmp_file=$(secure_temp_file "npm")
    printf '{"identity":"%s","scope":"user","secret":"%s"}' "$email" "$password" > "$tmp_file"
    
    resp=$(curl -s -X POST -H "Content-Type: application/json" \
        --data-binary @"$tmp_file" \
        "$NPM_URL/api/tokens")
    
    # 清理临时文件
    rm -f "$tmp_file"
    
    echo "$resp" | grep -o '"token":"[^"]*"' | cut -d'"' -f4
}
