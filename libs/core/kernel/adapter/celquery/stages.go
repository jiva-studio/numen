package celquery

import (
	"strconv"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/ast"
	"cel.dev/cel-go/common/types"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
)

func declareStage(f searchquery.Function) cel.EnvOption {
	lists := f.Overloads
	if lists == nil {
		lists = [][]searchquery.ValueType{f.Args}
	}
	overloads := make([]cel.FunctionOpt, len(lists))
	for i, list := range lists {
		args := []*cel.Type{cel.BoolType}
		for range list {
			args = append(args, cel.DynType)
		}
		overloads[i] = cel.MemberOverload(f.Name+"_bool_"+strconv.Itoa(len(list)), args, cel.BoolType)
	}
	return cel.Function(f.Name, overloads...)
}

const (
	messageStageLast    = "a stage comes after the whole query: rerank, top and sortBy are called on it last"
	messageStageLiteral = "the arguments of a stage are string or number literals"
)

func (c *converter) convertQuery(expr ast.Expr) searchquery.RawNode {
	var stages []searchquery.RawStage
	for isStageCall(expr) {
		call := expr.AsCall()
		stage := searchquery.RawStage{Name: call.FunctionName(), Position: c.getPosition(expr)}
		for _, arg := range call.Args() {
			stage.Args = append(stage.Args, c.convertArg(arg))
		}
		stages = append([]searchquery.RawStage{stage}, stages...)
		expr = call.Target()
	}
	root := c.convert(expr)
	root.Stages = stages
	return root
}

func isStageCall(expr ast.Expr) bool {
	return expr.Kind() == ast.CallKind && expr.AsCall().IsMemberFunction() && searchquery.IsStageName(expr.AsCall().FunctionName())
}

func (c *converter) convertArg(expr ast.Expr) searchquery.RawArg {
	arg := searchquery.RawArg{Position: c.getPosition(expr)}
	if expr.Kind() != ast.LiteralKind {
		c.problems = append(c.problems, searchquery.Problem{Position: arg.Position, Message: messageStageLiteral})
		return arg
	}
	switch value := expr.AsLiteral().(type) {
	case types.String:
		arg.Text = string(value)
	case types.Int:
		arg.Number, arg.IsInt = int64(value), true
	default:
		c.problems = append(c.problems, searchquery.Problem{Position: arg.Position, Message: messageStageLiteral})
	}
	return arg
}
