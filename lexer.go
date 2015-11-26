package lexer

import (
	"math"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	eof                 rune   = 0
	leftParenthesis     string = "("
	rightParenthesis           = ")"
	singleQuotationMark        = "'"
	doubleQuotationMark        = "\""
	comma                      = ","
	regexEqual                 = "~="
	stringEqual                = "=="
	not                        = "!"
	and                        = "&&"
	or                         = "||"
)

// TokenType represents a type of recognized token
type TokenType int

const (
	TokenError TokenType = iota
	TokenEof

	TokenLeftParenthesis
	TokenRightParenthesis
	TokenRegexEqual
	TokenStringEqual
	TokenNot
	TokenAnd
	TokenOr
	TokenStringLiteral
	TokenSha1
)

type Token struct {
	Type  TokenType
	Value string
}

type LexFn func(*Lexer) LexFn

type Lexer struct {
	Input    string
	Tokens   chan Token
	State    LexFn
	Start    int
	Position int
	Size     int
}

func (lexer *Lexer) Emit(tokenType TokenType) {
	lexer.Tokens <- Token{Type: tokenType, Value: lexer.Input[lexer.Start:lexer.Position]}
	lexer.Start = lexer.Position
}

func (lexer *Lexer) IncrementPosition() {
	lexer.Position++

	if lexer.Position >= utf8.RuneCountInString(lexer.Input) {
		lexer.Emit(TokenEof)
	}
}

func (lexer *Lexer) DecrementPosition() {
	lexer.Position = math.Max(lexer.Position-1, 0)
}

func (lexer *Lexer) RemainingInput() string {
	return lexer.Input[lexer.Position:]
}

func (lexer *Lexer) EatWhitespaces() {
	for {
		ch := lexer.Read()

		if !unicode.IsSpace(ch) {
			lexer.DecrementPosition()
			break
		}

		if ch == eof {
			lexer.Emit(TokenEof)
			break
		}
	}

}

func LexGroup(lexer *Lexer) LexFn {

}

func LexLeftParenthesis(lexer *Lexer) LexFn {
	lexer.IncrementPosition()
	lexer.Emit(TokenLeftParenthesis)
	return LexGroup
}

func LexBegin(lexer *Lexer) LexFn {
	lexer.EatWhitespaces()

	switch {
	case strings.HasPrefix(lexer.RemainingInput(), leftParenthesis):
		return LexLeftParenthesis
	}
}
