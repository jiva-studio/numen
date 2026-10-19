// Package toolcall tells a person about a call of a vault tool, by the declarations the host sends
// with a task.
package toolcall

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// Declaration is what the host declares about one vault tool: its title, its effect, and the
// argument a call is about.
type Declaration struct {
	Title, Kind, About, Element, Match, Text string
}

// IndexDeclarations indexes the declarations of the vault's tools by tool name.
func IndexDeclarations(declarations []*numenv1.ToolDeclaration) map[string]Declaration {
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

// Describe is a call of the tool the vault serves under name, as the person is told about it. A
// tool the vault declared is shown by its title, with its effect and what the call is about read by
// the arguments it declared. The arguments may not have finished arriving.
func Describe(words map[string]Declaration, name, call, args string) *numenv1.ToolCall {
	declared, isDeclared := words[name]
	described := &numenv1.ToolCall{Tool: name, Call: call, Written: int32(len([]rune(args)))}
	if !isDeclared {
		described.About, described.Path, described.Span = readCallDetails(args)
		return described
	}
	if declared.Title != "" {
		described.Tool = declared.Title
	}
	described.Kind = declared.Kind
	if declared.About == "" {
		described.Written = 0
		return described
	}
	described.About = readAbout(declared, args)
	if declared.About == "path" {
		described.Path, described.Span = described.About, readSpan(args)
	}
	return described
}

// readSpan is the run of a source's text a call names, read once the arguments parse whole.
func readSpan(args string) *numenv1.Span {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(args), &parsed); err != nil {
		return nil
	}
	start, _ := parsed["start"].(float64)
	length, _ := parsed["length"].(float64)
	if length <= 0 || start < 0 || start > math.MaxInt32-length {
		return nil
	}
	return &numenv1.Span{From: int32(start), To: int32(start + length)}
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
		first, _ = element[inside].(string)
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
func readCallDetails(args string) (string, string, *numenv1.Span) {
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

	var span *numenv1.Span
	start, err := strconv.ParseInt(ReadPartialField(args, "start"), 10, 32)
	if err != nil {
		return about, path, nil
	}
	length, err := strconv.ParseInt(ReadPartialField(args, "length"), 10, 32)
	if err == nil && start >= 0 && length >= 0 && start <= math.MaxInt32-length {
		span = &numenv1.Span{From: int32(start), To: int32(start + length)}
	}
	return about, path, span
}

// readWholeCallDetails is readCallDetails for arguments that parse whole.
func readWholeCallDetails(parsed map[string]any, args string) (string, string, *numenv1.Span) {
	var about, path string
	if p, ok := parsed["path"].(string); ok {
		path = p
	} else if f, ok := parsed["file"].(string); ok {
		path = f
	}

	if q, ok := parsed["query"].(string); ok {
		about = q
	} else if a, ok := parsed["about"].(string); ok {
		about = a
	} else if t, ok := parsed["title"].(string); ok {
		about = t
	} else if path != "" {
		about = path
	} else {
		about = args
	}

	startVal, hasStart := parsed["start"]
	lenVal, hasLen := parsed["length"]
	if !hasStart || !hasLen {
		return about, path, nil
	}
	from, _ := startVal.(float64)
	length, _ := lenVal.(float64)
	if from < 0 || length < 0 || from > math.MaxInt32-length {
		return about, path, nil
	}
	return about, path, &numenv1.Span{From: int32(from), To: int32(from + length)}
}
