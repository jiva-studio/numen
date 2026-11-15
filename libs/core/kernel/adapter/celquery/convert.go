package celquery

import (
	"fmt"

	"cel.dev/cel-go/common/ast"
	"cel.dev/cel-go/common/operators"
	"cel.dev/cel-go/common/overloads"
	"cel.dev/cel-go/common/types"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
)

// The messages of the shapes the language does not accept.
const (
	messageCondition = "the query is a condition over meaning, words, kind and path"
	messageKind      = `kind is compared with a string literal: kind == "pdf", kind != "pdf" or kind in ["pdf", "epub"]`
	messagePath      = `path is used as path.startsWith("folder/")`
	messageLiteral   = "%s takes a string literal"
)

// converter turns a checked CEL tree into a raw tree, collecting the shapes it does not accept.
type converter struct {
	source   *ast.SourceInfo
	problems []searchquery.Problem
}

func (c *converter) getPosition(expr ast.Expr) searchquery.Position {
	return getExprPosition(c.source, expr)
}

func (c *converter) reject(expr ast.Expr, message string) searchquery.RawNode {
	c.problems = append(c.problems, searchquery.Problem{Position: c.getPosition(expr), Message: message})
	return searchquery.RawNode{Operator: searchquery.OperatorAnd}
}

func (c *converter) newLeaf(expr ast.Expr, leaf searchquery.RawLeaf) searchquery.RawNode {
	leaf.Position = c.getPosition(expr)
	return searchquery.RawNode{Operator: searchquery.OperatorLeaf, Leaf: leaf}
}

func (c *converter) convert(expr ast.Expr) searchquery.RawNode {
	switch expr.Kind() {
	case ast.CallKind:
		return c.convertCall(expr)
	case ast.IdentKind:
		return c.reject(expr, getIdentMessage(expr.AsIdent()))
	case ast.UnspecifiedExprKind, ast.ComprehensionKind, ast.ListKind, ast.LiteralKind,
		ast.MapKind, ast.SelectKind, ast.StructKind:
		return c.reject(expr, messageCondition)
	}
	return c.reject(expr, messageCondition)
}

func getIdentMessage(name string) string {
	switch name {
	case searchquery.VariableKind:
		return messageKind
	case searchquery.VariablePath:
		return messagePath
	}
	return messageCondition
}

func (c *converter) convertCall(expr ast.Expr) searchquery.RawNode {
	call := expr.AsCall()
	switch call.FunctionName() {
	case operators.LogicalNot:
		return searchquery.RawNode{Operator: searchquery.OperatorNot, Children: []searchquery.RawNode{c.convert(call.Args()[0])}}
	case operators.LogicalAnd:
		return c.convertGroup(call.Args(), searchquery.OperatorAnd)
	case operators.LogicalOr:
		return c.convertGroup(call.Args(), searchquery.OperatorOr)
	case searchquery.FunctionMeaning:
		return c.convertText(expr, searchquery.LeafMeaning)
	case searchquery.FunctionWords:
		return c.convertText(expr, searchquery.LeafWords)
	case operators.Equals:
		return c.convertComparison(expr)
	case operators.NotEquals:
		return searchquery.RawNode{Operator: searchquery.OperatorNot, Children: []searchquery.RawNode{c.convertComparison(expr)}}
	case operators.In:
		return c.convertIn(expr)
	case overloads.StartsWith:
		return c.convertStartsWith(expr)
	}
	if isStageCall(expr) {
		return c.reject(expr, messageStageLast)
	}
	return c.reject(expr, messageCondition)
}

func (c *converter) convertGroup(args []ast.Expr, operator searchquery.Operator) searchquery.RawNode {
	group := searchquery.RawNode{Operator: operator}
	for _, arg := range args {
		group.Children = append(group.Children, c.convert(arg))
	}
	return group
}

func (c *converter) convertText(expr ast.Expr, kind searchquery.LeafKind) searchquery.RawNode {
	call := expr.AsCall()
	text, isLiteral := getStringLiteral(call.Args()[0])
	if !isLiteral {
		return c.reject(expr, fmt.Sprintf(messageLiteral, call.FunctionName()))
	}
	return c.newLeaf(expr, searchquery.RawLeaf{Kind: kind, Text: text})
}

func (c *converter) convertComparison(expr ast.Expr) searchquery.RawNode {
	args := expr.AsCall().Args()
	left, right := args[0], args[1]
	switch {
	case isIdent(left, searchquery.VariableKind) || isIdent(right, searchquery.VariableKind):
		other := right
		if isIdent(right, searchquery.VariableKind) {
			other = left
		}
		kind, isLiteral := getStringLiteral(other)
		if !isLiteral {
			return c.reject(expr, messageKind)
		}
		return c.newLeaf(expr, searchquery.RawLeaf{Kind: searchquery.LeafKindIn, Kinds: []string{kind}})
	case isIdent(left, searchquery.VariablePath) || isIdent(right, searchquery.VariablePath):
		return c.reject(expr, messagePath)
	}
	return c.reject(expr, messageCondition)
}

func (c *converter) convertIn(expr ast.Expr) searchquery.RawNode {
	args := expr.AsCall().Args()
	if !isIdent(args[0], searchquery.VariableKind) {
		return c.reject(expr, getIdentMessage(identName(args[0])))
	}
	if args[1].Kind() != ast.ListKind {
		return c.reject(expr, messageKind)
	}
	var kinds []string
	for _, element := range args[1].AsList().Elements() {
		kind, isLiteral := getStringLiteral(element)
		if !isLiteral {
			return c.reject(expr, messageKind)
		}
		kinds = append(kinds, kind)
	}
	return c.newLeaf(expr, searchquery.RawLeaf{Kind: searchquery.LeafKindIn, Kinds: kinds})
}

func (c *converter) convertStartsWith(expr ast.Expr) searchquery.RawNode {
	call := expr.AsCall()
	switch {
	case isIdent(call.Target(), searchquery.VariableKind):
		return c.reject(expr, messageKind)
	case !isIdent(call.Target(), searchquery.VariablePath):
		return c.reject(expr, messageCondition)
	}
	prefix, isLiteral := getStringLiteral(call.Args()[0])
	if !isLiteral {
		return c.reject(expr, messagePath)
	}
	return c.newLeaf(expr, searchquery.RawLeaf{Kind: searchquery.LeafPathPrefix, Prefix: prefix})
}

func identName(expr ast.Expr) string {
	if expr.Kind() != ast.IdentKind {
		return ""
	}
	return expr.AsIdent()
}

func isIdent(expr ast.Expr, name string) bool {
	return expr.Kind() == ast.IdentKind && expr.AsIdent() == name
}

func getStringLiteral(expr ast.Expr) (string, bool) {
	if expr.Kind() != ast.LiteralKind {
		return "", false
	}
	text, isString := expr.AsLiteral().(types.String)
	return string(text), isString
}
