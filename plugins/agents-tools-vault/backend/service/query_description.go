package service

import (
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
)

const (
	queryDescriptionHead = "Run a query of the search language over the index of all files in the vault: notes, books, PDFs, " +
		"scans and recordings. A query combines conditions with && (both), || (either) and ! (not), and stages " +
		"after a dot order or cut the results, for example " +
		"((meaning(\"how a river changes its course\") || words(\"meander\")) && kind == \"pdf\").rerank().top(10). " +
		"Returns a flat list of passages and whole files, best first, at most 20 a page. " +
		"isRunning true means the query runs on: call library_query_results with the id. " +
		"A query that is not valid returns errors with the line and column of each problem."
	queryFunctionsHead = "Functions and variables:"
	queryStagesHead    = "Stages:"
)

func createQueryDescription(functions host.SearchFunctions) string {
	var conditions, stages []string
	for _, f := range functions.Functions {
		line := "- " + strings.Join(f.Signatures, ", ") + ": " + f.Description + " Example: " + f.Example
		if f.Kind == "stage" {
			stages = append(stages, line)
			continue
		}
		conditions = append(conditions, line)
	}
	parts := []string{queryDescriptionHead}
	if len(conditions) > 0 {
		parts = append(parts, queryFunctionsHead+"\n"+strings.Join(conditions, "\n"))
	}
	if len(stages) > 0 {
		parts = append(parts, queryStagesHead+"\n"+strings.Join(stages, "\n"))
	}
	return strings.Join(parts, "\n")
}
