package service

import (
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/internal/testsupport"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/usecase/texts"
)

func newFollowedBook(t *testing.T) (*hosttest.Vault, *hosttest.Texts, *texts.Follower) {
	t.Helper()
	files, versions, writer := newTextVault(t, "book.epub", testsupport.CreateImageEpub(t))
	follower := texts.NewFollower(writer, func(id domain.VaultID) texts.Feed {
		return texts.Feed{Changes: files.With(id), Store: files.Store().With(id)}
	}, func(what string, err error) { t.Errorf("%s: %v", what, err) })
	t.Cleanup(follower.Close)
	return files, versions, follower
}

func listKinds(t *testing.T, versions *hosttest.Texts) []string {
	t.Helper()
	listing, err := versions.ListVersions(t.Context(), "book.epub")
	if err != nil {
		t.Fatal(err)
	}
	kinds := make([]string, 0, len(listing.Versions))
	for _, one := range listing.Versions {
		kinds = append(kinds, string(one.Kind))
	}
	return kinds
}

func TestAFilesChangedEventWritesTheTextsTheFeedNames(t *testing.T) {
	_, versions, follower := newFollowedBook(t)
	app := &App{follower: follower}

	if err := app.HandleEvent(t.Context(), hosttest.GetIndexChangedEvent("vault")); err != nil {
		t.Fatal(err)
	}
	follower.Wait()
	if kinds := listKinds(t, versions); len(kinds) != 0 {
		t.Fatalf("an onIndexChanged event made the texts %v", kinds)
	}

	if err := app.HandleEvent(t.Context(), "onFilesChanged:vault"); err != nil {
		t.Fatal(err)
	}
	follower.Wait()
	if kinds := listKinds(t, versions); !slices.Equal(kinds, []string{string(texts.Kind)}) {
		t.Errorf("the versions of book.epub are of kinds %v, want one of kind %q", kinds, texts.Kind)
	}
}

func TestAStartedPluginFollowsTheFeedOfTheShownVault(t *testing.T) {
	files, versions, follower := newFollowedBook(t)
	s := server.New()
	followShownVault(s, files.GetVault, follower)
	defer func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	}()

	deadline := time.After(10 * time.Second)
	for {
		follower.Wait()
		if kinds := listKinds(t, versions); slices.Equal(kinds, []string{string(texts.Kind)}) {
			return
		}
		select {
		case <-deadline:
			t.Error("the started plugin made no text of book.epub")
			return
		case <-time.After(10 * time.Millisecond):
		}
	}
}
