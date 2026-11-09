// Package when reads and evaluates the when conditions of commands and slot entries: CEL expressions
// over the variables of libs/core/protocol/when.json, such as target.isFolder && !target.isMultiple.
package when

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/types"
)

// kind is the type of a variable.
type kind int

const (
	kindBool kind = iota
	kindString
	kindMap
	kindList
)

// variable is one variable a condition may name.
type variable struct {
	name string
	kind kind
}

func (k kind) getType() *cel.Type {
	switch k {
	case kindBool:
		return cel.BoolType
	case kindString:
		return cel.StringType
	case kindMap:
		return cel.MapType(cel.StringType, cel.DynType)
	case kindList:
		return cel.ListType(cel.MapType(cel.StringType, cel.DynType))
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

func newEnvironment() (*cel.Env, error) {
	options := make([]cel.EnvOption, 0, len(listVariables()))
	for _, one := range listVariables() {
		options = append(options, cel.Variable(one.name, one.kind.getType()))
	}
	return cel.NewEnv(options...)
}

// Conditions reads conditions over one CEL environment, made on its first use, and keeps the condition
// read from each text, so a text is compiled once. It is safe to use from several goroutines.
type Conditions struct {
	getEnvironment func() (*cel.Env, error)

	mu       sync.Mutex
	compiled map[string]cacheEntry
}

// cacheEntry is the condition read from a text, or the error of reading it.
type cacheEntry struct {
	condition Condition
	err       error
}

// NewConditions makes a Conditions that has read no condition.
func NewConditions() *Conditions {
	return &Conditions{getEnvironment: sync.OnceValues(newEnvironment), compiled: map[string]cacheEntry{}}
}

// Parse reads the condition as the package's Parse does, and returns the condition read before for a
// text it has read.
func (c *Conditions) Parse(text string) (Condition, error) {
	if text == "" {
		return Condition{}, nil
	}
	c.mu.Lock()
	one, isCompiled := c.compiled[text]
	c.mu.Unlock()
	if isCompiled {
		return one.condition, one.err
	}
	environment, err := c.getEnvironment()
	if err != nil {
		return Condition{}, fmt.Errorf("the variables of when: %w", err)
	}
	condition, err := compile(environment, text)
	c.mu.Lock()
	c.compiled[text] = cacheEntry{condition: condition, err: err}
	c.mu.Unlock()
	return condition, err
}

// Parse reads the condition. It fails on an expression that is malformed, names a variable that is
// not declared, mixes types or gives no true or false. The empty text is a condition that always holds.
// It makes a new environment for the one text; a caller that evaluates conditions reads them through
// Conditions.
func Parse(text string) (Condition, error) {
	return NewConditions().Parse(text)
}

func compile(environment *cel.Env, text string) (Condition, error) {
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

// ListNames lists the variables the condition names, sorted.
func (c Condition) ListNames() []string {
	return slices.Clone(c.names)
}

// HasItems reports whether the condition names a variable of a text version: its command offers one
// item for each version of a file over which the condition holds.
func (c Condition) HasItems() bool {
	return c.hasVariablePrefix("version.")
}

// HasTargetVariable reports whether the condition names a variable of the target.
func (c Condition) HasTargetVariable() bool {
	return c.hasVariablePrefix("target.")
}

func (c Condition) hasVariablePrefix(prefix string) bool {
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
