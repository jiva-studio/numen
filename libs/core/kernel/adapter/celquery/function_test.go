package celquery_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/celquery"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const shelfPlugin = "md.test.shelf"

func getShelfCatalog() searchquery.Catalog {
	return searchquery.MergeCatalog(searchquery.GetBuiltinCatalog(), searchquery.ConvertManifestFunctions(shelfPlugin, manifest.SearchFunctions{
		Version: 1, Functions: []manifest.SearchFunction{
			{Name: "shelved", Args: []manifest.SearchFunctionArg{{Name: "shelf", Type: manifest.SearchFunctionArgString}}, Level: manifest.SearchFunctionLevelDocument},
			{Name: "isShelved", Level: manifest.SearchFunctionLevelDocument},
		},
	}))
}

func parseOver(text string, catalog searchquery.Catalog) (searchquery.Query, error) {
	limits := searchquery.DefaultLimits()
	raw, err := celquery.NewParser(limits).ParseQuery(text, catalog)
	if err != nil {
		return searchquery.Query{}, err
	}
	return searchquery.Normalize(raw, limits)
}

func TestAPluginFunctionParsesToOneCallWithItsNamedArguments(t *testing.T) {
	q, err := parseOver(`(shelved("east") && meaning("tide")) || (!shelved("east") && !isShelved() && kind == "pdf")`, getShelfCatalog())
	if err != nil {
		t.Fatal(err)
	}

	if len(q.Calls) != 2 {
		t.Fatalf("calls = %+v, want shelved and isShelved once each", q.Calls)
	}
	shelved := q.Calls[0]
	if shelved.PluginID != shelfPlugin || shelved.Function != "shelved" || !shelved.IsPositive ||
		len(shelved.Arguments) != 1 || shelved.Arguments[0] != (searchquery.Argument{Name: "shelf", Type: searchquery.TypeString, String: "east"}) {
		t.Errorf("shelved = %+v", shelved)
	}
	if shelved.Position != (searchquery.Position{Line: 1, Column: 2}) {
		t.Errorf("shelved is at %+v, want 1:2", shelved.Position)
	}
	if q.Calls[1].Function != "isShelved" || q.Calls[1].IsPositive {
		t.Errorf("isShelved = %+v, want a negated call", q.Calls[1])
	}
	if countLeaves(q.Root, searchquery.LeafFunction) != 3 {
		t.Errorf("the tree has %d function leaves, want 3", countLeaves(q.Root, searchquery.LeafFunction))
	}
}

func TestAFunctionOfAPluginNotInTheCatalogIsUnknown(t *testing.T) {
	_, err := parseOver(`shelved("east")`, searchquery.GetBuiltinCatalog())

	parseError, ok := errors.AsType[*searchquery.ParseError](err)
	if !ok || !hasProblem(parseError.Problems, "unknown function shelved") {
		t.Errorf("err = %v, want an unknown function", err)
	}
}

func TestAPluginFunctionAcceptsOnlyLiteralArgumentsOfItsTypes(t *testing.T) {
	for _, text := range []string{`shelved(1)`, `shelved("a", "b")`, `shelved(kind)`} {
		if _, err := parseOver(text, getShelfCatalog()); !errors.As(err, new(*searchquery.ParseError)) {
			t.Errorf("%s returned %v, want a ParseError", text, err)
		}
	}
}
