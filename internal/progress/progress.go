package progress

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Bar struct {
	total       int64
	current     int64
	width       int
	writer      io.Writer
	mu          sync.Mutex
	currentDirs map[string]bool
	dirMu       sync.Mutex
	enabled     bool
	lastUpdate  time.Time
}

func New(total int64) *Bar {
	return &Bar{
		total:       total,
		current:     0,
		width:       50,
		writer:      os.Stdout,
		currentDirs: make(map[string]bool),
		enabled:     true, // Always enabled - terminal detection can be unreliable
		lastUpdate:  time.Now(),
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
	if !b.enabled {
		return
	}

	b.dirMu.Lock()
	if !b.currentDirs[dir] {
		b.currentDirs[dir] = true
	}
	b.dirMu.Unlock()

	// Render outside of lock to avoid deadlock
	b.mu.Lock()
	b.render()
	b.mu.Unlock()
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

	// Get current directories being processed
	b.dirMu.Lock()
	dirs := make([]string, 0, len(b.currentDirs))
	for dir := range b.currentDirs {
		dirs = append(dirs, filepath.Base(dir))
	}
	b.dirMu.Unlock()

	var dirDisplay string
	if len(dirs) > 0 {
		if len(dirs) > 3 {
			dirDisplay = fmt.Sprintf(" | %s, %s, %s +%d more", dirs[0], dirs[1], dirs[2], len(dirs)-3)
		} else {
			dirDisplay = " | " + strings.Join(dirs, ", ")
		}
	}

	// Clear the line and write progress
	fmt.Fprintf(b.writer, "\r\033[K[%s] %3d%% (%d/%d)%s",
		bar, int(percent), b.current, b.total, dirDisplay)
}

func (b *Bar) Finish() {
	if !b.enabled {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.current = b.total
	b.render()
	fmt.Fprintf(b.writer, "\n")
}
