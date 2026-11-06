// Package when reads the conditions of when clauses: the keys of libs/protocol/when.json combined
// with !, && and || and parentheses; ! binds tightest, then &&, then ||.
package when

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// Operator is what a node of a condition does with its operands.
type Operator string

const (
	// OperatorKey is a key; the node has no operands.
	OperatorKey Operator = "key"
	// OperatorNot negates its one operand.
	OperatorNot Operator = "!"
	// OperatorAnd is true where both operands are.
	OperatorAnd Operator = "&&"
	// OperatorOr is true where either operand is.
	OperatorOr Operator = "||"
)

// Expression is a condition read into a tree.
type Expression struct {
	Operator Operator
	// Key is the key of an OperatorKey node.
	Key      string
	Operands []Expression
}

var errSyntax = errors.New("not an expression of keys, !, &&, || and parentheses")

// Parse reads the condition into a tree; it fails on a malformed condition and on a key outside the
// list of when.json.
func Parse(text string) (Expression, error) {
	tokens, err := split(text)
	if err != nil {
		return Expression{}, err
	}
	reader := parser{tokens: tokens}
	expression, err := reader.readOr()
	if err != nil {
		return Expression{}, err
	}
	if reader.next < len(tokens) {
		return Expression{}, fmt.Errorf("%w: %q after the end", errSyntax, tokens[reader.next])
	}
	return expression, nil
}

// split splits the condition into keys and the operators &&, ||, !, ( and ).
func split(text string) ([]string, error) {
	var tokens []string
	for start := 0; start < len(text); {
		switch char := rune(text[start]); {
		case unicode.IsSpace(char):
			start++
		case strings.HasPrefix(text[start:], "&&"), strings.HasPrefix(text[start:], "||"):
			tokens = append(tokens, text[start:start+2])
			start += 2
		case char == '!' || char == '(' || char == ')':
			tokens = append(tokens, string(char))
			start++
		case unicode.IsLetter(char):
			end := start
			for end < len(text) && (unicode.IsLetter(rune(text[end])) || unicode.IsDigit(rune(text[end]))) {
				end++
			}
			tokens = append(tokens, text[start:end])
			start = end
		default:
			return nil, fmt.Errorf("%w: %q", errSyntax, text[start:])
		}
	}
	return tokens, nil
}

type parser struct {
	tokens []string
	next   int
}

func (p *parser) peek() string {
	if p.next < len(p.tokens) {
		return p.tokens[p.next]
	}
	return ""
}

func (p *parser) readOr() (Expression, error) {
	return p.readBinary(OperatorOr, p.readAnd)
}

func (p *parser) readAnd() (Expression, error) {
	return p.readBinary(OperatorAnd, p.readUnary)
}

// readBinary reads operands joined by the operator, grouped from the left.
func (p *parser) readBinary(operator Operator, readOperand func() (Expression, error)) (Expression, error) {
	expression, err := readOperand()
	if err != nil {
		return Expression{}, err
	}
	for p.peek() == string(operator) {
		p.next++
		right, err := readOperand()
		if err != nil {
			return Expression{}, err
		}
		expression = Expression{Operator: operator, Operands: []Expression{expression, right}}
	}
	return expression, nil
}

func (p *parser) readUnary() (Expression, error) {
	token := p.peek()
	p.next++
	switch {
	case token == "!":
		operand, err := p.readUnary()
		if err != nil {
			return Expression{}, err
		}
		return Expression{Operator: OperatorNot, Operands: []Expression{operand}}, nil
	case token == "(":
		inner, err := p.readOr()
		if err != nil {
			return Expression{}, err
		}
		if p.peek() != ")" {
			return Expression{}, fmt.Errorf("%w: a parenthesis is not closed", errSyntax)
		}
		p.next++
		return inner, nil
	case token == "":
		return Expression{}, fmt.Errorf("%w: it ends where a key is expected", errSyntax)
	case IsKey(token):
		return Expression{Operator: OperatorKey, Key: token}, nil
	case unicode.IsLetter(rune(token[0])):
		return Expression{}, fmt.Errorf("%q is none of the keys %s", token, keyList)
	}
	return Expression{}, fmt.Errorf("%w: %q where a key is expected", errSyntax, token)
}
