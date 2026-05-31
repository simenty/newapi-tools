// NewAPI Tools - Application error types with structured error codes
package apperr

import (
	"fmt"

	"github.com/simenty/newapi-tools/internal/i18n"
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
	CodeUpdateCheckFail  = "U001" // 版本检查失败
	CodeUpdateSelfFail   = "U002" // 自更新失败
	CodeUpdateVerifyFail = "U003" // SHA256 校验失败
	CodeInstanceExists   = "I003" // 实例名已存在
	CodeInstanceNotFound = "I004" // 实例不存在
	CodeInstanceActive   = "I005" // 实例为当前活跃实例，无法删除
	CodeDoctorFailed     = "X001" // 诊断检查失败
)

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
// If suggestion is empty, it is populated from i18n.
func New(code, msg, suggestion string, cause error) *AppError {
	// Try i18n translation for the message
	translatedMsg := i18n.T(code, msg)
	if translatedMsg == code {
		// No i18n key matched, use the provided message
		translatedMsg = msg
	}

	// Fill suggestion from i18n if not provided
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
// The message is derived from cause.Error(), and the suggestion comes from i18n.
func Wrap(code, suggestion string, cause error) *AppError {
	msg := ""
	if cause != nil {
		msg = cause.Error()
	}
	return New(code, msg, suggestion, cause)
}

// GetSuggestion returns the fix suggestion for the given error code via i18n.
// Returns empty string if no suggestion is registered.
func GetSuggestion(code string) string {
	key := fmt.Sprintf("err.suggest.%s", code)
	result := i18n.T(key)
	if result == key {
		return ""
	}
	return result
}
