// NewAPI Tools - Docker management platform for newapi
package ui

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// Table represents a simple text-based table for CLI output.
type Table struct {
	headers []string
	rows    [][]string
	writer  io.Writer
}

// NewTable creates a new Table with the given headers.
func NewTable(headers ...string) *Table {
	return &Table{
		headers: headers,
		rows:    make([][]string, 0),
		writer:  os.Stdout,
	}
}

// SetWriter sets the output writer for the table.
func (t *Table) SetWriter(w io.Writer) *Table {
	t.writer = w
	return t
}

// AddRow adds a row to the table.
func (t *Table) AddRow(values ...string) *Table {
	t.rows = append(t.rows, values)
	return t
}

// Render prints the table to the writer with aligned columns.
func (t *Table) Render() {
	if len(t.headers) == 0 {
		return
	}

	// Calculate column widths
	colWidths := make([]int, len(t.headers))
	for i, h := range t.headers {
		colWidths[i] = len(h)
	}
	for _, row := range t.rows {
		for i, cell := range row {
			if i < len(colWidths) && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	// Print header
	t.printRow(t.headers, colWidths)

	// Print separator
	var sepParts []string
	for _, w := range colWidths {
		sepParts = append(sepParts, strings.Repeat("-", w))
	}
	fmt.Fprintf(t.writer, "  %s\n", strings.Join(sepParts, "  "))

	// Print rows
	for _, row := range t.rows {
		// Pad row if shorter than headers
		padded := make([]string, len(t.headers))
		for i := range padded {
			if i < len(row) {
				padded[i] = row[i]
			} else {
				padded[i] = ""
			}
		}
		t.printRow(padded, colWidths)
	}
}

// printRow prints a single row with proper alignment.
func (t *Table) printRow(values []string, widths []int) {
	var parts []string
	for i, v := range values {
		if i < len(widths) {
			parts = append(parts, fmt.Sprintf("%-*s", widths[i], v))
		} else {
			parts = append(parts, v)
		}
	}
	fmt.Fprintf(t.writer, "  %s\n", strings.Join(parts, "  "))
}

// KeyValue prints a key-value pair with aligned formatting.
func KeyValue(writer io.Writer, key string, value interface{}) {
	fmt.Fprintf(writer, "  %-20s %v\n", key+":", value)
}

// InfoBox prints a bordered info box with a message.
func InfoBox(writer io.Writer, title string, lines ...string) {
	maxWidth := len(title)
	for _, line := range lines {
		if len(line) > maxWidth {
			maxWidth = len(line)
		}
	}
	maxWidth += 4 // padding

	border := strings.Repeat("─", maxWidth)
	fmt.Fprintf(writer, "  ┌%s┐\n", border)
	fmt.Fprintf(writer, "  │ %-*s │\n", maxWidth-2, title)
	fmt.Fprintf(writer, "  ├%s┤\n", border)
	for _, line := range lines {
		fmt.Fprintf(writer, "  │ %-*s │\n", maxWidth-2, line)
	}
	fmt.Fprintf(writer, "  └%s┘\n", border)
}
