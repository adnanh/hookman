package parser

import (
	"github.com/adnanh/hookman/lexer"
)

type Parser struct {
	Lexer *lexer.Lexer
}

func New(lexer *Lexer) *Parser {
	return &Parser{Lexer: lexer}
}
