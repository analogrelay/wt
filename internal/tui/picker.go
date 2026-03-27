package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sahilm/fuzzy"
)

// PickerItem represents an item in the fuzzy picker.
type PickerItem struct {
	Text       string
	Annotation string
	Value      interface{}
}

// PickerModel is a bubbletea model for a fuzzy-filtering picker.
type PickerModel struct {
	items      []PickerItem
	filtered   []PickerItem
	matches    []fuzzy.Match
	cursor     int
	textInput  textinput.Model
	prompt     string
	selected   *PickerItem
	quitting   bool
	height     int // max visible list items
	termHeight int // full terminal height for bottom-anchoring

	// SyntheticEntry generates an additional picker entry based on the current
	// query text. Called when the query is non-empty and no filtered item's Text
	// exactly matches it. The returned item is appended to the filtered list.
	SyntheticEntry func(query string) *PickerItem
}

type pickerStrings []PickerItem

func (p pickerStrings) String(i int) string { return p[i].Text }
func (p pickerStrings) Len() int            { return len(p) }

// NewPicker creates a new fuzzy picker with the given items.
func NewPicker(items []PickerItem, prompt string, initialQuery string) PickerModel {
	ti := textinput.New()
	ti.Prompt = prompt
	ti.PromptStyle = PromptStyle
	ti.TextStyle = NormalStyle
	ti.Cursor.Style = CursorStyle
	ti.Focus()
	if initialQuery != "" {
		ti.SetValue(initialQuery)
	}

	m := PickerModel{
		items:     items,
		textInput: ti,
		prompt:    prompt,
		height:    20,
	}
	m.updateFilter()

	// Auto-select if initial query results in exactly one match
	if initialQuery != "" && len(m.filtered) == 1 {
		m.selected = &m.filtered[0]
		m.quitting = true
	}

	return m
}

func (m PickerModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m PickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termHeight = msg.Height
		m.height = msg.Height - 4 // leave room for input, count line, and margins
		if m.height < 5 {
			m.height = 5
		}
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.quitting = true
			return m, tea.Quit
		case tea.KeyEnter:
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				sel := m.filtered[m.cursor]
				m.selected = &sel
			}
			m.quitting = true
			return m, tea.Quit
		// Bottom-up: Up moves away from prompt (higher index), Down moves toward prompt (lower index)
		case tea.KeyUp, tea.KeyCtrlP:
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil
		case tea.KeyDown, tea.KeyCtrlN:
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	prevValue := m.textInput.Value()
	m.textInput, cmd = m.textInput.Update(msg)
	if m.textInput.Value() != prevValue {
		m.updateFilter()
	}

	return m, cmd
}

func (m *PickerModel) updateFilter() {
	query := m.textInput.Value()
	if query == "" {
		m.filtered = m.items
		m.matches = nil
	} else {
		results := fuzzy.FindFrom(query, pickerStrings(m.items))
		m.filtered = make([]PickerItem, len(results))
		m.matches = results
		for i, r := range results {
			m.filtered[i] = m.items[r.Index]
		}
	}

	// Append synthetic entry when query doesn't exactly match any result
	if m.SyntheticEntry != nil && query != "" {
		hasExact := false
		for _, item := range m.filtered {
			if strings.EqualFold(item.Text, query) {
				hasExact = true
				break
			}
		}
		if !hasExact {
			if entry := m.SyntheticEntry(query); entry != nil {
				m.filtered = append(m.filtered, *entry)
			}
		}
	}

	m.cursor = 0
}

func (m PickerModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	visible := m.height
	if visible > len(m.filtered) {
		visible = len(m.filtered)
	}

	// Compute scroll window
	start := 0
	if m.cursor >= visible {
		start = m.cursor - visible + 1
	}
	end := start + visible
	if end > len(m.filtered) {
		end = len(m.filtered)
		start = end - visible
		if start < 0 {
			start = 0
		}
	}

	// Count content lines: list items + count line + input line
	contentLines := 2 // count line + input line
	if len(m.filtered) == 0 {
		contentLines++ // "No matches" line
	} else {
		contentLines += end - start
	}

	// Pad top with empty lines to anchor content at bottom of viewport
	if m.termHeight > contentLines {
		b.WriteString(strings.Repeat("\n", m.termHeight-contentLines))
	}

	if len(m.filtered) == 0 {
		b.WriteString(DimStyle.Render("  No matches") + "\n")
	} else {
		// Bottom-up: render items in reverse so index 0 (best match) is closest to the input
		for i := end - 1; i >= start; i-- {
			item := m.filtered[i]
			cursor := "  "
			style := NormalStyle
			if i == m.cursor {
				cursor = SelectedStyle.Render("▸ ")
				style = SelectedStyle
			}

			line := style.Render(item.Text)
			if item.Annotation != "" {
				line += " " + AnnotationStyle.Render(item.Annotation)
			}
			b.WriteString(cursor + line + "\n")
		}
	}

	b.WriteString(DimStyle.Render(fmt.Sprintf("  %d/%d", len(m.filtered), len(m.items))))
	b.WriteString("\n")
	b.WriteString(m.textInput.View())

	return b.String()
}

// Selected returns the selected item, or nil if cancelled.
func (m PickerModel) Selected() *PickerItem {
	return m.selected
}

// RunPicker runs a picker and returns the selected item.
func RunPicker(items []PickerItem, prompt string, initialQuery string) (*PickerItem, error) {
	m := NewPicker(items, prompt, initialQuery)
	return RunPickerModel(m)
}

// RunPickerModel runs a pre-configured picker model and returns the selected item.
func RunPickerModel(m PickerModel) (*PickerItem, error) {
	// If auto-selected (single match with initial query), return immediately
	if m.selected != nil {
		return m.selected, nil
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	result := finalModel.(PickerModel)
	return result.Selected(), nil
}
