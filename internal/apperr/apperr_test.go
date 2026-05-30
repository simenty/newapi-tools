// NewAPI Tools - Application error type tests
package apperr

import (
	"errors"
	"fmt"
	"testing"
)

func TestNew(t *testing.T) {
	cause := errors.New("underlying error")
	appErr := New(CodeDockerNotFound, "Docker 未安装", "", cause)

	if appErr.Code != CodeDockerNotFound {
		t.Errorf("expected code %s, got %s", CodeDockerNotFound, appErr.Code)
	}
	if appErr.Message == "" {
		t.Error("expected non-empty message")
	}
	if appErr.Cause != cause {
		t.Error("expected cause to match")
	}
}

func TestNewWithSuggestion(t *testing.T) {
	appErr := New(CodeDockerNotFound, "Docker 未安装", "custom suggestion", nil)

	if appErr.Suggestion != "custom suggestion" {
		t.Errorf("expected custom suggestion, got %q", appErr.Suggestion)
	}
}

func TestNewAutoSuggestion(t *testing.T) {
	// When suggestion is empty, it should be populated from the suggestions map
	appErr := New(CodeDockerNotFound, "Docker 未安装", "", nil)

	if appErr.Suggestion == "" {
		t.Error("expected suggestion from map, got empty string")
	}
	if appErr.Suggestion != suggestions[CodeDockerNotFound] {
		t.Errorf("expected %q, got %q", suggestions[CodeDockerNotFound], appErr.Suggestion)
	}
}

func TestWrap(t *testing.T) {
	cause := errors.New("connection refused")
	appErr := Wrap(CodeDockerDaemonDown, "", cause)

	if appErr.Code != CodeDockerDaemonDown {
		t.Errorf("expected code %s, got %s", CodeDockerDaemonDown, appErr.Code)
	}
	if appErr.Message != "connection refused" {
		t.Errorf("expected message from cause, got %q", appErr.Message)
	}
	if appErr.Cause != cause {
		t.Error("expected cause to match")
	}
	if appErr.Suggestion == "" {
		t.Error("expected suggestion from map")
	}
}

func TestWrapNilCause(t *testing.T) {
	appErr := Wrap(CodeInstallFailed, "", nil)

	if appErr.Message != "" {
		t.Errorf("expected empty message for nil cause, got %q", appErr.Message)
	}
	if appErr.Cause != nil {
		t.Error("expected nil cause")
	}
}

func TestError(t *testing.T) {
	appErr := New(CodeDockerNotFound, "Docker 未安装", "", nil)

	got := appErr.Error()
	expected := "[D001] Docker 未安装"
	if got != expected {
		t.Errorf("Error() = %q, want %q", got, expected)
	}
}

func TestErrorEmptyCode(t *testing.T) {
	appErr := New("", "some message", "", nil)

	got := appErr.Error()
	if got != "some message" {
		t.Errorf("Error() with empty code = %q, want %q", got, "some message")
	}
}

func TestUnwrap(t *testing.T) {
	cause := errors.New("root cause")
	appErr := Wrap(CodeConfigLoad, "", cause)

	unwrapped := appErr.Unwrap()
	if unwrapped != cause {
		t.Error("Unwrap() should return the original cause")
	}
}

func TestUnwrapNil(t *testing.T) {
	appErr := New(CodeInstallFailed, "install failed", "", nil)

	unwrapped := appErr.Unwrap()
	if unwrapped != nil {
		t.Error("Unwrap() should return nil when no cause")
	}
}

func TestGetSuggestion(t *testing.T) {
	for code, expectedSuggestion := range suggestions {
		got := GetSuggestion(code)
		if got != expectedSuggestion {
			t.Errorf("GetSuggestion(%q) = %q, want %q", code, got, expectedSuggestion)
		}
	}
}

func TestGetSuggestionUnknown(t *testing.T) {
	got := GetSuggestion("Z999")
	if got != "" {
		t.Errorf("GetSuggestion for unknown code should return empty string, got %q", got)
	}
}

func TestAllCodesHaveSuggestions(t *testing.T) {
	codes := []string{
		CodeDockerNotFound, CodeDockerDaemonDown, CodeDockerGroupMiss,
		CodeConfigLoad, CodeConfigPerm,
		CodeInstallFailed, CodeInstallTimeout,
		CodeSystemArch,
		CodeMirrorApply,
		CodeBackupFailed, CodeRestoreFailed,
		CodePluginInit,
		CodeUpdateCheckFail, CodeUpdateSelfFail, CodeUpdateVerifyFail,
		CodeInstanceExists, CodeInstanceNotFound, CodeInstanceActive,
	}
	for _, code := range codes {
		s := GetSuggestion(code)
		if s == "" {
			t.Errorf("code %q has no suggestion registered", code)
		}
	}
}

func TestErrorFormatting(t *testing.T) {
	// Verify the formatted error string contains the code
	appErr := New(CodeInstallTimeout, "安装超时", "", nil)
	str := fmt.Sprintf("%v", appErr)
	if str != "[I002] 安装超时" {
		t.Errorf("formatted error = %q, want %q", str, "[I002] 安装超时")
	}
}

func TestErrorIs(t *testing.T) {
	cause := errors.New("root cause")
	appErr := Wrap(CodeDockerNotFound, "", cause)

	// errors.Unwrap should work
	unwrapped := errors.Unwrap(appErr)
	if unwrapped == nil {
		t.Error("errors.Unwrap should return the cause")
	}
}
