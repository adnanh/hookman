package main

import (
	"fmt"
	"strings"

	"github.com/adnanh/webhook/hook"
)

// AndRule that implements Stringer interface
type AndRule hook.AndRule

// OrRule that implements Stringer interface
type OrRule hook.OrRule

// NotRule that implements Stringer interface
type NotRule hook.NotRule

// MatchRule that implements Stringer interface
type MatchRule hook.MatchRule

// Rules that implements Stringer interface
type Rules hook.Rules

// Hook that implements Stringer interface
type Hook hook.Hook

// CompactHook that implements Stringer interface
type CompactHook hook.Hook

func (r OrRule) String() string {
	stringSlice := make([]string, len(r))

	for idx, rule := range r {
		stringSlice[idx] = fmt.Sprintf("%s", (Rules)(rule))
	}

	return fmt.Sprintf("(%s)", strings.Join(stringSlice, " or "))
}

func (r NotRule) String() string {
	return fmt.Sprintf("!(%s)", (Rules)(r))
}

func (r MatchRule) String() string {
	switch {
	case r.Type == hook.MatchValue:
		return fmt.Sprintf("matchValue(\"%s.%s\", \"%s\")", r.Parameter.Source, r.Parameter.Name, r.Value)
	case r.Type == hook.MatchRegex:
		return fmt.Sprintf("matchRegex(\"%s.%s\", \"%s\")", r.Parameter.Source, r.Parameter.Name, r.Regex)
	case r.Type == hook.MatchHashSHA1:
		return fmt.Sprintf("matchHashSHA1(\"%s.%s\", \"%s\")", r.Parameter.Source, r.Parameter.Name, r.Secret)
	default:
		return "Invalid match rule"
	}
}

func (r Rules) String() string {
	switch {
	case r.And != nil:
		return fmt.Sprintf("%s", (*AndRule)(r.And))
	case r.Or != nil:
		return fmt.Sprintf("%s", (*OrRule)(r.Or))
	case r.Not != nil:
		return fmt.Sprintf("%s", (*NotRule)(r.Not))
	case r.Match != nil:
		return fmt.Sprintf("%s", (*MatchRule)(r.Match))
	default:
		return "No rule"
	}
}

func (r *AndRule) String() string {
	stringSlice := make([]string, len(*r))

	for idx, rule := range *r {
		stringSlice[idx] = fmt.Sprintf("%s", Rules(rule))
	}

	return fmt.Sprintf("(%s)", strings.Join(stringSlice, " and "))
}

func (h Hook) String() string {
	var result []string

	result = append(result, fmt.Sprintf("HOOK ID:\n   %s\n\n", h.ID))

	if h.ExecuteCommand != "" {
		result = append(result, fmt.Sprintf("EXECUTE COMMAND:\n   %s\n\n", h.ExecuteCommand))
	}

	if h.CommandWorkingDirectory != "" {
		result = append(result, fmt.Sprintf("COMMAND WORKING DIRECTORY:\n   %s\n\n", h.CommandWorkingDirectory))
	}

	if h.JSONStringParameters != nil {
		result = append(result, fmt.Sprintf("JSON STRING PARAMETERS TO BE DECODED:\n   %s\n\n", argumentsToString(h.JSONStringParameters)))
	}

	if h.PassArgumentsToCommand != nil {
		result = append(result, fmt.Sprintf("PASS ARGUMENTS TO COMMAND:\n   %s\n\n", argumentsToString(h.PassArgumentsToCommand)))
	}

	if h.PassEnvironmentToCommand != nil {
		result = append(result, fmt.Sprintf("PASS ENVIRONMENT TO COMMAND:\n   %s\n\n", environmentToString(h.PassEnvironmentToCommand)))
	}

	if h.TriggerRule != nil {
		result = append(result, fmt.Sprintf("TRIGGER RULE:\n   %s\n\n", (*Rules)(h.TriggerRule)))
	}

	if h.ResponseMessage != "" {
		result = append(result, fmt.Sprintf("RESPONSE MESSAGE FOR THE HOOK INITIATOR:\n   %s\n\n", h.ResponseMessage))
	}

	result = append(result, fmt.Sprintf("INCLUDE COMMAND OUTPUT IN RESPONSE:\n   %t\n\n", h.CaptureCommandOutput))

	return strings.Join(result, "")
}

func (h CompactHook) String() string {
	var result []string

	result = append(result, fmt.Sprintf("%s", h.ID))

	if h.ExecuteCommand != "" {
		result = append(result, fmt.Sprintf("%s", h.ExecuteCommand))
	} else {
		result = append(result, "No command")
	}

	return strings.Join(result, " => ")
}

func argumentsToString(args []hook.Argument) string {
	argsSlice := make([]string, len(args))

	for idx, arg := range args {
		argsSlice[idx] = fmt.Sprintf("%s.%s", arg.Source, arg.Name)
	}

	return fmt.Sprintf("[%s]", strings.Join(argsSlice, ", "))
}

func environmentToString(args []hook.Argument) string {
	argsSlice := make([]string, len(args))

	for idx, arg := range args {
		argsSlice[idx] = fmt.Sprintf("HOOK_%s=%s.%s", arg.Name, arg.Source, arg.Name)
	}

	return fmt.Sprintf("[%s]", strings.Join(argsSlice, ", "))
}
