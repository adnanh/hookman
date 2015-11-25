package rule

import (
	"fmt"
	"strings"

	"github.com/adnanh/webhook/hook"
)

type List []string

func (o *List) last() string {
	if len(*o) == 0 {
		return ""
	}

	return (*o)[len(*o)-1]
}

func (o *List) pop() string {
	if len(*o) == 0 {
		return ""
	}

	var last string

	last, (*o) = (*o)[len(*o)-1], (*o)[:len(*o)-1]
	return last
}

type parserState struct {
	inLiteral                   bool
	literalPosition             int
	currentPosition             int
	shouldEscape                bool
	unclosedParenthesisCount    int
	literalCountBeforeMatchRule int
	literals                    List
	rules                       List
	openRules                   List
}

// ParserResult is a struct containing the list of rules and literals
type ParserResult struct {
	rules    List
	literals List
}

func closeRule(currentState *parserState) error {
	var err error

	switch {
	case currentState.openRules.last() == "matchValue":
		fallthrough
	case currentState.openRules.last() == "matchHashSHA1":
		fallthrough
	case currentState.openRules.last() == "matchRegex":
		if len(currentState.literals)-currentState.literalCountBeforeMatchRule != 2 {
			err = fmt.Errorf("argument count mismatch for %s rule at position %d", currentState.openRules.last(), currentState.currentPosition+1)
			return err
		}

		splitResult := strings.SplitN(currentState.literals[len(currentState.literals)-2], ".", 2)
		if len(splitResult) < 2 {
			err = fmt.Errorf("invalid parameter format for %s rule at position %d", currentState.openRules.last(), currentState.currentPosition+1)
			return err
		}

		if splitResult[0] != hook.SourceHeader && splitResult[0] != hook.SourcePayload && splitResult[0] != hook.SourceQuery {
			err = fmt.Errorf("invalid parameter source for %s rule at position %d", currentState.openRules.last(), currentState.currentPosition+1)
			return err
		}

		fallthrough
	case currentState.openRules.last() == "not":
		fallthrough
	case currentState.openRules.last() == "and":
		fallthrough
	case currentState.openRules.last() == "or":
		currentState.rules = append(currentState.rules, currentState.openRules.pop())
	}

	return err
}

// ParseString parses given string and returns a ParserResult
func ParseString(s string) (*ParserResult, error) {
	var currentState parserState

	reader := strings.NewReader(s)
	var err error
	for {
		var tok byte
		tok, err = reader.ReadByte()

		if err != nil {
			switch {
			case currentState.inLiteral:
				err = fmt.Errorf("unterminated string literal at position %d", currentState.literalPosition+1)
			case currentState.unclosedParenthesisCount > 0:
				err = fmt.Errorf("missing closing parenthesis at position %d", currentState.literalPosition+1)
			case len(currentState.openRules) == 1:
				err = closeRule(&currentState)
			case len(currentState.openRules) >= 1:
				err = fmt.Errorf("unterminated rule %s at position %d", currentState.openRules.last(), currentState.currentPosition+1)
			default:
				err = nil
			}
			break
		}

		switch {
		case tok == '\\':
			if currentState.inLiteral {
				if currentState.shouldEscape {
					currentState.shouldEscape = false
				} else {
					currentState.shouldEscape = true
				}
			} else {
				err = fmt.Errorf("unexpected escape character '\\' at position %d", currentState.currentPosition+1)
				break
			}
		case tok == '"':
			if currentState.inLiteral {
				if !currentState.shouldEscape {
					// close the literal, emit the string
					currentState.inLiteral = false
					currentState.literals = append(currentState.literals, strings.Replace(strings.Replace(s[currentState.literalPosition+1:currentState.currentPosition], "\\\"", "\"", -1), "\\\\", "\\", -1))
				}
			} else {
				// start the literal
				currentState.inLiteral = true
				currentState.literalPosition = currentState.currentPosition
			}

		case tok == '(':
			if !currentState.inLiteral {
				currentState.unclosedParenthesisCount++
			}
		case tok == ')':
			if !currentState.inLiteral {
				if currentState.unclosedParenthesisCount == 0 {
					err = fmt.Errorf("unexpected closing parenthesis ')' at position %d", currentState.currentPosition+1)
					break
				}
				currentState.unclosedParenthesisCount--
				err = closeRule(&currentState)
			}
		case tok == '!':
			if !currentState.inLiteral {
				currentState.openRules = append(currentState.openRules, "not")
			}
		case tok == 'o':
			if !currentState.inLiteral {
				r := s[currentState.currentPosition:len(s)]
				switch {
				case strings.HasPrefix(r, "or"):
					currentState.openRules = append(currentState.openRules, "or")
				default:
					err = fmt.Errorf("unexpected token at position %d", currentState.currentPosition+1)
					break
				}
			}
		case tok == 'a':
			if !currentState.inLiteral {
				r := s[currentState.currentPosition:len(s)]
				switch {
				case strings.HasPrefix(r, "and"):
					currentState.openRules = append(currentState.openRules, "and")
				default:
					err = fmt.Errorf("unexpected token at position %d", currentState.currentPosition+1)
					break
				}
			}
		case tok == 'm':
			if !currentState.inLiteral {
				r := s[currentState.currentPosition:len(s)]
				switch {
				case strings.HasPrefix(r, "matchValue"):
					currentState.openRules = append(currentState.openRules, "matchValue")

					for i := 0; i < 9; i++ {
						reader.ReadByte()
						currentState.currentPosition++
					}
				case strings.HasPrefix(r, "matchHashSHA1"):
					currentState.openRules = append(currentState.openRules, "matchHashSHA1")

					for i := 0; i < 12; i++ {
						reader.ReadByte()
						currentState.currentPosition++
					}
				case strings.HasPrefix(r, "matchRegex"):
					currentState.openRules = append(currentState.openRules, "matchRegex")

					for i := 0; i < 9; i++ {
						reader.ReadByte()
						currentState.currentPosition++
					}
				default:
					err = fmt.Errorf("unexpected rule at position %d", currentState.currentPosition+1)
					break
				}

				currentState.literalCountBeforeMatchRule = len(currentState.literals)
			}
		default:
			if currentState.shouldEscape {
				currentState.shouldEscape = false
			}
		}

		if err != nil {
			break
		}

		currentState.currentPosition++
	}

	return &ParserResult{currentState.rules, currentState.literals}, err
}

func NewParameter(source string) *hook.Argument {
	result := strings.SplitN(source, ".", 2)
	return &hook.Argument{Source: result[0], Name: result[1]}
}

func NewMatchRule(ruleName string, literals []string) *hook.MatchRule {
	var rule hook.MatchRule

	switch {
	case ruleName == "matchValue":
		rule.Type = hook.MatchValue
		rule.Value = literals[1]
		rule.Parameter = *NewParameter(literals[0])
	case ruleName == "matchRegex":
		rule.Type = hook.MatchRegex
		rule.Regex = literals[1]
		rule.Parameter = *NewParameter(literals[0])
	case ruleName == "matchHashSHA1":
		rule.Type = hook.MatchHashSHA1
		rule.Secret = literals[1]
		rule.Parameter = *NewParameter(literals[0])
	default:
		panic("invalid match rule")
	}

	return &rule
}

func NewFromParserResult(parserResult *ParserResult) *hook.Rules {
	if len(parserResult.rules) == 0 {
		return nil
	}
	var newRule hook.Rules

	var ruleName string

	ruleName, parserResult.rules = parserResult.rules[len(parserResult.rules)-1], parserResult.rules[:len(parserResult.rules)-1]

	switch {
	case strings.HasPrefix(ruleName, "match"):
		var literals []string
		literals, parserResult.literals = parserResult.literals[len(parserResult.literals)-2:len(parserResult.literals)], parserResult.literals[:len(parserResult.literals)-2]
		newRule.Match = NewMatchRule(ruleName, literals)
	case ruleName == "not":
		newRule.Not = (*hook.NotRule)(NewFromParserResult(parserResult))
	case ruleName == "and":
		var subRules []hook.Rules
		for {
			if len(parserResult.rules) == 0 {
				break
			}
			newSubRule := NewFromParserResult(parserResult)
			subRules = append([]hook.Rules{*newSubRule}, subRules...)
		}
		newRule.And = (*hook.AndRule)(&subRules)
	case ruleName == "or":
		var subRules []hook.Rules
		for {
			if len(parserResult.rules) == 0 {
				break
			}
			newSubRule := NewFromParserResult(parserResult)
			subRules = append([]hook.Rules{*newSubRule}, subRules...)
		}
		newRule.Or = (*hook.OrRule)(&subRules)
	}

	return &newRule
}
