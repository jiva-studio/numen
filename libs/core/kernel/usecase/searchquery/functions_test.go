package searchquery_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/searchfunction"

	searchquerydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchquery/searchquerytest"
)

const shelfPluginID = "md.test.shelf"

// shelfPlugin answers shelved(shelf) for the paths put on each shelf, and records every call.
type shelfPlugin struct {
	shelves map[string][]string
	warning string
	err     error
	cancel  context.CancelFunc

	mu      sync.Mutex
	finds   []searchquerydomain.FunctionCall
	limits  []int
	matches [][]string
	calls   []searchquerydomain.FunctionCall
}

func (p *shelfPlugin) getHeldPaths(call searchquerydomain.FunctionCall) []string {
	if len(call.Arguments) == 0 {
		var all []string
		for _, paths := range p.shelves {
			all = append(all, paths...)
		}
		slices.Sort(all)
		return slices.Compact(all)
	}
	return p.shelves[call.Arguments[0].String]
}

func (p *shelfPlugin) MatchDocuments(ctx context.Context, call searchquerydomain.FunctionCall, paths []string) (searchquerydomain.FunctionAnswer, error) {
	p.mu.Lock()
	p.matches = append(p.matches, slices.Clone(paths))
	p.calls = append(p.calls, call)
	p.mu.Unlock()
	if p.err != nil {
		return searchquerydomain.FunctionAnswer{}, p.err
	}
	if err := ctx.Err(); err != nil {
		return searchquerydomain.FunctionAnswer{}, err
	}
	held := p.getHeldPaths(call)
	return searchquerydomain.FunctionAnswer{Paths: slices.DeleteFunc(slices.Clone(paths), func(path string) bool { return !slices.Contains(held, path) }), Warning: p.warning}, nil
}

func (p *shelfPlugin) FindDocuments(ctx context.Context, call searchquerydomain.FunctionCall, limit int) (searchquerydomain.FunctionAnswer, error) {
	p.mu.Lock()
	p.finds = append(p.finds, call)
	p.limits = append(p.limits, limit)
	p.calls = append(p.calls, call)
	p.mu.Unlock()
	if p.cancel != nil {
		p.cancel()
	}
	if p.err != nil {
		return searchquerydomain.FunctionAnswer{}, p.err
	}
	if err := ctx.Err(); err != nil {
		return searchquerydomain.FunctionAnswer{}, err
	}
	held := p.getHeldPaths(call)
	return searchquerydomain.FunctionAnswer{Paths: held[:min(limit, len(held))], EstimatedCount: int64(len(held)), Warning: p.warning}, nil
}

func getShelfCatalog() searchquerydomain.Catalog {
	return searchquerydomain.MergeCatalog(searchquerydomain.GetBuiltinCatalog(), searchquerydomain.ConvertManifestFunctions(shelfPluginID, searchfunction.Set{
		Version: 1, Functions: []searchfunction.Function{
			{Name: "shelved", Args: []searchfunction.Arg{{Name: "shelf", Type: searchfunction.ArgString}}, Level: searchfunction.LevelDocument},
			{Name: "isShelved", Level: searchfunction.LevelDocument},
		},
	}))
}

func newShelfWorld(t *testing.T) world {
	t.Helper()
	w := newWorld(t, map[string]string{
		"east/a.md": "The tide comes in over the east rocks.",
		"east/b.md": "No water here, only sand.",
		"west/c.md": "The tide goes out by the west pier.",
		"west/d.md": "A tide table for the west harbor.",
	})
	for _, p := range []string{"east/a.md", "east/b.md", "west/c.md", "west/d.md"} {
		w.cutBySize(t, p, 400)
	}
	return w
}

func (w world) newShelfFind(plugin *shelfPlugin) *searchquery.FindSearchResults {
	deps := w.newDeps(nil)
	deps.GetCatalog = getShelfCatalog
	deps.Evaluate.Functions = plugin
	return searchquerytest.New(deps)
}

func newShelves() *shelfPlugin {
	return &shelfPlugin{shelves: map[string][]string{"east": {"east/a.md", "east/b.md"}, "west": {"west/c.md", "west/d.md"}}}
}

func TestAPositivePluginFunctionFindsTheDocumentsTheWordsAreSearchedIn(t *testing.T) {
	w := newShelfWorld(t)
	plugin := newShelves()
	find := w.newShelfFind(plugin)

	got, err := find.Execute(t.Context(), w.v, searchquerydomain.Request{Text: `shelved("east") && words("tide")`, IndexGeneration: "g7"})
	if err != nil {
		t.Fatal(err)
	}

	if paths := getPaths(got); !slices.Equal(paths, []string{"east/a.md"}) {
		t.Errorf("found %v, want east/a.md", paths)
	}
	if len(plugin.finds) != 1 || len(plugin.matches) != 0 || plugin.limits[0] != find.Limits.MaxDocuments+1 {
		t.Errorf("the plugin was asked to find %d times with limits %v and to match %v, want one find of MaxDocuments+1", len(plugin.finds), plugin.limits, plugin.matches)
	}
	call := plugin.finds[0]
	if call.PluginID != shelfPluginID || call.Function != "shelved" || call.VaultID != string(w.v.ID) || call.IndexGeneration != "g7" {
		t.Errorf("the call is %+v, want the vault and generation of the search", call)
	}
}

func TestANegatedPluginFunctionMatchesTheCandidatePaths(t *testing.T) {
	w := newShelfWorld(t)
	plugin := newShelves()

	got, err := w.newShelfFind(plugin).Execute(t.Context(), w.v, searchquerydomain.Request{Text: `!shelved("east") && words("tide")`})
	if err != nil {
		t.Fatal(err)
	}

	if paths := getPaths(got); !slices.Equal(paths, []string{"west/c.md", "west/d.md"}) {
		t.Errorf("found %v, want the west notes", paths)
	}
	if len(plugin.finds) != 0 || len(plugin.matches) == 0 {
		t.Fatalf("the plugin was asked to find %d times and to match %v, want matches only", len(plugin.finds), plugin.matches)
	}
	for _, batch := range plugin.matches {
		if len(batch) > 1000 {
			t.Errorf("a batch of %d paths, want at most 1000", len(batch))
		}
	}
}

func TestAFoundAnswerAboveTheDocumentLimitFiltersByMatchWithTheSameResults(t *testing.T) {
	w := newShelfWorld(t)
	exact, err := w.newShelfFind(newShelves()).Execute(t.Context(), w.v, searchquerydomain.Request{Text: `shelved("west") && words("tide")`})
	if err != nil {
		t.Fatal(err)
	}
	plugin := newShelves()
	find := w.newShelfFind(plugin)
	find.Limits.MaxDocuments = 1

	got, err := find.Execute(t.Context(), w.v, searchquerydomain.Request{Text: `shelved("west") && words("tide")`})
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(getPaths(got), getPaths(exact)) || !slices.Equal(getPaths(got), []string{"west/c.md", "west/d.md"}) {
		t.Errorf("found %v, want %v", getPaths(got), getPaths(exact))
	}
	if len(plugin.finds) != 1 || len(plugin.matches) == 0 {
		t.Errorf("the plugin was asked to find %d times and to match %v, want a find, then matches", len(plugin.finds), plugin.matches)
	}
}

func TestAQueryOfPluginFunctionsOnlyReturnsWholeFiles(t *testing.T) {
	w := newShelfWorld(t)

	got, err := w.newShelfFind(newShelves()).Execute(t.Context(), w.v, searchquerydomain.Request{Text: `isShelved() && !shelved("east")`})
	if err != nil {
		t.Fatal(err)
	}

	if paths := getPaths(got); !slices.Equal(paths, []string{"west/c.md", "west/d.md"}) {
		t.Errorf("found %v, want the west notes", paths)
	}
	for _, r := range got.Results {
		if len(r.Chunks) != 0 {
			t.Errorf("%s has chunks %v, want a whole file", r.Document.Path, r.Chunks)
		}
	}
}

func TestAWarningOfAPluginFunctionIsAQueryWarningOnceWithTheFunctionName(t *testing.T) {
	w := newShelfWorld(t)
	plugin := newShelves()
	plugin.warning = "the shelves may be out of date"

	got, err := w.newShelfFind(plugin).Execute(t.Context(), w.v, searchquerydomain.Request{Text: `shelved("west") && !shelved("east") && words("tide")`})
	if err != nil {
		t.Fatal(err)
	}

	want := "shelved: the shelves may be out of date"
	if count := slices.Index(got.Warnings, want); count < 0 || slices.Index(got.Warnings[count+1:], want) >= 0 {
		t.Errorf("warnings = %v, want %q once", got.Warnings, want)
	}
}

func TestInvalidArgumentsOfAPluginFunctionAreAProblemAtTheCall(t *testing.T) {
	w := newShelfWorld(t)
	plugin := newShelves()
	plugin.err = fmt.Errorf("%w: shelved needs a shelf", searchquerydomain.ErrInvalidFunctionArguments)

	_, err := w.newShelfFind(plugin).Execute(t.Context(), w.v, searchquerydomain.Request{Text: `words("tide") &&
  shelved("")`})

	parseError, ok := errors.AsType[*searchquerydomain.ParseError](err)
	if !ok || len(parseError.Problems) != 1 || parseError.Problems[0].Position != (searchquerydomain.Position{Line: 2, Column: 3}) ||
		!strings.Contains(parseError.Problems[0].Message, "shelved needs a shelf") {
		t.Fatalf("err = %v, want a problem at 2:3", err)
	}
}

func TestAFailedPluginFunctionFailsTheQueryNamingTheFunction(t *testing.T) {
	w := newShelfWorld(t)
	plugin := newShelves()
	plugin.err = errors.New("plugin md.test.shelf is unavailable")

	_, err := w.newShelfFind(plugin).Execute(t.Context(), w.v, searchquerydomain.Request{Text: `shelved("east")`})

	if err == nil || !strings.Contains(err.Error(), "the function shelved failed") || !strings.Contains(err.Error(), "md.test.shelf") {
		t.Errorf("err = %v, want the function and the plugin named", err)
	}
}

func TestACanceledSearchEndsThePluginCallWithTheContextError(t *testing.T) {
	w := newShelfWorld(t)
	ctx, cancel := context.WithCancel(t.Context())
	plugin := newShelves()
	plugin.cancel = cancel

	_, err := w.newShelfFind(plugin).Execute(ctx, w.v, searchquerydomain.Request{Text: `shelved("east")`})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}
