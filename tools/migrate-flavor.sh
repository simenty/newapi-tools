#!/bin/bash
# tools/migrate-flavor.sh — 跨 FLAVOR 路径迁移工具
# 帮助用户从 V2.1 旧路径（manage/ 平铺）迁移到 V2.2 新路径（manage/newapi/）
#
# 用法：
#   bash tools/migrate-flavor.sh              # 扫描模式：检测旧路径引用
#   bash tools/migrate-flavor.sh --fix        # 自动修复模式

set -eo pipefail

# ---------- 获取项目根目录 ----------
SCRIPT_DIR="$(cd "$(dirname "$(readlink -f "$0")")" && pwd)"
TOOLKIT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
PLAIN='\033[0m'

# 旧路径 → 新路径 映射
declare -A PATH_MAP=(
    ["manage/backup.sh"]="manage/newapi/backup.sh"
    ["manage/restore.sh"]="manage/newapi/restore.sh"
    ["manage/update.sh"]="manage/newapi/update.sh"
    ["manage/reinstall.sh"]="manage/newapi/reinstall.sh"
    ["manage/uninstall.sh"]="manage/newapi/uninstall.sh"
    ["manage/config.sh"]="manage/newapi/config.sh"
    ["manage/doctor.sh"]="manage/newapi/doctor.sh"
)

# 统计
FOUND_COUNT=0
FIXED_COUNT=0

# ---------- 扫描旧路径脚本文件 ----------
scan_old_scripts() {
    echo -e "${YELLOW}=== 1. 检测旧路径脚本文件 ===${PLAIN}"
    local found=0
    for old_path in "${!PATH_MAP[@]}"; do
        local full_path="${TOOLKIT_ROOT}/modules/${old_path}"
        if [[ -f "$full_path" ]]; then
            echo -e "  ${RED}[存在]${PLAIN}  modules/${old_path}  →  应迁移到 modules/${PATH_MAP[$old_path]}"
            found=$((found + 1))
        fi
    done

    if [[ $found -eq 0 ]]; then
        echo -e "  ${GREEN}[OK]${PLAIN} 未发现旧路径脚本文件"
    fi
    echo ""
    return $found
}

# ---------- 扫描旧路径引用 ----------
scan_references() {
    echo -e "${YELLOW}=== 2. 扫描旧路径引用 ===${PLAIN}"
    local found=0

    for old_path in "${!PATH_MAP[@]}"; do
        # 在 .sh 文件中搜索旧路径引用（排除 tools/ 和 .bak 文件）
        while IFS= read -r line; do
            if [[ -n "$line" ]]; then
                local file=$(echo "$line" | cut -d: -f1)
                local lineno=$(echo "$line" | cut -d: -f2)
                local content=$(echo "$line" | cut -d: -f3-)
                echo -e "  ${RED}[引用]${PLAIN}  ${file}:${lineno}  ${content}"
                found=$((found + 1))
            fi
        done < <(grep -rn "$old_path" "${TOOLKIT_ROOT}" --include="*.sh" \
            --exclude-dir="tools" --exclude="*.bak" 2>/dev/null || true)
    done

    FOUND_COUNT=$found
    if [[ $found -eq 0 ]]; then
        echo -e "  ${GREEN}[OK]${PLAIN} 未发现旧路径引用"
    fi
    echo ""
    return $found
}

# ---------- 自动修复 ----------
fix_references() {
    echo -e "${YELLOW}=== 3. 自动修复旧路径引用 ===${PLAIN}"
    local fixed=0

    for old_path in "${!PATH_MAP[@]}"; do
        local new_path="${PATH_MAP[$old_path]}"

        # 查找包含旧路径的文件
        local files=()
        while IFS= read -r f; do
            [[ -n "$f" ]] && files+=("$f")
        done < <(grep -rl "$old_path" "${TOOLKIT_ROOT}" --include="*.sh" \
            --exclude-dir="tools" --exclude="*.bak" 2>/dev/null || true)

        for file in "${files[@]}"; do
            # 使用 sed 替换
            local old_escaped
            old_escaped=$(printf '%s\n' "$old_path" | sed 's/[[\.*^$()+?{|]/\\&/g')
            local new_escaped
            new_escaped=$(printf '%s\n' "$new_path" | sed 's/[&/\]/\\&/g')

            if sed -i "s|${old_escaped}|${new_escaped}|g" "$file"; then
                echo -e "  ${GREEN}[修复]${PLAIN}  ${file##*/}: ${old_path} → ${new_path}"
                fixed=$((fixed + 1))
            else
                echo -e "  ${RED}[失败]${PLAIN}  ${file##*/}: 修复失败"
            fi
        done
    done

    FIXED_COUNT=$fixed
    echo ""
}

# ---------- 摘要报告 ----------
summary() {
    echo -e "${YELLOW}=== 迁移摘要 ===${PLAIN}"
    echo "  发现引用: ${FOUND_COUNT}"
    echo "  已修复:   ${FIXED_COUNT}"

    if [[ $FIXED_COUNT -gt 0 ]]; then
        echo -e "  ${GREEN}建议运行 bash -n 语法检查确认修复无误${PLAIN}"
    elif [[ $FOUND_COUNT -eq 0 ]]; then
        echo -e "  ${GREEN}项目已为 V2.2 新路径结构，无需迁移${PLAIN}"
    fi
    echo ""
}

# ---------- 主流程 ----------
main() {
    echo "NewAPI Tools V2.2 FLAVOR 路径迁移工具"
    echo "项目根目录: ${TOOLKIT_ROOT}"
    echo ""

    # 1. 扫描旧脚本文件
    local old_scripts=0
    scan_old_scripts || old_scripts=$?

    # 2. 扫描旧路径引用
    scan_references

    # 3. 如果指定 --fix，执行自动修复
    if [[ "${1:-}" == "--fix" ]]; then
        if [[ $FOUND_COUNT -gt 0 ]]; then
            fix_references
        else
            echo -e "${GREEN}无需修复，所有路径已是 V2.2 新结构${PLAIN}"
        fi
    elif [[ $FOUND_COUNT -gt 0 ]]; then
        echo -e "${YELLOW}提示: 使用 --fix 参数自动修复旧路径引用${PLAIN}"
        echo -e "  bash tools/migrate-flavor.sh --fix"
        echo ""
    fi

    # 4. 摘要
    summary
}

main "$@"
