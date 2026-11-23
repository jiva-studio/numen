package celquery

import (
	"fmt"
	"strings"

	"cel.dev/cel-go/common/ast"
	"cel.dev/cel-go/common/operators"
	"cel.dev/cel-go/common/overloads"
	"cel.dev/cel-go/common/types"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
)

// The messages of the shapes the language does not accept.
const (
	messageCondition = "A query is a condition over meaning, words, the document functions and the link functions"
	messageKind      = `Compare kind with a text in quotes: kind == "pdf", kind != "pdf" or kind in ["pdf", "epub"]`
	messagePath      = `Use path as path.startsWith("folder/")`
	messageLiteral   = "%s takes a text in quotes"
	messageTitle     = `Compare title with a text in quotes: title == "x", title in ["x", "y"], title.startsWith("x"), title.endsWith("x") or title.contains("x")`
	messageNoteType  = `Compare noteType with a text in quotes: noteType == "book", noteType != "book" or noteType in ["book", "article"]`
	messageModified  = `Compare modified with date(…): modified >= date("2026-01-01")`
	messageSize      = `Compare size with a whole number: size > 1000`
	messageProperty  = `property takes a key in quotes and is compared with a value: property("status") == "done", property("rating") >= 4 or "x" in property("tags")`
	messageDate      = `date takes a date in quotes: date("2026-01-01")`
	messageTarget    = "The target of %s is empty"
	messageArgument  = "Write the arguments of %s as texts in quotes or numbers"
)

// sink collects the problems of a conversion and builds its nodes.
type sink struct {
	source   *ast.SourceInfo
	text     queryText
	problems []searchquery.Problem
}

// comparer converts the comparisons and membership tests of a query.
type comparer struct{ *sink }

// linkConverter converts the link functions of a query.
type linkConverter struct{ *sink }

// converter turns a checked CEL tree into a raw tree, collecting the shapes it does not accept.
type converter struct {
	*sink
	catalog  searchquery.Catalog
	comparer comparer
	links    linkConverter
}

func (c *sink) getPosition(expr ast.Expr) searchquery.Position {
	return getExprPosition(c.text, c.source, expr)
}

func (c *sink) reject(expr ast.Expr, message string) searchquery.RawNode {
	c.problems = append(c.problems, searchquery.Problem{Position: c.getPosition(expr), Message: message})
	return searchquery.RawNode{Operator: searchquery.OperatorAnd}
}

func (c *sink) newLeaf(expr ast.Expr, leaf searchquery.RawLeaf) searchquery.RawNode {
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
	case searchquery.VariableTitle:
		return messageTitle
	case searchquery.VariableNoteType:
		return messageNoteType
	case searchquery.VariableModified:
		return messageModified
	case searchquery.VariableSize:
		return messageSize
	}
	return messageCondition
}

func (c *converter) convertCall(expr ast.Expr) searchquery.RawNode {
	call := expr.AsCall()
	switch name := call.FunctionName(); name {
	case operators.LogicalNot:
		return newNot(c.convert(call.Args()[0]))
	case operators.LogicalAnd:
		return c.convertGroup(call.Args(), searchquery.OperatorAnd)
	case operators.LogicalOr:
		return c.convertGroup(call.Args(), searchquery.OperatorOr)
	case searchquery.FunctionMeaning:
		return c.convertText(expr, searchquery.LeafMeaning)
	case searchquery.FunctionWords:
		return c.convertText(expr, searchquery.LeafWords)
	case searchquery.FunctionHasProperty, searchquery.FunctionHasHeading:
		return c.comparer.convertPresence(expr)
	case operators.NotEquals:
		return newNot(c.comparer.convertComparison(expr, searchquery.ComparisonEqual))
	case operators.In:
		return c.comparer.convertIn(expr)
	case overloads.StartsWith, overloads.EndsWith, overloads.Contains:
		return c.comparer.convertMethod(expr)
	default:
		return c.convertComparisonOrLink(expr, name)
	}
}

func (c *converter) convertComparisonOrLink(expr ast.Expr, name string) searchquery.RawNode {
	if comparison, isComparison := getComparison(name); isComparison {
		return c.comparer.convertComparison(expr, comparison)
	}
	if _, isLink := searchquery.FindLinkFunction(name); isLink {
		return c.links.convertLink(expr)
	}
	return c.convertOther(expr)
}

func getComparison(name string) (searchquery.Comparison, bool) {
	switch name {
	case operators.Equals:
		return searchquery.ComparisonEqual, true
	case operators.Less:
		return searchquery.ComparisonLess, true
	case operators.LessEquals:
		return searchquery.ComparisonLessOrEqual, true
	case operators.Greater:
		return searchquery.ComparisonGreater, true
	case operators.GreaterEquals:
		return searchquery.ComparisonGreaterOrEqual, true
	}
	return 0, false
}

func newNot(child searchquery.RawNode) searchquery.RawNode {
	return searchquery.RawNode{Operator: searchquery.OperatorNot, Children: []searchquery.RawNode{child}}
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

func (c *linkConverter) convertLink(expr ast.Expr) searchquery.RawNode {
	call := expr.AsCall()
	name := call.FunctionName()
	var texts []string
	for _, arg := range call.Args() {
		text, isLiteral := getStringLiteral(arg)
		if !isLiteral {
			return c.reject(arg, fmt.Sprintf(messageLiteral, name))
		}
		texts = append(texts, text)
	}
	if strings.TrimSpace(texts[0]) == "" {
		return c.reject(call.Args()[0], fmt.Sprintf(messageTarget, name))
	}
	leaf := searchquery.RawLeaf{Kind: searchquery.LeafSet, Function: name, Target: texts[0]}
	if len(texts) > 1 {
		leaf.LinkType = texts[1]
	}
	return c.newLeaf(expr, leaf)
}

func getIdentName(expr ast.Expr) string {
	if expr.Kind() != ast.IdentKind {
		return ""
	}
	return expr.AsIdent()
}

func isIdent(expr ast.Expr, name string) bool {
	return expr.Kind() == ast.IdentKind && expr.AsIdent() == name
}

func isCallOf(expr ast.Expr, name string) bool {
	return expr.Kind() == ast.CallKind && !expr.AsCall().IsMemberFunction() && expr.AsCall().FunctionName() == name
}

func getStringLiteral(expr ast.Expr) (string, bool) {
	if expr.Kind() != ast.LiteralKind {
		return "", false
	}
	text, isString := expr.AsLiteral().(types.String)
	return string(text), isString
}
