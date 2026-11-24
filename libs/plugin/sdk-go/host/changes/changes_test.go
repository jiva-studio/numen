package changes_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/changes"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
)

// follower records what Follow hands it.
type follower struct {
	reviews int
	batches [][]string
	refuse  error
}

func (f *follower) apply(_ context.Context, batch []host.FileChange) error {
	if f.refuse != nil {
		return f.refuse
	}
	paths := make([]string, 0, len(batch))
	for _, one := range batch {
		paths = append(paths, one.Path)
	}
	f.batches = append(f.batches, paths)
	return nil
}

func (f *follower) review(context.Context) error {
	f.reviews++
	return nil
}

func follow(t *testing.T, v *hosttest.Vault, f *follower) error {
	t.Helper()
	from := changes.Source{Feed: v, Store: v.Store(), CursorFile: "notes.cursor.json"}
	return changes.Follow(t.Context(), from, []string{host.EventFileAdded}, f.apply, f.review)
}

func TestTheFirstFollowReviewsAndTheNextOnesReadTheNewChanges(t *testing.T) {
	v := hosttest.New("garden")
	v.WriteFile("a.md", "A\n")
	f := &follower{}

	if err := follow(t, v, f); err != nil {
		t.Fatal(err)
	}
	if f.reviews != 1 || len(f.batches) != 0 {
		t.Fatalf("the first follow reviewed %d times and applied %v", f.reviews, f.batches)
	}
	v.WriteFile("b.md", "B\n")
	if err := follow(t, v, f); err != nil {
		t.Fatal(err)
	}
	if err := follow(t, v, f); err != nil {
		t.Fatal(err)
	}
	if f.reviews != 1 || !slices.EqualFunc(f.batches, [][]string{{"b.md"}}, slices.Equal) {
		t.Errorf("reviewed %d times and applied %v, want b.md once", f.reviews, f.batches)
	}
}

func TestAFailedApplyIsHandedTheSameChangesAgain(t *testing.T) {
	v := hosttest.New("garden")
	f := &follower{}
	if err := follow(t, v, f); err != nil {
		t.Fatal(err)
	}
	v.WriteFile("a.md", "A\n")

	f.refuse = errors.New("disk full")
	if err := follow(t, v, f); !errors.Is(err, f.refuse) {
		t.Fatalf("a failed apply returned %v", err)
	}
	f.refuse = nil
	if err := follow(t, v, f); err != nil {
		t.Fatal(err)
	}
	if !slices.EqualFunc(f.batches, [][]string{{"a.md"}}, slices.Equal) {
		t.Errorf("applied %v, want a.md after the failure", f.batches)
	}
}

func TestARebuiltIndexIsReviewedAgain(t *testing.T) {
	v := hosttest.New("garden")
	f := &follower{}
	if err := follow(t, v, f); err != nil {
		t.Fatal(err)
	}
	v.ResetFeed()
	if err := follow(t, v, f); err != nil {
		t.Fatal(err)
	}
	if f.reviews != 2 {
		t.Errorf("reviewed %d times, want once more after the index was rebuilt", f.reviews)
	}
}
