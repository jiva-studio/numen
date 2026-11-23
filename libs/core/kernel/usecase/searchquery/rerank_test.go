package searchquery_test

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protowire"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	modelcalldomain "github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	searchquerydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	connectionusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/connection/modelcalltest"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchquery/searchquerytest"
)

const (
	firstAddress  = "https://first.test/rerank"
	secondAddress = "https://second.test/rerank"
)

// services is the reranking services of a test: connections of a fake provider plugin reached through the
// real calls of the core.
type services struct {
	world  *modelcalltest.World
	first  modelcalltest.Entry
	second modelcalltest.Entry
}

func newServices(t *testing.T) *services {
	t.Helper()
	w := modelcalltest.New()
	s := &services{world: w, first: w.AddReranker(t, "First", firstAddress), second: w.AddReranker(t, "Second", secondAddress)}
	s.setRows(s.first, s.second)
	return s
}

func (s *services) setRows(entries ...modelcalltest.Entry) {
	rows := make([]connection.Row, len(entries))
	for i, one := range entries {
		rows[i] = connection.Row{ConnectionID: one.ID, Options: map[string]string{"model": "rerank-" + one.Name}}
	}
	s.world.Rows.Set(settingsdomain.AppOwner, modelcalltest.RerankOperation, rows...)
}

func (s *services) answerBy(address string, score func(text string) float32) {
	s.world.Provider.Answer(address, func(payload []byte) ([]byte, error) {
		_, documents := readRerankRequest(payload)
		scores := make([]float32, len(documents))
		for i, d := range documents {
			scores[i] = score(d)
		}
		return writeRerankResult(scores), nil
	})
}

func (s *services) sentTexts(address string) [][]string {
	var sent [][]string
	for _, call := range s.world.Provider.ListCallsTo(address) {
		_, documents := readRerankRequest(call.Payload)
		sent = append(sent, documents)
	}
	return sent
}

func (w world) newRerankingFind(s *services, limits searchquerydomain.StageLimits) *searchquery.FindSearchResults {
	deps := w.newDeps(nil)
	world := s.world
	deps.Stages.Calls = connectionusecase.NewCalls(
		connectionusecase.Connections{Resolve: world.Resolve, ReadSecrets: world.ReadSecrets, Rows: world.Rows.ListRows},
		world.Source, world.Clock, world.Waits.Wait, world.Notices.ReportSetAside,
	)
	deps.Stages.Rows = connectionusecase.NewRows(world.Resolve, world.Rows.ListRows, nil)
	deps.Stages.Resolve = world.Resolve
	deps.Stages.Results = w.db.OperationResults(1000)
	deps.Clock = world.Clock
	deps.Stages.Limits = limits
	return searchquerytest.New(deps)
}

func readRerankRequest(payload []byte) (string, []string) {
	var query string
	var documents []string
	for len(payload) > 0 {
		number, _, n := protowire.ConsumeTag(payload)
		payload = payload[n:]
		text, m := protowire.ConsumeString(payload)
		payload = payload[m:]
		switch number {
		case 1:
			query = text
		case 3:
			documents = append(documents, text)
		}
	}
	return query, documents
}

func writeRerankResult(scores []float32) []byte {
	var packed []byte
	for _, score := range scores {
		packed = protowire.AppendFixed32(packed, math.Float32bits(score))
	}
	out := protowire.AppendTag(nil, 1, protowire.BytesType)
	return protowire.AppendBytes(out, packed)
}

func newTideWorld(t *testing.T) world {
	t.Helper()
	w := newWorld(t, map[string]string{
		"a.md": "The tide comes over the sand at night.",
		"b.md": "A tide table, printed on paper, tide by tide.",
		"c.md": "Waves and the tide, far from land, tide tide tide.",
	})
	for _, p := range []string{"a.md", "b.md", "c.md"} {
		w.cutBySize(t, p, 400)
	}
	return w
}

func byLength(text string) float32 { return float32(len(text)) / 100 }

func TestTheSecondRowAnswersWhenTheFirstFailsAndTheResultIsInItsOrder(t *testing.T) {
	w := newTideWorld(t)
	s := newServices(t)
	s.world.Provider.FailAlways(firstAddress, &connection.TryNextError{Reason: connection.FailureReasonUnavailable})
	s.answerBy(secondAddress, func(text string) float32 { return -byLength(text) })

	got, err := w.newRerankingFind(s, searchquerydomain.StageLimits{}).Execute(t.Context(), w.v, searchquerydomain.Request{Text: `words("tide").rerank()`})

	if err != nil {
		t.Fatal(err)
	}
	if paths := getPaths(got); !slices.Equal(paths, []string{"a.md", "b.md", "c.md"}) {
		t.Errorf("results %v, want the shortest passage first", paths)
	}
	if got.ScoreKind != searchquerydomain.ScoreRerank || len(got.Reranks) != 1 || got.Reranks[0].ConnectionName != "Second" {
		t.Errorf("reports %+v, want one naming Second", got.Reranks)
	}
	if query, _ := readRerankRequest(s.world.Provider.ListCallsTo(secondAddress)[0].Payload); query != "tide" {
		t.Errorf("the rerank query is %q, want the text of the condition", query)
	}
}

func TestARerankWithNoRowOrWithEveryRowFailingIsAQueryError(t *testing.T) {
	w := newTideWorld(t)
	s := newServices(t)
	s.setRows()
	_, err := w.newRerankingFind(s, searchquerydomain.StageLimits{}).Execute(t.Context(), w.v, searchquerydomain.Request{Text: `words("tide").rerank()`})
	if !errors.Is(err, searchquerydomain.ErrNoReranker) {
		t.Errorf("with no row the query answered %v, want ErrNoReranker", err)
	}

	s.setRows(s.first, s.second)
	s.world.Provider.FailAlways(firstAddress, &connection.TryNextError{Reason: connection.FailureReasonUnavailable})
	s.world.Provider.FailAlways(secondAddress, &connection.StopError{Reason: connection.FailureReasonInvalidRequest})
	_, err = w.newRerankingFind(s, searchquerydomain.StageLimits{}).Execute(t.Context(), w.v, searchquerydomain.Request{Text: `words("tide").rerank()`})
	rerankError, ok := errors.AsType[*searchquerydomain.RerankError](err)
	if !ok || len(rerankError.Failures) != 2 || rerankError.Failures[0].ConnectionName != "First" || rerankError.Failures[1].ConnectionName != "Second" {
		t.Fatalf("with every row failing the query answered %v, want a RerankError naming both", err)
	}
	if want := "no reranking service answered: First: unavailable; Second: invalid request"; err.Error() != want {
		t.Errorf("the error reads %q, want %q", err.Error(), want)
	}
}

func TestASecondRunReusesTheScoresAndAChangedChunkIsSentAlone(t *testing.T) {
	w := newTideWorld(t)
	s := newServices(t)
	s.setRows(s.second)
	s.answerBy(secondAddress, byLength)
	find := w.newRerankingFind(s, searchquerydomain.StageLimits{})
	request := searchquerydomain.Request{Text: `words("tide").rerank()`}
	if _, err := find.Execute(t.Context(), w.v, request); err != nil {
		t.Fatal(err)
	}

	again, err := find.Execute(t.Context(), w.v, request)
	if err != nil {
		t.Fatal(err)
	}
	if calls := len(s.world.Provider.ListCallsTo(secondAddress)); calls != 1 {
		t.Errorf("the service was called %d times, want once", calls)
	}
	if report := again.Reranks[0]; report.ReusedCount != 3 || report.SentCount != 0 {
		t.Errorf("the second run reports %+v, want 3 reused and none sent", report)
	}

	writeFile(t, w, "b.md", "A tide table, printed on paper, read at the harbor.")
	w.cutBySize(t, "b.md", 400)
	third, err := find.Execute(t.Context(), w.v, request)
	if err != nil {
		t.Fatal(err)
	}
	sent := s.sentTexts(secondAddress)
	if len(sent) != 2 || !slices.Equal(sent[1], []string{"A tide table, printed on paper, read at the harbor."}) {
		t.Errorf("the third run sent %q, want the changed passage alone", sent[1:])
	}
	if report := third.Reranks[0]; report.ReusedCount != 2 || report.SentCount != 1 {
		t.Errorf("the third run reports %+v", report)
	}
}

func TestScoresOfUnchangedTextsAreReusedAfterTheFilesAreChunkedAgain(t *testing.T) {
	w := newTideWorld(t)
	s := newServices(t)
	s.setRows(s.second)
	s.answerBy(secondAddress, byLength)
	find := w.newRerankingFind(s, searchquerydomain.StageLimits{})
	request := searchquerydomain.Request{Text: `words("tide").rerank()`}
	if _, err := find.Execute(t.Context(), w.v, request); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"a.md", "b.md", "c.md"} {
		w.replace(t, p, nil)
		w.cutBySize(t, p, 400)
	}

	got, err := find.Execute(t.Context(), w.v, request)

	if err != nil {
		t.Fatal(err)
	}
	if report := got.Reranks[0]; report.ReusedCount != 3 || report.SentCount != 0 {
		t.Errorf("after cutting again the run reports %+v, want every score reused", report)
	}
}

func TestAWrongCountOfScoresFailsTheRowAndTheNextIsAsked(t *testing.T) {
	w := newTideWorld(t)
	s := newServices(t)
	s.world.Provider.Answer(firstAddress, func([]byte) ([]byte, error) { return writeRerankResult([]float32{1}), nil })
	s.answerBy(secondAddress, byLength)

	got, err := w.newRerankingFind(s, searchquerydomain.StageLimits{}).Execute(t.Context(), w.v, searchquerydomain.Request{Text: `words("tide").rerank()`})

	if err != nil {
		t.Fatal(err)
	}
	if got.Reranks[0].ConnectionName != "Second" || len(got.Results) != 3 {
		t.Errorf("results %v reported by %+v, want three by Second", getPaths(got), got.Reranks)
	}
}

func TestALongChunkIsSentTruncatedAndAWholeFileAsItsTitleAndTheStartOfItsText(t *testing.T) {
	long := strings.Repeat("tide ", 200)
	w := newWorld(t, map[string]string{
		"long.md": long,
		"Pier.md": "---\ntags: [harbor]\n---\nThe pier at dawn.",
	})
	w.cutRanges(t, "long.md", [2]int{0, len(long)})
	s := newServices(t)
	s.setRows(s.second)
	s.answerBy(secondAddress, byLength)
	find := w.newRerankingFind(s, searchquerydomain.StageLimits{RerankWindow: 60, MaxReranked: 300, MaxRerankedBytes: 300})

	if _, err := find.Execute(t.Context(), w.v, searchquerydomain.Request{Text: `words("tide").rerank()`}); err != nil {
		t.Fatal(err)
	}
	if _, err := find.Execute(t.Context(), w.v, searchquerydomain.Request{Text: `(kind == "note" && path.startsWith("Pier")).rerank("pier")`}); err != nil {
		t.Fatal(err)
	}

	sent := s.sentTexts(secondAddress)
	if len(sent) != 2 || len(sent[0]) != 1 || sent[0][0] != long[:300] {
		t.Fatalf("the first call sent %q, want the chunk cut at 300 bytes", sent)
	}
	if len(sent[1]) != 1 || sent[1][0] != "Pier\nThe pier at dawn." {
		t.Errorf("the whole file was sent as %q, want its title and its text", sent[1])
	}
}

func TestTheCountOfResultsFollowsTheLimitAndTheTopStage(t *testing.T) {
	files := map[string]string{}
	for i := range 40 {
		files["n"+strings.Repeat("x", i)+".md"] = "the tide again"
	}
	w := newWorld(t, files)
	for name := range files {
		w.cutBySize(t, name, 400)
	}
	find := w.newFind(nil)
	for _, c := range []struct {
		text  string
		limit int
		want  int
	}{
		{`words("tide")`, 0, 20},
		{`words("tide").top(3)`, 0, 3},
		{`words("tide").top(50)`, 10, 10},
		{`words("tide").top(30)`, 0, 30},
	} {
		got, err := find.Execute(t.Context(), w.v, searchquerydomain.Request{Text: c.text, Limit: c.limit})
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Results) != c.want {
			t.Errorf("%s with limit %d returned %d results, want %d", c.text, c.limit, len(got.Results), c.want)
		}
	}
	find.Limits.DefaultLimit = 30
	if got, err := find.Execute(t.Context(), w.v, searchquerydomain.Request{Text: `words("tide")`}); err != nil || len(got.Results) != 30 {
		t.Errorf("with a default of 30 the query returned %d results, %v", len(got.Results), err)
	}
}

func TestACanceledCallReturnsTheCancellationAndStoresNothing(t *testing.T) {
	w := newTideWorld(t)
	s := newServices(t)
	s.setRows(s.second)
	s.world.Provider.Block(secondAddress)
	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		<-s.world.Provider.GetEntered()
		cancel()
	}()

	_, err := w.newRerankingFind(s, searchquerydomain.StageLimits{}).Execute(ctx, w.v, searchquerydomain.Request{Text: `words("tide").rerank()`})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("the query answered %v, want context.Canceled", err)
	}
	removed, err := w.db.OperationResults(1000).ClearResults(t.Context())
	if err != nil || removed != 0 {
		t.Errorf("the canceled run stored %d results, %v", removed, err)
	}
}

func writeFile(t *testing.T, w world, name, text string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(w.v.Path, filepath.FromSlash(name)), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestTheResultsBeforeTheFirstRerankAreReportedBeforeTheServiceIsAsked(t *testing.T) {
	w := newTideWorld(t)
	s := newServices(t)
	s.setRows(s.second)
	s.answerBy(secondAddress, func(text string) float32 { return -byLength(text) })
	var partials [][]string
	var callsAtReport int
	report := func(partial searchquerydomain.SearchResults) {
		partials = append(partials, getPaths(partial))
		callsAtReport = len(s.world.Provider.ListCallsTo(secondAddress))
	}
	find := w.newRerankingFind(s, searchquerydomain.StageLimits{})
	words, err := find.Execute(t.Context(), w.v, searchquerydomain.Request{Text: `words("tide")`})
	if err != nil {
		t.Fatal(err)
	}

	got, err := find.Execute(t.Context(), w.v, searchquerydomain.Request{Text: `words("tide").rerank().rerank()`, ReportPartial: report})

	if err != nil {
		t.Fatal(err)
	}
	if len(partials) != 1 || !slices.Equal(partials[0], getPaths(words)) || callsAtReport != 0 {
		t.Fatalf("partials %v after %d calls, want once %v before any call", partials, callsAtReport, getPaths(words))
	}
	if paths := getPaths(got); !slices.Equal(paths, []string{"a.md", "b.md", "c.md"}) {
		t.Errorf("results %v, want the reranked order", paths)
	}
}

func newShoreWorld(t *testing.T, count int) world {
	t.Helper()
	files := map[string]string{}
	for i := range count {
		files[fmt.Sprintf("shore-%02d.md", i)] = fmt.Sprintf("The tide reaches shore number %d.", i)
	}
	w := newWorld(t, files)
	for name := range files {
		w.cutBySize(t, name, 400)
	}
	return w
}

func TestARerankSendsItsTextsInBatchesOfAtMost32(t *testing.T) {
	w := newShoreWorld(t, 40)
	s := newServices(t)
	s.setRows(s.second)
	s.answerBy(secondAddress, byLength)
	limits := searchquerydomain.StageLimits{RerankWindow: 100, MaxReranked: 300, MaxRerankedBytes: 4096}

	got, err := w.newRerankingFind(s, limits).Execute(t.Context(), w.v, searchquerydomain.Request{Text: `words("tide").rerank()`, Limit: 100})

	if err != nil {
		t.Fatal(err)
	}
	var sizes []int
	for _, sent := range s.sentTexts(secondAddress) {
		sizes = append(sizes, len(sent))
	}
	if !slices.Equal(sizes, []int{32, 8}) {
		t.Errorf("calls sent %v texts, want 32 and 8", sizes)
	}
	if len(got.Results) != 40 || got.ScoreKind != searchquerydomain.ScoreRerank {
		t.Errorf("%d results scored %v, want 40 reranked", len(got.Results), got.ScoreKind)
	}
}

func TestAWaitingInteractiveRerankGoesBeforeTheNextBatchOfABackgroundRerank(t *testing.T) {
	w := newShoreWorld(t, 40)
	s := newServices(t)
	s.setRows(s.second)
	s.answerBy(secondAddress, byLength)
	s.world.Provider.Block(secondAddress)
	find := w.newRerankingFind(s, searchquerydomain.StageLimits{RerankWindow: 100, MaxReranked: 300, MaxRerankedBytes: 4096})
	run := func(text string, priority modelcalldomain.Priority) chan error {
		done := make(chan error, 1)
		go func() {
			_, err := find.Execute(t.Context(), w.v, searchquerydomain.Request{Text: text, Limit: 100, Priority: priority})
			done <- err
		}()
		return done
	}

	background := run(`words("tide").rerank("background")`, modelcalldomain.PriorityBackground)
	<-s.world.Provider.GetEntered()
	interactive := run(`words("tide").rerank("person")`, modelcalldomain.PriorityInteractive)
	s.world.Waits.AwaitCount(t, 1)
	s.world.Provider.Release(secondAddress)
	<-s.world.Provider.GetEntered()

	calls := s.world.Provider.ListCallsTo(secondAddress)
	if query, _ := readRerankRequest(calls[len(calls)-1].Payload); query != "person" {
		t.Errorf("the call after the first background batch reranks for %q, want the interactive query", query)
	}
	s.world.Provider.ReleaseCalls(secondAddress)
	if err := <-interactive; err != nil {
		t.Errorf("the interactive rerank answered %v", err)
	}
	if err := <-background; err != nil {
		t.Errorf("the background rerank answered %v", err)
	}
}

func TestAPassageThatIsNotUTF8FailsTheRerankWithoutACall(t *testing.T) {
	w := newWorld(t, map[string]string{"a.md": "The tide comes over the sand \xff at night."})
	w.cutBySize(t, "a.md", 400)
	s := newServices(t)

	_, err := w.newRerankingFind(s, searchquerydomain.StageLimits{}).Execute(t.Context(), w.v, searchquerydomain.Request{Text: `words("tide").rerank()`})

	rerankError, ok := errors.AsType[*searchquerydomain.RerankError](err)
	if !ok || rerankError == nil {
		t.Fatalf("the query answered %v, want a RerankError", err)
	}
	if calls := len(s.world.Provider.ListCallsTo(firstAddress)) + len(s.world.Provider.ListCallsTo(secondAddress)); calls != 0 {
		t.Errorf("%d calls were made, want none for a request that cannot be written", calls)
	}
}
