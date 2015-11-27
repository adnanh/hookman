package parser

import (
	"fmt"

	"github.com/adnanh/hookman/lexer"
	"github.com/adnanh/webhook/hook"
)

type ExpressionType int

const (
	MatchValue ExpressionType = iota
	MatchRegex
	MatchHashSHA1
	And
	Or
	Not
	ExpressionGroupStart
	ExpressionGroupEnd
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

// (t && (t2 && t3 && (t4) || t5))
//

// New returns a new instance of Parser for given input string
func New(input string) *Parser {
	return &Parser{Lexer: lexer.New(input), GeneratedRule: &hook.Rules{}}
}

func (parser *Parser) HasPrefix(exprType ExpressionType) bool {
	switch {
	case exprType == And:
		return parser.Lexer.HasTokenTypeAt(parser.Position, lexer.TokenAnd)
	case exprType == Or:
		return parser.Lexer.HasTokenTypeAt(parser.Position, lexer.TokenOr)
	case exprType == Not:
		return parser.Lexer.HasTokenTypeAt(parser.Position, lexer.TokenNot) &&
			parser.Lexer.HasTokenTypeAt(parser.Position+1, lexer.TokenLeftParenthesis)
	case exprType == MatchValue:
		return (parser.Lexer.HasTokenTypeAt(parser.Position, lexer.TokenSingleQuotedStringLiteral) ||
			parser.Lexer.HasTokenTypeAt(parser.Position, lexer.TokenDoubleQuotedStringLiteral)) &&
			parser.Lexer.HasTokenTypeAt(parser.Position+1, lexer.TokenStringEqual) &&
			(parser.Lexer.HasTokenTypeAt(parser.Position+2, lexer.TokenSingleQuotedStringLiteral) ||
				parser.Lexer.HasTokenTypeAt(parser.Position+2, lexer.TokenDoubleQuotedStringLiteral))
	case exprType == MatchRegex:
		return (parser.Lexer.HasTokenTypeAt(parser.Position, lexer.TokenSingleQuotedStringLiteral) ||
			parser.Lexer.HasTokenTypeAt(parser.Position, lexer.TokenDoubleQuotedStringLiteral)) &&
			parser.Lexer.HasTokenTypeAt(parser.Position+1, lexer.TokenRegexEqual) &&
			(parser.Lexer.HasTokenTypeAt(parser.Position+2, lexer.TokenSingleQuotedStringLiteral) ||
				parser.Lexer.HasTokenTypeAt(parser.Position+2, lexer.TokenDoubleQuotedStringLiteral))
	case exprType == MatchHashSHA1:
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
	case exprType == ExpressionGroupStart:
		return parser.Lexer.HasTokenTypeAt(parser.Position, lexer.TokenLeftParenthesis)
	case exprType == ExpressionGroupEnd:
		return parser.Lexer.HasTokenTypeAt(parser.Position, lexer.TokenRightParenthesis)
	}

	return false
}

func (parser *Parser) ParseRule() hook.Rules {
	var rule hook.Rules
	var subRules []hook.Rules
	isAnd := false
	isOr := false
	done := false

	for {
		switch {
		case parser.HasPrefix(And):
			if isOr {
				isOr = false
				orRule := append(hook.OrRule{}, subRules...)
				subRules = []hook.Rules{hook.Rules{Or: &orRule}}
			}
			isAnd = true
			parser.Position++
		case parser.HasPrefix(Or):
			if isAnd {
				isAnd = false
				andRule := append(hook.AndRule{}, subRules...)
				subRules = []hook.Rules{hook.Rules{And: &andRule}}
			}
			isOr = true
			parser.Position++
		case parser.HasPrefix(Not):
			parser.Position += 2
			notRule := (hook.NotRule)(parser.ParseRule())
			subRules = append(subRules, hook.Rules{Not: &notRule})
		case parser.HasPrefix(MatchValue):
			subRules = append(subRules, hook.Rules{Match: &hook.MatchRule{Type: hook.MatchValue}})
			parser.Position += 3
		case parser.HasPrefix(MatchRegex):
			subRules = append(subRules, hook.Rules{Match: &hook.MatchRule{Type: hook.MatchRegex}})
			parser.Position += 3
		case parser.HasPrefix(MatchHashSHA1):
			subRules = append(subRules, hook.Rules{Match: &hook.MatchRule{Type: hook.MatchHashSHA1}})
			parser.Position += 6
		case parser.HasPrefix(ExpressionGroupStart):
			parser.Position++
			subRules = append(subRules, parser.ParseRule())
		case parser.HasPrefix(ExpressionGroupEnd):
			parser.Position++
			done = true
		default:
			done = true
		}

		if done {
			break
		}
	}

	if isAnd {
		rule.And = (*hook.AndRule)(&subRules)
	} else if isOr {
		rule.Or = (*hook.OrRule)(&subRules)
	} else {
		return subRules[0]
	}
	return rule
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
		return fmt.Errorf("Error while parsing input string: %s", errors[0])
	}

	rule := parser.ParseRule()

	printRule(&rule, 0)
	return nil
}
