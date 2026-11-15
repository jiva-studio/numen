package celquery

import (
	"slices"
	"strings"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/env"
	"cel.dev/cel-go/common/operators"
	"cel.dev/cel-go/common/overloads"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
)

func newEnvironment(catalog searchquery.Catalog, maxDepth, maxCodePoints int) (*cel.Env, error) {
	subset := &env.LibrarySubset{
		DisableMacros: true,
		IncludeFunctions: []*env.Function{
			{Name: operators.LogicalAnd}, {Name: operators.LogicalOr}, {Name: operators.LogicalNot},
			{Name: operators.Equals}, {Name: operators.NotEquals}, {Name: operators.In},
			{Name: operators.Less}, {Name: operators.LessEquals}, {Name: operators.Greater}, {Name: operators.GreaterEquals},
			{Name: overloads.StartsWith}, {Name: overloads.EndsWith}, {Name: overloads.Contains},
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
	overloads := []cel.FunctionOpt{newOverload(f.Name, f.Args, f.Result)}
	if len(f.OptionalArgs) > 0 {
		overloads = append(overloads, newOverload(f.Name, append(slices.Clone(f.Args), f.OptionalArgs...), f.Result))
	}
	return cel.Function(f.Name, overloads...)
}

func newOverload(name string, valueTypes []searchquery.ValueType, result searchquery.ValueType) cel.FunctionOpt {
	args := make([]*cel.Type, len(valueTypes))
	parts := []string{name}
	for i, arg := range valueTypes {
		args[i] = getType(arg)
		parts = append(parts, args[i].String())
	}
	return cel.Overload(strings.Join(parts, "_"), args, getType(result))
}

func getType(t searchquery.ValueType) *cel.Type {
	switch t {
	case searchquery.TypeString:
		return cel.StringType
	case searchquery.TypeInt:
		return cel.IntType
	case searchquery.TypeStringList:
		return cel.ListType(cel.StringType)
	case searchquery.TypeTimestamp:
		return cel.TimestampType
	case searchquery.TypeDyn:
		return cel.DynType
	// The checker of cel-go types && and || as bool only, so SearchResults is declared as bool.
	case searchquery.TypeBool, searchquery.TypeSearchResults:
		return cel.BoolType
	}
	return cel.BoolType
}
