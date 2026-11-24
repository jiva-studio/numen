package manifest

import (
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/catalog/searchfunction"
)

type (
	SearchFunctions       = searchfunction.Set
	SearchFunction        = searchfunction.Function
	SearchFunctionArg     = searchfunction.Arg
	SearchFunctionLevel   = searchfunction.Level
	SearchFunctionArgType = searchfunction.ArgType
)

const (
	SearchFunctionLevelDocument = searchfunction.LevelDocument
	SearchFunctionLevelChunk    = searchfunction.LevelChunk
	SearchFunctionArgString     = searchfunction.ArgString
	SearchFunctionArgInt        = searchfunction.ArgInt
	SearchFunctionArgBool       = searchfunction.ArgBool
)

func (m Manifest) listSearchFunctionIssues() []error {
	issues := searchfunction.ListIssues(m.Contributes.SearchFunctions)
	for i, issue := range issues {
		issues[i] = fmt.Errorf("%w: %w", ErrInvalidContribution, issue)
	}
	return issues
}
