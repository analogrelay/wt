package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// Column defines a table column.
type Column struct {
	Title    string
	MinWidth int
}

// Table renders a styled terminal table.
type Table struct {
	Columns []Column
	Rows    [][]string
}

// Render produces the styled table string.
func (t *Table) Render() string {
	if len(t.Columns) == 0 {
		return ""
	}

	// Calculate column widths
	widths := make([]int, len(t.Columns))
	for i, col := range t.Columns {
		widths[i] = len(col.Title)
		if col.MinWidth > widths[i] {
			widths[i] = col.MinWidth
		}
	}
	for _, row := range t.Rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Clamp to terminal width
	termWidth := 80
	if w, _, err := term.GetSize(0); err == nil && w > 0 {
		termWidth = w
	}

	gap := 2
	totalGaps := gap * (len(widths) - 1)
	totalWidth := totalGaps
	for _, w := range widths {
		totalWidth += w
	}

	// If too wide, shrink the widest shrinkable column (typically "branch")
	if totalWidth > termWidth {
		excess := totalWidth - termWidth
		// Find the widest column that's > 12
		maxIdx := -1
		maxW := 0
		for i, w := range widths {
			if w > maxW && w > 12 {
				maxW = w
				maxIdx = i
			}
		}
		if maxIdx >= 0 {
			widths[maxIdx] -= excess
			if widths[maxIdx] < 12 {
				widths[maxIdx] = 12
			}
		}
	}

	var b strings.Builder
	gapStr := strings.Repeat(" ", gap)

	// Header
	var headerParts []string
	for i, col := range t.Columns {
		headerParts = append(headerParts, HeaderStyle.Render(pad(col.Title, widths[i])))
	}
	b.WriteString(strings.Join(headerParts, gapStr))
	b.WriteString("\n")

	// Separator
	sepStyle := SeparatorStyle
	var sepParts []string
	for _, w := range widths {
		sepParts = append(sepParts, sepStyle.Render(strings.Repeat("─", w)))
	}
	b.WriteString(strings.Join(sepParts, gapStr))
	b.WriteString("\n")

	// Rows
	for _, row := range t.Rows {
		var parts []string
		for i := range t.Columns {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			// Truncate if needed
			if len(cell) > widths[i] {
				cell = cell[:widths[i]-1] + "…"
			}
			// Apply color styling based on content
			styled := styleCell(cell, i, len(t.Columns))
			parts = append(parts, pad(styled, widths[i]))
		}
		b.WriteString(strings.Join(parts, gapStr))
		b.WriteString("\n")
	}

	return b.String()
}

func pad(s string, width int) string {
	// Account for ANSI escape sequences in length calculation
	visible := lipgloss.Width(s)
	if visible >= width {
		return s
	}
	return s + strings.Repeat(" ", width-visible)
}

func styleCell(cell string, colIdx, totalCols int) string {
	// Style based on status indicators
	switch cell {
	case IndicatorClean:
		return SuccessStyle.Render(cell)
	case IndicatorDirty:
		return ErrorStyle.Render(cell)
	case IndicatorSession:
		return SessionStyle.Render(cell)
	case IndicatorNone:
		return DimStyle.Render(cell)
	}

	// Dirty count (e.g. "✗ 3")
	if strings.HasPrefix(cell, IndicatorDirty) {
		return ErrorStyle.Render(cell)
	}

	// Sync arrows
	if strings.Contains(cell, "↑") || strings.Contains(cell, "↓") {
		return WarningStyle.Render(cell)
	}

	// Orphan session labels
	if strings.HasPrefix(cell, "< ") {
		return WarningStyle.Render(cell)
	}

	return cell
}

// RenderStatusTable is a convenience function for status output.
func RenderStatusTable(rows [][]string) string {
	t := &Table{
		Columns: []Column{
			{Title: "REPO", MinWidth: 4},
			{Title: "WORKTREE", MinWidth: 8},
			{Title: "BRANCH", MinWidth: 6},
			{Title: "CLEAN", MinWidth: 5},
			{Title: "SYNC", MinWidth: 4},
			{Title: "SESSION", MinWidth: 7},
		},
		Rows: rows,
	}
	return fmt.Sprint(t.Render())
}
