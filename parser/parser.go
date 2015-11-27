package parser

import (
	"fmt"

	"github.com/adnanh/hookman/lexer"
	"github.com/adnanh/webhook/hook"
)

// Parser is a struct that contains Lexer and the GeneratedRule
type Parser struct {
	Lexer         *lexer.Lexer
	GeneratedRule *hook.Rules
}

// New returns a new instance of Parser for given input string
func New(input string) *Parser {
	return &Parser{Lexer: lexer.New(input), GeneratedRule: nil}
}

// Parse performs lexical analysis of the input string and generates rules based on the lexer output
func (parser *Parser) Parse() error {
	if errors := parser.Lexer.Lex(); len(errors) > 0 {
		return fmt.Errorf("Error while parsing input string: %s", errors[0])
	}

	return nil
}
