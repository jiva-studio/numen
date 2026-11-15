package celquery

import (
	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/env"
	"cel.dev/cel-go/common/operators"
	"cel.dev/cel-go/common/overloads"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
)

// newEnvironment declares the catalog in CEL. The checker of cel-go types && and || as bool only, so a
// function of type SearchResults is declared as bool, and the converted tree records whether the query names one.
func newEnvironment(catalog searchquery.Catalog, maxDepth, maxCodePoints int) (*cel.Env, error) {
	subset := &env.LibrarySubset{
		DisableMacros: true,
		IncludeFunctions: []*env.Function{
			{Name: operators.LogicalAnd}, {Name: operators.LogicalOr}, {Name: operators.LogicalNot},
			{Name: operators.Equals}, {Name: operators.NotEquals}, {Name: operators.In},
			{Name: overloads.StartsWith},
		},
	}
	options := []cel.EnvOption{
		cel.StdLib(cel.StdLibSubset(subset)),
		cel.ParserRecursionLimit(maxDepth),
		cel.ParserExpressionSizeLimit(maxCodePoints),
	}
	for _, f := range catalog.Functions {
		options = append(options, declare(f))
	}
	return cel.NewCustomEnv(options...)
}

func declare(f searchquery.Function) cel.EnvOption {
	if f.IsVariable {
		return cel.Variable(f.Name, getType(f.Result))
	}
	args := make([]*cel.Type, len(f.Args))
	overloadID := f.Name
	for i, arg := range f.Args {
		args[i] = getType(arg)
		overloadID += "_" + args[i].String()
	}
	return cel.Function(f.Name, cel.Overload(overloadID, args, getType(f.Result)))
}

func getType(t searchquery.ValueType) *cel.Type {
	switch t {
	case searchquery.TypeString:
		return cel.StringType
	case searchquery.TypeInt:
		return cel.IntType
	case searchquery.TypeStringList:
		return cel.ListType(cel.StringType)
	case searchquery.TypeBool, searchquery.TypeSearchResults:
		return cel.BoolType
	}
	return cel.BoolType
}
