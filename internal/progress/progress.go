package progress

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type Bar struct {
	total      int64
	current    int64
	width      int
	writer     io.Writer
	mu         sync.Mutex
	enabled    bool
	lastUpdate time.Time
}

func New(total int64) *Bar {
	return &Bar{
		total:      total,
		current:    0,
		width:      50,
		writer:     os.Stdout,
		enabled:    true, // Always enabled - terminal detection can be unreliable
		lastUpdate: time.Now(),
	}
}

func isTerminal() bool {
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	// Check if stdout is a terminal (character device)
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

func (b *Bar) SetDirectory(dir string) {
	// No-op: directory tracking removed for simpler display
}

func (b *Bar) Increment() {
	if !b.enabled {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.current++

	// Update at most every 100ms to reduce flickering
	now := time.Now()
	if now.Sub(b.lastUpdate) > 100*time.Millisecond || b.current == b.total {
		b.lastUpdate = now
		b.render()
	}
}

// render must be called with mu already locked
func (b *Bar) render() {
	if b.total == 0 {
		return
	}

	percent := float64(b.current) / float64(b.total) * 100
	filledWidth := int(float64(b.width) * float64(b.current) / float64(b.total))

	if filledWidth > b.width {
		filledWidth = b.width
	}

	bar := strings.Repeat("█", filledWidth) + strings.Repeat("░", b.width-filledWidth)

	// OSC 9;4 - macOS terminal progress bar (Ghostty 1.2+)
	// Format: \e]9;4;{state};{percentage}\e\\
	// state: 1 = show progress
	fmt.Fprintf(b.writer, "\033]9;4;1;%d\033\\", int(percent))

	// Clear the line and write progress
	fmt.Fprintf(b.writer, "\r\033[K[%s] %3d%% (%d/%d)",
		bar, int(percent), b.current, b.total)
}

func (b *Bar) Finish() {
	if !b.enabled {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.current = b.total
	b.render()

	// OSC 9;4 - Hide progress bar
	// Format: \e]9;4;0;0\e\\ (state 0 = hide)
	fmt.Fprintf(b.writer, "\033]9;4;0;0\033\\")

	fmt.Fprintf(b.writer, "\n")
}
