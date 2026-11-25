package searchquery

import (
	"errors"
	"slices"

	"github.com/jiva-studio/numen/libs/plugin/catalog/searchfunction"
)

// ErrInvalidFunctionArguments is a call of a plugin function whose arguments the plugin rejects.
var ErrInvalidFunctionArguments = errors.New("the function rejects these arguments")

// Argument is the value of one argument of a call of a plugin function, by the name the manifest gives it.
type Argument struct {
	Name   string
	Type   ValueType
	String string
	Int    int64
	Bool   bool
}

// Call is one distinct call of a plugin function in a query. IsPositive is set when it appears at least once
// without a negation.
type Call struct {
	PluginID   string
	Function   string
	Arguments  []Argument
	Position   Position
	IsPositive bool
}

func (c Call) isSame(other Call) bool {
	return c.PluginID == other.PluginID && c.Function == other.Function && slices.Equal(c.Arguments, other.Arguments)
}

// FunctionCall is a call of a plugin function as the plugin receives it: on one vault, for the search of one
// generation of its index.
type FunctionCall struct {
	PluginID        string
	Function        string
	Arguments       []Argument
	VaultID         string
	IndexGeneration string
}

// FunctionAnswer is the paths a plugin function holds for. EstimatedCount is how many documents it holds
// for in all, when the answer is cut at a limit.
type FunctionAnswer struct {
	Paths          []string
	EstimatedCount int64
	Warning        string
}

// FoundPaths are, for some calls of a query by their index, every path the call holds for.
type FoundPaths map[int][]string

// MergeCatalog returns the built-in functions followed by the contributed ones. A contributed function whose
// name is already in the catalog is left out.
func MergeCatalog(builtin Catalog, contributed []Function) Catalog {
	merged := Catalog{Functions: slices.Clone(builtin.Functions)}
	for _, f := range contributed {
		if _, isTaken := merged.FindFunction(f.Name); !isTaken {
			merged.Functions = append(merged.Functions, f)
		}
	}
	return merged
}

// ListFunctionNames returns the names of the functions, variables and stages of the catalog.
func ListFunctionNames(c Catalog) []string {
	names := make([]string, len(c.Functions))
	for i, f := range c.Functions {
		names[i] = f.Name
	}
	return names
}

// ConvertManifestFunctions returns the functions a plugin declares as entries of the catalog: each a
// document-level condition over its arguments.
func ConvertManifestFunctions(pluginID string, declared searchfunction.Set) []Function {
	out := make([]Function, 0, len(declared.Functions))
	for _, one := range declared.Functions {
		f := Function{
			Name: one.Name, Result: TypeBool, Level: LevelDocument, PluginID: pluginID,
			Description: one.Description, Example: one.Example, Args: []ValueType{}, ArgNames: []string{},
		}
		for _, arg := range one.Args {
			f.Args = append(f.Args, getArgumentType(arg.Type))
			f.ArgNames = append(f.ArgNames, arg.Name)
		}
		out = append(out, f)
	}
	return out
}

func getArgumentType(t searchfunction.ArgType) ValueType {
	switch t {
	case searchfunction.ArgInt:
		return TypeInt
	case searchfunction.ArgBool:
		return TypeBool
	case searchfunction.ArgString:
		return TypeString
	}
	return TypeString
}
