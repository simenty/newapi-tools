#!/bin/bash
# .env 文件加密/解密工具（AES-256-CBC — 资深开发优化版）
# 增强：密码确认、文件存在检查、安全清理
set -eo pipefail

# shellcheck source=lib/_init.sh
source "${TOOLKIT_ROOT}/lib/_init.sh"

ENV_FILE="${NEWAPI_HOME:-/home/new-api}/.env"
ENCRYPTED_FILE="${ENV_FILE}.enc"

check_dependency() {
    if ! command -v openssl &>/dev/null; then
        log_error "缺少依赖: openssl，请先安装: apt install openssl"
        exit 1
    fi
}

do_encrypt() {
    check_dependency

    if [[ ! -f "$ENV_FILE" ]]; then
        log_error "找不到 .env 文件: $ENV_FILE"
        log_info "请先部署 NewAPI，或检查 NEWAPI_HOME 配置"
        exit 1
    fi

    # 如果已存在加密文件，确认覆盖
    if [[ -f "$ENCRYPTED_FILE" ]]; then
        if ! ask_confirm "加密文件已存在，是否覆盖？"; then
            exit 0
        fi
    fi

    # 读取密码（带确认）
    read -s -r -p "请输入加密密码: " PASS; echo
    if [[ -z "$PASS" ]]; then
        log_error "密码不能为空"
        return 1
    fi
    read -s -r -p "请再次输入加密密码: " PASS_CONFIRM; echo
    if [[ "$PASS" != "$PASS_CONFIRM" ]]; then
        log_error "两次输入的密码不一致"
        unset PASS PASS_CONFIRM
        return 1
    fi

    # 执行加密（使用 stdin 传递密码，避免命令行泄露）
    if echo "$PASS" | openssl enc -aes-256-cbc -salt -in "$ENV_FILE" -out "$ENCRYPTED_FILE" -pass stdin 2>>"$LOG_FILE"; then
        log_success "加密完成: $ENCRYPTED_FILE"
        log_warn "建议删除明文 .env 文件，使用时通过解密挂载。"
        log_info "解密命令: newapi-tools encrypt-config.sh decrypt"
    else
        log_error "加密失败，请查看日志: $LOG_FILE"
        shred -u "$ENCRYPTED_FILE" 2>/dev/null || rm -f "$ENCRYPTED_FILE"
    fi

    # 安全清除密码变量
    unset PASS PASS_CONFIRM
}

do_decrypt() {
    check_dependency

    if [[ ! -f "$ENCRYPTED_FILE" ]]; then
        log_error "找不到加密文件: $ENCRYPTED_FILE"
        exit 1
    fi

    read -s -r -p "请输入解密密码: " PASS; echo
    if [[ -z "$PASS" ]]; then
        log_error "密码不能为空"
        return 1
    fi

    # 执行解密（输出到临时文件，验证成功后再替换）
    local tmp_file
    tmp_file=$(secure_temp_file "dec")
    if echo "$PASS" | openssl enc -aes-256-cbc -d -in "$ENCRYPTED_FILE" -out "$tmp_file" -pass stdin 2>/dev/null; then
        mv "$tmp_file" "$ENV_FILE"
        chmod 600 "$ENV_FILE"
        log_success "解密完成: $ENV_FILE（权限已设为 600）"
    else
        log_error "解密失败，密码错误或文件损坏"
        rm -f "$tmp_file"
    fi

    unset PASS
}

# ---------- 以下为主执行逻辑，仅在直接运行时执行 ----------
if [[ "${BASH_SOURCE[0]}" != "$0" ]]; then
    return 0 2>/dev/null || true
fi

# ---------- 主逻辑 ----------
case "${1:-}" in
    encrypt)
        do_encrypt
        ;;
    decrypt)
        do_decrypt
        ;;
    -h|--help)
        echo "用法: newapi-tools encrypt-config {encrypt|decrypt}"
        echo ""
        echo "  encrypt  - 加密 .env 文件为 .env.enc"
        echo "  decrypt  - 解密 .env.enc 为 .env"
        ;;
    *)
        log_error "未知操作: ${1:-}"
        echo "用法: $0 {encrypt|decrypt}"
        exit 1
        ;;
esac
