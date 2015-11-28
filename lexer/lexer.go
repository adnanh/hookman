package lexer

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	eof                        rune   = 0
	leftParenthesis            string = "("
	rightParenthesis                  = ")"
	singleQuotationMark               = "'"
	doubleQuotationMark               = "\""
	escapedSingleQuotationMark        = "\\'"
	escapedDoubleQuotationMark        = "\\\""
	comma                             = ","
	regexEqual                        = "~="
	stringEqual                       = "=="
	not                               = "!"
	and                               = "&&"
	or                                = "||"
	sha1                              = "sha1"
)

const (
	errorClosingSingleQuotationMarkIsMissing = "missing closing single quotation mark"
	errorClosingDoubleQuotationMarkIsMissing = "missing closing double quotation mark"
	errorClosingParenthesisMissing           = "missing closing parenthesis"
	errorUnexpectedToken                     = "unexpected token %c"
)

// TokenType represents a type of recognized token
type TokenType int

const (
	// TokenEOF is an End of File token
	TokenEOF TokenType = iota

	// TokenLeftParenthesis is a left parenthesis token
	TokenLeftParenthesis

	// TokenRightParenthesis is a right parenthesis token
	TokenRightParenthesis

	// TokenComma is a comma token
	TokenComma

	// TokenRegexEqual is a regex equal operator token
	TokenRegexEqual

	// TokenStringEqual is a string equal operator token
	TokenStringEqual

	// TokenNot is a negation logical operator token
	TokenNot

	// TokenAnd is an and logical operator token
	TokenAnd

	// TokenOr is an or logical operator token
	TokenOr

	// TokenSingleQuotedStringLiteral is a single quoted string literal token
	TokenSingleQuotedStringLiteral

	// TokenDoubleQuotedStringLiteral is a double quoted string literal token
	TokenDoubleQuotedStringLiteral

	// TokenSha1 is a sha1 function token
	TokenSha1
)

// Token is a structure that contains type of recognized token and it's value if applicable
type Token struct {
	Type  TokenType
	Value string
}

// LexFn is an interface that lexing functions have to implement
type LexFn func(*Lexer) LexFn

// Lexer is a structure that contains current state of the lexer
type Lexer struct {
	Input                string
	Tokens               []Token
	State                LexFn
	Errors               []error
	TokenStart           int
	Position             int
	OpenParenthesisCount int
}

// New returns a new lexer for the given input
func New(input string) *Lexer {
	return &Lexer{Input: input, State: LexBegin}
}

// Lex performs lexing
func (lexer *Lexer) Lex() []error {
	for {
		if lexer.State == nil {
			break
		}

		lexer.State = (lexer.State)(lexer)
	}

	if lexer.OpenParenthesisCount > 0 {
		lexer.Errorf(errorClosingParenthesisMissing)
	}

	return lexer.Errors
}

// IsEOF returns true if the lexer has reached the end of the input
func (lexer *Lexer) IsEOF() bool {
	return lexer.Position >= utf8.RuneCountInString(lexer.Input)
}

// HasTokenTypeAt returns true if the lexer recognized given token type at the given position
func (lexer *Lexer) HasTokenTypeAt(pos int, tokenType TokenType) bool {
	if pos < 0 || pos >= len(lexer.Tokens) {
		return false
	}

	return lexer.Tokens[pos].Type == tokenType
}

// Read returns current rune
func (lexer *Lexer) Read() rune {
	if lexer.IsEOF() {
		return eof
	}

	ch := rune(lexer.Input[lexer.Position])

	lexer.Position++

	return ch
}

// Emit appends given token to the lexer tokens slice
func (lexer *Lexer) Emit(tokenType TokenType) {
	lexer.Tokens = append(lexer.Tokens, Token{Type: tokenType, Value: lexer.Input[lexer.TokenStart:lexer.Position]})
}

// Errorf appends error with the given error message to the list of lexer errors
func (lexer *Lexer) Errorf(err string) LexFn {
	lexer.Errors = append(lexer.Errors, fmt.Errorf("%s at position: %d", err, lexer.Position))
	return nil
}

// RemainingInput returns a string of an unread remainder input string
func (lexer *Lexer) RemainingInput() string {
	return lexer.Input[lexer.Position:]
}

// EatWhitespaces skips all whitespaces
func (lexer *Lexer) EatWhitespaces() rune {
	var ch rune

	for {
		ch = lexer.Read()

		if ch == eof {
			break
		}

		if !unicode.IsSpace(ch) {
			lexer.Position--
			break
		}
	}

	return ch
}

// LexComma emits TokenComma
func LexComma(lexer *Lexer) LexFn {
	lexer.TokenStart = lexer.Position
	lexer.Position += len(comma)
	lexer.Emit(TokenComma)

	if lexer.IsEOF() {
		lexer.TokenStart = lexer.Position
		lexer.Emit(TokenEOF)
		return nil
	}

	return LexBegin
}

// LexLeftParenthesis emits TokenLeftParenthesis
func LexLeftParenthesis(lexer *Lexer) LexFn {
	lexer.TokenStart = lexer.Position
	lexer.Position += len(leftParenthesis)
	lexer.Emit(TokenLeftParenthesis)
	lexer.OpenParenthesisCount++

	if lexer.IsEOF() {
		lexer.TokenStart = lexer.Position
		lexer.Emit(TokenEOF)
		return lexer.Errorf(errorClosingParenthesisMissing)
	}

	return LexBegin
}

// LexRightParenthesis emits TokenRightParenthesis
func LexRightParenthesis(lexer *Lexer) LexFn {
	lexer.TokenStart = lexer.Position
	lexer.Position += len(rightParenthesis)
	lexer.Emit(TokenRightParenthesis)
	lexer.OpenParenthesisCount--

	if lexer.IsEOF() {
		lexer.TokenStart = lexer.Position
		lexer.Emit(TokenEOF)
		return nil
	}

	return LexBegin
}

// LexSingleQuotedString emits TokenSingleQuotedStringLiteral
func LexSingleQuotedString(lexer *Lexer) LexFn {
	lexer.Position += len(singleQuotationMark)
	lexer.TokenStart = lexer.Position
	for {
		if lexer.IsEOF() {
			lexer.TokenStart = lexer.Position
			lexer.Emit(TokenEOF)
			return lexer.Errorf(errorClosingSingleQuotationMarkIsMissing)
		}

		switch {
		case strings.HasPrefix(lexer.RemainingInput(), escapedSingleQuotationMark):
			lexer.Position += len(escapedSingleQuotationMark)
		case strings.HasPrefix(lexer.RemainingInput(), singleQuotationMark):
			lexer.Emit(TokenSingleQuotedStringLiteral)
			lexer.Position += len(singleQuotationMark)
			return LexBegin
		default:
			lexer.Position++
		}
	}
}

// LexDoubleQuotedString emits TokenDoubleQuotedStringLiteral
func LexDoubleQuotedString(lexer *Lexer) LexFn {
	lexer.Position += len(doubleQuotationMark)
	lexer.TokenStart = lexer.Position
	for {
		if lexer.IsEOF() {
			lexer.TokenStart = lexer.Position
			lexer.Emit(TokenEOF)
			return lexer.Errorf(errorClosingDoubleQuotationMarkIsMissing)
		}

		switch {
		case strings.HasPrefix(lexer.RemainingInput(), escapedDoubleQuotationMark):
			lexer.Position += len(escapedDoubleQuotationMark)
		case strings.HasPrefix(lexer.RemainingInput(), doubleQuotationMark):
			lexer.Emit(TokenDoubleQuotedStringLiteral)
			lexer.Position += len(doubleQuotationMark)
			return LexBegin
		default:
			lexer.Position++
		}
	}
}

// LexStringEqual emits TokenStringEqual
func LexStringEqual(lexer *Lexer) LexFn {
	lexer.TokenStart = lexer.Position
	lexer.Position += len(stringEqual)
	lexer.Emit(TokenStringEqual)

	if lexer.IsEOF() {
		lexer.TokenStart = lexer.Position
		lexer.Emit(TokenEOF)
		return nil
	}

	return LexBegin
}

// LexRegexEqual emits TokenRegexEqual
func LexRegexEqual(lexer *Lexer) LexFn {
	lexer.TokenStart = lexer.Position
	lexer.Position += len(regexEqual)
	lexer.Emit(TokenRegexEqual)

	if lexer.IsEOF() {
		lexer.TokenStart = lexer.Position
		lexer.Emit(TokenEOF)
		return nil
	}

	return LexBegin
}

// LexAnd emits TokenAnd
func LexAnd(lexer *Lexer) LexFn {
	lexer.TokenStart = lexer.Position
	lexer.Position += len(and)
	lexer.Emit(TokenAnd)

	if lexer.IsEOF() {
		lexer.TokenStart = lexer.Position
		lexer.Emit(TokenEOF)
		return nil
	}

	return LexBegin
}

// LexOr emits TokenOr
func LexOr(lexer *Lexer) LexFn {
	lexer.TokenStart = lexer.Position
	lexer.Position += len(or)
	lexer.Emit(TokenOr)

	if lexer.IsEOF() {
		lexer.TokenStart = lexer.Position
		lexer.Emit(TokenEOF)
		return nil
	}

	return LexBegin
}

// LexNot emits TokenNot
func LexNot(lexer *Lexer) LexFn {
	lexer.TokenStart = lexer.Position
	lexer.Position += len(not)
	lexer.Emit(TokenNot)

	if lexer.IsEOF() {
		lexer.TokenStart = lexer.Position
		lexer.Emit(TokenEOF)
		return nil
	}

	return LexBegin
}

// LexSha1 emits TokenSha1
func LexSha1(lexer *Lexer) LexFn {
	lexer.TokenStart = lexer.Position
	lexer.Position += len(sha1)
	lexer.Emit(TokenSha1)

	if lexer.IsEOF() {
		lexer.TokenStart = lexer.Position
		lexer.Emit(TokenEOF)
		return nil
	}

	return LexBegin
}

// LexBegin skips all whitespaces and returns a function that can lex the remaining input
func LexBegin(lexer *Lexer) LexFn {
	if lexer.EatWhitespaces(); lexer.IsEOF() {
		lexer.TokenStart = lexer.Position
		lexer.Emit(TokenEOF)
		return nil
	}

	remainingInput := lexer.RemainingInput()

	switch {
	case strings.HasPrefix(remainingInput, leftParenthesis):
		return LexLeftParenthesis
	case strings.HasPrefix(remainingInput, rightParenthesis):
		return LexRightParenthesis
	case strings.HasPrefix(remainingInput, comma):
		return LexComma
	case strings.HasPrefix(remainingInput, singleQuotationMark):
		return LexSingleQuotedString
	case strings.HasPrefix(remainingInput, doubleQuotationMark):
		return LexDoubleQuotedString
	case strings.HasPrefix(remainingInput, not):
		return LexNot
	case strings.HasPrefix(remainingInput, and):
		return LexAnd
	case strings.HasPrefix(remainingInput, or):
		return LexOr
	case strings.HasPrefix(remainingInput, stringEqual):
		return LexStringEqual
	case strings.HasPrefix(remainingInput, regexEqual):
		return LexRegexEqual
	case strings.HasPrefix(strings.ToLower(remainingInput), sha1):
		return LexSha1
	default:
		return lexer.Errorf(fmt.Sprintf(errorUnexpectedToken, remainingInput[0]))
	}
}
