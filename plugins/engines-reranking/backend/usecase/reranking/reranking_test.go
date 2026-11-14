package reranking_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/domain/profile"
	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/ports"
	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/usecase/reranking"
)

const (
	gteID = "onnx-community/gte-multilingual-reranker-base"
	bgeID = "onnx-community/bge-reranker-v2-m3-ONNX"
)

// waitLimit bounds a wait on a goroutine of the use case, so a test that fails does not hang.
const waitLimit = 10 * time.Second

type shippedSource struct{}

func (shippedSource) Load(func(file string, err error)) (*profile.Profiles, error) {
	return profile.GetShippedProfiles()
}

type scorePairs struct {
	query     string
	documents []string
}

type fakeScorer struct {
	mu       sync.Mutex
	profile  profile.Profile
	score    func(query string, documents []string) []float32
	calls    []scorePairs
	isClosed bool
}

func (f *fakeScorer) ScorePairs(_ context.Context, query string, documents []string) ([]float32, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, scorePairs{query: query, documents: slices.Clone(documents)})
	return f.score(query, documents), nil
}

func (f *fakeScorer) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.isClosed = true
	return nil
}

func (f *fakeScorer) getCalls() []scorePairs {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.calls)
}

func (f *fakeScorer) getIsClosed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.isClosed
}

// scoreByLength scores each document by its length in bytes, so every document has its own score.
func scoreByLength(_ string, documents []string) []float32 {
	scores := make([]float32, len(documents))
	for i, document := range documents {
		scores[i] = float32(len(document)) / 100
	}
	return scores
}

type opener struct {
	mu      sync.Mutex
	score   func(query string, documents []string) []float32
	err     error
	opened  []*fakeScorer
	release chan struct{}
	started chan context.Context
}

func newOpener() *opener {
	return &opener{score: scoreByLength}
}

func (o *opener) open(ctx context.Context, p profile.Profile) (ports.Scorer, error) {
	if o.started != nil {
		o.started <- ctx
	}
	if o.release != nil {
		<-o.release
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.err != nil {
		return nil, o.err
	}
	scorer := &fakeScorer{profile: p, score: o.score}
	o.opened = append(o.opened, scorer)
	return scorer, nil
}

func (o *opener) getOpened() []*fakeScorer {
	o.mu.Lock()
	defer o.mu.Unlock()
	return slices.Clone(o.opened)
}

type reports struct {
	mu    sync.Mutex
	texts []string
}

func (r *reports) report(what string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.texts = append(r.texts, what+": "+err.Error())
}

func (r *reports) getTexts() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.texts)
}

func newReranking(t *testing.T, open *opener, report *reports) *reranking.Reranking {
	t.Helper()
	r, err := reranking.New(shippedSource{}, open.open, report.report)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := r.Close(); err != nil {
			t.Error(err)
		}
	})
	return r
}

func callFor(model string) connection.Call {
	call := connection.Call{ConnectionID: "local-reranker:Local reranker", Type: "local-reranker"}
	if model != "" {
		call.Options = map[string]string{llm.OptionModel: model}
	}
	return call
}

func checkFailure(t *testing.T, err error, isTryNext bool, reason connection.Reason) {
	t.Helper()
	_, gotTryNext := connection.IsTryNext(err)
	if isTryNext && !gotTryNext {
		t.Fatalf("error = %v, want a try-next failure", err)
	}
	if !isTryNext && !connection.IsStop(err) {
		t.Fatalf("error = %v, want a stop failure", err)
	}
	if !strings.HasSuffix(err.Error(), reason.String()) {
		t.Errorf("error = %q, want the reason %q", err, reason)
	}
}

var gardenDocuments = []string{
	"Tomatoes need full sun.",
	"A tomato plant grown in a pot needs water every morning in July.",
	"The tram leaves at nine.",
}

func TestRerankScoresTheDocumentsInTheirOrder(t *testing.T) {
	open := newOpener()
	r := newReranking(t, open, &reports{})

	result, err := r.Rerank(t.Context(), callFor(""), &llm.RerankRequest{Query: "how often to water tomatoes", Documents: gardenDocuments})

	if err != nil {
		t.Fatal(err)
	}
	if want := scoreByLength("", gardenDocuments); !slices.Equal(result.GetScores(), want) {
		t.Errorf("scores = %v, want %v in the order of the documents", result.GetScores(), want)
	}
	opened := open.getOpened()
	if len(opened) != 1 || len(opened[0].getCalls()) != 1 || !slices.Equal(opened[0].getCalls()[0].documents, gardenDocuments) {
		t.Errorf("the scorer was not given the documents in their order: %+v", opened)
	}
}

func TestRerankWithNoModelRunsTheDefaultProfile(t *testing.T) {
	open := newOpener()
	r := newReranking(t, open, &reports{})

	if _, err := r.Rerank(t.Context(), callFor(""), &llm.RerankRequest{Query: "tomatoes", Documents: gardenDocuments}); err != nil {
		t.Fatal(err)
	}

	if opened := open.getOpened(); len(opened) != 1 || opened[0].profile.ID != profile.DefaultModel {
		t.Errorf("opened %+v, want the default profile %s", opened, profile.DefaultModel)
	}
}

func TestRerankOfAModelNamedByAliasRunsItsProfile(t *testing.T) {
	open := newOpener()
	r := newReranking(t, open, &reports{})

	if _, err := r.Rerank(t.Context(), callFor("BGE-Reranker-v2-M3"), &llm.RerankRequest{Query: "tomatoes", Documents: gardenDocuments}); err != nil {
		t.Fatal(err)
	}

	if opened := open.getOpened(); len(opened) != 1 || opened[0].profile.ID != bgeID || opened[0].profile.MaxTokens != 512 {
		t.Errorf("opened %+v, want the bge profile", opened)
	}
}

func TestRerankWithNoDocumentsLoadsNoModel(t *testing.T) {
	open := newOpener()
	r := newReranking(t, open, &reports{})

	result, err := r.Rerank(t.Context(), callFor(""), &llm.RerankRequest{Query: "tomatoes"})

	if err != nil || len(result.GetScores()) != 0 {
		t.Errorf("Rerank = %v, %v, want no scores", result.GetScores(), err)
	}
	if opened := open.getOpened(); len(opened) != 0 {
		t.Errorf("opened %d models for no document", len(opened))
	}
}

func TestRerankOfAnEndedCallWithNoDocumentsReturnsTheContextError(t *testing.T) {
	r := newReranking(t, newOpener(), &reports{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.Rerank(ctx, callFor(""), &llm.RerankRequest{Query: "tomatoes"})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
}

func TestRerankOfAnUnknownModelIsAStop(t *testing.T) {
	open := newOpener()
	report := &reports{}
	r := newReranking(t, open, report)

	_, err := r.Rerank(t.Context(), callFor("owner/unknown-reranker"), &llm.RerankRequest{Query: "tomatoes", Documents: gardenDocuments})

	checkFailure(t, err, false, connection.ReasonInvalidRequest)
	if strings.Contains(err.Error(), "owner/unknown-reranker") {
		t.Errorf("the failure %q carries the text of the plugin", err)
	}
	if texts := report.getTexts(); len(texts) != 1 || !strings.Contains(texts[0], "owner/unknown-reranker") {
		t.Errorf("reported %q, want the unknown model named", texts)
	}
	if opened := open.getOpened(); len(opened) != 0 {
		t.Errorf("opened %d models for an unknown one", len(opened))
	}
}

func TestRerankOfAModelThatDoesNotOpenIsTryNext(t *testing.T) {
	open := newOpener()
	open.err = errors.New("onnxruntime: the library at /opt/ort is not loadable")
	report := &reports{}
	r := newReranking(t, open, report)

	_, err := r.Rerank(t.Context(), callFor(""), &llm.RerankRequest{Query: "tomatoes", Documents: gardenDocuments})

	checkFailure(t, err, true, connection.ReasonUnavailable)
	if texts := report.getTexts(); len(texts) != 1 || !strings.Contains(texts[0], "not loadable") || !strings.Contains(texts[0], gteID) {
		t.Errorf("reported %q, want the open failure with the profile id", texts)
	}
}

func TestRerankPassesTheQueryWithoutTheInstruction(t *testing.T) {
	open := newOpener()
	r := newReranking(t, open, &reports{})

	_, err := r.Rerank(t.Context(), callFor(""), &llm.RerankRequest{
		Query: "how often to water tomatoes", Instruction: "Find the notes that answer the question", Documents: gardenDocuments,
	})

	if err != nil {
		t.Fatal(err)
	}
	opened := open.getOpened()
	if len(opened) != 1 {
		t.Fatalf("opened %d models, want 1", len(opened))
	}
	if calls := opened[0].getCalls(); len(calls) != 1 || calls[0].query != "how often to water tomatoes" {
		t.Errorf("scorer calls = %+v, want the query unchanged", calls)
	}
}

func TestRerankOfTheSameProfileKeepsTheLoadedModel(t *testing.T) {
	open := newOpener()
	r := newReranking(t, open, &reports{})
	request := &llm.RerankRequest{Query: "tomatoes", Documents: gardenDocuments}

	for _, model := range []string{"", gteID, "gte-multilingual-reranker-base"} {
		if _, err := r.Rerank(t.Context(), callFor(model), request); err != nil {
			t.Fatal(err)
		}
	}

	if opened := open.getOpened(); len(opened) != 1 || opened[0].getIsClosed() {
		t.Errorf("opened %d models, want one kept open", len(opened))
	}
}

func TestRerankOfAnotherProfileClosesTheLoadedModel(t *testing.T) {
	open := newOpener()
	r := newReranking(t, open, &reports{})
	request := &llm.RerankRequest{Query: "tomatoes", Documents: gardenDocuments}

	if _, err := r.Rerank(t.Context(), callFor(gteID), request); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Rerank(t.Context(), callFor(bgeID), request); err != nil {
		t.Fatal(err)
	}

	opened := open.getOpened()
	if len(opened) != 2 || opened[0].profile.ID != gteID || opened[1].profile.ID != bgeID {
		t.Fatalf("opened %+v, want gte then bge", opened)
	}
	if !opened[0].getIsClosed() || opened[1].getIsClosed() {
		t.Errorf("gte closed %v, bge closed %v, want gte closed and bge open", opened[0].getIsClosed(), opened[1].getIsClosed())
	}
}

func TestCloseClosesTheLoadedModel(t *testing.T) {
	open := newOpener()
	r, err := reranking.New(shippedSource{}, open.open, (&reports{}).report)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Rerank(t.Context(), callFor(""), &llm.RerankRequest{Query: "tomatoes", Documents: gardenDocuments}); err != nil {
		t.Fatal(err)
	}

	if err := r.Close(); err != nil {
		t.Fatal(err)
	}

	if opened := open.getOpened(); len(opened) != 1 || !opened[0].getIsClosed() {
		t.Errorf("the loaded model stays open after Close")
	}
}

func TestRerankWithTheWrongNumberOfScoresIsTryNext(t *testing.T) {
	open := newOpener()
	open.score = func(string, []string) []float32 { return []float32{0.9} }
	report := &reports{}
	r := newReranking(t, open, report)

	_, err := r.Rerank(t.Context(), callFor(""), &llm.RerankRequest{Query: "tomatoes", Documents: gardenDocuments})

	checkFailure(t, err, true, connection.ReasonUnavailable)
	if texts := report.getTexts(); len(texts) != 1 {
		t.Errorf("reported %q, want the wrong count reported once", texts)
	}
}

func TestCanceledCallLeavesTheLoadRunning(t *testing.T) {
	open := newOpener()
	open.release = make(chan struct{})
	open.started = make(chan context.Context, 2)
	r := newReranking(t, open, &reports{})
	request := &llm.RerankRequest{Query: "tomatoes", Documents: gardenDocuments}
	ctx, cancel := context.WithCancel(t.Context())
	first := make(chan error, 1)

	go func() {
		_, err := r.Rerank(ctx, callFor(""), request)
		first <- err
	}()
	var loadCtx context.Context
	select {
	case loadCtx = <-open.started:
	case <-time.After(waitLimit):
		t.Fatal("the first call did not start a load")
	}
	cancel()
	select {
	case err := <-first:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("canceled call = %v, want context.Canceled", err)
		}
	case <-time.After(waitLimit):
		t.Fatal("the canceled call waits for the load")
	}
	if loadCtx.Err() != nil {
		t.Errorf("the load was canceled with the call: %v", loadCtx.Err())
	}
	close(open.release)

	result, err := r.Rerank(t.Context(), callFor(""), request)

	if err != nil || len(result.GetScores()) != len(gardenDocuments) {
		t.Fatalf("Rerank after the load = %v, %v", result.GetScores(), err)
	}
	if opened := open.getOpened(); len(opened) != 1 {
		t.Errorf("opened %d models, want the one load the canceled call started", len(opened))
	}
}

func TestListModelsListsTheProfilesWithATier(t *testing.T) {
	open := newOpener()
	r := newReranking(t, open, &reports{})

	listed, err := r.ListModels(t.Context(), callFor(""), connection.OptionsQuery{Operations: []string{llm.OperationRerank}})

	if err != nil {
		t.Fatal(err)
	}
	want := []connection.Option{
		{Value: gteID, Title: "GTE multilingual reranker (default, 341 MB)"},
		{Value: bgeID, Title: "BGE reranker v2 M3 (larger, 571 MB)"},
	}
	if !slices.Equal(listed.Options, want) || listed.IsPartial {
		t.Errorf("listed %+v, want %+v and not partial", listed, want)
	}
	if opened := open.getOpened(); len(opened) != 0 {
		t.Errorf("listing the models opened %d of them", len(opened))
	}
}

func TestDetectConnectionsAnswersOneLocalReranker(t *testing.T) {
	open := newOpener()
	r := newReranking(t, open, &reports{})

	found, err := r.DetectConnections(t.Context(), connection.Call{Type: "local-reranker"})

	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].Name != "Local reranker" || len(found[0].Fields) != 0 || found[0].Path != "" {
		t.Errorf("detected %+v, want one Local reranker with no fields and no path", found)
	}
	if opened := open.getOpened(); len(opened) != 0 {
		t.Errorf("detecting the connection opened %d models", len(opened))
	}
}
