package container

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/embed/onnx"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/embed/openai"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/embedding"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/task"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/search"
)

// Embedders are what makes the vectors a vault is searched by and what makes
// the vector a question is asked with.
//
// They are one model where the settings name one placement. Naming a placement
// for questions is what puts a vault indexed over a network within reach of a
// machine that has none.
//
// A model on this machine is fetched and compiled behind this, and the window
// is drawn while it arrives. Until it is here the words answer alone, and the
// model and its width are known from the settings.
//
// An embedder is optional: an installation naming no placement answers with
// nothing. A placement that cannot be built — no key, no base URL — is a
// reason, and nothing is built at all.
func (c Config) Embedders(ctx context.Context, tasks *task.Tasks) (indexing, asking port.Embedder, close func() error, why error) {
	first, why := c.placed(ctx, c.Embedding.Indexing, forIndexing, tasks)
	if why != nil || first == nil {
		return nil, nil, nil, why
	}
	if c.Embedding.Query.Use == "" {
		return first.Filling(), first.Asking(), first.Close, nil
	}

	second, why := c.placed(ctx, c.Embedding.Query, forQuery, tasks)
	if why != nil {
		_ = first.Close()
		return nil, nil, nil, why
	}
	if second == nil {
		return first.Filling(), nil, first.Close, nil
	}
	// Two placements are asked whether they are one model, once both are here.
	go func() {
		if err := agreeing(ctx, first, second); err != nil {
			_ = second.Disown(err)
			failed(tasks, arriving(forQuery, c.Embedding.Query), err)
		}
	}()
	return first.Filling(), second.Asking(), both(first.Close, second.Close), nil
}

// Embedder is what makes the vectors a vault is searched by, waited for. A run
// with nowhere to show that a model is arriving waits for it instead.
func (c Config) Embedder(ctx context.Context) (port.Embedder, func() error, error) {
	held, err := c.placed(ctx, c.Embedding.Indexing, forIndexing, nil)
	if err != nil || held == nil {
		return nil, nil, err
	}
	return held.Filling(), held.Close, nil
}

// Asking is what embeds a question, for a run that fills no index. Only the
// placement that answers questions is opened, and it answers under the identity
// the index is filled with.
func (c Config) Asking(ctx context.Context) (port.Embedder, func() error, error) {
	held, err := c.placed(ctx, c.Embedding.Asking(), forQuery, nil)
	if err != nil || held == nil {
		return nil, nil, err
	}
	return held.Filling(), held.Close, nil
}

// Searching is the search a question is answered by, put together the one way:
// the passages, the vault's files, the text read out of books, and the model a
// question is embedded by.
//
// trouble is where a half that could not run is said. A search short of the
// half that asks by meaning is a search the words answer.
func (c Config) Searching(db *Index, asking port.Embedder, trouble func(error)) search.Search {
	return search.New(db.Passages(), c.VaultReaders(), c.DerivedStores(), c.Documents(),
		asking, c.Embedding.Floor, trouble)
}

// placed is what one placement makes: a service, which answers at once, or a
// model on this machine, which is loaded behind the window. Nothing for a
// placement that names neither.
//
// Whichever it is, it answers under the identity the index is filled with, and
// the two placements are held to it by being compared as one model.
func (c Config) placed(ctx context.Context, where embed.Placement, role string, tasks *task.Tasks) (*embedding.Embedding, error) {
	is := c.Embedding.Stored()
	switch where.Use {
	case embed.UseService:
		client, err := openai.New(is, where.Service)
		if err != nil {
			return nil, err
		}
		held := embedding.Arriving(is)
		held.Landed(client, nil)
		return held, nil

	case embed.UseLocal:
		held := embedding.Arriving(is)
		at := arriving(role, where)
		doing := preparing(tasks, at)
		doing(0, 0)
		go func() {
			model, err := onnx.Open(ctx, is, where.Local, doing)
			if err != nil {
				held.Landed(nil, err)
				failed(tasks, at, err)
				return
			}
			held.Landed(model, nil)
			ready(tasks, at)
		}()
		return held, nil

	case "":
		return nil, nil
	}
	// A word neither of them is a word nobody implements. Left to mean nothing,
	// it is a vault searched by its words and no reason given.
	return nil, fmt.Errorf("vectors are made %q, and they are made %q or %q",
		where.Use, embed.UseLocal, embed.UseService)
}

// agreeing is the two placements answering one text alike, once both are here.
//
// A question embedded in another space finds nothing the first indexed, and
// nothing in a settings file shows that two placements are one model. A
// comparison that did not happen is not agreement, and only a context that
// ended excuses one.
func agreeing(ctx context.Context, first, second *embedding.Embedding) error {
	// unchecked is a comparison nobody got an answer out of. A run somebody
	// stopped is owed no answer.
	unchecked := func(why error) error {
		if ctx.Err() != nil {
			return nil
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

	said, err := first.Filling().Embed(ctx, []string{embedding.Asked})
	if err != nil {
		return unchecked(err)
	}
	back, err := second.Filling().Embed(ctx, []string{embedding.Asked})
	if err != nil {
		return unchecked(err)
	}
	if len(said) != 1 || len(back) != 1 || !embedding.Agreed(said[0], back[0]) {
		return fmt.Errorf("%s and %s are not one model, and a question embedded by the second finds nothing the first indexed",
			first.Model(), second.Model())
	}
	return nil
}

// Which half of the work a placement is for. An arrival is called by its role
// and its name, and two placements naming one repository are two lines.
const (
	forIndexing = "indexing"
	forQuery    = "query"
)

// listing is one placement's arrival in the list of what is being done: what
// that line is called, and the name to show on it.
type listing struct {
	id, name string
}

// arriving is how one placement appears while it is on its way. A model on this
// machine is named by its repository and a service by the model it is asked
// for.
func arriving(role string, where embed.Placement) listing {
	name := where.Service.Name
	if where.Use == embed.UseLocal {
		name = where.Local.Name
	}
	return listing{id: "getting ready: " + role + ": " + name, name: name}
}

// preparing tells the list how far the model has got, counted in the bytes of
// it that are here. Fetching it and compiling it are one wait.
//
// The count is bytes and says so, and the sizes a person reads them in are the
// window's to write.
//
// A run with no list to tell is told nothing and still asks: what says how far
// the work has got is called wherever the work is, and a run in a terminal
// takes the same road as a window.
func preparing(tasks *task.Tasks, at listing) onnx.Fetching {
	if tasks == nil {
		return func(int64, int64) {}
	}
	return func(done, total int64) {
		tasks.Set(task.Task{
			ID: at.id, Doing: "Preparing the model", About: at.name,
			Done: done, Total: total, Counting: task.Bytes,
		})
	}
}

func ready(tasks *task.Tasks, at listing) {
	if tasks != nil {
		tasks.Done(at.id)
	}
}

// failed leaves the model in the list under what stopped it.
func failed(tasks *task.Tasks, at listing, why error) {
	if tasks != nil {
		tasks.Set(task.Task{
			ID: at.id, Doing: "Preparing the model", About: at.name,
			Failed: why.Error(),
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
