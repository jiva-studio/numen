package note_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func toName(value string, role domain.LinkRole) domain.Link {
	return domain.Link{Target: domain.Address{Scheme: domain.SchemeName, Value: value}, Role: role}
}

func TestABadLinkCostsOnlyItselfAndTheRestAreSavedPerNote(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{
		"Entropy.md": "# Entropy\n", "Heat.md": "# Heat\n", "Work.md": "# Work\n",
	})

	added, err := note.NewAddLinks(c.newEditLinks()).Execute(t.Context(), c.vault, []note.LinkAddition{
		{From: "Heat.md", Link: toName("Entropy", domain.RoleParent)},
		{From: "Heat.md", Link: toName("Work", "nonsense")},
		{From: "Heat.md", Link: toName("Work", domain.RoleJump)},
		{From: "Work.md", Link: toName("Entropy", domain.RoleRef)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(added) != 4 {
		t.Fatalf("want an outcome for each, got %d", len(added))
	}
	if added[1].Error == nil {
		t.Error("a link carrying no known role was written anyway")
	}
	for _, i := range []int{0, 2, 3} {
		if added[i].Error != nil {
			t.Errorf("link %d failed: %v", i, added[i].Error)
		}
	}
	if body := c.read(t, "Heat.md"); !strings.Contains(body, "Entropy") || !strings.Contains(body, "Work") {
		t.Errorf("the links of one note were not written together:\n%s", body)
	}
}

func TestAddLinks_RejectsLinksWithConflictingFingerprints(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"Heat.md": "# Heat\n"})
	first := domain.FileStat{Size: 1}

	added, err := note.NewAddLinks(c.newEditLinks()).Execute(t.Context(), c.vault, []note.LinkAddition{
		{From: "Heat.md", FileStat: first, Link: toName("Entropy", domain.RoleRef)},
		{From: "Heat.md", FileStat: domain.FileStat{Size: 2}, Link: toName("Work", domain.RoleRef)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if added[1].Error == nil || !strings.Contains(added[1].Error.Error(), "two different fingerprints") {
		t.Errorf("the second fingerprint was not rejected for naming another: %v", added[1].Error)
	}
}

func TestAnUnreadableFingerprintIsReportedAfterTheLinksOwnError(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"Heat.md": "# Heat\n"})
	unreadable := errors.New("not a fingerprint")

	added, err := note.NewAddLinks(c.newEditLinks()).Execute(t.Context(), c.vault, []note.LinkAddition{
		{From: "Heat.md", Link: toName("Entropy", "nonsense"), FileStatError: unreadable},
		{From: "Heat.md", Link: toName("Entropy", domain.RoleRef), FileStatError: unreadable},
	})
	if err != nil {
		t.Fatal(err)
	}
	if added[0].Error == nil || errors.Is(added[0].Error, unreadable) {
		t.Errorf("the link's own error did not come first: %v", added[0].Error)
	}
	if !errors.Is(added[1].Error, unreadable) {
		t.Errorf("the unreadable fingerprint was not reported: %v", added[1].Error)
	}
	if body := c.read(t, "Heat.md"); body != "# Heat\n" {
		t.Errorf("the note was written anyway:\n%s", body)
	}
}

func TestAWriteTheNoteChangedBelowIsRefusedForEveryLinkOfIt(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"Heat.md": "# Heat\n"})

	added, err := note.NewAddLinks(c.newEditLinks()).Execute(t.Context(), c.vault, []note.LinkAddition{
		{From: "Heat.md", FileStat: domain.FileStat{Size: 1}, Link: toName("Entropy", domain.RoleRef)},
		{From: "Heat.md", FileStat: domain.FileStat{Size: 1}, Link: toName("Work", domain.RoleRef)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if added[0].Error == nil || added[1].Error == nil || !errors.Is(added[0].Error, added[1].Error) {
		t.Errorf("the links of one note did not fail together: %v, %v", added[0].Error, added[1].Error)
	}
}

func TestACallCanceledBeforeTheFirstWriteReturnsNotAttemptedAndWritesNothing(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"Heat.md": "# Heat\n"})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	added, err := note.NewAddLinks(c.newEditLinks()).Execute(ctx, c.vault, []note.LinkAddition{
		{From: "Heat.md", Link: toName("Entropy", domain.RoleRef)},
	})
	if err != nil {
		t.Fatalf("Execute returned %v, want the results and no error", err)
	}
	if len(added) != 1 || !errors.Is(added[0].Error, plugin.ErrNotAttempted) {
		t.Fatalf("Execute returned %+v, want one result with ErrNotAttempted", added)
	}
	if body := c.read(t, "Heat.md"); body != "# Heat\n" {
		t.Errorf("the note was written anyway:\n%s", body)
	}
}

func TestACallCanceledAfterTheFirstNoteReturnsItsResultAndNotAttemptedForTheRest(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{
		"Heat.md": "# Heat\n", "Work.md": "# Work\n", "Order.md": "# Order\n",
	})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	canceling := c
	canceling.index = func(ctx context.Context, v domain.Vault, paths []string) error {
		err := c.index(ctx, v, paths)
		cancel()
		return err
	}

	added, err := note.NewAddLinks(canceling.newEditLinks()).Execute(ctx, c.vault, []note.LinkAddition{
		{From: "Heat.md", Link: toName("Entropy", domain.RoleRef)},
		{From: "Work.md", Link: toName("Entropy", domain.RoleRef)},
		{From: "Order.md", Link: toName("Entropy", domain.RoleRef)},
	})
	if err != nil {
		t.Fatalf("Execute returned %v, want the results and no error", err)
	}
	if len(added) != 3 {
		t.Fatalf("Execute returned %d results, want 3", len(added))
	}
	heat := c.read(t, "Heat.md")
	if added[0].Error != nil || added[0].FileStat.Size != int64(len(heat)) {
		t.Errorf("result 0 = %+v, want nil error and the file stat of the written note (size %d)", added[0], len(heat))
	}
	if !strings.Contains(heat, "Entropy") {
		t.Errorf("the first note was not written:\n%s", heat)
	}
	for i, path := range map[int]string{1: "Work.md", 2: "Order.md"} {
		if !errors.Is(added[i].Error, plugin.ErrNotAttempted) {
			t.Errorf("result %d error = %v, want ErrNotAttempted", i, added[i].Error)
			continue
		}
		if !strings.Contains(added[i].Error.Error(), path) {
			t.Errorf("result %d error = %q, want it to name %s", i, added[i].Error, path)
		}
		if added[i].FileStat != (domain.FileStat{}) {
			t.Errorf("result %d carries the file stat %+v, want none", i, added[i].FileStat)
		}
	}
	if body := c.read(t, "Work.md"); body != "# Work\n" {
		t.Errorf("Work.md was written after the cancellation:\n%s", body)
	}
	if body := c.read(t, "Order.md"); body != "# Order\n" {
		t.Errorf("Order.md was written after the cancellation:\n%s", body)
	}
}
