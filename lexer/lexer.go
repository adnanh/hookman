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
	ErrorClosingSingleQuotationMarkIsMissing = "Missing closing single quotation mark"
	ErrorClosingDoubleQuotationMarkIsMissing = "Missing closing double quotation mark"
	ErrorClosingParenthesisMissing           = "Missing closing parenthesis"
	ErrorUnexpectedToken                     = "Unexpected token %c"
)

// TokenType represents a type of recognized token
type TokenType int

const (
	TokenEOF TokenType = iota

	TokenLeftParenthesis
	TokenRightParenthesis
	TokenComma
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
	Errors               []error
	TokenStart           int
	Position             int
	OpenParenthesisCount int
}

func New(input string) *Lexer {
	return &Lexer{Input: input, State: LexBegin}
}

func (lexer *Lexer) Lex() []error {
	for {
		if lexer.State == nil {
			break
		}

		lexer.State = (lexer.State)(lexer)
	}

	if lexer.OpenParenthesisCount > 0 {
		lexer.Errorf(ErrorClosingParenthesisMissing)
	}

	return lexer.Errors
}

func (lexer *Lexer) Emit(tokenType TokenType) {
	lexer.Tokens = append(lexer.Tokens, Token{Type: tokenType, Value: lexer.Input[lexer.TokenStart:lexer.Position]})
}

func (lexer *Lexer) RemainingInput() string {
	return lexer.Input[lexer.Position:]
}

func (lexer *Lexer) Read() rune {
	if lexer.Position++; lexer.Position >= utf8.RuneCountInString(lexer.Input) {
		return eof
	}

	return rune(lexer.Input[lexer.Position-1])
}

func (lexer *Lexer) EatWhitespaces() rune {
	var ch rune

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

func LexLeftParenthesis(lexer *Lexer) LexFn {
	lexer.TokenStart = lexer.Position
	lexer.Position += len(leftParenthesis)
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

func (lexer *Lexer) IsEOF() bool {
	return lexer.Position >= utf8.RuneCountInString(lexer.Input)
}

func (lexer *Lexer) Errorf(err string) LexFn {
	lexer.Errors = append(lexer.Errors, fmt.Errorf("%s at position: %d", err, lexer.Position))
	return nil
}

func LexSingleQuotedString(lexer *Lexer) LexFn {
	lexer.Position += len(singleQuotationMark)
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
			lexer.Position += len(singleQuotationMark)
			return LexBegin
		default:
			lexer.Position++
		}
	}
}

func LexDoubleQuotedString(lexer *Lexer) LexFn {
	lexer.Position += len(doubleQuotationMark)
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
			lexer.Position += len(doubleQuotationMark)
			return LexBegin
		default:
			lexer.Position++
		}
	}
}

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

func LexBegin(lexer *Lexer) LexFn {
	lexer.EatWhitespaces()

	if lexer.IsEOF() {
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
	case strings.HasPrefix(remainingInput, stringEqual):
		return LexStringEqual
	case strings.HasPrefix(remainingInput, regexEqual):
		return LexRegexEqual
	case strings.HasPrefix(remainingInput, and):
		return LexAnd
	case strings.HasPrefix(remainingInput, or):
		return LexOr
	case strings.HasPrefix(remainingInput, not):
		return LexNot
	case strings.HasPrefix(strings.ToLower(remainingInput), sha1):
		return LexSha1
	default:
		return lexer.Errorf(fmt.Sprintf(ErrorUnexpectedToken, remainingInput[0]))
	}
}
