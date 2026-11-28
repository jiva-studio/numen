package manifest

import (
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/catalog/searchfunction"
)

func (m Manifest) listSearchFunctionIssues() []error {
	issues := searchfunction.ListIssues(m.Contributes.SearchFunctions)
	for i, issue := range issues {
		issues[i] = fmt.Errorf("%w: %w", ErrInvalidContribution, issue)
	}
	return issues
}
