package note

import (
	"testing"
	"time"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/naming"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func TestEveryRelationIsSavedFromOne(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.Relation]domain.Relation{
		v1.Relation_RELATION_PARENT:    domain.RelationParent,
		v1.Relation_RELATION_CHILD:     domain.RelationChild,
		v1.Relation_RELATION_JUMP:      domain.RelationJump,
		v1.Relation_RELATION_SIBLING:   domain.RelationSibling,
		v1.Relation_RELATION_REFERENCE: domain.RelationReference,
	}, convertRelationToWire)
}

func TestConvertNoteTypeToWire_MapsPlainNoteToEmpty(t *testing.T) {
	for typed, want := range map[domain.NoteType]string{
		domain.TypeNote: "",
		"sample":        "sample",
		"sheet":         "sheet",
	} {
		if got := convertNoteTypeToWire(typed); got != want {
			t.Errorf("%q is written as %q, want %q", typed, got, want)
		}
	}
}

func TestEveryNamingIsSavedFromOne(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.NameSource]naming.Source{
		v1.NameSource_NAME_SOURCE_FRONTMATTER: naming.ByFrontmatter,
		v1.NameSource_NAME_SOURCE_FILENAME:    naming.ByFilename,
	}, newWireNameSource)
}

func TestTheSchemaCarriesNanosecondsSinceTheEpoch(t *testing.T) {
	const nanos = 1788433445123456789
	ref := domain.FileStat{
		Path: "library/Compost.pdf", Size: 12, ModTime: time.Date(2026, 9, 3, 11, 4, 5, 123456789, time.UTC),
	}

	if target := convertFileStatToWire(ref).GetMtime(); target != nanos {
		t.Errorf("the schema carries mtime %d, want %d", target, nanos)
	}
}

func TestAFileStampedByAClockIsStillItselfThroughTheSchema(t *testing.T) {
	ref := domain.FileStat{Path: "recordings/Compost.mp3", Size: 3, ModTime: testsupport.GetMachine().Now()}
	if back := convertFileStatFromWire(convertFileStatToWire(ref)); !back.IsUnchanged(ref) {
		t.Errorf("the message carried the file back as %+v, and it is %+v", back, ref)
	}
}

func TestAnEmptyFileStatIsNothingOnTheWireAndNotTheEpoch(t *testing.T) {
	if back := convertFileStatFromWire(convertFileStatToWire(domain.FileStat{})); !back.IsZero() {
		t.Errorf("nothing said about a file came back as %+v", back)
	}
	if target := stamp(time.Time{}); target != 0 {
		t.Errorf("no stamp goes on the wire as %d", target)
	}
}

func TestEveryLinkDirectionIsSavedFromOne(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.LinkDirection]domain.LinkDirection{
		v1.LinkDirection_LINK_DIRECTION_OUTGOING:    domain.DirectionOutgoing,
		v1.LinkDirection_LINK_DIRECTION_INCOMING:    domain.DirectionIncoming,
		v1.LinkDirection_LINK_DIRECTION_BOTH:        domain.DirectionBoth,
		v1.LinkDirection_LINK_DIRECTION_UNSPECIFIED: "",
	}, convertDirectionToWire)
}
