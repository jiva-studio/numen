package mcp_test

import (
	"context"
	"net/http"
	"path/filepath"
	"sync"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/mcp"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/check"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/search"
	usecase "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/vault"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/testsupport"
)

// sequence is what happened, in the order it happened.
type sequence struct {
	mu   sync.Mutex
	said []string
}

func (s *sequence) at(what string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.said = append(s.said, what)
}

func (s *sequence) taken() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.said...)
}

// presenting is what an agent shows to be let in.
type presenting struct{ token string }

func (p presenting) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+p.token)
	return http.DefaultTransport.RoundTrip(r)
}

// TestAnAgentWriteInFlightAtTheQuitLandsBeforeTheDatabaseCloses. The transport
// is cut off under a bound, and the calls it was in the middle of are not: a
// drain that runs out of time says nothing about what is still writing.
func TestAnAgentWriteInFlightAtTheQuitLandsBeforeTheDatabaseCloses(t *testing.T) {
	v := testsupport.NewVault(t, map[string]string{"Note.md": "---\ntitle: Note\n---\n\n# Note\n"})
	db, err := container.Config{
		IndexPath: filepath.Join(t.TempDir(), "index.db"),
	}.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	readers, writers := filesystem.Readers{}, filesystem.Writers{}
	scan := usecase.Scan{
		Readers: readers, Vaults: db.Vaults(), Notes: db.Notes(),
		Known: db.Queries(), Maintenance: db.Maintenance(),
	}
	if _, err := scan.Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}

	recorded := &sequence{}
	begun := make(chan struct{}, 1)
	until := make(chan struct{})
	var once sync.Once
	// release lets the held write through, from now on.
	release := func() { once.Do(func() { close(until) }) }
	refresh := usecase.Refresh{Readers: readers, Notes: db.Notes()}
	// The index hook is where a write reaches the database. Held open, it is a
	// write that has not finished at the moment the application is asked to go.
	index := func(ctx context.Context, v domain.Vault, paths []string) error {
		select {
		case begun <- struct{}{}:
		default:
		}
		<-until
		if _, err := refresh.Execute(ctx, v, paths); err != nil {
			return err
		}
		recorded.at("levelled")
		return nil
	}

	queries := db.Queries()
	core := mcp.Core{
		Showing: mcp.One(v, v.Path), Readers: readers, Notes: queries,
		Search:        search.New(db.Passages(), readers, nil, nil, nil, 0, nil),
		Neighbourhood: note.ShowNeighbourhood{Links: db.Links(), Notes: queries},
		Links:         note.ShowLinks{Links: db.Links()},
		Problems:      check.Standard(db.Problems()),
		Create:        note.Create{Writers: writers, Names: queries, Index: index},
		Write:         note.Write{Readers: readers, Writers: writers, Index: index},
		Move:          note.Move{Readers: readers, Writers: writers, Links: db.Links(), Sources: db.Sources(), Index: index},
		Rename:        note.Rename{Move: note.Move{Readers: readers, Writers: writers, Links: db.Links(), Sources: db.Sources(), Index: index}},
		Remove:        note.Remove{Writers: writers, Links: db.Links(), Known: db.SourcesKnown(), Index: index},
		Linking:       note.Linking{Readers: readers, Writers: writers, Index: index},
	}

	const secret = "the-token"
	endpoint, err := mcp.ServeHTTP(t.Context(), "127.0.0.1:0", secret, core, nil)
	if err != nil {
		t.Fatal(err)
	}

	client := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "v0"}, nil)
	session, err := client.Connect(t.Context(), &sdk.StreamableClientTransport{
		Endpoint:   endpoint.URL,
		HTTPClient: &http.Client{Transport: presenting{token: secret}},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	// Registered after the session, so a test that gives up lets the write through
	// before the session it is answered over is closed under it.
	defer release()

	wrote := make(chan error, 1)
	go func() {
		res, err := session.CallTool(context.Background(), &sdk.CallToolParams{
			Name:      "note_write",
			Arguments: map[string]any{"path": "Note.md", "body": "# What the agent wrote\n"},
		})
		switch {
		case err != nil:
			wrote <- err
		case res.IsError:
			wrote <- errFrom(res)
		default:
			wrote <- nil
		}
	}()

	// The write has reached the index and is held there.
	select {
	case <-begun:
	case <-time.After(5 * time.Second):
		t.Fatal("the write never reached the index")
	}

	// The drain is shorter than the call, so it runs out of time with the write
	// still going. What follows must wait for the write and not for the drain.
	drain := 200 * time.Millisecond
	shut := make(chan struct{})
	go func() {
		defer close(shut)
		ctx, cancel := context.WithTimeout(context.Background(), drain)
		defer cancel()
		endpoint.Close(ctx)
		db.Close()
		recorded.at("closed")
	}()

	select {
	case <-shut:
		t.Fatal("the database closed with an agent's write still going")
	case <-time.After(4 * drain):
	}

	release()

	select {
	case <-shut:
	case <-time.After(5 * time.Second):
		t.Fatal("closing never finished")
	}
	if err := <-wrote; err != nil {
		t.Fatalf("the agent's write was refused: %v", err)
	}

	if got := recorded.taken(); len(got) != 2 || got[0] != "levelled" || got[1] != "closed" {
		t.Errorf("the quit went %v", got)
	}
}

// errFrom is what a tool said when it answered with a failure.
func errFrom(res *sdk.CallToolResult) error {
	return &toolError{said: text(res)}
}

type toolError struct{ said string }

func (e *toolError) Error() string { return e.said }
