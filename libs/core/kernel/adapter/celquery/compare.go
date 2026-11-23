package celquery

import (
	"cel.dev/cel-go/common/ast"
	"cel.dev/cel-go/common/overloads"
	"cel.dev/cel-go/common/types"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
)

func getAttribute(name string) (searchquery.Attribute, bool) {
	switch name {
	case searchquery.VariableTitle:
		return searchquery.AttributeTitle, true
	case searchquery.VariableNoteType:
		return searchquery.AttributeNoteType, true
	case searchquery.VariableModified:
		return searchquery.AttributeModified, true
	case searchquery.VariableSize:
		return searchquery.AttributeSize, true
	}
	return 0, false
}

func reverseComparison(comparison searchquery.Comparison) searchquery.Comparison {
	switch comparison {
	case searchquery.ComparisonLess:
		return searchquery.ComparisonGreater
	case searchquery.ComparisonLessOrEqual:
		return searchquery.ComparisonGreaterOrEqual
	case searchquery.ComparisonGreater:
		return searchquery.ComparisonLess
	case searchquery.ComparisonGreaterOrEqual:
		return searchquery.ComparisonLessOrEqual
	default:
		return comparison
	}
}

func (c *comparer) convertComparison(expr ast.Expr, comparison searchquery.Comparison) searchquery.RawNode {
	args := expr.AsCall().Args()
	subject, other := args[0], args[1]
	if isValueExpr(subject) && !isValueExpr(other) {
		subject, other = other, subject
		comparison = reverseComparison(comparison)
	}
	switch name := getIdentName(subject); {
	case name == searchquery.VariableKind:
		return c.convertKind(expr, other, comparison)
	case name == searchquery.VariablePath:
		return c.reject(expr, messagePath)
	case name != "":
		return c.convertAttribute(expr, name, comparison, other)
	case isCallOf(subject, searchquery.FunctionProperty):
		return c.convertProperty(expr, subject, comparison, []ast.Expr{other})
	}
	return c.reject(expr, messageCondition)
}

func (c *comparer) convertKind(expr, other ast.Expr, comparison searchquery.Comparison) searchquery.RawNode {
	kind, isLiteral := getStringLiteral(other)
	if !isLiteral || comparison != searchquery.ComparisonEqual {
		return c.reject(expr, messageKind)
	}
	return c.newLeaf(expr, searchquery.RawLeaf{Kind: searchquery.LeafKindIn, Kinds: []string{kind}})
}

func (c *comparer) convertAttribute(expr ast.Expr, name string, comparison searchquery.Comparison, other ast.Expr) searchquery.RawNode {
	attribute, isAttribute := getAttribute(name)
	if !isAttribute {
		return c.reject(expr, messageCondition)
	}
	value, isValue := c.getValue(other)
	if !isValue || !isAttributeOperand(attribute, comparison, value) {
		return c.reject(expr, getIdentMessage(name))
	}
	return c.newLeaf(expr, searchquery.RawLeaf{
		Kind: searchquery.LeafCompare, Attribute: attribute, Comparison: comparison, Operands: []searchquery.Value{value},
	})
}

func isAttributeOperand(attribute searchquery.Attribute, comparison searchquery.Comparison, value searchquery.Value) bool {
	switch attribute {
	case searchquery.AttributeTitle, searchquery.AttributeNoteType:
		return value.Kind == searchquery.ValueString && comparison == searchquery.ComparisonEqual
	case searchquery.AttributeModified:
		return value.Kind == searchquery.ValueDate
	case searchquery.AttributeSize:
		return value.Kind == searchquery.ValueInt
	}
	return false
}

func (c *comparer) convertProperty(expr, call ast.Expr, comparison searchquery.Comparison, others []ast.Expr) searchquery.RawNode {
	key, isLiteral := getStringLiteral(call.AsCall().Args()[0])
	if !isLiteral {
		return c.reject(call.AsCall().Args()[0], messageProperty)
	}
	operands := make([]searchquery.Value, len(others))
	for i, other := range others {
		value, isValue := c.getValue(other)
		if !isValue {
			return c.reject(expr, messageProperty)
		}
		operands[i] = value
	}
	return c.newLeaf(call.AsCall().Args()[0], searchquery.RawLeaf{
		Kind: searchquery.LeafSet, Function: searchquery.FunctionProperty, Key: key, Comparison: comparison, Operands: operands,
	})
}

func (c *comparer) convertPresence(expr ast.Expr) searchquery.RawNode {
	call := expr.AsCall()
	text, isLiteral := getStringLiteral(call.Args()[0])
	if !isLiteral {
		return c.reject(call.Args()[0], "hasProperty and hasHeading take a string literal")
	}
	leaf := searchquery.RawLeaf{Kind: searchquery.LeafSet, Function: call.FunctionName(), Comparison: searchquery.ComparisonPresent}
	if call.FunctionName() == searchquery.FunctionHasHeading {
		leaf.Text = text
	} else {
		leaf.Key = text
	}
	return c.newLeaf(call.Args()[0], leaf)
}

func (c *comparer) convertIn(expr ast.Expr) searchquery.RawNode {
	args := expr.AsCall().Args()
	element, list := args[0], args[1]
	switch {
	case isCallOf(list, searchquery.FunctionProperty):
		return c.convertProperty(expr, list, searchquery.ComparisonContains, []ast.Expr{element})
	case list.Kind() != ast.ListKind:
		return c.reject(expr, getIdentMessage(getIdentName(element)))
	case isCallOf(element, searchquery.FunctionProperty):
		return c.convertProperty(expr, element, searchquery.ComparisonIn, list.AsList().Elements())
	case isIdent(element, searchquery.VariableKind):
		return c.convertKindList(expr, list)
	case isIdent(element, searchquery.VariableTitle), isIdent(element, searchquery.VariableNoteType):
		return c.convertTextList(expr, element, list)
	}
	return c.reject(expr, getIdentMessage(getIdentName(element)))
}

func (c *comparer) convertKindList(expr, list ast.Expr) searchquery.RawNode {
	var kinds []string
	for _, element := range list.AsList().Elements() {
		kind, isLiteral := getStringLiteral(element)
		if !isLiteral {
			return c.reject(expr, messageKind)
		}
		kinds = append(kinds, kind)
	}
	return c.newLeaf(expr, searchquery.RawLeaf{Kind: searchquery.LeafKindIn, Kinds: kinds})
}

func (c *comparer) convertTextList(expr, element, list ast.Expr) searchquery.RawNode {
	var operands []searchquery.Value
	for _, item := range list.AsList().Elements() {
		text, isLiteral := getStringLiteral(item)
		if !isLiteral {
			return c.reject(expr, getIdentMessage(element.AsIdent()))
		}
		operands = append(operands, searchquery.Value{Kind: searchquery.ValueString, Text: text})
	}
	return c.newLeaf(expr, searchquery.RawLeaf{
		Kind: searchquery.LeafCompare, Attribute: getElementAttribute(element), Comparison: searchquery.ComparisonIn, Operands: operands,
	})
}

func getMethodComparison(name string) searchquery.Comparison {
	switch name {
	case overloads.EndsWith:
		return searchquery.ComparisonEndsWith
	case overloads.Contains:
		return searchquery.ComparisonSubstring
	}
	return searchquery.ComparisonStartsWith
}

func (c *comparer) convertMethod(expr ast.Expr) searchquery.RawNode {
	call := expr.AsCall()
	target := call.Target()
	text, isLiteral := getStringLiteral(call.Args()[0])
	switch {
	case isIdent(target, searchquery.VariablePath) && call.FunctionName() == overloads.StartsWith:
		if !isLiteral {
			return c.reject(expr, messagePath)
		}
		return c.newLeaf(expr, searchquery.RawLeaf{Kind: searchquery.LeafPathPrefix, Prefix: text})
	case isIdent(target, searchquery.VariableTitle):
		if !isLiteral {
			return c.reject(expr, messageTitle)
		}
		return c.newLeaf(expr, searchquery.RawLeaf{
			Kind: searchquery.LeafCompare, Attribute: searchquery.AttributeTitle, Comparison: getMethodComparison(call.FunctionName()),
			Operands: []searchquery.Value{{Kind: searchquery.ValueString, Text: text}},
		})
	case isCallOf(target, searchquery.FunctionProperty):
		return c.reject(expr, messageProperty)
	}
	return c.reject(expr, getIdentMessage(getIdentName(target)))
}

func (c *comparer) getValue(expr ast.Expr) (searchquery.Value, bool) {
	if isCallOf(expr, searchquery.FunctionDate) {
		arg := expr.AsCall().Args()[0]
		text, isLiteral := getStringLiteral(arg)
		if !isLiteral {
			c.reject(arg, messageDate)
			return searchquery.Value{Kind: searchquery.ValueDate}, true
		}
		return searchquery.Value{Kind: searchquery.ValueDate, Text: text, Position: c.getPosition(arg)}, true
	}
	if expr.Kind() != ast.LiteralKind {
		return searchquery.Value{}, false
	}
	switch v := expr.AsLiteral().(type) {
	case types.String:
		return searchquery.Value{Kind: searchquery.ValueString, Text: string(v)}, true
	case types.Int:
		return searchquery.Value{Kind: searchquery.ValueInt, Int: int64(v)}, true
	case types.Double:
		return searchquery.Value{Kind: searchquery.ValueDouble, Double: float64(v)}, true
	case types.Bool:
		return searchquery.Value{Kind: searchquery.ValueBool, Bool: bool(v)}, true
	case types.Null:
		return searchquery.Value{Kind: searchquery.ValueNull}, true
	}
	return searchquery.Value{}, false
}

func isValueExpr(expr ast.Expr) bool {
	return expr.Kind() == ast.LiteralKind || isCallOf(expr, searchquery.FunctionDate)
}

func getElementAttribute(element ast.Expr) searchquery.Attribute {
	attribute, _ := getAttribute(element.AsIdent())
	return attribute
}
