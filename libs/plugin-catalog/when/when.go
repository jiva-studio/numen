// Package when reads and evaluates the when conditions of commands and slot entries: CEL expressions
// over the variables of libs/protocol/when.json, such as target.isFolder && !target.isMultiple.
package when

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/types"
)

// kind is the type of a variable.
type kind int

const (
	kindBool kind = iota
	kindString
	kindMap
)

// variable is one variable a condition may name.
type variable struct {
	name string
	kind kind
}

func (k kind) getType() *cel.Type {
	switch k {
	case kindString:
		return cel.StringType
	case kindMap:
		return cel.MapType(cel.StringType, cel.DynType)
	}
	return cel.BoolType
}

// Result is the value of a condition.
type Result int

const (
	// False is a condition that does not hold.
	False Result = iota
	// True is a condition that holds.
	True
	// Unknown is a condition whose value depends on a variable that was not given.
	Unknown
)

// Condition is a when condition read once and evaluated over many values.
type Condition struct {
	program cel.Program
	names   []string
}

var errNotCondition = errors.New("the expression gives no true or false")

// Parse reads the condition. It fails on an expression that is malformed, names a variable that is
// not declared, mixes types or gives no true or false. The empty text is a condition that always holds.
func Parse(text string) (Condition, error) {
	if text == "" {
		return Condition{}, nil
	}
	options := make([]cel.EnvOption, 0, len(listVariables()))
	for _, one := range listVariables() {
		options = append(options, cel.Variable(one.name, one.kind.getType()))
	}
	environment, err := cel.NewEnv(options...)
	if err != nil {
		return Condition{}, fmt.Errorf("the variables of when: %w", err)
	}
	checked, issues := environment.Compile(text)
	if issues.Err() != nil {
		return Condition{}, fmt.Errorf("%q: %w", text, issues.Err())
	}
	if !checked.OutputType().IsExactType(cel.BoolType) {
		return Condition{}, fmt.Errorf("%q: %w", text, errNotCondition)
	}
	program, err := environment.Program(checked, cel.EvalOptions(cel.OptPartialEval))
	if err != nil {
		return Condition{}, fmt.Errorf("%q: %w", text, err)
	}
	var names []string
	for _, reference := range checked.NativeRep().ReferenceMap() {
		if reference.Name != "" && !slices.Contains(names, reference.Name) {
			names = append(names, reference.Name)
		}
	}
	slices.Sort(names)
	return Condition{program: program, names: names}, nil
}

// Names gives the variables the condition names, sorted.
func (c Condition) Names() []string {
	return slices.Clone(c.names)
}

// HasItems reports whether the condition names a variable of a text version: its command offers one
// item for each version of a file over which the condition holds.
func (c Condition) HasItems() bool {
	return c.isNaming("version.")
}

// NamesTarget reports whether the condition names a variable of the target.
func (c Condition) NamesTarget() bool {
	return c.isNaming("target.")
}

func (c Condition) isNaming(prefix string) bool {
	return slices.ContainsFunc(c.names, func(name string) bool { return strings.HasPrefix(name, prefix) })
}

// Evaluate gives the value of the condition over the values, by the qualified names of the variables,
// such as "target.isNote", and "config" for the map of the settings. A variable the values do not give
// makes the condition Unknown where its value decides it. A setting config does not hold makes it False.
func (c Condition) Evaluate(values map[string]any) Result {
	if c.program == nil {
		return True
	}
	activation := map[string]any{}
	var unknown []*cel.AttributePatternType
	for _, name := range c.names {
		if value, isGiven := values[name]; isGiven {
			activation[name] = value
		} else {
			unknown = append(unknown, cel.AttributePattern(name))
		}
	}
	partial, err := cel.PartialVars(activation, unknown...)
	if err != nil {
		return Unknown
	}
	out, _, err := c.program.Eval(partial)
	switch {
	case out != nil && types.IsUnknown(out):
		return Unknown
	case err != nil:
		return False
	case out.Value() == true:
		return True
	}
	return False
}
