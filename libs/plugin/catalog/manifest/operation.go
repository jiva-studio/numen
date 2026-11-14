package manifest

import (
	"fmt"
	"strings"
)

// Operation titles an operation of connection types in words a person reads. Name is lowercase
// words joined by dots; Title is a lowercase verb phrase that completes "This type cannot ...".
type Operation struct {
	Name  string
	Title string
}

func (m Manifest) listOperationIssues() []error {
	var issues []error
	observed := map[string]bool{}
	for _, one := range m.Contributes.Operations {
		switch {
		case !connectionOperationPattern.MatchString(one.Name):
			issues = append(issues, newOperationIssue("%q is not lowercase words joined by dots", one.Name))
		case observed[one.Name]:
			issues = append(issues, newOperationIssue("%q is declared twice", one.Name))
		}
		observed[one.Name] = true
		if strings.TrimSpace(one.Title) == "" {
			issues = append(issues, newOperationIssue("%q has no title", one.Name))
		}
	}
	return issues
}

func newOperationIssue(format string, args ...any) error {
	return fmt.Errorf("%w: operation %s", ErrInvalidContribution, fmt.Sprintf(format, args...))
}
