package editor

import (
	"slices"
	"testing"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/settings"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// Every value of every enum this package reads or writes reaches something.
//
// The values are walked from the schema, so a value added there is a test that
// fails and not a field that quietly carries the unspecified value.

func TestEverySearchModeIsRead(t *testing.T) {
	testsupport.CheckHandled(t, func(mode v1.SearchMode) bool {
		_, named := convertSearchModeFromWire(mode)
		return named
	})
}

func TestEveryRoleIsRead(t *testing.T) {
	testsupport.CheckHandled(t, func(role v1.Role) bool {
		_, named := convertRoleFromWire(role)
		return named
	})
}

func TestEveryRelationIsSavedFromOne(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.Relation]domain.Relation{
		v1.Relation_RELATION_PARENT:  domain.RelationParent,
		v1.Relation_RELATION_CHILD:   domain.RelationChild,
		v1.Relation_RELATION_JUMP:    domain.RelationJump,
		v1.Relation_RELATION_SIBLING: domain.RelationSibling,
	}, convertRelationToWire)
}

func TestANoteTypeIsSavedAsTheWordItContainsAndAnOrdinaryNoteAsNone(t *testing.T) {
	for typed, want := range map[domain.NoteType]string{
		domain.TypeNote: "",
		"deck":          "deck",
		"sheet":         "sheet",
	} {
		if got := convertNoteTypeToWire(typed); got != want {
			t.Errorf("%q is written as %q, want %q", typed, got, want)
		}
	}
}

func TestEverySourceKindIsSavedFromOne(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.SourceKind]domain.SourceKind{
		v1.SourceKind_SOURCE_KIND_NOTE:      domain.KindNote,
		v1.SourceKind_SOURCE_KIND_BOOK:      domain.KindBook,
		v1.SourceKind_SOURCE_KIND_RECORDING: domain.KindRecording,
		v1.SourceKind_SOURCE_KIND_URL:       domain.KindURL,
	}, convertSourceKindToWire)
}

func TestEveryBookFormatIsSavedFromOne(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.BookFormat]string{
		v1.BookFormat_BOOK_FORMAT_EPUB: "library/a.epub",
		v1.BookFormat_BOOK_FORMAT_PDF:  "library/a.pdf",
	}, func(path string) v1.BookFormat { return convertBookFormatToWire(path, domain.KindBook) })
}

func TestEveryNamingIsSavedFromOne(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.NameSource]note.NameSource{
		v1.NameSource_NAME_SOURCE_FRONTMATTER: note.ByFrontmatter,
		v1.NameSource_NAME_SOURCE_FILENAME:    note.ByFilename,
	}, newWireNameSource)
}

func TestEveryPresenceIsSavedFromOne(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.Presence]port.Presence{
		v1.Presence_PRESENCE_PRESENT:          port.Present,
		v1.Presence_PRESENCE_NOT_FETCHED:      port.NotFetched,
		v1.Presence_PRESENCE_NOTHING_TO_FETCH: port.NothingToFetch,
	}, func(is port.Presence) v1.Presence { return presences[is] })
}

func TestEveryCompletionConsentIsSavedFromOne(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.CompletionConsent]settings.Consent{
		v1.CompletionConsent_COMPLETION_CONSENT_UNANSWERED: settings.ConsentUnanswered,
		v1.CompletionConsent_COMPLETION_CONSENT_ASKED:      settings.ConsentAsked,
		v1.CompletionConsent_COMPLETION_CONSENT_ALLOWED:    settings.ConsentAllowed,
		v1.CompletionConsent_COMPLETION_CONSENT_REFUSED:    settings.ConsentRefused,
	}, func(is settings.Consent) v1.CompletionConsent { return consents[is] })
}

func TestEveryVaultsErrorCodeIsSavedFromOne(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.VaultsErrorCode]error{
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNREADABLE: vaults.ErrUnreadable,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_COPY:       vaults.ErrCopy,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_OVERLAPS:   domain.ErrOverlaps,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_NAME_TAKEN: vaults.ErrNameTaken,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_LAST_VAULT: vaults.ErrLastVault,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_SHOWING:    vaults.ErrShowing,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNKNOWN:    vaults.ErrUnknown,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_NO_TRASH:   port.ErrNoTrash,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_ASKING:     errAnswerPending,
	}, func(err error) v1.VaultsErrorCode {
		reason, _ := vaultsErrorCodeBy(err)
		return reason
	})
}

// and what a run just set going is. Each schema value names the call that
// writes it.
func TestEveryArtifactStateIsSavedFromOne(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.State]func() v1.State{
		v1.State_STATE_NONE:    getStateAt(source.RunProgress{State: source.RunNone}),
		v1.State_STATE_DONE:    getStateAt(source.RunProgress{State: source.RunComplete}),
		v1.State_STATE_RUNNING: getStateAt(source.RunProgress{State: source.RunActive}),
		v1.State_STATE_STOPPED: getStateAt(source.RunProgress{State: source.RunStopped}),
		v1.State_STATE_EMPTY:   getStateAt(source.RunProgress{State: source.RunNoSpeech}),
		v1.State_STATE_FAILED:  getStateAt(source.RunProgress{State: source.RunUnopened}),
		v1.State_STATE_QUEUED:  func() v1.State { return getStartState(port.Queued) },
	}, func(writes func() v1.State) v1.State { return writes() })
}

// Every artifact the schema names is placed under a name in the store, and is
// carried by a file of some kind.
func TestEveryArtifactTheSchemaNamesExistsSomewhere(t *testing.T) {
	testsupport.CheckHandled(t, hasArtifact)
	testsupport.CheckHandled(t, func(of v1.ArtifactKind) bool {
		// A url carries what was fetched from the address it holds, and which
		// of the two texts that is follows from what fetched it.
		return slices.Contains(getArtifactKinds(domain.KindBook, ""), of) ||
			slices.Contains(getArtifactKinds(domain.KindRecording, ""), of) ||
			slices.Contains(getArtifactKinds(domain.KindURL, domain.Captions), of) ||
			slices.Contains(getArtifactKinds(domain.KindURL, domain.Article), of)
	})
}

// getStateAt is what an artifact over a source standing here is answered with.
func getStateAt(got source.RunProgress) func() v1.State {
	return func() v1.State {
		return newRunArtifact(v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT, got).GetState()
	}
}

// Which extensions are books and which reader cuts each of them are two lists,
// and a book the vault names with no reader behind it opens in the reader made
// picture reader.
func TestEveryBookTheVaultNamesOpensInAReader(t *testing.T) {
	for _, extension := range filesystem.DefaultBookExtensions {
		path := "library/a" + extension
		if got := convertBookFormatToWire(path, domain.KindBook); got == v1.BookFormat_BOOK_FORMAT_UNSPECIFIED {
			t.Errorf("%s is a book of the vault and opens in no reader", extension)
		}
	}
}
