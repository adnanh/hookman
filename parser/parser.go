package parser

import (
	"fmt"
	"strings"

	"github.com/adnanh/hookman/lexer"
	"github.com/adnanh/webhook/hook"
)

type expressionType int

const (
	matchValue expressionType = iota
	matchRegex
	matchHashSHA1
	and
	or
	not
	expressionGroupStart
	expressionGroupEnd
)

const (
	unexpectedToken             string = "unexpected token %s"
	invalidRule                        = "invalid rule %s"
	expectedValidRule                  = "expected valid rule %s"
	errorParsingExpressionGroup        = "error parsing expression group %%s\n%s%s"
	errorParsingNotRule                = "error parsing not rule %%s\n%s%s"
	invalidArgumentFormat              = "syntax error %%s\n%sargument literal must be in format: paramsource.param.name.path"
	invalidParameterSource             = "syntax error %%s\n%sparameter source must be one of [%s]"
	invalidParameterName               = "syntax error %%s\n%sparameter name cannot be blank"
	invalidSha1Target                  = "syntax error %%s\n%ssha1 target must be payload"
)

// Parser is a struct that contains Lexer and the GeneratedRule
type Parser struct {
	Lexer         *lexer.Lexer
	Position      int
	GeneratedRule *hook.Rules
}

type expression struct {
	tokens         []lexer.Token
	subExpressions []expression
}

// New returns a new instance of Parser for given input string
func New(input string) *Parser {
	return &Parser{Lexer: lexer.New(input), GeneratedRule: &hook.Rules{}}
}

func (parser *Parser) hasPrefix(exprType expressionType) bool {
	switch {
	case exprType == and:
		return parser.Lexer.HasTokenTypeAt(parser.Position, lexer.TokenAnd)
	case exprType == or:
		return parser.Lexer.HasTokenTypeAt(parser.Position, lexer.TokenOr)
	case exprType == not:
		return parser.Lexer.HasTokenTypeAt(parser.Position, lexer.TokenNot) &&
			parser.Lexer.HasTokenTypeAt(parser.Position+1, lexer.TokenLeftParenthesis)
	case exprType == matchValue:
		return (parser.Lexer.HasTokenTypeAt(parser.Position, lexer.TokenSingleQuotedStringLiteral) ||
			parser.Lexer.HasTokenTypeAt(parser.Position, lexer.TokenDoubleQuotedStringLiteral)) &&
			parser.Lexer.HasTokenTypeAt(parser.Position+1, lexer.TokenStringEqual) &&
			(parser.Lexer.HasTokenTypeAt(parser.Position+2, lexer.TokenSingleQuotedStringLiteral) ||
				parser.Lexer.HasTokenTypeAt(parser.Position+2, lexer.TokenDoubleQuotedStringLiteral))
	case exprType == matchRegex:
		return (parser.Lexer.HasTokenTypeAt(parser.Position, lexer.TokenSingleQuotedStringLiteral) ||
			parser.Lexer.HasTokenTypeAt(parser.Position, lexer.TokenDoubleQuotedStringLiteral)) &&
			parser.Lexer.HasTokenTypeAt(parser.Position+1, lexer.TokenRegexEqual) &&
			(parser.Lexer.HasTokenTypeAt(parser.Position+2, lexer.TokenSingleQuotedStringLiteral) ||
				parser.Lexer.HasTokenTypeAt(parser.Position+2, lexer.TokenDoubleQuotedStringLiteral))
	case exprType == matchHashSHA1:
		return (parser.Lexer.HasTokenTypeAt(parser.Position, lexer.TokenSingleQuotedStringLiteral) ||
			parser.Lexer.HasTokenTypeAt(parser.Position, lexer.TokenDoubleQuotedStringLiteral)) &&
			parser.Lexer.HasTokenTypeAt(parser.Position+1, lexer.TokenStringEqual) &&
			parser.Lexer.HasTokenTypeAt(parser.Position+2, lexer.TokenSha1) &&
			parser.Lexer.HasTokenTypeAt(parser.Position+3, lexer.TokenLeftParenthesis) &&
			(parser.Lexer.HasTokenTypeAt(parser.Position+4, lexer.TokenSingleQuotedStringLiteral) ||
				parser.Lexer.HasTokenTypeAt(parser.Position+4, lexer.TokenDoubleQuotedStringLiteral)) &&
			parser.Lexer.HasTokenTypeAt(parser.Position+5, lexer.TokenComma) &&
			(parser.Lexer.HasTokenTypeAt(parser.Position+6, lexer.TokenSingleQuotedStringLiteral) ||
				parser.Lexer.HasTokenTypeAt(parser.Position+6, lexer.TokenDoubleQuotedStringLiteral)) &&
			parser.Lexer.HasTokenTypeAt(parser.Position+7, lexer.TokenRightParenthesis)
	case exprType == expressionGroupStart:
		return parser.Lexer.HasTokenTypeAt(parser.Position, lexer.TokenLeftParenthesis)
	case exprType == expressionGroupEnd:
		return parser.Lexer.HasTokenTypeAt(parser.Position, lexer.TokenRightParenthesis)
	}

	return false
}

func (parser *Parser) Error(err string, tokenPos int) error {
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

func (parser *Parser) parseRule(depth int) (*hook.Rules, error) {
	var err error
	var rule hook.Rules
	var subRules []hook.Rules

	isAnd := false
	isOr := false
	done := false

	expectingRule := false

	for {
		switch {
		case parser.hasPrefix(and):
			if len(subRules) == 0 || expectingRule {
				return nil, parser.Error(unexpectedToken, parser.Position)
			}

			if isOr {
				isOr = false
				orRule := append(hook.OrRule{}, subRules...)
				subRules = []hook.Rules{hook.Rules{Or: &orRule}}
			}

			isAnd = true
			expectingRule = true
			parser.Position++
		case parser.hasPrefix(or):
			if len(subRules) == 0 || expectingRule {
				return nil, parser.Error(unexpectedToken, parser.Position)
			}

			if isAnd {
				isAnd = false
				andRule := append(hook.AndRule{}, subRules...)
				subRules = []hook.Rules{hook.Rules{And: &andRule}}
			}

			isOr = true
			expectingRule = true
			parser.Position++
		case parser.hasPrefix(not):
			parser.Position += 2

			notRule, err := parser.parseRule(depth + 1)

			if err != nil {
				return nil, parser.Error(fmt.Sprintf(errorParsingNotRule, strings.Repeat("\t", depth), err), parser.Position-2)
			}

			subRules = append(subRules, hook.Rules{Not: (*hook.NotRule)(notRule)})
			expectingRule = false
		case parser.hasPrefix(matchValue):
			source := parser.Lexer.Tokens[parser.Position].Value
			value := parser.Lexer.Tokens[parser.Position+2].Value

			result := strings.SplitN(source, ".", 2)

			if len(result) != 2 {
				return nil, parser.Error(fmt.Sprintf(invalidArgumentFormat, strings.Repeat("\t", depth)), parser.Position)
			}

			if result[0] != hook.SourceHeader && result[0] != hook.SourcePayload && result[0] != hook.SourceQuery && result[0] != hook.SourceString {
				return nil, parser.Error(fmt.Sprintf(invalidParameterSource, strings.Repeat("\t", depth), strings.Join([]string{hook.SourceHeader, hook.SourcePayload, hook.SourceQuery, hook.SourceString}, ", ")), parser.Position)
			}

			if result[1] == "" {
				return nil, parser.Error(fmt.Sprintf(invalidParameterName, strings.Repeat("\t", depth)), parser.Position)
			}

			argument := hook.Argument{Source: result[0], Name: result[1]}

			matchRule := &hook.MatchRule{Type: hook.MatchValue, Value: value, Parameter: argument}

			subRules = append(subRules, hook.Rules{Match: matchRule})
			expectingRule = false
			parser.Position += 3
		case parser.hasPrefix(matchRegex):
			source := parser.Lexer.Tokens[parser.Position].Value
			value := parser.Lexer.Tokens[parser.Position+2].Value

			result := strings.SplitN(source, ".", 2)

			if len(result) != 2 {
				return nil, parser.Error(fmt.Sprintf(invalidArgumentFormat, strings.Repeat("\t", depth)), parser.Position)
			}

			if result[0] != hook.SourceHeader && result[0] != hook.SourcePayload && result[0] != hook.SourceQuery && result[0] != hook.SourceString {
				return nil, parser.Error(fmt.Sprintf(invalidParameterSource, strings.Repeat("\t", depth), strings.Join([]string{hook.SourceHeader, hook.SourcePayload, hook.SourceQuery, hook.SourceString}, ", ")), parser.Position)
			}

			if result[1] == "" {
				return nil, parser.Error(fmt.Sprintf(invalidParameterName, strings.Repeat("\t", depth)), parser.Position)
			}

			argument := hook.Argument{Source: result[0], Name: result[1]}

			matchRule := &hook.MatchRule{Type: hook.MatchRegex, Regex: value, Parameter: argument}

			subRules = append(subRules, hook.Rules{Match: matchRule})

			expectingRule = false
			parser.Position += 3
		case parser.hasPrefix(matchHashSHA1):
			source := parser.Lexer.Tokens[parser.Position].Value
			target := parser.Lexer.Tokens[parser.Position+4].Value
			secret := parser.Lexer.Tokens[parser.Position+6].Value

			if target != hook.SourcePayload {
				return nil, parser.Error(fmt.Sprintf(invalidSha1Target, strings.Repeat("\t", depth)), parser.Position+4)
			}

			result := strings.SplitN(source, ".", 2)

			if len(result) != 2 {
				return nil, parser.Error(fmt.Sprintf(invalidArgumentFormat, strings.Repeat("\t", depth)), parser.Position)
			}

			if result[0] != hook.SourceHeader && result[0] != hook.SourcePayload && result[0] != hook.SourceQuery && result[0] != hook.SourceString {
				return nil, parser.Error(fmt.Sprintf(invalidParameterSource, strings.Repeat("\t", depth), strings.Join([]string{hook.SourceHeader, hook.SourcePayload, hook.SourceQuery, hook.SourceString}, ", ")), parser.Position)
			}

			if result[1] == "" {
				return nil, parser.Error(fmt.Sprintf(invalidParameterName, strings.Repeat("\t", depth)), parser.Position)
			}

			argument := hook.Argument{Source: result[0], Name: result[1]}

			matchRule := &hook.MatchRule{Type: hook.MatchHashSHA1, Secret: secret, Parameter: argument}

			subRules = append(subRules, hook.Rules{Match: matchRule})

			expectingRule = false
			parser.Position += 8
		case parser.hasPrefix(expressionGroupStart):
			parser.Position++

			expression, err := parser.parseRule(depth + 1)

			if err != nil {
				return nil, parser.Error(fmt.Sprintf(errorParsingExpressionGroup, strings.Repeat("\t", depth), err), parser.Position-1)
			}

			subRules = append(subRules, *expression)
			expectingRule = false
		case parser.hasPrefix(expressionGroupEnd):
			parser.Position++
			done = true
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

	if len(subRules) == 0 {
		return nil, parser.Error(invalidRule, parser.Position)
	}

	if expectingRule {
		return nil, parser.Error(expectedValidRule, parser.Position)
	}

	if isAnd {
		if len(subRules) == 1 {
			return nil, parser.Error(unexpectedToken, parser.Position-1)
		}
		rule.And = (*hook.AndRule)(&subRules)
	} else if isOr {
		if len(subRules) == 1 {
			return nil, parser.Error(unexpectedToken, parser.Position-1)
		}
		rule.Or = (*hook.OrRule)(&subRules)
	} else {
		return &subRules[0], err
	}

	return &rule, err
}

func printRules(r *[]hook.Rules, indent int) {
	for _, rule := range *r {
		printRule(&rule, indent)
	}
}

func printRule(r *hook.Rules, indent int) {
	for i := 0; i < indent; i++ {
		fmt.Printf(" ")
	}
	switch {
	case r.And != nil:
		fmt.Printf("And:\n")
		printRules((*[]hook.Rules)(r.And), indent+1)
	case r.Or != nil:
		fmt.Printf("Or:\n")
		printRules((*[]hook.Rules)(r.Or), indent+1)
	case r.Not != nil:
		fmt.Printf("Not:\n")
		printRule((*hook.Rules)(r.Not), indent+1)
	case r.Match != nil:
		fmt.Printf("Match: %+v\n", r.Match)
	}
}

// Parse performs lexical analysis of the input string and generates rules based on the lexer output
func (parser *Parser) Parse() error {
	if errors := parser.Lexer.Lex(); len(errors) > 0 {
		return fmt.Errorf("error while parsing input string:\n\t%s", errors[0])
	}

	rule, err := parser.parseRule(1)

	if err == nil {
		parser.GeneratedRule = rule
	}

	return err
}
