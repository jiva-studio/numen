package celquery

import (
	"fmt"
	"regexp"
	"strings"

	"cel.dev/cel-go/common"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
)

const (
	messageArguments = "%s takes %s"
	messageNoArgs    = "%s takes nothing in its parentheses"
	messageOperator  = "The operator %s is not allowed in a query"
	messageOperands  = "The values on both sides of %s do not fit together"
	messageWrongType = "This part of the query does not fit here"
)

var (
	noMatchingOverload  = regexp.MustCompile(`found no matching overload for '([^']+)'`)
	undeclaredReference = regexp.MustCompile(`undeclared reference to '([^']+)'`)
)

func newCheckProblems(source queryText, errs []*common.Error, catalog searchquery.Catalog) []searchquery.Problem {
	problems := make([]searchquery.Problem, len(errs))
	for i, err := range errs {
		offset := source.findNameStart(source.getOffset(err.Location))
		problems[i] = searchquery.Problem{Position: source.getPosition(offset), Message: getCheckMessage(err.Message, catalog)}
	}
	return problems
}

func getCheckMessage(message string, catalog searchquery.Catalog) string {
	if match := undeclaredReference.FindStringSubmatch(message); match != nil {
		if isOperator(match[1]) {
			return fmt.Sprintf(messageOperator, getOperatorText(match[1]))
		}
		return "Unknown name " + match[1]
	}
	match := noMatchingOverload.FindStringSubmatch(message)
	if match == nil {
		return messageWrongType
	}
	if isOperator(match[1]) {
		return fmt.Sprintf(messageOperands, getOperatorText(match[1]))
	}
	f, ok := catalog.FindFunction(match[1])
	if !ok {
		return messageWrongType
	}
	if len(f.Args) == 0 {
		return fmt.Sprintf(messageNoArgs, f.Name)
	}
	args := make([]string, len(f.Args))
	for i, arg := range f.Args {
		args[i] = getTypeText(arg)
	}
	text := strings.Join(args, " and ")
	if len(f.OptionalArgs) > 0 {
		optional := make([]string, len(f.OptionalArgs))
		for i, arg := range f.OptionalArgs {
			optional[i] = getTypeText(arg)
		}
		text += ", and optionally " + strings.Join(optional, " and ")
	}
	return fmt.Sprintf(messageArguments, f.Name, text)
}

func getOperatorText(name string) string {
	return strings.Trim(name, "_@")
}

func getTypeText(t searchquery.ValueType) string {
	switch t {
	case searchquery.TypeString:
		return "one text in quotes"
	case searchquery.TypeInt:
		return "one number"
	case searchquery.TypeStringList:
		return "one list of texts in quotes"
	case searchquery.TypeBool, searchquery.TypeSearchResults:
		return "one condition"
	}
	return "one value"
}
