package indexing

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vaultreading"
	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
)

// scriptedVectors embeds sources in order, as the embed pipeline does, and
// reports the sources in flight. The source named in blockOn is held until the
// context of the pass ends.
type scriptedVectors struct {
	order    []string
	blockOn  string
	inFlight func(path string) (current string, active []string)
	started  chan struct{}

	mu       sync.Mutex
	embedded []string
}

func (s *scriptedVectors) EmbedWithProgress(
	ctx context.Context, _ domain.Vault, onProgress func(vaultreading.EmbedResult), isSkipped func(path, hash string) bool,
) (vaultreading.EmbedResult, error) {
	var res vaultreading.EmbedResult
	for _, path := range s.order {
		if slices.Contains(s.getEmbeddedPaths(), path) {
			continue
		}
		if isSkipped != nil && isSkipped(path, "hash") {
			res.Skipped++
			continue
		}
		res.CurrentSource, res.Active = s.inFlight(path)
		onProgress(res)
		if path == s.blockOn {
			signal(s.started)
			<-ctx.Done()
			return res, ctx.Err()
		}
		s.mu.Lock()
		s.embedded = append(s.embedded, path)
		s.mu.Unlock()
		res.Embedded++
	}
	return res, nil
}

func (s *scriptedVectors) getEmbeddedPaths() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.embedded)
}

// A cancel holds the sources in flight and no other: the first source is done,
// the second is embedding, and the third waits. The next pass embeds the
// first source's successor only if it was not in flight, and the held sources
// once their change is reported.
func TestCancelMarksTheSourcesInFlight(t *testing.T) {
	one := func(path string) (string, []string) { return path, []string{path} }
	tests := []struct {
		name     string
		inFlight func(path string) (string, []string)
		wantHeld []string
		wantNext []string
	}{
		{
			name:     "one batch in flight",
			inFlight: one,
			wantHeld: []string{"B.pdf"},
			wantNext: []string{"A.pdf", "C.pdf"},
		},
		{
			name:     "a report that names only the current source",
			inFlight: func(path string) (string, []string) { return path, nil },
			wantHeld: []string{"B.pdf"},
			wantNext: []string{"A.pdf", "C.pdf"},
		},
		{
			name: "two sources in one batch",
			inFlight: func(path string) (string, []string) {
				if path == "B.pdf" {
					return path, []string{"B.pdf", "C.pdf"}
				}
				return one(path)
			},
			wantHeld: []string{"B.pdf", "C.pdf"},
			wantNext: []string{"A.pdf"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := newRecorder()
			script := &scriptedVectors{
				order:    []string{"A.pdf", "B.pdf", "C.pdf"},
				blockOn:  "B.pdf",
				inFlight: tt.inFlight,
				started:  make(chan struct{}, 1),
			}
			worker := NewWorker(Config{
				Reading: func(context.Context, embeddingport.Embedder, domain.Vault, bool) (ReadWholeVault, error) {
					return NewReadWholeVault(nil, nil, script), nil
				},
				Embedder: &fakeEmbedder{dims: 8},
				Events:   events,
				Quiet:    time.Hour,
				After:    time.After,
			})
			p := worker.newPass(worker.NewSession(), domain.Vault{ID: "v1"}, false)

			ended := make(chan struct{})
			go func() {
				defer close(ended)
				p.embed(t.Context())
			}()
			select {
			case <-script.started:
			case <-time.After(10 * time.Second):
				t.Fatal("the pass never reached the second source")
			}
			if !events.tasks.Cancel(work.MakingVectors) {
				t.Fatalf("the vector pass is not cancellable: %+v", events.tasks.List())
			}
			select {
			case <-ended:
			case <-time.After(10 * time.Second):
				t.Fatal("the pass did not return after its task was canceled")
			}

			script.blockOn = ""
			p.embed(t.Context())
			if got := script.getEmbeddedPaths(); !slices.Equal(got, tt.wantNext) {
				t.Errorf("after the cancel the passes embedded %v, want %v", got, tt.wantNext)
			}

			worker.ClearSkips(tt.wantHeld)
			p.embed(t.Context())
			got := script.getEmbeddedPaths()
			slices.Sort(got)
			if want := []string{"A.pdf", "B.pdf", "C.pdf"}; !slices.Equal(got, want) {
				t.Errorf("after the change was reported the passes embedded %v, want %v", got, want)
			}
		})
	}
}
