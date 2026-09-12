package source

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/internal/transcript"
	"github.com/jiva-studio/numen/modules/libs/core/port"
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
	Readers port.VaultReaders
	Sources port.SourceRepository
	Derived port.DerivedStores
	By      port.Transcriber

	// Area is the store the artifact is kept in. Empty means the default.
	Area string

	// Batch is how many stretches of speech are heard before what was heard is
	// written down. A recording is an hour's listening, and a run that is
	// stopped keeps what it had. Zero takes the default.
	Batch int

	// Again throws away what a run before this one heard and listens from the
	// start. It is how a person asks for a recording to be heard by whatever
	// model is configured now, and nothing sets it on its own.
	Again bool

	// Cut is optional. It makes a source's chunks, and is called as speech is
	// written down, so what has been heard is searchable before the rest of it
	// is. Where nothing cuts, the source is recorded as owing its text and the
	// next scan cuts it.
	Cut func(ctx context.Context, v domain.Vault, path string) error

	OnProgress func(TranscribeResult)
}

// NewTranscribe is what a recording is listened to through: the vault it is
// read out of, where what the vault holds is recorded, the store the transcript
// is written into, and the model that hears it.
func NewTranscribe(
	readers port.VaultReaders,
	sources port.SourceRepository,
	derived port.DerivedStores,
	by port.Transcriber,
) Transcribe {
	return Transcribe{Readers: readers, Sources: sources, Derived: derived, By: by}
}

// TranscribeResult reports what transcribing did.
type TranscribeResult struct {
	Path     string // the recording being transcribed
	Length   int    // how long it is, in milliseconds
	Heard    int    // how much of it has been written down, this run and before it
	Resumed  int    // how much a run before this one had already written down
	Silent   bool   // it carries no speech, and that is what was written
	Unopened bool   // nothing here can open it, and that is what was written
	Busy     bool   // somebody else is transcribing these bytes, and nothing was done
}

// DefaultHeard is how many stretches of speech are heard before they are
// written down.
const DefaultHeard = 16

// Execute listens to one recording.
func (u Transcribe) Execute(ctx context.Context, v domain.Vault, path string) (TranscribeResult, error) {
	res := TranscribeResult{Path: path}
	reader, err := u.Readers.Open(v)
	if err != nil {
		return res, err
	}
	ref, err := reader.Stat(ctx, path)
	if err != nil {
		return res, fmt.Errorf("stat %s: %w", path, err)
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return res, fmt.Errorf("read %s: %w", path, err)
	}
	store, err := u.Derived.Open(v)
	if err != nil {
		return res, err
	}

	hash := text.Fingerprint(raw)
	area := u.area()
	final, partial := text.Artifact(area, hash), text.Partial(area, hash)

	// One run to a recording. The name is the hash of its bytes, and it is held
	// for as long as the listening takes.
	release, err := store.Claim(ctx, partial)
	if errors.Is(err, port.ErrClaimed) {
		res.Busy = true
		return res, nil
	}
	if err != nil {
		return res, err
	}
	defer release()

	if u.Again {
		// Somebody asked for this recording to be heard afresh. What a run
		// before this one made goes, and the listening starts from the top.
		for _, name := range []string{final, partial, text.Answer(area, hash)} {
			if err := store.Remove(ctx, name); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return res, err
			}
		}
	}

	// A recording already listened to is not listened to again. The name is the
	// hash of what was heard, so this holds however the file was renamed or
	// moved since.
	if held, err := store.Read(ctx, final); err == nil {
		if _, cues := transcript.Parse(held); len(cues) > 0 {
			res.Heard = cues[len(cues)-1].To
			res.Length = res.Heard
		}
		return res, u.stand(ctx, v, ref, hash, area)
	}
	// A recording that gave no words gave an answer all the same, and it is
	// recorded. Taking the record away is how a person asks for it again.
	if held, err := store.Read(ctx, text.Answer(area, hash)); err == nil {
		gave, _ := text.ReadAnswer(held)
		res.Silent = gave == text.Silent
		res.Unopened = gave == text.Unopened
		return res, u.stand(ctx, v, ref, hash, "")
	}

	recording, err := u.By.Open(ctx, raw)
	if err != nil {
		res.Unopened = true
		return res, u.answer(ctx, v, ref, hash, area, store, text.Unopened+": "+describeFailure(err.Error()))
	}
	defer recording.Close()
	res.Length = recording.Length()

	from, size, err := leftOff(ctx, store, partial)
	if err != nil {
		return res, err
	}
	res.Resumed, res.Heard = from, from
	u.progress(res)

	// write puts a run of cues down and cuts the source from everything the
	// recording has said so far.
	//
	// The note stands after the cues it claims, so a batch that did not land
	// whole is one no note claims, and the next run transcribes those stretches
	// again.
	opened := size > 0
	write := func(cues []transcript.Cue, transcribed int) error {
		body := trimHeader(transcript.Marshal(cues), opened)
		if err := store.Append(ctx, partial, append(body, transcript.Reaches(transcribed)...)); err != nil {
			return err
		}
		opened = true
		return u.cut(ctx, v, path)
	}

	for {
		if err := ctx.Err(); err != nil {
			// What has been heard is on disk already. Stopping is a thing a
			// person did, and the next run begins where this one stopped.
			return res, err
		}
		segments, err := recording.Segments(ctx, from, u.batch())
		if err != nil {
			return res, fmt.Errorf("transcribe %s at %s: %w", path, transcript.Stamp(from), err)
		}
		if len(segments) == 0 {
			break
		}

		reached := from
		cues := make([]transcript.Cue, 0, len(segments))
		for _, audio := range segments {
			words, err := u.By.Transcribe(ctx, audio)
			if err != nil {
				return res, fmt.Errorf("hear %s at %s: %w", path, transcript.Stamp(audio.From), err)
			}
			cues = append(cues, transcript.Cue{Text: words, From: audio.From, To: audio.To})
			if audio.To > reached {
				reached = audio.To
			}
		}
		// A batch reaching no further than the last one is a recording that has
		// stopped moving, and there is nothing after it to ask for.
		if reached <= from {
			break
		}

		from = reached
		if err := write(cues, from); err != nil {
			return res, err
		}
		res.Heard = from
		u.progress(res)
	}

	whole, err := store.Read(ctx, partial)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return res, err
	}
	if words, _ := transcript.Parse(whole); strings.TrimSpace(words) == "" {
		// A recording carrying no speech says so, and nothing stands as its
		// text. An artifact with no cues in it is a transcription that worked.
		res.Silent = true
		return res, u.answer(ctx, v, ref, hash, area, store, text.Silent)
	}

	if err := store.Write(ctx, final, whole); err != nil {
		return res, err
	}
	if err := u.record(ctx, store, area, hash); err != nil {
		return res, err
	}
	// The source stands on the artifact before the partial goes.
	if err := u.stand(ctx, v, ref, hash, area); err != nil {
		return res, err
	}
	return res, store.Remove(ctx, partial)
}

// answer records what a recording with no words to give gave, takes away what
// listening to it produced, and leaves the source standing on nothing.
func (u Transcribe) answer(
	ctx context.Context,
	v domain.Vault,
	ref domain.Fingerprint,
	hash, area string,
	store port.DerivedStore,
	gave string,
) error {
	if err := store.Write(ctx, text.Answer(area, hash), []byte(gave+"\n")); err != nil {
		return err
	}
	if err := u.record(ctx, store, area, hash); err != nil {
		return err
	}
	if err := store.Remove(ctx, text.Partial(area, hash)); err != nil {
		return err
	}
	return u.stand(ctx, v, ref, hash, "")
}

// stand puts the source on the text a producer made.
//
// Cutting writes which text a source stands on together with the chunks cut
// from it, in one statement, and the two are one fact. Where nothing cuts here
// the source is recorded as owing its text, and the scan that cuts it writes
// both.
func (u Transcribe) stand(ctx context.Context, v domain.Vault, ref domain.Fingerprint, hash, from string) error {
	if u.Cut != nil {
		return u.Cut(ctx, v, ref.Path)
	}
	return u.Sources.SaveSource(ctx, v.ID, domain.Source{Fingerprint: ref, Hash: hash, Producer: from})
}

// cut makes this source's chunks from what has been heard so far.
func (u Transcribe) cut(ctx context.Context, v domain.Vault, path string) error {
	if u.Cut == nil {
		return nil
	}
	return u.Cut(ctx, v, path)
}

// record keeps what listened to the recording beside what it heard. Nothing on
// any path that answers a question reads it: it is there so that a person can
// ask what produced a text, and so that everything a transcriber now known to
// be bad produced can be found again.
func (u Transcribe) record(ctx context.Context, store port.DerivedStore, area, hash string) error {
	named := u.By.Transcription()
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
	return store.Write(ctx, text.Beside(area, hash), append(raw, '\n'))
}

// describeFailure is a failure as one line, which is what a file holding one
// line takes.
func describeFailure(why string) string {
	return strings.Join(strings.Fields(why), " ")
}

// trimHeader is a run of cues as they are added to a file that has been written
// to already. The header stands once, at the top.
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
	ctx context.Context, store port.DerivedStore, partial string,
) (transcribed, size int, err error) {
	raw, err := store.Read(ctx, partial)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, err
	}
	ms, end := transcript.ReadReached(raw)
	if end < len(raw) {
		// Written back to the note. Bytes past it are cues no note claims.
		if err := store.Write(ctx, partial, raw[:end]); err != nil {
			return 0, 0, err
		}
	}
	return ms, end, nil
}

func (u Transcribe) area() string {
	if u.Area == "" {
		return text.ASR
	}
	return u.Area
}

func (u Transcribe) batch() int {
	if u.Batch <= 0 {
		return DefaultHeard
	}
	return u.Batch
}

func (u Transcribe) progress(res TranscribeResult) {
	if u.OnProgress != nil {
		u.OnProgress(res)
	}
}
