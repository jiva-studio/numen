// Package toolcall describes a vault tool call to the user, using the declarations the host sends
// with a task.
package toolcall

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
)

// Declaration is what the host declares about one vault tool: its title, its effect, and the
// argument a call is about.
type Declaration struct {
	Title, Kind, About, Element, Match, Text string
}

// ToolDeclaration is the declaration of one vault tool, by the name the vault serves it under.
type ToolDeclaration struct {
	Name, Title, Kind, About, Element, Match, Text string
}

// Range is the run of a source's text a call names, counted in bytes.
type Range struct {
	Start, End int32
}

// ToolCall is a tool call as shown to the user. One call is reported more than once while its
// arguments arrive.
type ToolCall struct {
	// Tool is the name of the tool, or its title where the vault declared one.
	Tool string
	// About is what the call is about. Empty when the call says nothing worth showing.
	About string
	// Length is how much of the call has been written, in characters.
	Length int32
	// Path is the source the call works on, by the path the vault files it under.
	Path string
	// Range is the run of the source's text the call names. Nil when the call names none.
	Range *Range
	// Call is the name the agent gave the call. Every report of one call carries the same name.
	Call string
	// Kind is what the call does to the vault: read, edit, remove, move or search. Empty is unknown.
	Kind string
}

// IndexDeclarations indexes the declarations of the vault's tools by tool name.
func IndexDeclarations(declarations []ToolDeclaration) map[string]Declaration {
	words := make(map[string]Declaration, len(declarations))
	for _, d := range declarations {
		words[d.Name] = Declaration{
			Title: d.Title, Kind: d.Kind, About: d.About, Element: d.Element, Match: d.Match, Text: d.Text,
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
func readRange(args string) *Range {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(args), &parsed); err != nil {
		return nil
	}
	start, length := readNumber(parsed, "start"), readNumber(parsed, "length")
	if length <= 0 || start < 0 || start > math.MaxInt32-length {
		return nil
	}
	return &Range{Start: int32(start), End: int32(start + length)}
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
func readCallDetails(args string) (string, string, *Range) {
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

	var span *Range
	start, err := strconv.ParseInt(ReadPartialField(args, "start"), 10, 32)
	if err != nil {
		return about, path, nil
	}
	length, err := strconv.ParseInt(ReadPartialField(args, "length"), 10, 32)
	if err == nil && start >= 0 && length >= 0 && start <= math.MaxInt32-length {
		span = &Range{Start: int32(start), End: int32(start + length)}
	}
	return about, path, span
}

// readWholeCallDetails is readCallDetails for arguments that parse whole.
func readWholeCallDetails(parsed map[string]any, args string) (string, string, *Range) {
	path, _ := readFirstString(parsed, "path", "file")
	about, hasAbout := readFirstString(parsed, "query", "about", "title")
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
	return about, path, &Range{Start: int32(from), End: int32(from + length)}
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

// readFirstString returns the first of keys whose value in parsed is a string.
func readFirstString(parsed map[string]any, keys ...string) (string, bool) {
	for _, key := range keys {
		if value, ok := parsed[key].(string); ok {
			return value, true
		}
	}
	return "", false
}
