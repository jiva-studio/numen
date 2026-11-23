package cli

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
)

const queryUsage = "usage: numen-cli query [--limit N] <vault> <query>"

// errInvalidQuery is a query the command reported the problems of.
var errInvalidQuery = errors.New("the query is not valid")

type queryCommand struct {
	session Session
}

func (queryCommand) Name() string { return "query" }

func (queryCommand) Synopsis() string {
	return "  numen-cli query [--limit N] <vault> <query>  run a search query within one vault"
}

func (c queryCommand) Run(ctx context.Context, term terminal, args []string) error {
	request, vaultName, err := parseQueryArgs(args)
	if err != nil {
		return err
	}
	diagnostics := newReporter(term.errOut)
	errorHandler := func(err error) { diagnostics.WriteFormat("warning: %v\n", err) }
	indexed, err := c.session.OpenIndexed(ctx, IndexOptions{ErrorHandler: errorHandler})
	if err != nil {
		return err
	}
	results, err := indexed.Query.ExecuteByName(ctx, vaultName, request)
	if parseError, ok := errors.AsType[*searchquery.ParseError](err); ok {
		for _, p := range parseError.Problems {
			diagnostics.WriteFormat("query:%s\n", searchquery.FormatProblem(p))
		}
		return errors.Join(errInvalidQuery, diagnostics.Err())
	}
	if err != nil {
		return formatVaultError(err)
	}
	for _, warning := range results.Warnings {
		diagnostics.WriteFormat("warning: %s\n", warning)
	}
	for _, report := range results.Reranks {
		diagnostics.WriteLine(formatRerankReport(report))
	}
	r := newReporter(term.out)
	printResults(r, results)
	return errors.Join(r.Err(), diagnostics.Err())
}

func parseQueryArgs(args []string) (searchquery.Request, string, error) {
	request := searchquery.Request{Priority: modelcall.PriorityInteractive}
	if len(args) > 0 && args[0] == "--limit" {
		if len(args) < 2 {
			return request, "", errors.New(queryUsage)
		}
		limit, err := strconv.Atoi(args[1])
		if err != nil || limit <= 0 {
			return request, "", fmt.Errorf("--limit takes a positive count of results, got %q; %s", args[1], queryUsage)
		}
		request.Limit = limit
		args = args[2:]
	}
	if len(args) != 2 {
		return request, "", errors.New(queryUsage)
	}
	request.Text = args[1]
	return request, args[0], nil
}

func printResults(r *reporter, results searchquery.SearchResults) {
	if len(results.Results) == 0 {
		r.WriteLine("nothing found")
		return
	}
	for _, one := range results.Results {
		line := fmt.Sprintf("%-5s  %s  %s  %s", formatScore(one.Score), one.Document.Path, one.Document.Kind, getPlace(one))
		if one.IsWholeFile() {
			r.WriteLine(line)
			continue
		}
		r.WriteFormat("%s  meaning %s  words %s\n    %s\n", line, formatScore(one.MeaningScore), formatScore(one.WordScore), getExcerpt(one.Text))
	}
}

func formatScore(score *float64) string {
	if score == nil {
		return "-"
	}
	return strconv.FormatFloat(searchquery.Round(*score), 'f', 3, 64)
}

func getPlace(r searchquery.SearchResult) string {
	switch {
	case r.IsWholeFile():
		return "whole file"
	case r.PlaceLabel != "":
		return r.PlaceLabel
	}
	return "line " + strconv.Itoa(r.MatchLine+1)
}

func formatRerankReport(report searchquery.RerankReport) string {
	options := make([]string, 0, len(report.Options))
	for _, key := range slices.Sorted(maps.Keys(report.Options)) {
		options = append(options, key+"="+report.Options[key])
	}
	return fmt.Sprintf("rerank: %s (%s) sent %d, reused %d", report.ConnectionName, strings.Join(options, ", "), report.SentCount, report.ReusedCount)
}
