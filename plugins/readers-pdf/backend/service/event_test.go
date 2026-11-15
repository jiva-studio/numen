package service

import (
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/internal/testsupport"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/usecase/extract"
)

func TestAFilesChangedEventWritesTheTextLayersTheFeedNames(t *testing.T) {
	f := testsupport.NewFollowedVault(t, map[string][]byte{"books/tiny.pdf": testsupport.ReadFixture(t, "tiny.pdf")})
	follower := f.NewFollower(t)
	app := &App{follower: follower}

	if err := app.HandleEvent(t.Context(), "onIndexChanged:vault"); err != nil {
		t.Fatal(err)
	}
	follower.Wait()
	if got := f.Texts.TakeChanges(); len(got) != 0 {
		t.Fatalf("an onIndexChanged event made the text layers of %v", got)
	}

	if err := app.HandleEvent(t.Context(), "onFilesChanged:vault"); err != nil {
		t.Fatal(err)
	}
	follower.Wait()
	if text := testsupport.ReadLive(t, f.Texts, "books/tiny.pdf"); text.Version().Kind != extract.TextKind {
		t.Errorf("the live version is of kind %q, want %q", text.Version().Kind, extract.TextKind)
	}
}

func TestAStartedPluginFollowsTheFeedOfTheShownVault(t *testing.T) {
	f := testsupport.NewFollowedVault(t, map[string][]byte{"books/tiny.pdf": testsupport.ReadFixture(t, "tiny.pdf")})
	follower := f.NewFollower(t)
	s := server.New()

	startFollowing(s, f.Vault, follower)
	defer func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	}()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		follower.Wait()
		if got := f.Texts.TakeChanges(); slices.Equal(got, []string{"books/tiny.pdf"}) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Error("the started plugin made no text layer of books/tiny.pdf")
}
