package catalog

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/plugintest"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/model"
)

// listerFunc is the outer edge of the catalog: the program that lists the models.
type listerFunc func(ctx context.Context, command, env []string) ([]model.Model, error)

func (f listerFunc) ListModels(ctx context.Context, command, env []string) ([]model.Model, error) {
	return f(ctx, command, env)
}

func newCatalog(t *testing.T, lister listerFunc, isAsync bool, extra ...cache.Option) *Catalog {
	t.Helper()
	c, err := New(lister, func(err error) { t.Logf("the catalog fell back: %v", err) }, isAsync, extra...)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func neverCalled(t *testing.T) listerFunc {
	t.Helper()
	return func(context.Context, []string, []string) ([]model.Model, error) {
		t.Error("the program was asked for its models")
		return nil, errors.New("not expected")
	}
}

func readFixtureModels(t *testing.T) []model.Model {
	t.Helper()
	raw, err := os.ReadFile("../../model/testdata/initialize-response.json")
	if err != nil {
		t.Fatal(err)
	}
	list, err := model.Parse(strings.NewReader(string(raw)), "r1")
	if err != nil {
		t.Fatal(err)
	}
	return list
}

func TestNewRefusesAMissingLister(t *testing.T) {
	if _, err := New(nil, func(error) {}, false); err == nil {
		t.Error("a catalog without a lister was created")
	}
}

func TestDepthLimitedToModelLevels(t *testing.T) {
	c := newCatalog(t, neverCalled(t), false)
	c.lists.Put("claude", readFixtureModels(t))
	cases := []struct{ model, requestedLevel, want string }{
		{"claude-sonnet-4-6", "xhigh", ""},
		{"claude-sonnet-4-6", "max", "max"},
		{"claude-haiku-4-5-20251001", "high", ""},
		{"", "high", "high"},
		{"unlisted", "high", "high"},
		{"unlisted", "bogus", ""},
	}
	for _, one := range cases {
		if got := c.GetReasoning(one.model, one.requestedLevel); got != one.want {
			t.Errorf("%q requested %q got %q, want %q", one.model, one.requestedLevel, got, one.want)
		}
	}
}

func TestAnEmptyModelNameIsTheDefaultModel(t *testing.T) {
	c := newCatalog(t, neverCalled(t), false)
	c.lists.Put("claude", []model.Model{
		{Value: model.DefaultName, SupportsEffort: true, EffortLevels: []string{"low", "high"}},
	})
	if got := c.GetReasoning("", "xhigh"); got != "" {
		t.Errorf("GetReasoning(\"\", \"xhigh\") = %q, want empty when default model lacks xhigh", got)
	}
}

func TestCachedListIsNotRequeried(t *testing.T) {
	c := newCatalog(t, neverCalled(t), false)
	c.lists.Put("x", []model.Model{{Value: "m"}})
	got, err := c.Get(t.Context(), []string{"x"}, nil)
	if err != nil || len(got) != 1 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestASynchronousCatalogReturnsTheErrorOfTheProgram(t *testing.T) {
	c := newCatalog(t, func(context.Context, []string, []string) ([]model.Model, error) {
		return nil, errors.New("sync fetch failed")
	}, false)
	if _, err := c.Get(t.Context(), []string{"claude"}, nil); err == nil {
		t.Error("expected Get to return the synchronous error")
	}
}

func TestTheProgramIsAskedWithTheCommandAndTheEnvironment(t *testing.T) {
	var asked []string
	c := newCatalog(t, func(_ context.Context, command, env []string) ([]model.Model, error) {
		asked = append(append([]string{}, command...), env...)
		return []model.Model{{Value: "m"}}, nil
	}, false)
	if _, err := c.Get(t.Context(), []string{"claude", "--extra"}, []string{"A=1"}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(asked, " ") != "claude --extra A=1" {
		t.Errorf("the program was asked %v", asked)
	}
}

// The load runs in the background and fails when the test lets it. The cache reports the failure
// to its error handler after it has recorded it, so waiting for the report is waiting for the
// failure.
func TestAsyncReturnsEmptyThenFallbackOnFailure(t *testing.T) {
	plugintest.GrantStorage(t)
	fail := make(chan struct{})
	reported := make(chan error, 2)
	c := newCatalog(t, func(ctx context.Context, _, _ []string) ([]model.Model, error) {
		select {
		case <-fail:
			return nil, errors.New("the program lists no model")
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}, true, cache.WithErrorHandler(func(err error) { reported <- err }))
	command := []string{"claude"}
	got, err := c.Get(t.Context(), command, nil)
	if err != nil || len(got) != 0 {
		t.Fatalf("in-flight async cache returned %v, %v", got, err)
	}
	close(fail)
	select {
	case reportErr := <-reported:
		if reportErr == nil {
			t.Error("the load reported no failure")
		}
	case <-t.Context().Done():
		t.Fatal("the load did not report its failure")
	}
	got, err = c.Get(t.Context(), command, nil)
	if err != nil || len(got) != len(model.GetFallback()) {
		t.Fatalf("failed async cache returned %v, %v", got, err)
	}
}

func TestAProgramThatIsNotInstalledListsTheFallbackModels(t *testing.T) {
	var reported error
	c, err := New(neverCalled(t), func(err error) { reported = err }, false)
	if err != nil {
		t.Fatal(err)
	}
	list := c.List(t.Context(), "", nil, nil)
	if reported == nil {
		t.Error("expected List to report error when program is not installed")
	}
	if len(list) != len(model.GetFallback()) {
		t.Fatalf("the models are %v", list)
	}
	for place, one := range list {
		if one.Value != model.GetFallback()[place].Value || len(one.GetLevels()) != 0 {
			t.Errorf("the model %v is not a fallback alias", one)
		}
	}
}
