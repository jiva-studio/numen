package celquery

import (
	"fmt"

	"cel.dev/cel-go/common/ast"
	"cel.dev/cel-go/common/types"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
)

func (c *converter) convertFunction(expr ast.Expr, f searchquery.Function) searchquery.RawNode {
	args := expr.AsCall().Args()
	call := searchquery.Call{PluginID: f.PluginID, Function: f.Name, Position: c.getPosition(expr)}
	for i, arg := range args {
		if i >= len(f.Args) || i >= len(f.ArgNames) {
			return c.reject(expr, fmt.Sprintf(messageArgument, f.Name))
		}
		value, isLiteral := getArgument(arg, f.ArgNames[i], f.Args[i])
		if !isLiteral {
			return c.reject(expr, fmt.Sprintf(messageArgument, f.Name))
		}
		call.Arguments = append(call.Arguments, value)
	}
	return c.newLeaf(expr, searchquery.RawLeaf{Kind: searchquery.LeafFunction, Call: call})
}

func getArgument(expr ast.Expr, name string, t searchquery.ValueType) (searchquery.Argument, bool) {
	if expr.Kind() != ast.LiteralKind {
		return searchquery.Argument{}, false
	}
	argument := searchquery.Argument{Name: name, Type: t}
	switch value := expr.AsLiteral().(type) {
	case types.String:
		argument.String = string(value)
		return argument, t == searchquery.TypeString
	case types.Int:
		argument.Int = int64(value)
		return argument, t == searchquery.TypeInt
	case types.Bool:
		argument.Bool = bool(value)
		return argument, t == searchquery.TypeBool
	}
	return searchquery.Argument{}, false
}

func (c *converter) convertOther(expr ast.Expr) searchquery.RawNode {
	call := expr.AsCall()
	if f, isFound := c.catalog.FindFunction(call.FunctionName()); isFound && f.PluginID != "" && !call.IsMemberFunction() {
		return c.convertFunction(expr, f)
	}
	if isStageCall(expr) {
		return c.reject(expr, messageStageLast)
	}
	return c.reject(expr, messageCondition)
}
