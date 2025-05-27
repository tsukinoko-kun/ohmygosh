package ansicompiler

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/tsukinoko-kun/ohmygosh/internal/term"
)

// Token represents either a rune or an ANSI escape sequence
type Token interface {
	IsToken()
}

// RuneToken represents a single character
type RuneToken struct {
	Rune rune
}

func (RuneToken) IsToken() {}

// AnsiToken represents an ANSI escape sequence
type AnsiToken struct {
	Sequence string
}

func (AnsiToken) IsToken() {}

// Cell represents a character in the buffer with its styling
type Cell struct {
	Rune  rune
	Style string
}

// Buffer represents a 2D grid of cells with cursor tracking
type Buffer struct {
	cells        [][]Cell
	cursorRow    int
	cursorCol    int
	currentStyle string
	softWrap     int
}

// NewBuffer creates a new buffer
func NewBuffer() *Buffer {
	return &Buffer{
		cells:        make([][]Cell, 0),
		cursorRow:    0,
		cursorCol:    0,
		currentStyle: "",
		softWrap:     int(term.Cols),
	}
}

// ensureSize ensures the buffer is large enough for the given position
func (b *Buffer) ensureSize(row, col int) {
	// Expand rows if needed
	for len(b.cells) <= row {
		b.cells = append(b.cells, make([]Cell, 0))
	}

	// Expand columns if needed
	for len(b.cells[row]) <= col {
		b.cells[row] = append(b.cells[row], Cell{Rune: ' ', Style: ""})
	}
}

// setCursor sets the cursor position
func (b *Buffer) setCursor(row, col int) {
	if row < 0 {
		row = 0
	}
	if col < 0 {
		col = 0
	}
	b.cursorRow = row
	b.cursorCol = col
}

// writeRune writes a rune at the current cursor position and advances
func (b *Buffer) writeRune(r rune) {
	b.ensureSize(b.cursorRow, b.cursorCol)
	b.cells[b.cursorRow][b.cursorCol] = Cell{
		Rune:  r,
		Style: b.currentStyle,
	}
	b.cursorCol++
}

// Tokenize splits the input string into tokens
func Tokenize(input string) []Token {
	var tokens []Token
	i := 0
	runes := []rune(input)

	for i < len(runes) {
		if runes[i] == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			// Found start of ANSI sequence
			start := i
			i += 2 // Skip \x1b[

			// Find the end of the sequence (first letter after parameters)
			for i < len(runes) && !isAnsiTerminator(runes[i]) {
				i++
			}
			if i < len(runes) {
				i++ // Include the terminator
			}

			sequence := string(runes[start:i])
			tokens = append(tokens, AnsiToken{Sequence: sequence})
		} else {
			// Regular character
			tokens = append(tokens, RuneToken{Rune: runes[i]})
			i++
		}
	}

	return tokens
}

// isAnsiTerminator checks if a character terminates an ANSI sequence
func isAnsiTerminator(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}

// CompileAnsi processes tokens and returns the final rendered string
func CompileAnsi(input string) string {
	tokens := Tokenize(input)
	buffer := NewBuffer()

	for _, token := range tokens {
		switch t := token.(type) {
		case RuneToken:
			if t.Rune == '\n' {
				buffer.cursorRow++
				buffer.cursorCol = 0
			} else if t.Rune == '\r' {
				buffer.cursorCol = 0
			} else {
				buffer.writeRune(t.Rune)
			}
		case AnsiToken:
			processAnsiSequence(buffer, t.Sequence)
		}
	}

	return renderBuffer(buffer)
}

// processAnsiSequence handles ANSI escape sequences
func processAnsiSequence(buffer *Buffer, sequence string) {
	if !strings.HasPrefix(sequence, "\x1b[") {
		return
	}

	// Remove \x1b[ prefix
	content := sequence[2:]
	if len(content) == 0 {
		return
	}

	// Get the command (last character)
	cmd := content[len(content)-1]
	params := content[:len(content)-1]

	switch cmd {
	case 'H': // Cursor position
		handleCursorPosition(buffer, params)
	case 'A': // Cursor up
		handleCursorUp(buffer, params)
	case 'B': // Cursor down
		handleCursorDown(buffer, params)
	case 'C': // Cursor forward
		handleCursorForward(buffer, params)
	case 'D': // Cursor backward
		handleCursorBackward(buffer, params)
	case 'J': // Erase in display
		handleEraseInDisplay(buffer, params)
	case 'K': // Erase in line
		handleEraseInLine(buffer, params)
	case 'm': // SGR (styling)
		handleStyling(buffer, sequence)
	default:
		// For unknown sequences, check if they might be styling
		if isStylingSequence(sequence) {
			handleStyling(buffer, sequence)
		}
	}
}

// handleCursorPosition processes cursor positioning commands
func handleCursorPosition(buffer *Buffer, params string) {
	if params == "" {
		buffer.setCursor(0, 0)
		return
	}

	parts := strings.Split(params, ";")
	row := 0
	col := 0

	if len(parts) >= 1 {
		if r, err := strconv.Atoi(parts[0]); err == nil && r > 0 {
			row = r - 1 // Convert to 0-based
		}
	}
	if len(parts) >= 2 {
		if c, err := strconv.Atoi(parts[1]); err == nil && c > 0 {
			col = c - 1 // Convert to 0-based
		}
	}

	buffer.setCursor(row, col)
}

// handleCursorUp moves cursor up
func handleCursorUp(buffer *Buffer, params string) {
	n := 1
	if params != "" {
		if parsed, err := strconv.Atoi(params); err == nil {
			n = parsed
		}
	}
	buffer.setCursor(buffer.cursorRow-n, buffer.cursorCol)
}

// handleCursorDown moves cursor down
func handleCursorDown(buffer *Buffer, params string) {
	n := 1
	if params != "" {
		if parsed, err := strconv.Atoi(params); err == nil {
			n = parsed
		}
	}
	buffer.setCursor(buffer.cursorRow+n, buffer.cursorCol)
}

// handleCursorForward moves cursor forward
func handleCursorForward(buffer *Buffer, params string) {
	n := 1
	if params != "" {
		if parsed, err := strconv.Atoi(params); err == nil {
			n = parsed
		}
	}
	buffer.setCursor(buffer.cursorRow, buffer.cursorCol+n)
}

// handleCursorBackward moves cursor backward
func handleCursorBackward(buffer *Buffer, params string) {
	n := 1
	if params != "" {
		if parsed, err := strconv.Atoi(params); err == nil {
			n = parsed
		}
	}
	buffer.setCursor(buffer.cursorRow, buffer.cursorCol-n)
}

// handleEraseInDisplay erases the display
func handleEraseInDisplay(buffer *Buffer, params string) {
	n, err := strconv.Atoi(params)
	if err != nil {
		n = 0
	}
	switch n {
	case 0:
		// Erase from the cursor to the end of the display
		buffer.cells = buffer.cells[:buffer.cursorRow+1]
	case 1:
		// Erase from the start of the display to the cursor
		buffer.cells = buffer.cells[buffer.cursorRow+1:]
	case 2:
		// Erase the entire display
		buffer.cells = make([][]Cell, 0)
	}
}

// handleEraseInLine erases the line
func handleEraseInLine(buffer *Buffer, params string) {
	n, err := strconv.Atoi(params)
	if err != nil {
		n = 0
	}
	switch n {
	case 0:
		// Erase from the cursor to the end of the line
		buffer.cells[buffer.cursorRow] = buffer.cells[buffer.cursorRow][:buffer.cursorCol]
	case 1:
		// Erase from the start of the line to the cursor
		buffer.cells[buffer.cursorRow] = buffer.cells[buffer.cursorRow][buffer.cursorCol:]
	case 2:
		// Erase the entire line
		buffer.cells[buffer.cursorRow] = make([]Cell, buffer.cursorCol)
	}
}

var stylingPattern = regexp.MustCompile(`^\x1b\[[0-9;]*m$`)

// isStylingSequence checks if an ANSI sequence is for styling
func isStylingSequence(sequence string) bool {
	return stylingPattern.MatchString(sequence)
}

// handleStyling processes styling sequences
func handleStyling(buffer *Buffer, sequence string) {
	buffer.currentStyle += sequence
}

// renderBuffer converts the buffer back to a string
func renderBuffer(buffer *Buffer) string {
	var result strings.Builder
	var lastStyle string

	for rowIndex, row := range buffer.cells {
		if rowIndex > 0 {
			result.WriteString("\n")
		}

		// Trim trailing empty cells
		lastNonEmpty := -1
		for i := len(row) - 1; i >= 0; i-- {
			if row[i].Rune != ' ' || row[i].Style != "" {
				lastNonEmpty = i
				break
			}
		}

		for colIndex := 0; colIndex <= lastNonEmpty; colIndex++ {
			cell := row[colIndex]

			if colIndex != 0 && colIndex%buffer.softWrap == 0 {
				result.WriteString("\n")
				lastStyle = ""
			}

			// Apply style changes
			if cell.Style != lastStyle {
				if lastStyle != "" {
					result.WriteString("\x1b[0m") // Reset previous style
				}
				result.WriteString(cell.Style)
				lastStyle = cell.Style
			}

			result.WriteRune(cell.Rune)
		}
	}

	// Reset at the end
	if lastStyle != "" {
		result.WriteString("\x1b[0m")
	}

	return result.String()
}
