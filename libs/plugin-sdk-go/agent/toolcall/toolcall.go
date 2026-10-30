// Package toolcall tells a person about a call of a vault tool, by the declarations the host sends
// with a task.
package toolcall

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
)

// Declaration is what the host declares about one vault tool: its title, its effect, and the
// argument a call is about.
type Declaration struct {
	Title, Kind, About, Element, Match, Text string
}

type ToolDeclaration = v1.ToolDeclaration

type ToolCall = v1.ToolCall

type Range = v1.Range

// IndexDeclarations indexes the declarations of the vault's tools by tool name.
func IndexDeclarations(declarations []*ToolDeclaration) map[string]Declaration {
	words := make(map[string]Declaration, len(declarations))
	for _, d := range declarations {
		words[d.GetName()] = Declaration{
			Title:   d.GetTitle(),
			Kind:    d.GetKind(),
			About:   d.GetAbout(),
			Element: d.GetElement(),
			Match:   d.GetMatch(),
			Text:    d.GetText(),
		}
	}
	return words
}

// tool the vault declared is shown by its title, with its effect and what the call is about read by
// the arguments it declared. The arguments may not have finished arriving.
func Describe(words map[string]Declaration, name, call, args string) *ToolCall {
	declared, isDeclared := words[name]
	described := &ToolCall{Tool: name, Call: call, Length: int32(len([]rune(args)))}
	if !isDeclared {
		described.About, described.Path, described.Range = readCallDetails(args)
		return described
	}
	if declared.Title != "" {
		described.Tool = declared.Title
	}
	described.Kind = declared.Kind
	if declared.About == "" {
		described.Length = 0
		return described
	}
	described.About = readAbout(declared, args)
	if declared.About == "path" {
		described.Path, described.Range = described.About, readRange(args)
	}
	return described
}

// readRange is the run of a source's text a call names, read once the arguments parse whole.
func readRange(args string) *v1.Range {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(args), &parsed); err != nil {
		return nil
	}
	start, length := readNumber(parsed, "start"), readNumber(parsed, "length")
	if length <= 0 || start < 0 || start > math.MaxInt32-length {
		return nil
	}
	return &v1.Range{Start: int32(start), End: int32(start + length)}
}

// readAbout is what a call is about, read from the arguments as far as they have arrived.
func readAbout(words Declaration, args string) string {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(args), &parsed); err != nil {
		if partial := ReadPartialField(args, words.Element); partial != "" {
			return partial
		}
		return ReadPartialField(args, words.About)
	}
	switch value := parsed[words.About].(type) {
	case string:
		return value
	case []any:
		return describeCollection(value, words.Element)
	}
	return ""
}

// describeCollection is the first element of a collection by the name it holds, and how many
// others there are.
func describeCollection(value []any, inside string) string {
	if len(value) == 0 {
		return ""
	}
	first := ""
	switch element := value[0].(type) {
	case string:
		first = element
	case map[string]any:
		if inside == "" {
			return ""
		}
		first = readText(element[inside])
	}
	if first == "" {
		return ""
	}
	if len(value) > 1 {
		return fmt.Sprintf("%s and %d more", first, len(value)-1)
	}
	return first
}

// readCallDetails is what an undeclared call is about, the file it names and the span inside it,
// read from the arguments whole or as far as they have arrived.
func readCallDetails(args string) (string, string, *v1.Range) {
	if args == "" {
		return "", "", nil
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(args), &parsed); err == nil {
		return readWholeCallDetails(parsed, args)
	}

	path := ReadPartialField(args, "path")
	if path == "" {
		path = ReadPartialField(args, "file")
	}
	about := ReadPartialField(args, "query")
	if about == "" {
		about = ReadPartialField(args, "about")
	}
	if about == "" {
		about = ReadPartialField(args, "title")
	}
	if about == "" {
		about = path
	}

	var span *v1.Range
	start, err := strconv.ParseInt(ReadPartialField(args, "start"), 10, 32)
	if err != nil {
		return about, path, nil
	}
	length, err := strconv.ParseInt(ReadPartialField(args, "length"), 10, 32)
	if err == nil && start >= 0 && length >= 0 && start <= math.MaxInt32-length {
		span = &v1.Range{Start: int32(start), End: int32(start + length)}
	}
	return about, path, span
}

// readWholeCallDetails is readCallDetails for arguments that parse whole.
func readWholeCallDetails(parsed map[string]any, args string) (string, string, *v1.Range) {
	path, _ := readFirstText(parsed, "path", "file")
	about, hasAbout := readFirstText(parsed, "query", "about", "title")
	if !hasAbout {
		about = args
		if path != "" {
			about = path
		}
	}

	_, hasStart := parsed["start"]
	_, hasLen := parsed["length"]
	if !hasStart || !hasLen {
		return about, path, nil
	}
	from, length := readNumber(parsed, "start"), readNumber(parsed, "length")
	if from < 0 || length < 0 || from > math.MaxInt32-length {
		return about, path, nil
	}
	return about, path, &v1.Range{Start: int32(from), End: int32(from + length)}
}

// readNumber returns the number under key, or 0 when the key is absent or not a number.
func readNumber(values map[string]any, key string) float64 {
	number, isNumber := values[key].(float64)
	if !isNumber {
		return 0
	}
	return number
}

// readText returns value when it is a string, or "" otherwise.
func readText(value any) string {
	text, isString := value.(string)
	if !isString {
		return ""
	}
	return text
}

// readFirstText returns the string under the first of keys that holds one, and whether any does.
func readFirstText(values map[string]any, keys ...string) (string, bool) {
	for _, key := range keys {
		if text, isString := values[key].(string); isString {
			return text, true
		}
	}
	return "", false
}
