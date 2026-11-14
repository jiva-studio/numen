package query

import (
	"fmt"
	"strings"

	"cel.dev/cel-go/common/ast"
	"cel.dev/cel-go/common/operators"
	"cel.dev/cel-go/common/overloads"
	"cel.dev/cel-go/common/types"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
)

// converter turns the checked CEL tree into the tree of the plugin, pushing every negation down to the
// leaves and collecting the distinct leaves and the problems it finds.
type converter struct {
	source   *ast.SourceInfo
	leaves   []Leaf
	problems []Problem
}

func (c *converter) getProblem(id int64, message string) Problem {
	location := c.source.GetStartLocation(id)
	return Problem{
		Line:    max(location.Line(), firstLine),
		Column:  max(location.Column()+celColumnToOneBased, firstColumn),
		Message: message,
	}
}

func (c *converter) refuse(expr ast.Expr, message string) Node {
	c.problems = append(c.problems, c.getProblem(expr.ID(), message))
	return c.newLeafNode(expr, newTerm(TermWords), false)
}

func (c *converter) newLeafNode(expr ast.Expr, term Term, isNegated bool) Node {
	location := c.source.GetStartLocation(expr.ID())
	term.Line = max(location.Line(), firstLine)
	term.Column = max(location.Column()+celColumnToOneBased, firstColumn)
	return Node{Operator: OperatorLeaf, Children: nil, Term: term, IsNegated: isNegated}
}

func newTerm(termType TermType) Term {
	return Term{Kind: termType, LeafIndex: 0, Kinds: nil, Prefix: "", Line: 0, Column: 0}
}

func (c *converter) convert(expr ast.Expr, isNegated bool) Node {
	switch expr.Kind() {
	case ast.CallKind:
		return c.convertCall(expr, isNegated)
	case ast.IdentKind:
		return c.refuse(expr, getIdentMessage(expr.AsIdent()))
	case ast.UnspecifiedExprKind, ast.ComprehensionKind, ast.ListKind, ast.LiteralKind,
		ast.MapKind, ast.SelectKind, ast.StructKind:
		return c.refuse(expr, messageCondition)
	}
	return c.refuse(expr, messageCondition)
}

func getIdentMessage(name string) string {
	switch name {
	case variableKind:
		return messageKind
	case variablePath:
		return messagePath
	}
	return messageCondition
}

func (c *converter) convertCall(expr ast.Expr, isNegated bool) Node {
	call := expr.AsCall()
	switch call.FunctionName() {
	case operators.LogicalNot:
		return c.convert(call.Args()[0], !isNegated)
	case operators.LogicalAnd:
		return c.convertGroup(call.Args(), OperatorAnd, isNegated)
	case operators.LogicalOr:
		return c.convertGroup(call.Args(), OperatorOr, isNegated)
	case functionMeaning, functionWords:
		return c.convertText(expr, isNegated)
	case operators.Equals:
		return c.convertComparison(expr, isNegated)
	case operators.NotEquals:
		return c.convertComparison(expr, !isNegated)
	case operators.In:
		return c.convertIn(expr, isNegated)
	case overloads.StartsWith:
		return c.convertStartsWith(expr, isNegated)
	}
	return c.refuse(expr, messageCondition)
}

func (c *converter) convertGroup(args []ast.Expr, operator Operator, isNegated bool) Node {
	if isNegated {
		operator = getOtherOperator(operator)
	}
	group := Node{Operator: operator, Children: nil, Term: newTerm(TermWords), IsNegated: false}
	for _, arg := range args {
		child := c.convert(arg, isNegated)
		if child.Operator == operator {
			group.Children = append(group.Children, child.Children...)
			continue
		}
		group.Children = append(group.Children, child)
	}
	return group
}

func getOtherOperator(operator Operator) Operator {
	if operator == OperatorAnd {
		return OperatorOr
	}
	return OperatorAnd
}

func (c *converter) convertText(expr ast.Expr, isNegated bool) Node {
	call := expr.AsCall()
	name := call.FunctionName()
	text, isLiteral := getStringLiteral(call.Args()[0])
	if !isLiteral {
		return c.refuse(expr, fmt.Sprintf(messageLiteralText, name))
	}
	if strings.TrimSpace(text) == "" || len(text) > host.MaxConditionBytes {
		return c.refuse(expr, fmt.Sprintf(messageTextSize, name, host.MaxConditionBytes))
	}
	isMeaning := name == functionMeaning
	index, isAdded := c.addLeaf(isMeaning, text, !isNegated)
	if !isAdded {
		return c.refuse(expr, getLeafCountMessage())
	}
	term := newTerm(TermWords)
	if isMeaning {
		term.Kind = TermMeaning
	}
	term.LeafIndex = index
	return c.newLeafNode(expr, term, isNegated)
}

func (c *converter) addLeaf(isMeaning bool, text string, isPositive bool) (int, bool) {
	for i, leaf := range c.leaves {
		if leaf.IsMeaning == isMeaning && leaf.Text == text {
			c.leaves[i].IsPositive = leaf.IsPositive || isPositive
			return i, true
		}
	}
	if len(c.leaves) == host.MaxScoredConditions {
		return 0, false
	}
	c.leaves = append(c.leaves, Leaf{IsMeaning: isMeaning, Text: text, IsPositive: isPositive})
	return len(c.leaves) - 1, true
}

func (c *converter) convertComparison(expr ast.Expr, isNegated bool) Node {
	args := expr.AsCall().Args()
	left, right := args[0], args[1]
	switch {
	case isIdent(left, variableKind) || isIdent(right, variableKind):
		other := right
		if isIdent(right, variableKind) {
			other = left
		}
		kind, isLiteral := getStringLiteral(other)
		if !isLiteral || !isKind(kind) {
			return c.refuse(expr, messageKind)
		}
		term := newTerm(TermKind)
		term.Kinds = []string{kind}
		return c.newLeafNode(expr, term, isNegated)
	case isIdent(left, variablePath) || isIdent(right, variablePath):
		return c.refuse(expr, messagePath)
	}
	return c.refuse(expr, messageCondition)
}

func (c *converter) convertIn(expr ast.Expr, isNegated bool) Node {
	args := expr.AsCall().Args()
	if !isIdent(args[0], variableKind) {
		return c.refuse(expr, getInMessage(args[0]))
	}
	if args[1].Kind() != ast.ListKind {
		return c.refuse(expr, messageKind)
	}
	var kinds []string
	for _, element := range args[1].AsList().Elements() {
		kind, isLiteral := getStringLiteral(element)
		if !isLiteral || !isKind(kind) {
			return c.refuse(expr, messageKind)
		}
		kinds = append(kinds, kind)
	}
	term := newTerm(TermKind)
	term.Kinds = kinds
	return c.newLeafNode(expr, term, isNegated)
}

func getInMessage(expr ast.Expr) string {
	if isIdent(expr, variablePath) {
		return messagePath
	}
	return messageCondition
}

func (c *converter) convertStartsWith(expr ast.Expr, isNegated bool) Node {
	call := expr.AsCall()
	switch {
	case isIdent(call.Target(), variableKind):
		return c.refuse(expr, messageKind)
	case !isIdent(call.Target(), variablePath):
		return c.refuse(expr, messageCondition)
	}
	prefix, isLiteral := getStringLiteral(call.Args()[0])
	if !isLiteral {
		return c.refuse(expr, messagePath)
	}
	term := newTerm(TermPath)
	term.Prefix = prefix
	return c.newLeafNode(expr, term, isNegated)
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
