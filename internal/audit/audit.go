// NewAPI Tools - Audit logging for command execution tracking
package audit

import (
	"encoding/json"
	"fmt"
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
	if err := os.MkdirAll(filepath.Dir(a.path), 0700); err != nil {
		return fmt.Errorf("audit: failed to create directory: %w", err)
	}

	// Rotate if needed
	if err := a.rotate(); err != nil {
		return fmt.Errorf("audit: rotation failed: %w", err)
	}

	// Marshal entry to JSON
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("audit: failed to marshal entry: %w", err)
	}

	// Open file in append mode
	f, err := os.OpenFile(a.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("audit: failed to open log file: %w", err)
	}
	defer f.Close()

	// Write JSON line with newline
	if _, err := f.Write(append(data, '\n')); err != nil {
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

// rotatedFiles returns the paths of rotated log files, sorted from oldest to newest.
// e.g. audit.log.5, audit.log.4, ..., audit.log.1
func (r *AuditReader) rotatedFiles() []string {
	var files []string
	for i := 1; ; i++ {
		rotated := fmt.Sprintf("%s.%d", r.path, i)
		if _, err := os.Stat(rotated); err != nil {
			break
		}
		files = append(files, rotated)
	}
	// Reverse so oldest file comes first: .1 is newest rotated, .N is oldest
	for i, j := 0, len(files)-1; i < j; i, j = i+1, j-1 {
		files[i], files[j] = files[j], files[i]
	}
	return files
}

// decodeEntries reads all AuditEntry values from a file.
func decodeEntries(filePath string) ([]AuditEntry, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var entries []AuditEntry
	for decoder.More() {
		var entry AuditEntry
		if err := decoder.Decode(&entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// List returns audit log entries matching the given options.
// When opt.Last > 0, uses a ring buffer to avoid loading all entries into memory.
// Reads from all rotated files (oldest first) and the main file to provide complete results.
func (r *AuditReader) List(opt ListOption) ([]AuditEntry, error) {
	// Gather all files to read (rotated files oldest-first, then main)
	rotated := r.rotatedFiles()
	allPaths := append(rotated, r.path)

	// When opt.Last > 0, use a ring buffer across all files to keep only the N most recent matching entries
	if opt.Last > 0 {
		ring := make([]AuditEntry, 0, opt.Last)
		for _, filePath := range allPaths {
			file, err := os.Open(filePath)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, fmt.Errorf("open audit log: %w", err)
			}

			decoder := json.NewDecoder(file)
			for decoder.More() {
				var entry AuditEntry
				if err := decoder.Decode(&entry); err != nil {
					continue
				}
				if !r.matchesFilter(entry, opt) {
					continue
				}
				if len(ring) < opt.Last {
					ring = append(ring, entry)
				} else {
					ring = append(ring[1:], entry)
				}
			}
			file.Close()
		}
		// Reverse to show newest first
		for i, j := 0, len(ring)-1; i < j; i, j = i+1, j-1 {
			ring[i], ring[j] = ring[j], ring[i]
		}
		return ring, nil
	}

	// No Last limit: read all entries from all files
	var entries []AuditEntry
	for _, filePath := range allPaths {
		fileEntries, err := decodeEntries(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("open audit log: %w", err)
		}
		entries = append(entries, fileEntries...)
	}

	// Apply non-Last filters (Command, Since)
	filtered := r.filterEntries(entries, opt)

	return filtered, nil
}

// matchesFilter checks if a single entry matches the command and since filters.
func (r *AuditReader) matchesFilter(entry AuditEntry, opt ListOption) bool {
	if opt.Command != "" {
		if !strings.Contains(strings.ToLower(entry.Command), strings.ToLower(opt.Command)) {
			return false
		}
	}
	if !opt.Since.IsZero() {
		if entry.Timestamp.Before(opt.Since) {
			return false
		}
	}
	return true
}

func (r *AuditReader) filterEntries(entries []AuditEntry, opt ListOption) []AuditEntry {
	var result []AuditEntry

	for _, entry := range entries {
		if !r.matchesFilter(entry, opt) {
			continue
		}
		result = append(result, entry)
	}

	// Reverse to show newest first
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result
}
