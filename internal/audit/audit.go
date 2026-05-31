// NewAPI Tools - Audit logging for command execution tracking
package audit

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// AuditEntry represents a single audit log entry for command execution.
type AuditEntry struct {
	Timestamp  time.Time `json:"ts"`
	Command    string    `json:"cmd"`
	User       string    `json:"user"`
	Args       []string  `json:"args"`
	Result     string    `json:"result"` // "ok" | "error"
	Error      string    `json:"error,omitempty"`
	DurationMs int64     `json:"duration_ms"`
}

// AuditLogger writes audit entries as JSON Lines to a log file with rotation.
type AuditLogger struct {
	path    string
	maxSize int64 // maximum file size in bytes before rotation
	keep    int   // number of rotated files to keep
	mu      sync.Mutex
}

// DefaultAuditPath returns the default audit log file path:
// ~/.config/newapi-tools/audit.log
func DefaultAuditPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "audit.log"
	}
	return filepath.Join(home, ".config", "newapi-tools", "audit.log")
}

// NewAuditLogger creates a new AuditLogger that writes to the given path.
// If path is empty, the default path (~/.config/newapi-tools/audit.log) is used.
// The default max size is 10MB and the default keep count is 5.
func NewAuditLogger(path string) *AuditLogger {
	if path == "" {
		path = DefaultAuditPath()
	}
	return &AuditLogger{
		path:    path,
		maxSize: 10 * 1024 * 1024, // 10 MB
		keep:    5,
	}
}

// Log writes an AuditEntry as a JSON line to the audit log file.
// It performs rotation before writing if the file exceeds maxSize.
// If writing fails, the error is returned to the caller so the caller can decide
// whether to treat audit failures as fatal.
func (a *AuditLogger) Log(entry AuditEntry) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(a.path), 0755); err != nil {
		slog.Warn("audit: failed to create directory", "path", filepath.Dir(a.path), "error", err)
		return fmt.Errorf("audit: failed to create directory: %w", err)
	}

	// Rotate if needed
	if err := a.rotate(); err != nil {
		slog.Warn("audit: rotation failed", "error", err)
		// Continue writing even if rotation fails
	}

	// Marshal entry to JSON
	data, err := json.Marshal(entry)
	if err != nil {
		slog.Warn("audit: failed to marshal entry", "error", err)
		return fmt.Errorf("audit: failed to marshal entry: %w", err)
	}

	// Open file in append mode
	f, err := os.OpenFile(a.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		slog.Warn("audit: failed to open log file", "path", a.path, "error", err)
		return fmt.Errorf("audit: failed to open log file: %w", err)
	}
	defer f.Close()

	// Write JSON line with newline
	if _, err := f.Write(append(data, '\n')); err != nil {
		slog.Warn("audit: failed to write entry", "error", err)
		return fmt.Errorf("audit: failed to write entry: %w", err)
	}

	return nil
}

// rotate checks if the current log file exceeds maxSize and rotates it if needed.
// Rotation renames audit.log -> audit.log.1, audit.log.1 -> audit.log.2, etc.
// Files beyond the keep count are deleted.
func (a *AuditLogger) rotate() error {
	info, err := os.Stat(a.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist yet, no rotation needed
		}
		return fmt.Errorf("audit: failed to stat log file: %w", err)
	}

	if info.Size() < a.maxSize {
		return nil // File is small enough, no rotation needed
	}

	// Delete the oldest rotated file if it exists
	oldest := fmt.Sprintf("%s.%d", a.path, a.keep)
	if _, err := os.Stat(oldest); err == nil {
		if err := os.Remove(oldest); err != nil {
			return fmt.Errorf("audit: failed to remove oldest rotated file: %w", err)
		}
	}

	// Shift rotated files: .N-1 -> .N, .N-2 -> .N-1, ..., .1 -> .2
	for i := a.keep - 1; i >= 1; i-- {
		src := fmt.Sprintf("%s.%d", a.path, i)
		dst := fmt.Sprintf("%s.%d", a.path, i+1)
		if _, err := os.Stat(src); err == nil {
			if err := os.Rename(src, dst); err != nil {
				return fmt.Errorf("audit: failed to rename rotated file %s -> %s: %w", src, dst, err)
			}
		}
	}

	// Rename current log file to .1
	if err := os.Rename(a.path, a.path+".1"); err != nil {
		return fmt.Errorf("audit: failed to rename current log file: %w", err)
	}

	return nil
}

// Path returns the audit log file path.
func (a *AuditLogger) Path() string {
	return a.path
}

// --- Audit Reader ---

// AuditReader reads and queries audit log entries
type AuditReader struct {
	path string
}

// NewAuditReader creates an AuditReader for the given path
func NewAuditReader(path string) *AuditReader {
	if path == "" {
		path = DefaultAuditPath()
	}
	return &AuditReader{path: path}
}

// ListOption contains filtering options for listing audit entries
type ListOption struct {
	Last    int       // Number of most recent entries to return (0 = all)
	Command string    // Filter by command name (substring match)
	Since   time.Time // Filter entries after this time (zero = no filter)
}

// List returns audit log entries matching the given options
func (r *AuditReader) List(opt ListOption) ([]AuditEntry, error) {
	// Open log file
	file, err := os.Open(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []AuditEntry{}, nil // No log file, return empty
		}
		return nil, fmt.Errorf("open audit log: %w", err)
	}
	defer file.Close()

	var entries []AuditEntry
	decoder := json.NewDecoder(file)

	// Read all entries
	for decoder.More() {
		var entry AuditEntry
		if err := decoder.Decode(&entry); err != nil {
			// Skip malformed lines
			continue
		}
		entries = append(entries, entry)
	}

	// Apply filters
	filtered := r.filterEntries(entries, opt)

	return filtered, nil
}

func (r *AuditReader) filterEntries(entries []AuditEntry, opt ListOption) []AuditEntry {
	var result []AuditEntry

	for _, entry := range entries {
		// Filter by command
		if opt.Command != "" {
			if !strings.Contains(strings.ToLower(entry.Command), strings.ToLower(opt.Command)) {
				continue
			}
		}

		// Filter by time
		if !opt.Since.IsZero() {
			if entry.Timestamp.Before(opt.Since) {
				continue
			}
		}

		result = append(result, entry)
	}

	// Apply Last filter (take most recent N)
	if opt.Last > 0 && len(result) > opt.Last {
		result = result[len(result)-opt.Last:]
	}

	// Reverse to show newest first
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result
}
