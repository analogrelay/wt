package tui

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var spinnerFrames = []string{"✦", "✧", "★", "☆", "✶", "✸"}
var spinnerColors = []lipgloss.Color{Blue, Purple, lipgloss.Color("51")} // blue, magenta, cyan

// Spinner provides a simple API to show/hide a spinner on stderr.
type Spinner struct {
	mu      sync.Mutex
	stop    chan struct{}
	stopped chan struct{}
	message string
}

// StartSpinner starts a spinner with the given message.
// In non-TTY environments, prints a plain status line instead.
func StartSpinner(message string) *Spinner {
	s := &Spinner{
		stop:    make(chan struct{}),
		stopped: make(chan struct{}),
		message: message,
	}

	if !IsFancyErr() {
		fmt.Fprintf(os.Stderr, "⏳ %s\n", message)
		close(s.stopped)
		return s
	}

	go s.run()
	return s
}

func (s *Spinner) run() {
	defer close(s.stopped)

	frame := 0
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.stop:
			// Clear the spinner line
			fmt.Fprintf(os.Stderr, "\r\033[K")
			return
		case <-ticker.C:
			char := spinnerFrames[frame%len(spinnerFrames)]
			color := spinnerColors[frame%len(spinnerColors)]
			style := lipgloss.NewStyle().Foreground(color)
			fmt.Fprintf(os.Stderr, "\r%s %s", style.Render(char), DimStyle.Render(s.message))
			frame++
		}
	}
}

// Stop stops the spinner.
func (s *Spinner) Stop() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-s.stop:
		// already stopped
	default:
		close(s.stop)
	}
	<-s.stopped
}

