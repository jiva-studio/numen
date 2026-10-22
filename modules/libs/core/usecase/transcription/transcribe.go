package transcription

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/internal/transcript"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// Transcribe listens to a recording with a model and writes down what it heard.
//
// A recording carries no text of its own, so what the model heard is the only
// text there is. It is written into the vault as an artifact and the source is
// cut from it.
//
// A queue calls this without asking anybody, and every recording it is handed
// ends in an answer. Words are one answer, a recording holding no speech is
// another, and a file nothing here can open is a third. All three are written
// down, and a recording that has answered is not listened to again.
type Transcribe struct {
	readers port.VaultReaders
	sources port.SourceRepository
	derived port.DerivedStores
	by      port.Transcriber

	// Area is the store the artifact is kept in. Empty means the default.
	Area string

	// Batch is how many stretches of speech are heard before what was heard is
	// written down. A recording is an hour's listening, and a run that is
	// stopped keeps what it had. Zero takes the default.
	Batch int

	// IsRepeat throws away what a run before this one heard and listens from the
	// start. It is how a person asks for a recording to be heard by whatever
	// model is configured now, and nothing sets it on its own.
	IsRepeat bool

	// Cut is optional. It makes a source's chunks, and is called as speech is
	// written down, so what has been heard is searchable before the rest of it
	// is. Where nothing cuts, the source is recorded as owing its text and the
	// next scan cuts it.
	Cut func(ctx context.Context, v domain.Vault, path string) error

	OnProgress func(TranscribeResult)
}

// NewTranscribe is what a recording is listened to through: the vault it is
// read out of, where what the vault holds is recorded, the store the transcript
func NewTranscribe(
	readers port.VaultReaders,
	sources port.SourceRepository,
	derived port.DerivedStores,
	by port.Transcriber,
) Transcribe {
	return Transcribe{readers: readers, sources: sources, derived: derived, by: by}
}

// By returns the transcriber.
func (u Transcribe) By() port.Transcriber { return u.by }

// TranscribeResult reports what transcribing did.
type TranscribeResult struct {
	Path       string // the recording being transcribed
	Length     int    // how long it is, in milliseconds
	Heard      int    // how much of it has been written down, this run and before it
	Resumed    int    // how much a run before this one had already written down
	IsSilent   bool   // it carries no speech, and that is what was written
	IsUnopened bool   // nothing here can open it, and that is what was written
	IsBusy     bool   // somebody else is transcribing these bytes, and nothing was done
}

// DefaultHeard is how many stretches of speech are heard before they are
// written down.
const DefaultHeard = 16

// Execute listens to one recording.
func (u Transcribe) Execute(ctx context.Context, v domain.Vault, path string) (TranscribeResult, error) {
	res := TranscribeResult{Path: path}
	reader, err := u.readers.Open(v)
	if err != nil {
		return res, err
	}
	ref, err := reader.GetFingerprint(ctx, path)
	if err != nil {
		return res, fmt.Errorf("stat %s: %w", path, err)
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return res, fmt.Errorf("read %s: %w", path, err)
	}
	store, err := u.derived.Open(v)
	if err != nil {
		return res, err
	}

	hash := text.Fingerprint(raw)
	area := run.GetArea(u.Area, domain.ASR)
	final := domain.NewArtifactName(domain.ArtifactText, area, hash)
	partial := domain.NewArtifactName(domain.ArtifactPartial, area, hash)

	// One run to a recording. The name is the hash of its bytes, and it is held
	// for as long as the listening takes.
	release, err := store.ClaimArtifact(ctx, partial)
	if errors.Is(err, port.ErrClaimed) {
		res.IsBusy = true
		return res, nil
	}
	if err != nil {
		return res, err
	}
	defer release()

	if u.IsRepeat {
		// Somebody asked for this recording to be heard afresh. What a run
		// before this one made goes, and the listening starts from the top.
		answer := domain.NewArtifactName(domain.ArtifactAnswer, area, hash)
		if err := removeAllNames(ctx, store, final, partial, answer); err != nil {
			return res, err
		}
	}

	if isDone, err := u.recordSource(ctx, v, store, ref, hash, area, &res); isDone {
		return res, err
	}

	recording, err := u.by.Open(ctx, raw)
	if err != nil {
		res.IsUnopened = true
		return res, u.answer(ctx, v, ref, hash, area, store, text.Unopened+": "+describeFailure(err.Error()))
	}
	defer recording.Close()
	res.Length = recording.GetDuration()

	if err := u.transcribeSegments(ctx, v, store, recording, path, partial, &res); err != nil {
		return res, err
	}

	err = u.saveTranscript(ctx, v, store, ref, hash, area, &res)
	return res, err
}

// saveTranscript stands the source on the artifact made from the partial, or answers
// that the recording carries no speech.
func (u Transcribe) saveTranscript(
	ctx context.Context,
	v domain.Vault,
	store port.DerivedStore,
	ref domain.Fingerprint,
	hash, area string,
	res *TranscribeResult,
) error {
	partial := domain.NewArtifactName(domain.ArtifactPartial, area, hash)
	whole, err := store.ReadArtifact(ctx, partial)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if words, _ := transcript.Parse(whole); strings.TrimSpace(words) == "" {
		// A recording carrying no speech says so, and nothing stands as its
		// text. An artifact with no cues in it is a transcription that worked.
		res.IsSilent = true
		return u.answer(ctx, v, ref, hash, area, store, text.Silent)
	}

	if err := store.WriteArtifact(ctx, domain.NewArtifactName(domain.ArtifactText, area, hash), whole); err != nil {
		return err
	}
	if err := u.record(ctx, store, area, hash); err != nil {
		return err
	}
	// The source relies on the artifact before the partial goes.
	if err := run.Stand(ctx, u.Cut, u.sources, v, ref, hash, area); err != nil {
		return err
	}
	return store.RemoveArtifact(ctx, partial)
}

// removeAllNames removes each name from the store and passes over names that
// are already absent.
func removeAllNames(ctx context.Context, store port.DerivedStore, names ...domain.ArtifactName) error {
	for _, name := range names {
		if err := store.RemoveArtifact(ctx, name); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	return nil
}

// recordSource stands the source on a transcript or answer an earlier run
// stored for these bytes, and says whether one was there.
//
// The name is the hash of what was heard, so a recording is not listened to
// again however the file was renamed or moved since. A recording that gave no
// words gave an answer all the same, and taking the record away is how a person
// asks for it again.
func (u Transcribe) recordSource(
	ctx context.Context,
	v domain.Vault,
	store port.DerivedStore,
	ref domain.Fingerprint,
	hash, area string,
	res *TranscribeResult,
) (bool, error) {
	if target, err := store.ReadArtifact(ctx, domain.NewArtifactName(domain.ArtifactText, area, hash)); err == nil {
		if _, cues := transcript.Parse(target); len(cues) > 0 {
			res.Heard = cues[len(cues)-1].To
			res.Length = res.Heard
		}
		return true, run.Stand(ctx, u.Cut, u.sources, v, ref, hash, area)
	}
	if targetVal, err := store.ReadArtifact(ctx, domain.NewArtifactName(domain.ArtifactAnswer, area, hash)); err == nil {
		gave, _ := text.ReadAnswer(targetVal)
		res.IsSilent = gave == text.Silent
		res.IsUnopened = gave == text.Unopened
		return true, run.Stand(ctx, u.Cut, u.sources, v, ref, hash, "")
	}
	return false, nil
}

// transcribeSegments transcribes the recording from where the partial left off until it
// stops moving, writing each batch of cues as it is heard.
//
// The note stands after the cues it claims, so a batch that did not land whole
// is one no note claims, and the next run transcribes those stretches again.
func (u Transcribe) transcribeSegments(
	ctx context.Context,
	v domain.Vault,
	store port.DerivedStore,
	recording port.Recording,
	path string,
	partial domain.ArtifactName,
	res *TranscribeResult,
) error {
	from, size, err := leftOff(ctx, store, partial)
	if err != nil {
		return err
	}
	res.Resumed, res.Heard = from, from
	run.Report(u.OnProgress, *res)

	hasContent := size > 0
	for {
		if err := ctx.Err(); err != nil {
			// What has been heard is on disk already, and the next run begins
			// where this one stopped.
			return err
		}
		cues, reached, err := u.transcribeBatch(ctx, recording, path, from)
		if err != nil {
			return err
		}
		// A batch reaching no further than the last one is a recording that has
		// stopped moving, and there is nothing after it to ask for.
		if reached <= from {
			return nil
		}

		from = reached
		body := trimHeader(transcript.Marshal(cues), hasContent)
		if err := store.AppendArtifact(ctx, partial, append(body, transcript.GetReachMarker(from)...)); err != nil {
			return err
		}
		hasContent = true
		if err := run.Cut(ctx, u.Cut, v, path); err != nil {
			return err
		}
		res.Heard = from
		run.Report(u.OnProgress, *res)
	}
}

// transcribeBatch transcribes the next segments after from, and returns their cues
// with the furthest point they reach. A recording with nothing after from
// returns no cues and from itself.
func (u Transcribe) transcribeBatch(
	ctx context.Context,
	recording port.Recording,
	path string,
	from int,
) ([]domain.TranscriptCue, int, error) {
	segments, err := recording.GetSegments(ctx, from, run.GetSize(u.Batch, DefaultHeard))
	if err != nil {
		return nil, from, fmt.Errorf("transcribe %s at %s: %w", path, transcript.FormatTimestamp(from), err)
	}
	reached := from
	cues := make([]domain.TranscriptCue, 0, len(segments))
	for _, audio := range segments {
		words, err := u.by.Transcribe(ctx, audio)
		if err != nil {
			return nil, from, fmt.Errorf("hear %s at %s: %w", path, transcript.FormatTimestamp(audio.From), err)
		}
		cues = append(cues, domain.TranscriptCue{Text: words, From: audio.From, To: audio.To})
		reached = max(reached, audio.To)
	}
	return cues, reached, nil
}

// answer records what a recording with no words to give gave, takes away what
// listening to it produced, and leaves the source standing on nothing.
func (u Transcribe) answer(
	ctx context.Context,
	v domain.Vault,
	ref domain.Fingerprint,
	hash, area string,
	store port.DerivedStore,
	transcript string,
) error {
	if err := store.WriteArtifact(ctx, domain.NewArtifactName(domain.ArtifactAnswer, area, hash), []byte(transcript+"\n")); err != nil {
		return err
	}
	if err := u.record(ctx, store, area, hash); err != nil {
		return err
	}
	if err := store.RemoveArtifact(ctx, domain.NewArtifactName(domain.ArtifactPartial, area, hash)); err != nil {
		return err
	}
	return run.Stand(ctx, u.Cut, u.sources, v, ref, hash, "")
}

// any path that answers a question reads it: it is there so that a person can
// ask what produced a text, and so that everything a transcriber now known to
// be bad produced can be found again.
func (u Transcribe) record(ctx context.Context, store port.DerivedStore, area, hash string) error {
	named := u.by.GetTranscriptionModel()
	raw, err := json.MarshalIndent(struct {
		Model     string `json:"model"`
		Segmenter string `json:"segmenter"`
		Cutting   string `json:"cutting"`
		From      string `json:"from"`
		Heard     string `json:"heard"`
		Recipe    string `json:"recipe"`
	}{named.Model, named.Segmenter, named.Cutting, named.From, named.String(), named.Recipe()}, "", "  ")
	if err != nil {
		return err
	}
	return store.WriteArtifact(ctx, domain.NewArtifactName(domain.ArtifactProducer, area, hash), append(raw, '\n'))
}

// describeFailure is a failure as one line, which is what a file holding one
// line takes.
func describeFailure(why string) string {
	return strings.Join(strings.Fields(why), " ")
}

// trimHeader is a run of cues as they are added to a file that has been written
// to already. The header appears once, at the top.
func trimHeader(raw []byte, opened bool) []byte {
	if !opened {
		return raw
	}
	return bytes.TrimPrefix(raw, []byte(transcript.Head+"\n"))
}

// leftOff is how far a run before this one got and how long what it left is,
// with anything past the last note cut away.
//
// A note stands after the cues it claims, so what follows the last one is a
// batch that did not land whole.
func leftOff(
	ctx context.Context, store port.DerivedStore, partial domain.ArtifactName,
) (transcribed, size int, err error) {
	raw, err := store.ReadArtifact(ctx, partial)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, err
	}
	ms, end := transcript.ReadReached(raw)
	if end < len(raw) {
		// Written back to the note. Bytes past it are cues no note claims.
		if err := store.WriteArtifact(ctx, partial, raw[:end]); err != nil {
			return 0, 0, err
		}
	}
	return ms, end, nil
}
