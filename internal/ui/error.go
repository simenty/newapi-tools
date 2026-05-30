// NewAPI Tools - Unified error display for user-facing output
package ui

import (
	"fmt"
	"os"

	"github.com/simenty/newapi-tools/internal/apperr"
)

// ANSI color codes for terminal output.
const (
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorReset  = "\033[0m"
)

// PrintError prints a formatted error to stderr.
// For *AppError, it displays the error code, message, and suggestion with colors:
//
//	❌ [D001] Docker 未安装
//	💡 请先安装 Docker: https://docs.docker.com/engine/install/
//
// For other error types, it simply prints the error message.
func PrintError(err error) {
	if err == nil {
		return
	}

	appErr, ok := err.(*apperr.AppError)
	if !ok {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		return
	}

	// Colored output for AppError
	fmt.Fprintf(os.Stderr, "%s❌%s %s\n", colorRed, colorReset, appErr.Error())
	if appErr.Suggestion != "" {
		fmt.Fprintf(os.Stderr, "%s💡%s %s\n", colorYellow, colorReset, appErr.Suggestion)
	}
}
