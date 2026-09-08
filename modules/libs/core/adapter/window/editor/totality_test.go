package editor

import (
	"slices"
	"testing"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/format"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	derived "github.com/jiva-studio/numen/modules/libs/core/text"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// Every value of every enum this package reads or writes reaches something.
//
// The values are walked from the schema, so a value added there is a test that
// fails and not a field that quietly carries the unspecified value.

func TestEverySearchModeIsRead(t *testing.T) {
	testsupport.Handled(t, func(mode v1.SearchMode) bool {
		_, named := modeOf(mode)
		return named
	})
}

func TestEveryRoleIsRead(t *testing.T) {
	testsupport.Handled(t, func(role v1.Role) bool {
		_, named := roleOf(role)
		return named
	})
}

func TestEveryFaultIsWrittenFromOne(t *testing.T) {
	testsupport.Produced(t, map[v1.Fault]format.Fault{
		v1.Fault_FAULT_FIELD_DECLARED_TWICE:   format.FaultTwoFields,
		v1.Fault_FAULT_STENCIL_WITHOUT_FIELDS: format.FaultNoFields,
		v1.Fault_FAULT_FACE_MISSING_A_SIDE:    format.FaultFaceSide,
		v1.Fault_FAULT_PLACEHOLDER_UNDECLARED: format.FaultPlaceholder,
		v1.Fault_FAULT_CARD_WITHOUT_A_STENCIL: format.FaultNoStencil,
		v1.Fault_FAULT_STENCIL_IS_NOT_ONE:     format.FaultNotAStencil,
		v1.Fault_FAULT_FIELD_WRITTEN_TWICE:    format.FaultTwoValues,
		v1.Fault_FAULT_FIELD_NOT_RENAMED:      format.FaultNotWritten,
		v1.Fault_FAULT_MARK_CARRIED_TWICE:     format.FaultTwoMarks,
	}, func(fault format.Fault) v1.Fault {
		named, _ := faultOf(fault)
		return named
	})
}

func TestEverySeatIsWrittenFromOne(t *testing.T) {
	testsupport.Produced(t, map[v1.Seat]domain.Relation{
		v1.Seat_SEAT_PARENT:  domain.SeatParent,
		v1.Seat_SEAT_CHILD:   domain.SeatChild,
		v1.Seat_SEAT_JUMP:    domain.SeatJump,
		v1.Seat_SEAT_SIBLING: domain.SeatSibling,
	}, seatOf)
}

func TestEveryNoteTypeIsWrittenFromOne(t *testing.T) {
	testsupport.Produced(t, map[v1.NoteType]domain.NoteType{
		v1.NoteType_NOTE_TYPE_DECK:    domain.TypeDeck,
		v1.NoteType_NOTE_TYPE_STENCIL: domain.TypeStencil,
		v1.NoteType_NOTE_TYPE_PRESET:  domain.TypePreset,
	}, typeOf)
}

func TestEverySourceKindIsWrittenFromOne(t *testing.T) {
	testsupport.Produced(t, map[v1.SourceKind]domain.SourceKind{
		v1.SourceKind_SOURCE_KIND_NOTE:      domain.KindNote,
		v1.SourceKind_SOURCE_KIND_BOOK:      domain.KindBook,
		v1.SourceKind_SOURCE_KIND_RECORDING: domain.KindRecording,
		v1.SourceKind_SOURCE_KIND_URL:       domain.KindURL,
	}, kindOf)
}

func TestEveryBookFormatIsWrittenFromOne(t *testing.T) {
	testsupport.Produced(t, map[v1.BookFormat]string{
		v1.BookFormat_BOOK_FORMAT_EPUB: "library/a.epub",
		v1.BookFormat_BOOK_FORMAT_PDF:  "library/a.pdf",
	}, func(path string) v1.BookFormat { return formatOf(path, domain.KindBook) })
}

func TestEveryNamingIsWrittenFromOne(t *testing.T) {
	testsupport.Produced(t, map[v1.NamedBy]note.NameSource{
		v1.NamedBy_NAMED_BY_FRONTMATTER: note.ByFrontmatter,
		v1.NamedBy_NAMED_BY_FILENAME:    note.ByFilename,
	}, namedByOf)
}

func TestEveryPresenceIsWrittenFromOne(t *testing.T) {
	testsupport.Produced(t, map[v1.Presence]port.Presence{
		v1.Presence_PRESENCE_PRESENT:          port.Present,
		v1.Presence_PRESENCE_NOT_FETCHED:      port.NotFetched,
		v1.Presence_PRESENCE_NOTHING_TO_FETCH: port.NothingToFetch,
	}, func(is port.Presence) v1.Presence { return presences[is] })
}

func TestEveryVaultsRefusalIsWrittenFromOne(t *testing.T) {
	testsupport.Produced(t, map[v1.VaultsRefusal]error{
		v1.VaultsRefusal_VAULTS_REFUSAL_UNREADABLE: vaults.ErrUnreadable,
		v1.VaultsRefusal_VAULTS_REFUSAL_COPY:       vaults.ErrCopy,
		v1.VaultsRefusal_VAULTS_REFUSAL_OVERLAPS:   domain.ErrOverlaps,
		v1.VaultsRefusal_VAULTS_REFUSAL_NAME_TAKEN: vaults.ErrNameTaken,
		v1.VaultsRefusal_VAULTS_REFUSAL_LAST_VAULT: vaults.ErrLastVault,
		v1.VaultsRefusal_VAULTS_REFUSAL_SHOWING:    errShowing,
		v1.VaultsRefusal_VAULTS_REFUSAL_UNKNOWN:    vaults.ErrUnknown,
		v1.VaultsRefusal_VAULTS_REFUSAL_NO_TRASH:   port.ErrNoTrash,
		v1.VaultsRefusal_VAULTS_REFUSAL_ASKING:     errAsking,
	}, func(err error) v1.VaultsRefusal {
		refusal, _ := vaultRefusedBy(err)
		return refusal
	})
}

// What an artifact stands at is written in two places: how far a run has got,
// and what a run just set going is. Each schema value names the call that
// writes it.
func TestEveryArtifactStateIsWrittenFromOne(t *testing.T) {
	testsupport.Produced(t, map[v1.State]func() v1.State{
		v1.State_STATE_NONE:    standingAt(reached{stands: untouched}),
		v1.State_STATE_DONE:    standingAt(reached{stands: done}),
		v1.State_STATE_RUNNING: standingAt(reached{stands: under}),
		v1.State_STATE_STOPPED: standingAt(reached{stands: stopped}),
		v1.State_STATE_EMPTY:   standingAt(reached{stands: silent}),
		v1.State_STATE_FAILED:  standingAt(reached{stands: unopened}),
		v1.State_STATE_QUEUED:  func() v1.State { return beginning(port.Queued) },
	}, func(writes func() v1.State) v1.State { return writes() })
}

// Every artifact the schema names stands under a name in the store, and is
// carried by a file of some kind.
func TestEveryArtifactTheSchemaNamesStandsSomewhere(t *testing.T) {
	testsupport.Handled(t, func(of v1.ArtifactKind) bool {
		id, named := standing(of)
		return named && id != ""
	})
	testsupport.Handled(t, func(of v1.ArtifactKind) bool {
		// A url carries what was fetched from the address it holds, and which
		// of the two texts that is follows from what fetched it.
		return slices.Contains(carried(domain.KindBook, ""), of) ||
			slices.Contains(carried(domain.KindRecording, ""), of) ||
			slices.Contains(carried(domain.KindURL, derived.Captions), of) ||
			slices.Contains(carried(domain.KindURL, derived.Article), of)
	})
}

// standingAt is what an artifact over a source standing here is answered with.
func standingAt(got reached) func() v1.State {
	return func() v1.State {
		return stood(v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT, got).GetState()
	}
}
