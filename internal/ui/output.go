// NewAPI Tools - UI output utilities and logger setup
package ui

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/Bonus520/newapi-tools/internal/core"
	"github.com/Bonus520/newapi-tools/internal/i18n"
)

// SetupLogger initializes the global slog logger based on the provided config.
// It supports "text" and "json" formats, and "debug", "info", "warn", "error" levels.
func SetupLogger(cfg *core.LogConfig) {
	var level slog.Level
	switch strings.ToLower(cfg.Level) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	switch strings.ToLower(cfg.Format) {
	case "json":
		handler = slog.NewJSONHandler(os.Stderr, opts)
	default:
		handler = slog.NewTextHandler(os.Stderr, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
}

// L returns the default slog.Logger.
func L() *slog.Logger {
	return slog.Default()
}

// PrintStep prints a progress step indicator to stdout.
// Format: [step/total] message
// The message is translated via i18n.T() if a matching key exists.
func PrintStep(step, total int, msg string) {
	fmt.Printf("[%d/%d] %s\n", step, total, i18n.T(msg))
}
