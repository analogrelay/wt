package tui

import (
	"os"

	"golang.org/x/term"
)

// IsTerminal reports whether the given file descriptor is a terminal.
func IsTerminal(fd uintptr) bool {
	return term.IsTerminal(int(fd))
}

// IsFancyOutput reports whether stdout supports styled output
// (is a TTY and NO_COLOR is not set).
func IsFancyOutput() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	return IsTerminal(os.Stdout.Fd())
}

// IsFancyErr reports whether stderr supports animated output (is a TTY).
func IsFancyErr() bool {
	return IsTerminal(os.Stderr.Fd())
}
