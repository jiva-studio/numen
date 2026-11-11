package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/adapter/rpc"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/vaulttest"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/cards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/note"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/wire/cardcommands"
)

type faultyVault struct {
	*hosttest.Vault
	getVaultErr        error
	scanErr            error
	getPathsErr        error
	getNotesErr        error
	getLinksErr        error
	ensureIndexedErr   error
	watchMovesErr      error
	watchMovesChan     <-chan host.Move
	emptyGetNotesCalls int
}

func (f *faultyVault) GetLinks(ctx context.Context, from string) ([]domain.ResolvedLink, error) {
	if f.getLinksErr != nil {
		return nil, f.getLinksErr
	}
	return f.Vault.GetLinks(ctx, from)
}

func (f *faultyVault) With(id domain.VaultID) *faultyVault {
	cp := *f
	cp.Vault = f.Vault.With(id)
	return &cp
}

func (f *faultyVault) GetVault(ctx context.Context) (domain.Vault, error) {
	if f.getVaultErr != nil {
		return domain.Vault{}, f.getVaultErr
	}
	return f.Vault.GetVault(ctx)
}

func (f *faultyVault) Scan(ctx context.Context, paths []string) error {
	if f.scanErr != nil {
		return f.scanErr
	}
	return f.Vault.Scan(ctx, paths)
}

func (f *faultyVault) GetPathsOfType(ctx context.Context, noteType domain.NoteType) ([]string, error) {
	if f.getPathsErr != nil {
		return nil, f.getPathsErr
	}
	return f.Vault.GetPathsOfType(ctx, noteType)
}

func (f *faultyVault) GetNotes(ctx context.Context, paths []string) (map[string]domain.NoteRef, error) {
	if len(paths) == 1 && paths[0] == "" {
		f.emptyGetNotesCalls++
	}
	if f.getNotesErr != nil {
		return nil, f.getNotesErr
	}
	return f.Vault.GetNotes(ctx, paths)
}

func (f *faultyVault) EnsureIndexed(ctx context.Context) (host.IndexState, error) {
	if f.ensureIndexedErr != nil {
		return host.IndexState{}, f.ensureIndexedErr
	}
	return f.Vault.EnsureIndexed(ctx)
}

func (f *faultyVault) WatchMoves(ctx context.Context) (<-chan host.Move, error) {
	if f.watchMovesErr != nil {
		return nil, f.watchMovesErr
	}
	if f.watchMovesChan != nil {
		return f.watchMovesChan, nil
	}
	return f.Vault.WatchMoves(ctx)
}

func TestBackendCoresAndVaultHelpers(t *testing.T) {
	env := vaulttest.New(t, map[string]string{"Term.md": getDeck()["Term.md"]})
	prev := runtime.GOMAXPROCS(2)
	defer runtime.GOMAXPROCS(prev)

	for _, tc := range []struct {
		cores int
		want  int
	}{
		{0, 2},
		{-1, 2},
		{1, 1},
		{3, 3},
	} {
		b := New(Config{Vaults: vaulttest.Vaults(env.Host), Store: vaulttest.Folders(env.Host), Now: vaulttest.NewClock(), Cores: tc.cores})
		if got := b.flash.Get().Reviews.ReviewCards.Cores; got != tc.want {
			t.Fatalf("Cores=%d: expected %d, got %d", tc.cores, tc.want, got)
		}
		if got := runtime.GOMAXPROCS(0); got != 2 {
			t.Fatalf("expected GOMAXPROCS to remain 2, got %d", got)
		}
	}

	countingVault := &faultyVault{Vault: env.Host}
	bCountEmpty := New(Config{
		Vaults: countingVault.vaults(),
		Store:  vaulttest.Folders(env.Host),
		Now:    vaulttest.NewClock(),
	})
	if got := bCountEmpty.getTitle(t.Context(), env.Vault, ""); got != "" {
		t.Fatalf("expected empty title for empty path, got %q", got)
	}
	if countingVault.emptyGetNotesCalls != 0 {
		t.Fatalf("expected no GetNotes call for an empty path, got %d", countingVault.emptyGetNotesCalls)
	}
	if got := bCountEmpty.getTitle(t.Context(), env.Vault, "Term.md"); got != "Term" {
		t.Fatalf("expected Term title, got %q", got)
	}

	bNotesErr := New(Config{
		Vaults: (&faultyVault{Vault: env.Host, getNotesErr: errors.New("notes boom")}).vaults(),
		Store:  vaulttest.Folders(env.Host),
		Now:    vaulttest.NewClock(),
	})
	if got := bNotesErr.getTitle(t.Context(), env.Vault, "Term.md"); got != "" {
		t.Fatalf("expected empty title on GetNotes error, got %q", got)
	}

	bVaultErr := New(Config{
		Vaults: (&faultyVault{Vault: env.Host, getVaultErr: errors.New("vault boom")}).vaults(),
		Store:  vaulttest.Folders(env.Host),
		Now:    vaulttest.NewClock(),
	})
	if v, err := bVaultErr.getCurrentVault(t.Context()); v != (domain.Vault{}) || codeOf(err) != server.CodeInternal {
		t.Fatalf("expected CodeInternal from getCurrentVault, got %+v, %v", v, err)
	}
	if v, err := bVaultErr.getVault(t.Context(), string(env.Vault.ID)); v != (domain.Vault{}) || codeOf(err) != server.CodeInternal {
		t.Fatalf("expected CodeInternal from getVault, got %+v, %v", v, err)
	}

	env.Host.Hide()
	bNoVault := New(Config{Vaults: vaulttest.Vaults(env.Host), Store: vaulttest.Folders(env.Host), Now: vaulttest.NewClock()})
	if v, err := bNoVault.getCurrentVault(t.Context()); v != (domain.Vault{}) || codeOf(err) != server.CodeFailedPrecondition {
		t.Fatalf("expected CodeFailedPrecondition from getCurrentVault, got %+v, %v", v, err)
	}
}

func TestCreateCardNoteReadsTheEndOfTheScenario(t *testing.T) {
	env := vaulttest.New(t, map[string]string{})
	b := New(Config{Vaults: vaulttest.Vaults(env.Host), Store: vaulttest.Folders(env.Host), Now: vaulttest.NewClock()})
	ctx := t.Context()

	made, code, outdated, err := b.createCardNote(ctx, func(domain.Vault, cards.New) (cards.CreateNoteResult, error) {
		return cards.CreateNoteResult{Path: "Deck.md"}, nil
	}, "Deck", "")
	if err != nil || made.Path != "Deck.md" || code != api.ErrorCodeUnspecified || outdated {
		t.Fatalf("unexpected createCardNote ok: %+v, %v, %v, %v", made, code, outdated, err)
	}

	made, code, outdated, err = b.createCardNote(ctx, func(domain.Vault, cards.New) (cards.CreateNoteResult, error) {
		return cards.CreateNoteResult{Path: "Deck.md"}, note.ErrIndexOutdated
	}, "Deck", "")
	if err != nil || made.Path != "Deck.md" || code != api.ErrorCodeUnspecified || !outdated {
		t.Fatalf("unexpected createCardNote outdated: %+v, %v, %v, %v", made, code, outdated, err)
	}

	made, code, outdated, err = b.createCardNote(ctx, func(domain.Vault, cards.New) (cards.CreateNoteResult, error) {
		return cards.CreateNoteResult{}, cards.ErrNoFields
	}, "Model", "")
	if codeOf(err) != server.CodeInvalidArgument || made != (cards.CreateNoteResult{}) || code != api.ErrorCodeUnspecified || outdated {
		t.Fatalf("unexpected createCardNote ErrNoFields: %+v, %v, %v, %v", made, code, outdated, err)
	}

	made, code, outdated, err = b.createCardNote(ctx, func(domain.Vault, cards.New) (cards.CreateNoteResult, error) {
		return cards.CreateNoteResult{}, ports.ErrAlreadyExists
	}, "Deck", "")
	if err != nil || made != (cards.CreateNoteResult{}) || code != api.ErrorCodeAlreadyExists || outdated {
		t.Fatalf("unexpected createCardNote ErrAlreadyExists: %+v, %v, %v, %v", made, code, outdated, err)
	}

	made, code, outdated, err = b.createCardNote(ctx, func(domain.Vault, cards.New) (cards.CreateNoteResult, error) {
		return cards.CreateNoteResult{}, errors.New("disk boom")
	}, "Deck", "")
	if codeOf(err) != server.CodeInternal || made != (cards.CreateNoteResult{}) || code != api.ErrorCodeUnspecified || outdated {
		t.Fatalf("unexpected createCardNote internal error: %+v, %v, %v, %v", made, code, outdated, err)
	}

	env.Host.Hide()
	made, code, outdated, err = b.createCardNote(ctx, func(domain.Vault, cards.New) (cards.CreateNoteResult, error) {
		return cards.CreateNoteResult{Path: "Deck.md"}, nil
	}, "Deck", "")
	if codeOf(err) != server.CodeFailedPrecondition || made != (cards.CreateNoteResult{}) || code != api.ErrorCodeUnspecified || outdated {
		t.Fatalf("unexpected createCardNote no vault: %+v, %v, %v, %v", made, code, outdated, err)
	}
}

func TestCardsAndPresetsHandlersErrorPaths(t *testing.T) {
	ctx := t.Context()

	t.Run("no vault open", func(t *testing.T) {
		env := vaulttest.New(t, map[string]string{})
		env.Host.Hide()
		b := New(Config{Vaults: vaulttest.Vaults(env.Host), Store: vaulttest.Folders(env.Host), Now: vaulttest.NewClock()})
		cardsAPI := b.newCardsHandler()
		presetsAPI := b.newPresetsHandler()

		if _, err := cardsAPI.ListCardModels(ctx, &api.ListCardModelsRequest{}); codeOf(err) != server.CodeFailedPrecondition {
			t.Fatalf("ListCardModels: %v", err)
		}
		if _, err := cardsAPI.CreateCardModel(ctx, &api.CreateCardModelRequest{Title: "M", Fields: []string{"F"}}); codeOf(err) != server.CodeFailedPrecondition {
			t.Fatalf("CreateCardModel: %v", err)
		}
		if _, err := cardsAPI.CreateDeck(ctx, &api.CreateDeckRequest{Title: "D"}); codeOf(err) != server.CodeFailedPrecondition {
			t.Fatalf("CreateDeck: %v", err)
		}
		if _, err := cardsAPI.RenameCardModelField(ctx, &api.RenameCardModelFieldRequest{Path: "M.md", From: "F", To: "G"}); codeOf(err) != server.CodeFailedPrecondition {
			t.Fatalf("RenameCardModelField: %v", err)
		}
		if _, err := cardsAPI.ReadCardModel(ctx, &api.ReadCardModelRequest{Path: "M.md"}); codeOf(err) != server.CodeFailedPrecondition {
			t.Fatalf("ReadCardModel: %v", err)
		}
		if _, err := cardsAPI.ReadDeck(ctx, &api.ReadDeckRequest{Path: "D.md"}); codeOf(err) != server.CodeFailedPrecondition {
			t.Fatalf("ReadDeck: %v", err)
		}
		if _, err := cardsAPI.WriteDeck(ctx, &api.WriteDeckRequest{Path: "D.md"}); codeOf(err) != server.CodeFailedPrecondition {
			t.Fatalf("WriteDeck: %v", err)
		}
		if _, err := cardsAPI.WriteCardModel(ctx, &api.WriteCardModelRequest{Path: "M.md", Fields: []string{"F"}}); codeOf(err) != server.CodeFailedPrecondition {
			t.Fatalf("WriteCardModel: %v", err)
		}

		if _, err := presetsAPI.GetDeckPreset(ctx, &api.GetDeckPresetRequest{Deck: "D.md"}); codeOf(err) != server.CodeFailedPrecondition {
			t.Fatalf("GetDeckPreset: %v", err)
		}
		if _, err := presetsAPI.ListPresets(ctx, &api.ListPresetsRequest{}); codeOf(err) != server.CodeFailedPrecondition {
			t.Fatalf("ListPresets: %v", err)
		}
		if _, err := presetsAPI.CreatePreset(ctx, &api.CreatePresetRequest{Title: "P"}); codeOf(err) != server.CodeFailedPrecondition {
			t.Fatalf("CreatePreset: %v", err)
		}
		if _, err := presetsAPI.ScheduleDeck(ctx, &api.ScheduleDeckRequest{Deck: "D.md"}); codeOf(err) != server.CodeFailedPrecondition {
			t.Fatalf("ScheduleDeck: %v", err)
		}
		if _, err := presetsAPI.ReadPreset(ctx, &api.ReadPresetRequest{Path: "P.md"}); codeOf(err) != server.CodeFailedPrecondition {
			t.Fatalf("ReadPreset: %v", err)
		}
		if _, err := presetsAPI.WritePreset(ctx, &api.WritePresetRequest{Path: "P.md"}); codeOf(err) != server.CodeFailedPrecondition {
			t.Fatalf("WritePreset: %v", err)
		}
		if _, err := presetsAPI.ComputeCurve(ctx, &api.ComputeCurveRequest{Path: "P.md"}); codeOf(err) != server.CodeFailedPrecondition {
			t.Fatalf("ComputeCurve: %v", err)
		}
	})

	t.Run("list failure", func(t *testing.T) {
		env := vaulttest.New(t, map[string]string{})
		b := New(Config{
			Vaults: (&faultyVault{Vault: env.Host, getPathsErr: errors.New("paths boom"), getLinksErr: errors.New("links boom")}).vaults(),
			Store:  vaulttest.Folders(env.Host),
			Now:    vaulttest.NewClock(),
		})
		if _, err := (b.newCardsHandler()).ListCardModels(ctx, &api.ListCardModelsRequest{}); codeOf(err) != server.CodeInternal {
			t.Fatalf("expected CodeInternal from ListCardModels, got %v", err)
		}
		if _, err := (b.newPresetsHandler()).ListPresets(ctx, &api.ListPresetsRequest{}); codeOf(err) != server.CodeInternal {
			t.Fatalf("expected CodeInternal from ListPresets, got %v", err)
		}
		if _, err := (b.newPresetsHandler()).GetDeckPreset(ctx, &api.GetDeckPresetRequest{Deck: "D.md"}); codeOf(err) != server.CodeInternal {
			t.Fatalf("expected CodeInternal from GetDeckPreset, got %v", err)
		}
	})

	t.Run("index outdated on writes", func(t *testing.T) {
		env := vaulttest.New(t, getPointed())
		b := New(Config{
			Vaults: (&faultyVault{Vault: env.Host, scanErr: errors.New("scan boom")}).vaults(),
			Store:  vaulttest.Folders(env.Host),
			Now:    vaulttest.NewClock(),
		})
		cardsAPI := b.newCardsHandler()
		presetsAPI := b.newPresetsHandler()

		created, err := cardsAPI.CreateDeck(ctx, &api.CreateDeckRequest{Title: "Fresh"})
		if err != nil || !created.IsIndexOutdated || created.Path != "Fresh.md" || created.Error != api.ErrorCodeUnspecified {
			t.Fatalf("unexpected CreateDeck outdated: %+v, %v", created, err)
		}

		renamed, err := cardsAPI.RenameCardModelField(ctx, &api.RenameCardModelFieldRequest{
			Path: "Term.md", From: "Word", To: "TermWord",
		})
		if err != nil || !renamed.IsIndexOutdated || renamed.At == nil || renamed.Error != api.ErrorCodeUnspecified {
			t.Fatalf("unexpected RenameCardModelField outdated: %+v, %v", renamed, err)
		}

		wroteDeck, err := cardsAPI.WriteDeck(ctx, &api.WriteDeckRequest{Path: "decks/Words.md"})
		if err != nil || !wroteDeck.IsIndexOutdated || wroteDeck.At == nil || wroteDeck.Bound != 0 {
			t.Fatalf("unexpected WriteDeck outdated: %+v, %v", wroteDeck, err)
		}

		wroteModel, err := cardsAPI.WriteCardModel(ctx, &api.WriteCardModelRequest{
			Path: "Term.md", Fields: []string{"TermWord", "Meaning"},
		})
		if err != nil || !wroteModel.IsIndexOutdated || wroteModel.At == nil {
			t.Fatalf("unexpected WriteCardModel outdated: %+v, %v", wroteModel, err)
		}

		scheduled, err := presetsAPI.ScheduleDeck(ctx, &api.ScheduleDeckRequest{
			Deck: "decks/Words.md", Preset: "Sanskrit.md",
		})
		if err != nil || !scheduled.IsIndexOutdated || scheduled.At == nil {
			t.Fatalf("unexpected ScheduleDeck outdated: %+v, %v", scheduled, err)
		}

		readPreset, err := presetsAPI.ReadPreset(ctx, &api.ReadPresetRequest{Path: "Sanskrit.md"})
		if err != nil || readPreset.Preset == nil {
			t.Fatalf("unexpected ReadPreset: %+v, %v", readPreset, err)
		}
		wrotePreset, err := presetsAPI.WritePreset(ctx, &api.WritePresetRequest{
			Path: "Sanskrit.md", Settings: readPreset.Preset.Settings,
		})
		if err != nil || !wrotePreset.IsIndexOutdated || wrotePreset.At == nil {
			t.Fatalf("unexpected WritePreset outdated: %+v, %v", wrotePreset, err)
		}
	})

	t.Run("validation, bounds, and error codes", func(t *testing.T) {
		notes := getPointed()
		notes["Huge.md"] = "---\ntype: deck\n---\n\n" + strings.Repeat("x", int(cards.MaxBytes)+10)
		env := vaulttest.New(t, notes)
		b := New(Config{Vaults: vaulttest.Vaults(env.Host), Store: vaulttest.Folders(env.Host), Now: vaulttest.NewClock()})
		cardsAPI := b.newCardsHandler()
		presetsAPI := b.newPresetsHandler()

		if _, err := cardsAPI.CreateCardModel(ctx, &api.CreateCardModelRequest{Title: "Empty"}); codeOf(err) != server.CodeInvalidArgument {
			t.Fatalf("expected CodeInvalidArgument for empty fields, got %v", err)
		}

		for _, tc := range []struct{ path, from, to, label string }{
			{"Term.md", "MissingField", "NewField", "ErrNoSuchField"},
			{"Term.md", "Word", "Meaning", "ErrFieldTaken"},
			{"../../outside.md", "Word", "Other", "outside path"},
		} {
			if _, err := cardsAPI.RenameCardModelField(ctx, &api.RenameCardModelFieldRequest{
				Path: tc.path, From: tc.from, To: tc.to,
			}); codeOf(err) != server.CodeInvalidArgument {
				t.Fatalf("expected CodeInvalidArgument for %s, got %v", tc.label, err)
			}
		}
		renamedStale, err := cardsAPI.RenameCardModelField(ctx, &api.RenameCardModelFieldRequest{
			Path:     "Term.md",
			From:     "Word",
			To:       "Other",
			Baseline: &api.FileStat{Path: "Term.md", Size: 999999, Mtime: 1},
		})
		if err != nil || renamedStale.Error != api.ErrorCodeConflict || renamedStale.Cards != 0 || renamedStale.IsIndexOutdated {
			t.Fatalf("unexpected RenameCardModelField on stale: %+v, %v", renamedStale, err)
		}

		readOK, err := cardsAPI.ReadDeck(ctx, &api.ReadDeckRequest{Path: "decks/Words.md"})
		if err != nil || readOK.Error != api.ErrorCodeUnspecified || readOK.Bound != 0 {
			t.Fatalf("unexpected ReadDeck ok: %+v, %v", readOK, err)
		}
		readMissing, err := cardsAPI.ReadDeck(ctx, &api.ReadDeckRequest{Path: "Missing.md"})
		if err != nil || readMissing.Error != api.ErrorCodeNotFound || readMissing.Bound != 0 {
			t.Fatalf("unexpected ReadDeck missing: %+v, %v", readMissing, err)
		}
		readHuge, err := cardsAPI.ReadDeck(ctx, &api.ReadDeckRequest{Path: "Huge.md"})
		if err != nil || readHuge.Error != api.ErrorCodeDeckTooLarge || readHuge.Bound != cards.MaxBytes {
			t.Fatalf("unexpected ReadDeck huge: %+v, %v", readHuge, err)
		}

		writeOK, err := cardsAPI.WriteDeck(ctx, &api.WriteDeckRequest{Path: "decks/Words.md"})
		if err != nil || writeOK.Error != api.ErrorCodeUnspecified || writeOK.Bound != 0 || writeOK.IsIndexOutdated {
			t.Fatalf("unexpected WriteDeck ok: %+v, %v", writeOK, err)
		}
		writeHuge, err := cardsAPI.WriteDeck(ctx, &api.WriteDeckRequest{
			Path:     "decks/Words.md",
			Preamble: strings.Repeat("x", int(cards.MaxBytes)+10),
		})
		if err != nil || writeHuge.Error != api.ErrorCodeDeckTooLarge || writeHuge.Bound != cards.MaxBytes || writeHuge.IsIndexOutdated {
			t.Fatalf("unexpected WriteDeck huge: %+v, %v", writeHuge, err)
		}
		writeMissing, err := cardsAPI.WriteDeck(ctx, &api.WriteDeckRequest{Path: "Missing.md"})
		if err != nil || writeMissing.Error != api.ErrorCodeNotFound || writeMissing.Bound != 0 || writeMissing.IsIndexOutdated {
			t.Fatalf("unexpected WriteDeck missing: %+v, %v", writeMissing, err)
		}
		if _, err := cardsAPI.WriteDeck(ctx, &api.WriteDeckRequest{Path: "../../outside.md"}); codeOf(err) != server.CodeInvalidArgument {
			t.Fatalf("expected CodeInvalidArgument for WriteDeck outside, got %v", err)
		}
		if _, err := cardsAPI.WriteCardModel(ctx, &api.WriteCardModelRequest{Path: "../../outside.md", Fields: []string{"Word"}}); codeOf(err) != server.CodeInvalidArgument {
			t.Fatalf("expected CodeInvalidArgument for WriteCardModel outside, got %v", err)
		}

		deckPreset, err := presetsAPI.GetDeckPreset(ctx, &api.GetDeckPresetRequest{Deck: "decks/Words.md"})
		if err != nil || deckPreset.At == nil || deckPreset.At.Path != "Sanskrit.md" {
			t.Fatalf("expected GetDeckPreset At for Sanskrit.md, got %+v, %v", deckPreset, err)
		}
		if _, err := presetsAPI.CreatePreset(ctx, &api.CreatePresetRequest{Title: "P", Path: "../outside"}); codeOf(err) != server.CodeInvalidArgument {
			t.Fatalf("expected CodeInvalidArgument for CreatePreset outside, got %v", err)
		}
		if _, err := presetsAPI.ScheduleDeck(ctx, &api.ScheduleDeckRequest{Deck: "../../outside.md"}); codeOf(err) != server.CodeInvalidArgument {
			t.Fatalf("expected CodeInvalidArgument for ScheduleDeck outside, got %v", err)
		}
		if _, err := presetsAPI.ReadPreset(ctx, &api.ReadPresetRequest{Path: "../../outside.md"}); codeOf(err) != server.CodeInvalidArgument {
			t.Fatalf("expected CodeInvalidArgument for ReadPreset outside, got %v", err)
		}

		validSettings := deckPreset.Preset.Settings
		notAPreset, err := presetsAPI.WritePreset(ctx, &api.WritePresetRequest{
			Path: "decks/Words.md", Settings: validSettings,
		})
		if err != nil || notAPreset.Error != api.ErrorCodeNotAPreset || notAPreset.At != nil || notAPreset.IsIndexOutdated {
			t.Fatalf("unexpected WritePreset on deck: %+v, %v", notAPreset, err)
		}
		if _, err := presetsAPI.WritePreset(ctx, &api.WritePresetRequest{
			Path: "../../outside.md", Settings: validSettings,
		}); codeOf(err) != server.CodeInvalidArgument {
			t.Fatalf("expected CodeInvalidArgument for WritePreset outside, got %v", err)
		}

		if _, err := presetsAPI.WritePreset(ctx, &api.WritePresetRequest{
			Path: "Sanskrit.md", Settings: &api.Settings{ByDate: "not-a-date"},
		}); codeOf(err) != server.CodeInvalidArgument {
			t.Fatalf("expected CodeInvalidArgument for WritePreset invalid ByDate, got %v", err)
		}
		if _, err := presetsAPI.ComputeCurve(ctx, &api.ComputeCurveRequest{
			Path: "Sanskrit.md", Settings: &api.Settings{ByDate: "not-a-date"},
		}); codeOf(err) != server.CodeInvalidArgument {
			t.Fatalf("expected CodeInvalidArgument for ComputeCurve invalid settings, got %v", err)
		}
		badRetention := *validSettings
		badRetention.Goal = api.GoalRetention
		badRetention.Retention = 1.5
		if _, err := presetsAPI.ComputeCurve(ctx, &api.ComputeCurveRequest{
			Path: "Sanskrit.md", Settings: &badRetention,
		}); err == nil {
			t.Fatal("expected error from ComputeCurve with out-of-bounds retention")
		}
	})
}

func TestACommandWithNoVaultOpenFails(t *testing.T) {
	env := vaulttest.New(t, map[string]string{})
	env.Host.Hide()
	cmds, err := New(Config{Vaults: vaulttest.Vaults(env.Host), Store: vaulttest.Folders(env.Host), Now: vaulttest.NewClock()}).Commands()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cmds.RunCommand(t.Context(), command.Request{
		Command: cardcommands.CreateDeck, Arguments: command.Arguments{cardcommands.ArgumentTitle: "D"},
	}); codeOf(err) != server.CodeFailedPrecondition {
		t.Fatalf("expected CodeFailedPrecondition from command with no vault, got %v", err)
	}
}

func TestReviewInternalPaths(t *testing.T) {
	ctx := t.Context()

	env := vaulttest.New(t, getDeck())
	bEnsureErr := New(Config{
		Vaults: (&faultyVault{Vault: env.Host, ensureIndexedErr: errors.New("ensure boom")}).vaults(),
		Store:  vaulttest.Folders(env.Host),
		Now:    vaulttest.NewClock(),
	})
	hEnsureErr := bEnsureErr.newReviewHandler()
	if got := hEnsureErr.countVault(ctx, env.Vault); got == nil || got.Unread != "ensure boom" {
		t.Fatalf("expected Unread='ensure boom', got %+v", got)
	}

	bDueErr := New(Config{
		Vaults: (&faultyVault{Vault: env.Host, getPathsErr: errors.New("due boom")}).vaults(),
		Store:  vaulttest.Folders(env.Host),
		Now:    vaulttest.NewClock(),
	})
	hDueErr := bDueErr.newReviewHandler()
	if got := hDueErr.countVault(ctx, env.Vault); got == nil || got.Unread != "due boom" {
		t.Fatalf("expected Unread='due boom', got %+v", got)
	}

	handler, target := newAPI(t, getDeck())
	v := target[0]
	sess := startSession(t, handler, v)
	card := sess.Cards[0]
	if _, err := handler.RecordReview(ctx, &api.RecordReviewRequest{
		Vault: string(v.ID), Run: sess.Run, CardID: card.CardID, Template: card.Template, Rating: api.RatingEasy,
	}); err != nil {
		t.Fatal(err)
	}
	daysOut, err := handler.ListReviewDays(ctx, &api.ListReviewDaysRequest{Vault: string(v.ID)})
	if err != nil || len(daysOut.Due) != 1 {
		t.Fatalf("unexpected ListReviewDays: %+v, %v", daysOut, err)
	}
	dueDay := daysOut.Due[0]
	if dueDay.Reviewed != 1 || dueDay.Again != 0 || dueDay.Hard != 0 || dueDay.Good != 0 ||
		dueDay.Easy != 0 || dueDay.Tested != 0 || dueDay.Recalled != 0 {
		t.Fatalf("unexpected Due ReviewDay fields: %+v", dueDay)
	}

	bWatchErr := New(Config{
		Vaults: (&faultyVault{Vault: env.Host, watchMovesErr: errors.New("watch boom")}).vaults(),
		Store:  vaulttest.Folders(env.Host),
		Now:    vaulttest.NewClock(),
	})
	if err := (bWatchErr.newReviewHandler()).WatchReloads(ctx, &api.WatchReloadsRequest{}, func(*api.WatchReloadsResponse) error {
		return nil
	}); codeOf(err) != server.CodeUnavailable {
		t.Fatalf("expected CodeUnavailable when WatchMoves fails, got %v", err)
	}

	closedMoves := make(chan host.Move)
	close(closedMoves)
	bClosedMoves := New(Config{
		Vaults: (&faultyVault{Vault: env.Host, watchMovesChan: closedMoves}).vaults(),
		Store:  vaulttest.Folders(env.Host),
		Now:    vaulttest.NewClock(),
	})
	if err := (bClosedMoves.newReviewHandler()).WatchReloads(ctx, &api.WatchReloadsRequest{}, func(*api.WatchReloadsResponse) error {
		return nil
	}); err != nil {
		t.Fatalf("expected nil when WatchMoves channel closes cleanly, got %v", err)
	}

	errMoves := make(chan host.Move, 1)
	errMoves <- host.Move{Err: errors.New("move boom")}
	bErrMoves := New(Config{
		Vaults: (&faultyVault{Vault: env.Host, watchMovesChan: errMoves}).vaults(),
		Store:  vaulttest.Folders(env.Host),
		Now:    vaulttest.NewClock(),
	})
	if err := (bErrMoves.newReviewHandler()).WatchReloads(ctx, &api.WatchReloadsRequest{}, func(*api.WatchReloadsResponse) error {
		return nil
	}); codeOf(err) != server.CodeUnavailable {
		t.Fatalf("expected CodeUnavailable when move.Err != nil, got %v", err)
	}
}

func TestMountAndAllEndpoints(t *testing.T) {
	env := vaulttest.New(t, getPointed())
	b := New(Config{Vaults: vaulttest.Vaults(env.Host), Store: vaulttest.Folders(env.Host), Now: vaulttest.NewClock()})

	const tok = "test-mount-token"
	if _, err := server.NewLinkedHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": tok}, func(s *server.Server) error {
		return rpc.Mount(s.NewAPI(""), b.Handlers())
	}); err == nil {
		t.Fatal("expected rpc.Mount with invalid service name to return error")
	}

	handler, err := server.NewLinkedHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": tok}, b.Mount)
	if err != nil {
		t.Fatal(err)
	}

	post := func(path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	if rec := post("/numen.v1.PluginToolService/ListTools", `{}`); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "card_model_list") {
		t.Fatalf("expected ListTools 200 OK with tools, got %d: %s", rec.Code, rec.Body)
	}
	if rec := post("/numen.v1.PluginCommandService/RunCommand", `{"command":"deck.create","arguments":{"title":"ViaCommand"}}`); rec.Code != http.StatusOK {
		t.Fatalf("expected RunCommand 200 OK, got %d: %s", rec.Code, rec.Body)
	}

	for _, method := range []string{
		"ListCardModels",
		"CreateCardModel",
		"ReadCardModel",
		"WriteCardModel",
		"RenameCardModelField",
		"CreateDeck",
		"ReadDeck",
		"WriteDeck",
		"GetDeckPreset",
		"ListPresets",
		"CreatePreset",
		"ScheduleDeck",
		"ReadPreset",
		"WritePreset",
		"ComputeCurve",
		"StartSession",
		"RecordReview",
		"UndoReview",
		"ListReviewDays",
		"GetDeckNeighborhood",
		"GetVaultDeckPreset",
		"WatchCardsDue",
	} {
		rec := post("/"+ServiceName+"/"+method, `null`)
		if rec.Code == http.StatusNotFound && !strings.Contains(rec.Body.String(), `"code":"not_found"`) {
			t.Fatalf("method %s not mounted (404): %s", method, rec.Body)
		}
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/"+ServiceName+"/WatchReloads", strings.NewReader(`null`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code == http.StatusNotFound {
		t.Fatalf("WatchReloads not mounted (404): %s", rec.Body)
	}

	emptyEnv := vaulttest.New(t, map[string]string{})
	emptyB := New(Config{Vaults: vaulttest.Vaults(emptyEnv.Host), Store: vaulttest.Folders(emptyEnv.Host), Now: vaulttest.NewClock()})
	emptyHandler, err := server.NewLinkedHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": tok}, emptyB.Mount)
	if err != nil {
		t.Fatal(err)
	}
	reqModels := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/"+ServiceName+"/ListCardModels", strings.NewReader(`null`))
	reqModels.Header.Set("Authorization", "Bearer "+tok)
	reqModels.Header.Set("Content-Type", "application/json")
	recModels := httptest.NewRecorder()
	emptyHandler.ServeHTTP(recModels, reqModels)
	if recModels.Code != http.StatusOK || !strings.Contains(recModels.Body.String(), `"cardModels":[]`) {
		t.Fatalf("expected cardModels:[], got %d: %s", recModels.Code, recModels.Body)
	}
}

func (f *faultyVault) vaults() func(domain.VaultID) ports.HostVault {
	return func(id domain.VaultID) ports.HostVault {
		on := f
		if id != "" {
			on = f.With(id)
		}
		return ports.HostVault{Files: on, Writer: on, Index: on, Notes: on, Links: on, Cache: on, Selector: on}
	}
}
