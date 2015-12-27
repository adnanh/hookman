package parser

import (
	"fmt"
	"strings"

	"github.com/adnanh/hookman/lexer"
	"github.com/adnanh/webhook/hook"
)

type argumentExpressionType int

const (
	argument argumentExpressionType = iota
	comma
)

const (
	expectedArgument string = "expected argument %s"
	expectedComma    string = "expected , %s"
)

// ArgumentParser is a struct that contains Lexer and the GeneratedArguments
type ArgumentParser struct {
	Lexer              *lexer.Lexer
	Position           int
	GeneratedArguments []hook.Argument
}

// NewArgumentParser returns a new instance of ArgumentParser for given input string
func NewArgumentParser(input string) *ArgumentParser {
	return &ArgumentParser{Lexer: lexer.New(input), GeneratedArguments: make([]hook.Argument, 0)}
}

func (parser *ArgumentParser) hasPrefix(exprType argumentExpressionType) bool {
	switch {
	case exprType == comma:
		return parser.Lexer.HasTokenTypeAt(parser.Position, lexer.TokenComma)
	case exprType == argument:
		return (parser.Lexer.HasTokenTypeAt(parser.Position, lexer.TokenSingleQuotedStringLiteral) ||
			parser.Lexer.HasTokenTypeAt(parser.Position, lexer.TokenDoubleQuotedStringLiteral))
	}
	return false
}

func (parser *ArgumentParser) Error(err string, tokenPos int) error {
	absolutePosition := 0
	token := parser.Lexer.Tokens[tokenPos]
	tokenValue := token.Value

	if token.Type == lexer.TokenEOF {
		tokenValue = "<EOF>"
	}

	for i, t := range parser.Lexer.Tokens {
		if i == tokenPos {
			break
		}

		absolutePosition += len(t.Value)
	}

	return fmt.Errorf(err, fmt.Sprintf("(token: %s, pos: %d)", tokenValue, absolutePosition))
}

func (parser *ArgumentParser) parseArguments() ([]hook.Argument, error) {
	var arguments []hook.Argument
	expectingArgument := true
	done := false

	for {
		switch {
		case parser.hasPrefix(argument):
			if !expectingArgument {
				return nil, parser.Error(expectedComma, parser.Position)
			}

			source := parser.Lexer.Tokens[parser.Position].Value

			result := strings.SplitN(source, ".", 2)

			if len(result) != 2 {
				return nil, parser.Error(fmt.Sprintf(invalidArgumentFormat, "\t"), parser.Position)
			}

			if result[0] != hook.SourceHeader && result[0] != hook.SourcePayload && result[0] != hook.SourceQuery && result[0] != hook.SourceString {
				return nil, parser.Error(fmt.Sprintf(invalidParameterSource, "\t", strings.Join([]string{hook.SourceHeader, hook.SourcePayload, hook.SourceQuery, hook.SourceString}, ", ")), parser.Position)
			}

			if result[1] == "" {
				return nil, parser.Error(fmt.Sprintf(invalidParameterName, "\t"), parser.Position)
			}

			argument := hook.Argument{Source: result[0], Name: result[1]}
			arguments = append(arguments, argument)

			expectingArgument = false

			parser.Position++
		case parser.hasPrefix(comma):
			if expectingArgument {
				return nil, parser.Error(unexpectedToken, parser.Position)
			}

			expectingArgument = true

			parser.Position++
		default:
			if !parser.Lexer.HasTokenTypeAt(parser.Position, lexer.TokenEOF) {
				return nil, parser.Error(unexpectedToken, parser.Position)
			}

			done = true
		}

		if done {
			break
		}
	}

	if expectingArgument {
		return nil, parser.Error(expectedArgument, parser.Position)
	}

	return arguments, nil
}

// Parse performs lexical analysis of the input string and generates arguments based on the lexer output
func (parser *ArgumentParser) Parse() error {
	if errors := parser.Lexer.Lex(); len(errors) > 0 {
		return fmt.Errorf("error while parsing input string:\n\t%s", errors[0])
	}

	arguments, err := parser.parseArguments()

	if err == nil {
		parser.GeneratedArguments = arguments
	}

	return err
}
