package manifest

import (
	"fmt"
	"regexp"
	"strings"
)

// SearchFunctionsVersion is the schema version of contributes.searchFunctions this build reads.
const SearchFunctionsVersion = 1

const (
	maxSearchFunctionNameBytes   = 64
	maxSearchFunctionDescription = 300
)

var searchFunctionNamePattern = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)

// SearchFunctions are the query functions the plugin's backend answers, under a schema version.
type SearchFunctions struct {
	Version   int
	Functions []SearchFunction
}

// SearchFunction is one function of the query language a plugin answers.
type SearchFunction struct {
	Name        string
	Args        []SearchFunctionArg
	Level       SearchFunctionLevel
	Description string
	Example     string
}

// SearchFunctionArg is one positional argument, named for the description and the SDK.
type SearchFunctionArg struct {
	Name string
	Type SearchFunctionArgType
}

// SearchFunctionLevel is where a query function decides: on a whole document or on one chunk.
type SearchFunctionLevel string

const (
	SearchFunctionLevelDocument SearchFunctionLevel = "document"
	SearchFunctionLevelChunk    SearchFunctionLevel = "chunk"
)

// SearchFunctionArgType is the type of the value of an argument.
type SearchFunctionArgType string

const (
	SearchFunctionArgString SearchFunctionArgType = "string"
	SearchFunctionArgInt    SearchFunctionArgType = "int"
	SearchFunctionArgBool   SearchFunctionArgType = "bool"
)

func (m Manifest) listSearchFunctionIssues() []error {
	declared := m.Contributes.SearchFunctions
	if len(declared.Functions) == 0 {
		return nil
	}
	if declared.Version != SearchFunctionsVersion {
		return []error{newSearchFunctionIssue("schema version %d is not supported", declared.Version)}
	}
	var issues []error
	observed := map[string]bool{}
	for _, f := range declared.Functions {
		switch {
		case !isSearchFunctionName(f.Name):
			issues = append(issues, newSearchFunctionIssue("name %q is not a lowercase letter followed by letters and digits", f.Name))
		case observed[f.Name]:
			issues = append(issues, newSearchFunctionIssue("%q is declared twice", f.Name))
		}
		observed[f.Name] = true
		issues = append(issues, listSearchFunctionFieldIssues(f)...)
	}
	return issues
}

func listSearchFunctionFieldIssues(f SearchFunction) []error {
	var issues []error
	args := map[string]bool{}
	for _, arg := range f.Args {
		switch {
		case !isSearchFunctionName(arg.Name):
			issues = append(issues, newSearchFunctionIssue("%q has an argument name %q that is not a lowercase letter followed by letters and digits", f.Name, arg.Name))
		case args[arg.Name]:
			issues = append(issues, newSearchFunctionIssue("%q names the argument %q twice", f.Name, arg.Name))
		}
		args[arg.Name] = true
		if arg.Type != SearchFunctionArgString && arg.Type != SearchFunctionArgInt && arg.Type != SearchFunctionArgBool {
			issues = append(issues, newSearchFunctionIssue("%q has an argument of type %q; the type is string, int or bool", f.Name, arg.Type))
		}
	}
	switch f.Level {
	case SearchFunctionLevelDocument:
	case SearchFunctionLevelChunk:
		issues = append(issues, newSearchFunctionIssue("%q: chunk-level search functions are not supported", f.Name))
	default:
		issues = append(issues, newSearchFunctionIssue("%q has the level %q; the level is document", f.Name, f.Level))
	}
	if d := f.Description; strings.TrimSpace(d) == "" || len(d) > maxSearchFunctionDescription || strings.ContainsAny(d, "\r\n") {
		issues = append(issues, newSearchFunctionIssue("%q needs a description of one line of 1 to %d bytes", f.Name, maxSearchFunctionDescription))
	}
	if !strings.Contains(f.Example, f.Name+"(") {
		issues = append(issues, newSearchFunctionIssue("%q needs an example that calls it", f.Name))
	}
	return issues
}

func isSearchFunctionName(name string) bool {
	return len(name) <= maxSearchFunctionNameBytes && searchFunctionNamePattern.MatchString(name)
}

func newSearchFunctionIssue(format string, args ...any) error {
	return fmt.Errorf("%w: search function %s", ErrInvalidContribution, fmt.Sprintf(format, args...))
}
