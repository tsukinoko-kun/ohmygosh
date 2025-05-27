package vimtextinput

import (
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tsukinoko-kun/ohmygosh/internal/config"
	"github.com/tsukinoko-kun/ohmygosh/internal/history"
	"github.com/tsukinoko-kun/ohmygosh/internal/prompt"
	"github.com/tsukinoko-kun/ohmygosh/internal/shell"
	"github.com/tsukinoko-kun/ohmygosh/internal/term"
)

// Mode represents the current input mode
type Mode int

const (
	ModeNormal Mode = iota
	ModeInsert
	ModeVisual
)

// Model represents the text input model with Vim motions
type Model struct {
	cursorStyle lipgloss.Style
	promptStyle lipgloss.Style
	textStyle   lipgloss.Style
	recentKeys  []string
	lastUpdate  time.Time
	value       string
	cursor      int
	mode        Mode
	visualStart int
	width       int
	focused     bool
}

// New creates a new vim text input model
func New() Model {
	return Model{
		value:       "",
		cursor:      0,
		mode:        ModeNormal,
		visualStart: 0,
		width:       20,
		focused:     false,
		recentKeys:  nil,
		cursorStyle: lipgloss.NewStyle().Background(lipgloss.Color(config.Get.Ui.CursorColor)).Foreground(lipgloss.Color(config.Get.Ui.CursorColorText)),
		promptStyle: lipgloss.NewStyle(),
		textStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color(config.Get.Ui.TextColor)),
	}
}

// SetValue sets the value of the text input
func (m *Model) SetValue(s string) {
	m.value = s
	if m.cursor >= len(s) {
		m.cursor = len(s) - 1
	}
}

// Value returns the current value
func (m Model) Value() string {
	return m.value
}

// SetCursor sets the cursor position
func (m *Model) SetCursor(pos int) {
	if pos < 0 {
		pos = 0
	}
	if pos > len(m.value) {
		pos = len(m.value)
	}
	m.cursor = pos
}

func (m *Model) Cursor() int {
	return m.cursor
}

// SetWidth sets the width of the input
func (m *Model) SetWidth(w int) {
	m.width = w
}

// SetMode sets the current mode
func (m *Model) SetMode(mode Mode) {
	m.mode = mode
}

// Focus sets the focus state
func (m *Model) Focus() tea.Cmd {
	m.focused = true
	return nil
}

// Blur removes focus
func (m *Model) Blur() {
	m.focused = false
}

// Focused returns whether the input is focused
func (m Model) Focused() bool {
	return m.focused
}

// Update handles key events and returns updated model
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}

	return m, nil
}

// handleKeyMsg processes key messages based on current mode
func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	if len(m.recentKeys) != 0 && time.Since(m.lastUpdate) > 2*time.Second {
		m.recentKeys = nil
	}
	if m.cursor < 0 {
		m.cursor = 0
	} else if m.cursor > len(m.value) {
		m.cursor = len(m.value)
	}

	switch msg.String() {
	case "up":
		newValue := history.Peek()
		if newValue != "" {
			m.SetValue(newValue)
			m.cursor = max(0, len(m.value))
		}
		return m, nil
	case "down":
		m.SetValue(history.PeekReverse())
		m.cursor = max(0, len(m.value))
		if m.value == "" {
			history.SetFilter("")
		}
		return m, nil
	}

	newModel := m
	var cmd tea.Cmd = nil

	prevRecentKeys := len(m.recentKeys)
	prevValue := m.value
	prevCursor := m.cursor
	prevMode := m.mode
	prevVisualStart := m.visualStart

	switch m.mode {
	case ModeNormal:
		newModel, cmd = m.handleNormalMode(msg)
	case ModeInsert:
		newModel, cmd = m.handleInsertMode(msg)
	case ModeVisual:
		newModel, cmd = m.handleVisualMode(msg)
	}
	if prevValue != newModel.value || prevCursor != newModel.cursor || prevMode != newModel.mode || prevVisualStart != newModel.visualStart {
		newModel.recentKeys = nil
	} else if prevRecentKeys != len(newModel.recentKeys) {
		newModel.lastUpdate = time.Now()
	}
	// if prevValue != newModel.value {
	// 	history.SetFilter(newModel.value)
	// }
	return newModel, cmd
}

// handleNormalMode handles keys in normal mode
func (m Model) handleNormalMode(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "h", "left":
		if len(m.recentKeys) == 1 && m.recentKeys[0] == "d" {
			m.deleteCharBefore()
		} else {
			m.moveCursorLeft()
		}
	case "l", "right":
		if len(m.recentKeys) == 1 && m.recentKeys[0] == "d" {
			m.deleteChar()
		} else {
			m.moveCursorRight()
		}
	case "0", "home", "_":
		if len(m.recentKeys) == 1 && m.recentKeys[0] == "d" {
			m.value = m.value[m.cursor:]
		}
		m.cursor = 0
	case "$", "end":
		if len(m.recentKeys) == 1 && m.recentKeys[0] == "d" {
			m.value = m.value[0:m.cursor]
		}
		m.cursor = len(m.value)
	case "w":
		m.moveWordForward()
	case "b":
		m.moveWordBackward()
	case "e":
		m.moveWordEnd()
	case "i":
		m.mode = ModeInsert
	case "a":
		m.mode = ModeInsert
		m.moveCursorRight()
	case "I":
		m.mode = ModeInsert
		m.cursor = 0
	case "A":
		m.mode = ModeInsert
		m.cursor = len(m.value)
	case "x":
		m.deleteChar()
	case "X":
		m.deleteCharBefore()
	case "v":
		m.mode = ModeVisual
		m.visualStart = m.cursor
	case "y":
		_ = clipboard.WriteAll(m.value)
	case "p":
		text, err := clipboard.ReadAll()
		if err == nil {
			m.InsertText(text)
		}
	case "P":
		text, err := clipboard.ReadAll()
		if err == nil {
			m.cursor--
			if m.cursor < 0 {
				m.cursor = 0
			}
			m.InsertText(text)
		}
	case "D":
		m.value = m.value[0:m.cursor]
		m.cursor = len(m.value)
	case "d":
		if len(m.recentKeys) == 1 && m.recentKeys[0] == "d" {
			m.value = ""
			m.cursor = 0
		} else {
			m.recentKeys = append(m.recentKeys, "d")
		}
	case "s":
		if len(m.value) > m.cursor {
			m.value = m.value[:m.cursor] + m.value[m.cursor+1:]
		}
		m.mode = ModeInsert
	case "c":
		if len(m.recentKeys) == 1 && m.recentKeys[0] == "c" {
			m.value = m.value[0:m.cursor]
			m.cursor = len(m.value)
			m.mode = ModeInsert
		} else {
			m.recentKeys = append(m.recentKeys, "c")
		}
	case "C":
		m.value = m.value[0:m.cursor]
		m.cursor = len(m.value)
		m.mode = ModeInsert
	}
	return m, nil
}

// handleInsertMode handles keys in insert mode
func (m Model) handleInsertMode(msg tea.KeyMsg) (Model, tea.Cmd) {
	if msg.Paste {
		cb, _ := clipboard.ReadAll()
		m.InsertText(cb)
		return m, nil
	}

	switch msg.String() {
	case "esc":
		m.mode = ModeNormal
		m.moveCursorLeft()
	case "backspace":
		m.deleteCharBefore()
	case "delete":
		m.deleteChar()
	case "left":
		m.moveCursorLeft()
	case "right":
		m.moveCursorRight()
	case "home":
		m.cursor = 0
	case "end":
		m.cursor = len(m.value)
	default:
		if len(msg.String()) == 1 {
			m.InsertText(msg.String())
		}
	}
	history.SetFilter(m.value)
	return m, nil
}

// handleVisualMode handles keys in visual mode
func (m Model) handleVisualMode(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = ModeNormal
	case "h", "left":
		m.moveCursorLeft()
	case "l", "right":
		m.moveCursorRight()
	case "0", "home":
		m.cursor = 0
	case "$", "end":
		m.cursor = len(m.value)
	case "w":
		m.moveWordForward()
	case "b":
		m.moveWordBackward()
	case "e":
		m.moveWordEnd()
	case "y":
		start, end := m.getVisualSelection()
		if start != end {
			_ = clipboard.WriteAll(m.value[start:end])
		}
		m.mode = ModeNormal
	case "d", "x":
		start, end := m.getVisualSelection()
		if start != end {
			_ = clipboard.WriteAll(m.value[start:end])
			m.value = m.value[:start] + m.value[end:]
			m.cursor = start
		}
		m.mode = ModeNormal
	case "c", "s":
		start, end := m.getVisualSelection()
		if end >= len(m.value) {
			end = len(m.value)
		}
		m.value = m.value[:start] + m.value[end:]
		m.cursor = start
		m.mode = ModeInsert
	}
	return m, nil
}

// Movement helper functions
func (m *Model) moveCursorLeft() {
	if m.cursor > 0 {
		m.cursor--
	}
}

func (m *Model) moveCursorRight() {
	if m.cursor < len(m.value) {
		m.cursor++
	}
}

// moveWordForward implements 'w' - move to beginning of next word
func (m *Model) moveWordForward() {
	if m.cursor >= len(m.value) {
		return
	}

	tokens := tokenize(m.value)
	if len(tokens) == 0 {
		return
	}

	currentTokenIdx := findTokenAtPosition(tokens, m.cursor)
	currentToken := tokens[currentTokenIdx]

	// If we're in the middle of a non-space token, first move to the next token
	if m.cursor >= currentToken.Start && m.cursor < currentToken.End && currentToken.Type != TokenSpace {
		currentTokenIdx++
	}

	// Skip whitespace tokens
	for currentTokenIdx < len(tokens) && tokens[currentTokenIdx].Type == TokenSpace {
		currentTokenIdx++
	}

	// Move to the beginning of the next non-space token
	if currentTokenIdx < len(tokens) {
		m.cursor = tokens[currentTokenIdx].Start
	} else {
		m.cursor = len(m.value)
	}
}

// moveWordBackward implements 'b' - move to beginning of current or previous word
func (m *Model) moveWordBackward() {
	if m.cursor <= 0 {
		return
	}

	tokens := tokenize(m.value)
	if len(tokens) == 0 {
		return
	}

	currentTokenIdx := findTokenAtPosition(tokens, m.cursor-1)
	currentToken := tokens[currentTokenIdx]

	// If we're at the beginning of a non-space token, move to previous token
	if m.cursor == currentToken.Start && currentToken.Type != TokenSpace {
		currentTokenIdx--
	}

	// Skip whitespace tokens backwards
	for currentTokenIdx >= 0 && tokens[currentTokenIdx].Type == TokenSpace {
		currentTokenIdx--
	}

	// Move to the beginning of the found non-space token
	if currentTokenIdx >= 0 {
		m.cursor = tokens[currentTokenIdx].Start
	} else {
		m.cursor = 0
	}
}

// moveWordEnd implements 'e' - move to end of current or next word
func (m *Model) moveWordEnd() {
	if m.cursor >= len(m.value) {
		return
	}

	tokens := tokenize(m.value)
	if len(tokens) == 0 {
		return
	}

	currentTokenIdx := findTokenAtPosition(tokens, m.cursor)
	currentToken := tokens[currentTokenIdx]

	// If we're at the end of a non-space token, move to next token
	if m.cursor == currentToken.End-1 && currentToken.Type != TokenSpace {
		currentTokenIdx++
	} else if currentToken.Type == TokenSpace {
		// If we're in whitespace, find the next non-space token
		for currentTokenIdx < len(tokens) && tokens[currentTokenIdx].Type == TokenSpace {
			currentTokenIdx++
		}
	}

	// Skip whitespace tokens
	for currentTokenIdx < len(tokens) && tokens[currentTokenIdx].Type == TokenSpace {
		currentTokenIdx++
	}

	// Move to the end of the found non-space token
	if currentTokenIdx < len(tokens) {
		m.cursor = tokens[currentTokenIdx].End - 1
	} else {
		m.cursor = len(m.value) - 1
	}

	// Ensure cursor doesn't go beyond text length
	if m.cursor >= len(m.value) {
		m.cursor = len(m.value) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// Text manipulation functions
func (m *Model) InsertText(text string) {
	text = strings.ReplaceAll(text, "\n", " ")
	if m.cursor >= len(m.value)-1 {
		m.value += text
		m.cursor = len(m.value)
	} else if m.cursor <= 0 {
		m.value = text + m.value
		m.cursor += len(text)
	} else {
		m.value = m.value[:m.cursor] + text + m.value[m.cursor:]
		m.cursor += len(text)
	}
}

func (m *Model) deleteChar() {
	if m.cursor < len(m.value) {
		m.value = m.value[:m.cursor] + m.value[m.cursor+1:]
	}
}

func (m *Model) deleteCharBefore() {
	if m.cursor > 0 {
		m.value = m.value[:m.cursor-1] + m.value[m.cursor:]
		m.cursor--
	}
}

// getVisualSelection returns the start and end positions of visual selection
func (m Model) getVisualSelection() (int, int) {
	start, end := m.visualStart, m.cursor
	if start > end {
		start, end = end, start
	}
	return start, end + 1
}

// View renders the text input
func (m Model) View() string {
	var b strings.Builder

	b.WriteString(term.SetSessionTitle(prompt.Boring()))
	b.WriteString(term.CWDReportString(shell.Wd))
	b.WriteString(prompt.Get())
	b.WriteString("\n")

	// Add mode indicator
	switch m.mode {
	case ModeInsert:
		b.WriteString(
			m.promptStyle.Background(lipgloss.Color(config.Get.Ui.InsertColorBg)).
				Foreground(lipgloss.Color(config.Get.Ui.InsertColorFg)).
				Render(" I "))
	case ModeNormal:
		b.WriteString(
			m.promptStyle.Background(lipgloss.Color(config.Get.Ui.NormalColorBg)).
				Foreground(lipgloss.Color(config.Get.Ui.NormalColorFg)).
				Render(" N "))
	case ModeVisual:
		b.WriteString(
			m.promptStyle.Background(lipgloss.Color(config.Get.Ui.VisualColorBg)).
				Foreground(lipgloss.Color(config.Get.Ui.VisualColorFg)).
				Render(" V "))
	}
	b.WriteString(" ")

	if len(m.value) == 0 {
		if m.focused {
			switch m.mode {
			case ModeInsert:
				b.WriteString(m.cursorStyle.Render(" "))
			case ModeNormal, ModeVisual:
				b.WriteString(m.cursorStyle.Render(" "))
			}
		}
		return b.String()
	}

	// Render text with cursor
	for i, char := range m.value {
		charStr := string(char)

		if i == m.cursor && m.focused {
			b.WriteString(m.textStyle.Background(lipgloss.Color(config.Get.Ui.CursorColor)).Foreground(lipgloss.Color("0")).Render(charStr))
		} else {
			if m.mode == ModeVisual && m.focused {
				start, end := m.getVisualSelection()
				if i >= start && i < end {
					b.WriteString(m.textStyle.Background(lipgloss.Color(config.Get.Ui.VisualSelectionBg)).Render(charStr))
				} else {
					b.WriteString(m.textStyle.Render(charStr))
				}
			} else {
				b.WriteString(m.textStyle.Render(charStr))
			}
		}
	}

	// Cursor at end
	if m.cursor == len(m.value) && m.focused {
		b.WriteString(m.cursorStyle.Render(" "))
	}

	return b.String()
}

// Reset clears the input and resets to initial state
func (m *Model) Reset() {
	m.value = ""
	m.cursor = 0
	m.mode = ModeInsert
	m.visualStart = 0
}
