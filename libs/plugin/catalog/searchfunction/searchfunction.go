// Package searchfunction is the model of the query functions a plugin declares in contributes.searchFunctions, and the rules they follow.
package searchfunction

import (
	"fmt"
	"regexp"
	"strings"
)

// SupportedVersion is the schema version of contributes.searchFunctions this build reads.
const SupportedVersion = 1

const (
	maxNameBytes   = 64
	maxDescription = 300
)

var namePattern = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)

// Set are the query functions the plugin's backend answers, under a schema version.
type Set struct {
	Version   int
	Functions []Function
}

// Function is one function of the query language a plugin answers.
type Function struct {
	Name        string
	Args        []Arg
	Level       Level
	Description string
	Example     string
}

// Arg is one positional argument, named for the description and the SDK.
type Arg struct {
	Name string
	Type ArgType
}

// Level is where a query function decides: on a whole document or on one chunk.
type Level string

const (
	LevelDocument Level = "document"
	LevelChunk    Level = "chunk"
)

// ArgType is the type of the value of an argument.
type ArgType string

const (
	ArgString ArgType = "string"
	ArgInt    ArgType = "int"
	ArgBool   ArgType = "bool"
)

// ListIssues returns one error for each rule the declared functions break.
func ListIssues(declared Set) []error {
	if len(declared.Functions) == 0 {
		return nil
	}
	if declared.Version != SupportedVersion {
		return []error{newIssue("schema version %d is not supported", declared.Version)}
	}
	var issues []error
	observed := map[string]bool{}
	for _, f := range declared.Functions {
		switch {
		case !isName(f.Name):
			issues = append(issues, newIssue("name %q is not a lowercase letter followed by letters and digits", f.Name))
		case observed[f.Name]:
			issues = append(issues, newIssue("%q is declared twice", f.Name))
		}
		observed[f.Name] = true
		issues = append(issues, listFieldIssues(f)...)
	}
	return issues
}

func listFieldIssues(f Function) []error {
	issues := listArgIssues(f)
	switch f.Level {
	case LevelDocument:
	case LevelChunk:
		issues = append(issues, newIssue("%q: chunk-level search functions are not supported", f.Name))
	default:
		issues = append(issues, newIssue("%q has the level %q; the level is document", f.Name, f.Level))
	}
	if d := f.Description; strings.TrimSpace(d) == "" || len(d) > maxDescription || strings.ContainsAny(d, "\r\n") {
		issues = append(issues, newIssue("%q needs a description of one line of 1 to %d bytes", f.Name, maxDescription))
	}
	if !strings.Contains(f.Example, f.Name+"(") {
		issues = append(issues, newIssue("%q needs an example that calls it", f.Name))
	}
	return issues
}

func listArgIssues(f Function) []error {
	var issues []error
	args := map[string]bool{}
	for _, arg := range f.Args {
		switch {
		case !isName(arg.Name):
			issues = append(issues, newIssue("%q has an argument name %q that is not a lowercase letter followed by letters and digits", f.Name, arg.Name))
		case args[arg.Name]:
			issues = append(issues, newIssue("%q names the argument %q twice", f.Name, arg.Name))
		}
		args[arg.Name] = true
		if arg.Type != ArgString && arg.Type != ArgInt && arg.Type != ArgBool {
			issues = append(issues, newIssue("%q has an argument of type %q; the type is string, int or bool", f.Name, arg.Type))
		}
	}
	return issues
}

func isName(name string) bool {
	return len(name) <= maxNameBytes && namePattern.MatchString(name)
}

func newIssue(format string, args ...any) error {
	return fmt.Errorf("search function %s", fmt.Sprintf(format, args...))
}
