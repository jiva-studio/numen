package query

import (
	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/env"
	"cel.dev/cel-go/common/operators"
	"cel.dev/cel-go/common/overloads"
)

// The names the language declares besides the operators.
const (
	functionMeaning = "meaning"
	functionWords   = "words"
	variableKind    = "kind"
	variablePath    = "path"
)

func newEnvironment() (*cel.Env, error) {
	subset := &env.LibrarySubset{
		DisableMacros: true,
		IncludeFunctions: []*env.Function{
			{Name: operators.LogicalAnd}, {Name: operators.LogicalOr}, {Name: operators.LogicalNot},
			{Name: operators.Equals}, {Name: operators.NotEquals}, {Name: operators.In},
			{Name: overloads.StartsWith},
		},
	}
	return cel.NewCustomEnv(
		cel.StdLib(cel.StdLibSubset(subset)),
		cel.Variable(variableKind, cel.StringType),
		cel.Variable(variablePath, cel.StringType),
		cel.Function(functionMeaning, cel.Overload("meaning_string", []*cel.Type{cel.StringType}, cel.BoolType)),
		cel.Function(functionWords, cel.Overload("words_string", []*cel.Type{cel.StringType}, cel.BoolType)),
	)
}
