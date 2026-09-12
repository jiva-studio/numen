package embedders

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/core/internal/embedding"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/task"
)

// OpenModel is a model on this machine becoming answerable. It is told how far
// fetching and compiling it has got, counted in the bytes of it that are here.
type OpenModel func(ctx context.Context, tell func(done, total int64)) (port.Embedder, error)

// Provider is one place vectors are made: the identity what it answers is kept
// under, the name to call it by while it arrives, and the one way it is opened.
//
// The zero Provider is an installation that names no provider for this half.
type Provider struct {
	model port.EmbeddingModel
	name  string

	// One of the two, for a provider that is named at all. A service answers
	// the moment there is something to ask; a model on this machine answers
	// once it is here.
	held  port.Embedder
	fetch OpenModel
}

// NewReached is a provider that answers at once.
func NewReached(model port.EmbeddingModel, name string, held port.Embedder) Provider {
	return Provider{model: model, name: name, held: held}
}

// NewFetched is a model on this machine, opened behind whoever asked.
func NewFetched(model port.EmbeddingModel, name string, open OpenModel) Provider {
	return Provider{model: model, name: name, fetch: open}
}

// isPlaced says whether this half was placed anywhere.
func (p Provider) isPlaced() bool { return p.held != nil || p.fetch != nil }

// Which half of the work a provider is for. An arrival is called by its role
// and its name, and two providers naming one repository are two lines.
const (
	forIndexing = "indexing"
	forQuery    = "query"
)

// Open is what fills a vault's index and what a question is asked with, from the
// two providers an installation was given, and what lets go of them.
//
// They are one model where only the first is named. Where both are, the second
// is held to the first by being asked one text once both are here, and a second
// answering it differently is let go of: a question embedded in another space
// finds nothing the first indexed.
//
// Naming no provider for the index is embedding with nothing, and there is
// nothing to close.
func Open(
	ctx context.Context, tasks *task.Tasks, indexing, query Provider,
) (filling, asking port.Embedder, close func() error) {
	if !indexing.isPlaced() {
		return nil, nil, nil
	}
	first := open(ctx, tasks, indexing.newArrival(forIndexing))
	if !query.isPlaced() {
		return first.GetWaitingEmbedder(), first.GetImpatientEmbedder(), first.Close
	}

	at := query.newArrival(forQuery)
	second := open(ctx, tasks, at)
	go func() {
		if err := compareModels(ctx, first, second); err != nil {
			_ = second.Disown(err)
			reportError(tasks, at, err)
		}
	}()
	return first.GetWaitingEmbedder(), second.GetImpatientEmbedder(), both(first.Close, second.Close)
}

// One is a single provider opened for a run with nowhere to show that a model
// is arriving, which waits for it instead. It answers under the identity the
// index is filled with, whichever half of the work it was named for.
func One(ctx context.Context, from Provider) (port.Embedder, func() error) {
	if !from.isPlaced() {
		return nil, nil
	}
	held := open(ctx, nil, arrival{from: from})
	return held.GetWaitingEmbedder(), held.Close
}

// open is one provider held under the identity its vectors are kept under. What
// it is is known before it is here, so the index is fitted and a vector claimed
// under the right recipe while the weights are still coming down.
func open(ctx context.Context, tasks *task.Tasks, at arrival) *embedding.Embedder {
	held := embedding.NewArriving(at.from.model)
	if at.from.fetch == nil {
		held.ReportArrival(at.from.held, nil)
		return held
	}

	tell := newProgressReport(tasks, at)
	tell(0, 0)
	go func() {
		model, err := at.from.fetch(ctx, tell)
		if err != nil {
			held.ReportArrival(nil, err)
			reportError(tasks, at, err)
			return
		}
		held.ReportArrival(model, nil)
		ready(tasks, at)
	}()
	return held
}

// compareModels is the two providers answering one text alike, once both are
// here.
//
// A comparison that did not happen is not agreement, and only a context that
// ended excuses one.
func compareModels(ctx context.Context, first, second *embedding.Embedder) error {
	// unchecked is a comparison nobody got an answer out of. A run somebody
	// stopped is owed no answer.
	unchecked := func(why error) error {
		if ctx.Err() != nil {
			return nil //nolint:nilerr // a run somebody stopped is owed no answer
		}
		return fmt.Errorf("%s and %s were not compared as one model: %w",
			first.Model(), second.Model(), why)
	}

	if err := first.Wait(ctx); err != nil {
		return unchecked(err)
	}
	if err := second.Wait(ctx); err != nil {
		return unchecked(err)
	}

	said, err := first.GetWaitingEmbedder().Embed(ctx, []string{embedding.Asked})
	if err != nil {
		return unchecked(err)
	}
	back, err := second.GetWaitingEmbedder().Embed(ctx, []string{embedding.Asked})
	if err != nil {
		return unchecked(err)
	}
	if len(said) != 1 || len(back) != 1 || !embedding.IsAgreed(said[0], back[0]) {
		return fmt.Errorf("%s and %s are not one model, and a question embedded by the second finds nothing the first indexed",
			first.Model(), second.Model())
	}
	return nil
}

// arrival is one provider on its way: the provider itself, and its line in the
// list of what is being done — what that line is called, and the name to show
// on it.
type arrival struct {
	from     Provider
	id, name string
}

// newArrival is how one provider appears while it is on its way, under the name
// it is reached by and the half of the work it was named for.
func (p Provider) newArrival(role string) arrival {
	return arrival{from: p, id: "getting ready: " + role + ": " + p.name, name: p.name}
}

// newProgressReport tells the list how far the model has got, counted in the
// bytes of it that are here. Fetching it and compiling it are one wait.
//
// The count is bytes and says so, and the sizes a person reads them in are the
// window's to write. A share is drawn once some of the model is here.
//
// A run with no list to tell is told nothing and still asks.
func newProgressReport(tasks *task.Tasks, at arrival) func(done, total int64) {
	if tasks == nil {
		return func(int64, int64) {}
	}
	return func(done, total int64) {
		held := task.Task{ID: at.id, Doing: "Preparing the model", About: at.name}
		if done > 0 {
			held.Count, held.Total, held.Unit = done, total, task.Bytes
		}
		tasks.Set(held)
	}
}

func ready(tasks *task.Tasks, at arrival) {
	if tasks != nil {
		tasks.Remove(at.id)
	}
}

// reportError leaves the model in the list under what stopped it.
func reportError(tasks *task.Tasks, at arrival, why error) {
	if tasks != nil {
		tasks.Set(task.Task{
			ID: at.id, Doing: "Preparing the model", About: at.name,
			Error: why.Error(),
		})
	}
}

// both is one closer for two, letting go of the second whatever the first says.
func both(first, second func() error) func() error {
	return func() error {
		var why error
		if second != nil {
			why = second()
		}
		if first != nil {
			if err := first(); why == nil {
				why = err
			}
		}
		return why
	}
}
