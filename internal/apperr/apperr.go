// NewAPI Tools - Application error types with structured error codes
package apperr

import (
	"fmt"

	"github.com/Bonus520/newapi-tools/internal/i18n"
)

// Error code constants for categorizing application errors.
const (
	CodeDockerNotFound   = "D001" // Docker is not installed
	CodeDockerDaemonDown = "D002" // Docker daemon is not running
	CodeDockerGroupMiss  = "D003" // Current user not in docker group
	CodeConfigLoad       = "C001" // Configuration load failed
	CodeConfigPerm       = "C002" // Config file permissions too wide
	CodeInstallFailed    = "I001" // Installation failed
	CodeInstallTimeout   = "I002" // Installation timeout
	CodeSystemArch       = "S001" // Unsupported architecture
	CodeMirrorApply      = "M001" // Mirror apply failed
	CodeBackupFailed     = "B001" // Backup failed
	CodeRestoreFailed    = "B002" // Restore failed
	CodePluginInit       = "P001" // Plugin initialization failed
)

// suggestions maps error codes to user-friendly fix suggestions.
// These are hardcoded for now and can be migrated to i18n later.
var suggestions = map[string]string{
	"D001": "请先安装 Docker: https://docs.docker.com/engine/install/",
	"D002": "请启动 Docker 服务: sudo systemctl start docker",
	"D003": "请将当前用户加入 docker 组: sudo usermod -aG docker $USER",
	"C001": "请检查配置文件格式是否正确",
	"C002": "建议设置配置文件权限: chmod 600 <config-file>",
	"I001": "请检查网络连接和 Docker 状态",
	"I002": "容器启动超时，请手动检查: docker ps",
	"S001": "当前架构不支持，请使用 amd64 或 arm64 系统",
	"M001": "请检查镜像源地址是否可用",
	"B001": "请检查磁盘空间和备份路径权限",
	"B002": "请检查备份文件是否完整",
	"P001": "请检查插件目录和 metadata.yml 格式",
}

// AppError represents a structured application error with a code, message, and suggestion.
type AppError struct {
	Code       string // Error code, e.g. "D001", "I002"
	Message    string // User-visible description (already translated via i18n.T())
	Suggestion string // Fix suggestion
	Cause      error  // Underlying error
}

// Error returns the string representation of the error in the format "[CODE] message".
func (e *AppError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("[%s] %s", e.Code, e.Message)
	}
	return e.Message
}

// Unwrap returns the underlying cause error, enabling errors.Is/As chaining.
func (e *AppError) Unwrap() error {
	return e.Cause
}

// New creates a new AppError with the given code, message, suggestion, and cause.
// The message is translated via i18n.T() if a matching key exists.
// If suggestion is empty, it is populated from the suggestions map.
func New(code, msg, suggestion string, cause error) *AppError {
	// Try i18n translation for the message
	translatedMsg := i18n.T(code, msg)
	if translatedMsg == code {
		// No i18n key matched, use the provided message
		translatedMsg = msg
	}

	// Fill suggestion from map if not provided
	if suggestion == "" {
		suggestion = GetSuggestion(code)
	}

	return &AppError{
		Code:       code,
		Message:    translatedMsg,
		Suggestion: suggestion,
		Cause:      cause,
	}
}

// Wrap creates a new AppError that wraps an existing error.
// The message is derived from cause.Error(), and the suggestion comes from the suggestions map.
func Wrap(code, suggestion string, cause error) *AppError {
	msg := ""
	if cause != nil {
		msg = cause.Error()
	}
	return New(code, msg, suggestion, cause)
}

// GetSuggestion returns the fix suggestion for the given error code.
// Returns empty string if no suggestion is registered.
func GetSuggestion(code string) string {
	return suggestions[code]
}
