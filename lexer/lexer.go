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
	ErrorClosingParenthesisMissing           = "Missing closing parenthesis"
	ErrorUnexpectedToken                     = "Unexpected token %c"
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
	Input                string
	Tokens               []Token
	State                LexFn
	Error                error
	TokenStart           int
	Position             int
	OpenParenthesisCount int
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
	lexer.Tokens = append(lexer.Tokens, Token{Type: tokenType, Value: lexer.Input[lexer.TokenStart:lexer.Position]})
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

func (lexer *Lexer) EatWhitespaces() rune {
	var ch rune = eof
	for {
		ch = lexer.Read()

		if !unicode.IsSpace(ch) {
			lexer.Position--
			break
		}

		if ch == eof {
			break
		}
	}

	return ch
}

func LexLeftParenthesis(lexer *Lexer) LexFn {
	lexer.TokenStart = lexer.Position
	lexer.Position++
	lexer.Emit(TokenLeftParenthesis)
	lexer.OpenParenthesisCount++

	if lexer.IsEOF() {
		lexer.TokenStart = lexer.Position
		lexer.Emit(TokenEOF)
		return lexer.Errorf(ErrorClosingParenthesisMissing)
	}

	return LexBegin
}

func LexRightParenthesis(lexer *Lexer) LexFn {
	lexer.TokenStart = lexer.Position
	lexer.Position++
	lexer.Emit(TokenRightParenthesis)
	lexer.OpenParenthesisCount--

	if lexer.IsEOF() {
		lexer.TokenStart = lexer.Position
		lexer.Emit(TokenEOF)
		return nil
	}

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
	lexer.Position++
	lexer.TokenStart = lexer.Position
	for {
		if lexer.IsEOF() {
			lexer.TokenStart = lexer.Position
			lexer.Emit(TokenEOF)
			return lexer.Errorf(ErrorClosingSingleQuotationMarkIsMissing)
		}

		switch {
		case strings.HasPrefix(lexer.RemainingInput(), escapedSingleQuotationMark):
			lexer.Position += len(escapedSingleQuotationMark)
		case strings.HasPrefix(lexer.RemainingInput(), singleQuotationMark):
			lexer.Emit(TokenSingleQuotedStringLiteral)
			lexer.Position++
			return LexBegin
		default:
			lexer.Position++
		}
	}
}

func LexDoubleQuotedString(lexer *Lexer) LexFn {
	lexer.Position++
	lexer.TokenStart = lexer.Position
	for {
		if lexer.IsEOF() {
			lexer.TokenStart = lexer.Position
			lexer.Emit(TokenEOF)
			return lexer.Errorf(ErrorClosingDoubleQuotationMarkIsMissing)
		}

		switch {
		case strings.HasPrefix(lexer.RemainingInput(), escapedDoubleQuotationMark):
			lexer.Position += len(escapedDoubleQuotationMark)
		case strings.HasPrefix(lexer.RemainingInput(), doubleQuotationMark):
			lexer.Emit(TokenDoubleQuotedStringLiteral)
			lexer.Position++
			return LexBegin
		default:
			lexer.Position++
		}
	}
}

func LexBegin(lexer *Lexer) LexFn {
	lexer.EatWhitespaces()

	if lexer.IsEOF() {
		lexer.TokenStart = lexer.Position
		lexer.Emit(TokenEOF)

		if lexer.OpenParenthesisCount > 0 {
			return lexer.Errorf(ErrorClosingParenthesisMissing)
		}

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
		return lexer.Errorf(fmt.Sprintf(ErrorUnexpectedToken, remainingInput[0]))
	}
}
