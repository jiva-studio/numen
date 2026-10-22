package source

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/core/internal/transcript"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// ErrNotCorrectable is a kind whose corrections are not the text itself. A
// recognition's are records keyed to the printed line each one puts right, and the
// coordinates on the page hang off those numbers.
var ErrNotCorrectable = errors.New(
	"only a transcript is written here; a recognition is put right by a proofreading run")

// ErrNoCue is words with no cue in them.
var ErrNoCue = errors.New("no cue was read out of those words")

// Correction is a span of a transcript as it should read. Words is WebVTT, and
// the span is Length milliseconds from Start; a Length of zero or less takes the
// span the cues cover.
type Correction struct {
	Path   string
	Kind   string
	Words  string
	Start  int
	Length int
}

// WriteCorrections writes a span of a transcript as it should read. What was
// transcribed stays as it was and the corrections are kept beside it.
type WriteCorrections struct {
	store   artifactStore
	changed func(path string)
}

// NewWriteCorrections writes corrections through the index that knows each
// file's hash and the store the transcripts are kept in. Changed, when it is
// not nil, is told the path of each file whose transcript was written.
func NewWriteCorrections(
	queries port.SourceQueries, derived port.DerivedStores, changed func(path string),
) *WriteCorrections {
	return &WriteCorrections{store: artifactStore{queries: queries, derived: derived}, changed: changed}
}

// Execute writes the cues in place of every cue beginning inside the span they
// cover, and returns how many cues the transcript now holds.
func (u *WriteCorrections) Execute(ctx context.Context, v domain.Vault, in Correction) (int, error) {
	if in.Kind != KindTranscript {
		return 0, ErrNotCorrectable
	}
	_, cues := transcript.Parse([]byte(in.Words))
	if len(cues) == 0 {
		return 0, ErrNoCue
	}

	set, err := u.store.getArtifacts(ctx, v, in.Path)
	if err != nil {
		return 0, err
	}
	for _, one := range set.Artifacts {
		if one.Kind != KindTranscript {
			continue
		}
		return u.write(ctx, set.Store, in, one.Producer, set.Hash, cues)
	}
	return 0, fmt.Errorf("nothing of the kind %q was made from %s", in.Kind, in.Path)
}

func (u *WriteCorrections) write(
	ctx context.Context, store port.DerivedStore, in Correction, producer, hash string, placed []domain.TranscriptCue,
) (int, error) {
	words, err := getArtifactWords(ctx, store, producer, hash)
	if err != nil {
		return 0, err
	}

	// A run appends to what it is writing down, and what is being appended to is
	// not corrected underneath.
	release, err := store.ClaimArtifact(ctx, domain.NewArtifactName(domain.ArtifactPartial, producer, hash))
	if errors.Is(err, port.ErrClaimed) {
		return 0, ErrTranscriptBusy
	}
	if err != nil {
		return 0, err
	}
	defer release()

	_, status := transcript.Parse([]byte(words))
	whole := spliceCues(status, placed, in.Start, in.Length)
	content := append(transcript.Marshal(whole), transcript.GetHandWrittenMark()...)
	if err := store.WriteArtifact(ctx, domain.NewArtifactName(domain.ArtifactCorrections, producer, hash), content); err != nil {
		return 0, err
	}
	if u.changed != nil {
		u.changed(in.Path)
	}
	return len(whole), nil
}

// spliceCues is the cues as they now read: the ones outside the span as they
// were, and the ones sent in place of every cue beginning inside it.
func spliceCues(current, applied []domain.TranscriptCue, start, length int) []domain.TranscriptCue {
	// The cues sent are what they cover: a call naming no span replaces what
	if length <= 0 {
		start, length = applied[0].From, applied[len(applied)-1].To-applied[0].From
	}
	end := start + length
	out := make([]domain.TranscriptCue, 0, len(current)+len(applied))
	for _, one := range current {
		if one.From < start {
			out = append(out, one)
		}
	}
	out = append(out, applied...)
	for _, one := range current {
		if one.From >= end {
			out = append(out, one)
		}
	}
	return out
}
