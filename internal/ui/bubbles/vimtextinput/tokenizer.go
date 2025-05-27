package vimtextinput

import "unicode"

// Token represents a word token with its position information
type Token struct {
	Start int
	End   int
	Type  TokenType
	Text  string
}

type TokenType int

const (
	TokenWord  TokenType = iota // letters, digits, underscores
	TokenPunct                  // other non-blank characters
	TokenSpace                  // whitespace
)

// tokenize splits the text into tokens similar to how Vim handles words
func tokenize(text string) []Token {
	if len(text) == 0 {
		return nil
	}

	var tokens []Token
	runes := []rune(text)
	i := 0

	for i < len(runes) {
		start := i
		r := runes[i]

		if unicode.IsSpace(r) {
			// Consume all consecutive whitespace
			for i < len(runes) && unicode.IsSpace(runes[i]) {
				i++
			}
			tokens = append(tokens, Token{
				Start: start,
				End:   i,
				Type:  TokenSpace,
				Text:  string(runes[start:i]),
			})
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			// Consume word characters (letters, digits, underscores)
			for i < len(runes) && (unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i]) || runes[i] == '_') {
				i++
			}
			tokens = append(tokens, Token{
				Start: start,
				End:   i,
				Type:  TokenWord,
				Text:  string(runes[start:i]),
			})
		} else {
			// Consume punctuation characters
			for i < len(runes) && !unicode.IsSpace(runes[i]) && !unicode.IsLetter(runes[i]) && !unicode.IsDigit(runes[i]) && runes[i] != '_' {
				i++
			}
			tokens = append(tokens, Token{
				Start: start,
				End:   i,
				Type:  TokenPunct,
				Text:  string(runes[start:i]),
			})
		}
	}

	return tokens
}

// findTokenAtPosition finds the token that contains or is closest to the given position
func findTokenAtPosition(tokens []Token, pos int) int {
	for i, token := range tokens {
		if pos >= token.Start && pos < token.End {
			return i
		}
		if pos < token.Start {
			return max(0, i-1)
		}
	}
	return len(tokens) - 1
}
