// NewAPI Tools - Docker management platform for newapi
package ui

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// Progress represents a simple progress indicator for long-running operations.
type Progress struct {
	total     int
	current   int
	prefix    string
	writer    io.Writer
	startTime time.Time
}

// NewProgress creates a new Progress instance.
func NewProgress(writer io.Writer, prefix string, total int) *Progress {
	return &Progress{
		total:     total,
		current:   0,
		prefix:    prefix,
		writer:    writer,
		startTime: time.Now(),
	}
}

// Add increments the progress counter by n and displays the progress bar.
func (p *Progress) Add(n int) {
	p.current += n
	if p.current > p.total {
		p.current = p.total
	}
	p.render()
}

// Done marks the progress as complete and prints a newline.
func (p *Progress) Done() {
	p.current = p.total
	p.render()
	fmt.Fprintln(p.writer)
}

// Elapsed returns the time elapsed since the progress started.
func (p *Progress) Elapsed() time.Duration {
	return time.Since(p.startTime)
}

// render draws the progress bar to the writer.
func (p *Progress) render() {
	percent := 0
	if p.total > 0 {
		percent = p.current * 100 / p.total
	}

	barWidth := 30
	filled := barWidth * percent / 100

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	fmt.Fprintf(p.writer, "\r%s [%s] %d%% (%d/%d)", p.prefix, bar, percent, p.current, p.total)
}

// Spinner shows a simple spinner animation for indeterminate progress.
type Spinner struct {
	frames  []rune
	current int
	prefix  string
	writer  io.Writer
	active  bool
	stopCh  chan struct{}
}

// NewSpinner creates a new Spinner instance.
func NewSpinner(writer io.Writer, prefix string) *Spinner {
	return &Spinner{
		frames:  []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'},
		current: 0,
		prefix:  prefix,
		writer:  writer,
		active:  false,
		stopCh:  make(chan struct{}),
	}
}

// Start begins the spinner animation.
func (s *Spinner) Start() {
	if s.active {
		return
	}
	s.active = true
	go func() {
		for {
			select {
			case <-s.stopCh:
				return
			default:
				frame := s.frames[s.current%len(s.frames)]
				fmt.Fprintf(s.writer, "\r%s %c", s.prefix, frame)
				s.current++
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
}

// Stop halts the spinner and clears the line.
func (s *Spinner) Stop() {
	if !s.active {
		return
	}
	s.active = false
	s.stopCh <- struct{}{}
	fmt.Fprintf(s.writer, "\r%s", strings.Repeat(" ", len(s.prefix)+3))
	fmt.Fprintf(s.writer, "\r")
}
