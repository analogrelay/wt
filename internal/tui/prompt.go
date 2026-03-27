package tui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Confirm shows a y/N prompt and returns true if the user answers yes.
func Confirm(message string) bool {
	fmt.Printf("%s [y/N] ", message)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		ans := strings.TrimSpace(scanner.Text())
		return strings.HasPrefix(strings.ToLower(ans), "y")
	}
	return false
}

// ConfirmDefault shows a Y/n prompt and returns true unless the user answers no.
func ConfirmDefault(message string) bool {
	fmt.Printf("%s [Y/n] ", message)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		ans := strings.TrimSpace(scanner.Text())
		return !strings.HasPrefix(strings.ToLower(ans), "n")
	}
	return true
}

// Prompt shows a prompt and returns the user's input.
func Prompt(message string) string {
	fmt.Print(message)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}
