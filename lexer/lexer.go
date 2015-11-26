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
)

const (
	ErrorClosingSingleQuotationMarkIsMissing = "Missing closing single quotation mark"
	ErrorClosingDoubleQuotationMarkIsMissing = "Missing closing double quotation mark"
)

// TokenType represents a type of recognized token
type TokenType int

const (
	TokenError TokenType = iota
	TokenEOF

	TokenLeftParenthesis
	TokenRightParenthesis
	TokenRegexEqual
	TokenStringEqual
	TokenNot
	TokenAnd
	TokenOr
	TokenSingleQuotedStringLiteral
	TokenDoubleQuotedStringLiteral
	TokenSha1
)

type Token struct {
	Type  TokenType
	Value string
}

type LexFn func(*Lexer) LexFn

type Lexer struct {
	Input    string
	Tokens   []Token
	State    LexFn
	Error    error
	Start    int
	Position int
}

func New(input string) *Lexer {
	return &Lexer{Input: input, State: LexBegin}
}

func (lexer *Lexer) Lex() error {
	for {
		if lexer.State == nil {
			break
		}

		lexer.State = (lexer.State)(lexer)
	}

	return lexer.Error
}

func (lexer *Lexer) Emit(tokenType TokenType) {
	lexer.Tokens = append(lexer.Tokens, Token{Type: tokenType, Value: lexer.Input[lexer.Start:lexer.Position]})
	lexer.Start = lexer.Position
}

func (lexer *Lexer) IncrementPosition() {
	lexer.Position++

	if lexer.Position >= utf8.RuneCountInString(lexer.Input) {
		lexer.Emit(TokenEOF)
	}
}

func (lexer *Lexer) DecrementPosition() {
	if lexer.Position--; lexer.Position < 0 {
		lexer.Position = 0
	}
}

func (lexer *Lexer) RemainingInput() string {
	return lexer.Input[lexer.Position:]
}

func (lexer *Lexer) Read() rune {
	lexer.Position++

	if lexer.Position >= utf8.RuneCountInString(lexer.Input) {
		return eof
	}

	return rune(lexer.Input[lexer.Position])
}

func (lexer *Lexer) EatWhitespaces() {
	for {
		ch := lexer.Read()

		if !unicode.IsSpace(ch) {
			lexer.DecrementPosition()
			break
		}

		if ch == eof {
			lexer.Emit(TokenEOF)
			break
		}
	}
}

func LexLeftParenthesis(lexer *Lexer) LexFn {
	lexer.Start = lexer.Position
	lexer.IncrementPosition()
	lexer.Emit(TokenLeftParenthesis)
	return LexBegin
}

func LexRightParenthesis(lexer *Lexer) LexFn {
	lexer.Start = lexer.Position
	lexer.IncrementPosition()
	lexer.Emit(TokenRightParenthesis)
	return LexBegin
}

func (lexer *Lexer) IsEOF() bool {
	return lexer.Position >= utf8.RuneCountInString(lexer.Input)
}

func (lexer *Lexer) Errorf(err string) LexFn {
	lexer.Error = fmt.Errorf("%s at position: %d", err, lexer.Position)
	return nil
}

func LexSingleQuotedString(lexer *Lexer) LexFn {
	lexer.IncrementPosition()
	lexer.Start = lexer.Position
	for {
		if lexer.IsEOF() {
			lexer.Errorf(ErrorClosingSingleQuotationMarkIsMissing)
			return nil
		}

		switch {
		case strings.HasPrefix(lexer.RemainingInput(), escapedSingleQuotationMark):
			lexer.Position += 2
		case strings.HasPrefix(lexer.RemainingInput(), singleQuotationMark):
			lexer.Emit(TokenSingleQuotedStringLiteral)
			lexer.IncrementPosition()
			return LexBegin
		default:
			lexer.IncrementPosition()
		}
	}
}

func LexDoubleQuotedString(lexer *Lexer) LexFn {
	lexer.IncrementPosition()
	lexer.Start = lexer.Position
	for {
		if lexer.IsEOF() {
			lexer.Errorf(ErrorClosingDoubleQuotationMarkIsMissing)
			return nil
		}

		switch {
		case strings.HasPrefix(lexer.RemainingInput(), escapedDoubleQuotationMark):
			lexer.Position += 2
		case strings.HasPrefix(lexer.RemainingInput(), doubleQuotationMark):
			lexer.Emit(TokenDoubleQuotedStringLiteral)
			lexer.IncrementPosition()
			return LexBegin
		default:
			lexer.IncrementPosition()
		}
	}
}

func LexBegin(lexer *Lexer) LexFn {
	lexer.EatWhitespaces()

	if lexer.IsEOF() {
		return nil
	}

	remainingInput := lexer.RemainingInput()

	switch {
	case strings.HasPrefix(remainingInput, leftParenthesis):
		return LexLeftParenthesis
	case strings.HasPrefix(remainingInput, rightParenthesis):
		return LexRightParenthesis
	case strings.HasPrefix(remainingInput, singleQuotationMark):
		return LexSingleQuotedString
	case strings.HasPrefix(remainingInput, doubleQuotationMark):
		return LexDoubleQuotedString
	default:
		return nil
	}
}
