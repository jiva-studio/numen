package webui

import (
	"context"
	"errors"
	"io/fs"
	"net/http"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	derived "github.com/jiva-studio/numen/modules/libs/core/text"
)

// A source carrying no text a person typed is put through a run: a scan is
// read, a recording is heard. The window asks for one and is told at once what
// became of the ask; how far a run gets is in the list of what is being done.
//
// A source a person named is never turned away for want of a turn. It waits in
// line and is taken up before anything the vault set itself.

// A Run reads a source a model has to produce the text of. Reading a scan and
// hearing a recording are one shape here, and one at a time of each.
type Run interface {
	// Start takes on one source a person named, and says whether it began now
	// or waits its turn. It runs under the application, so whoever asked is
	// answered at once.
	Start(v domain.Vault, path string) port.Taking
}

// errNoReading is a build with nothing to read a scan with, and errNoListening
// one with nothing to hear a recording with.
var (
	errNoReading   = errors.New("this build cannot read a scan")
	errNoListening = errors.New("this build cannot hear a recording")
)

// errNotPosted is a run asked for by any other method. A run changes the vault.
var errNotPosted = errors.New("a run is asked for with POST")

// What became of an ask, as the window reads it. The window draws the sentence
// and needs none of these; they are here so that it may draw a source waiting
// in line differently from one being worked on.
const (
	outcomeStarted = "started"
	outcomeQueued  = "queued"
	outcomeRunning = "running"
	outcomeDone    = "done"
	outcomeUnfit   = "unfit"
	// answered is a run having got no words out of this source and written
	// down what it got instead. Asking again gets the same.
	outcomeAnswered = "answered"
	// unheard is a proofreading asked for over a recording nothing has
	// listened to, which holds no words to put right.
	outcomeUnheard = "unheard"
	// byHand is a proofreading asked for over words a person wrote themselves,
	// which a model does not correct.
	outcomeByHand = "byHand"
)

// The sentences the window shows, one for every outcome and told apart by what
// they say.
//
// They answer a person who chose Transcribe or Recognise from a menu, and they
// say the word that person chose.
const (
	notAScan      = "Only a scan is recognised, and this file is not one."
	notARecording = "Only a recording is transcribed, and this file is not one."

	readAlready  = "This scan has already been recognised."
	heardAlready = "This recording has already been transcribed."

	readingNow = "This scan is being recognised now."
	hearingNow = "This recording is being transcribed now."

	readingQueued = "This scan is in line, behind the one being recognised now."
	hearingQueued = "This recording is in line, behind the one being transcribed now."

	readingSilent = "Nothing was read in this scan."
	hearingSilent = "No speech was heard in this recording."

	// What the run said about bytes it could not open stands after these, and
	// asking again gets the same until that record is taken away.
	readingUnopened = "This scan could not be opened:"
	hearingUnopened = "This recording could not be opened:"
)

// began is what the window is told of a run it asked for: what became of the
// ask, and the one sentence a person is shown for it.
type began struct {
	Path   string `json:"path"`
	Answer string `json:"answer"`
	Why    string `json:"why"`
}

// telling is the sentences one kind of run answers with, by what became of the
// ask.
type telling struct {
	// unfit is the file not being what this run reads, and done the whole of
	// the text already standing.
	unfit string
	done  string
	// running is a run holding this very source now, and queued the source
	// waiting behind one.
	running string
	queued  string
	// silent is a run having got no words out of this source, and unopened
	// bytes nothing here can open. What the run said about those bytes follows
	// unopened.
	silent   string
	unopened string
}

// Recognise begins reading the scan at a path.
func (a *API) Recognise(w http.ResponseWriter, r *http.Request, path string) {
	reads := a.recognises()
	if reads == nil {
		http.Error(w, errNoReading.Error(), http.StatusNotImplemented)
		return
	}
	a.begin(w, r, path, domain.KindBook, reads, telling{
		unfit:    notAScan,
		done:     readAlready,
		running:  readingNow,
		queued:   readingQueued,
		silent:   readingSilent,
		unopened: readingUnopened,
	})
}

// Transcribe begins listening to the recording at a path.
func (a *API) Transcribe(w http.ResponseWriter, r *http.Request, path string) {
	hears := a.transcribes()
	if hears == nil {
		http.Error(w, errNoListening.Error(), http.StatusNotImplemented)
		return
	}
	a.begin(w, r, path, domain.KindRecording, hears, telling{
		unfit:    notARecording,
		done:     heardAlready,
		running:  hearingNow,
		queued:   hearingQueued,
		silent:   hearingSilent,
		unopened: hearingUnopened,
	})
}

// begin takes on a run over the source at a path and says what became of the
// ask.
//
// Every outcome a person can act on is an answer carrying a sentence. A path
// the vault does not hold is not one of those and is refused as an error.
func (a *API) begin(
	w http.ResponseWriter,
	r *http.Request,
	path string,
	kind domain.SourceKind,
	by Run,
	says telling,
) {
	if r.Method != http.MethodPost {
		http.Error(w, errNotPosted.Error(), http.StatusMethodNotAllowed)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), patience)
	defer cancel()

	showing, ref, err := a.held(ctx, path)
	if err != nil {
		refuse(w, err)
		return
	}
	if ref.Kind != kind {
		answer(w, began{Path: ref.Path, Answer: outcomeUnfit, Why: says.unfit})
		return
	}
	got, err := a.far(ctx, showing, ref.Path, ref.Kind)
	if err != nil {
		refuse(w, err)
		return
	}
	switch {
	case got.done:
		answer(w, began{Path: ref.Path, Answer: outcomeDone, Why: says.done})
		return
	case got.under:
		answer(w, began{Path: ref.Path, Answer: outcomeRunning, Why: says.running})
		return
	case got.gave != "":
		answer(w, began{Path: ref.Path, Answer: outcomeAnswered, Why: says.about(got)})
		return
	}

	// What this machine has fetched is not asked about. A run comes up in its
	// turn and fetches what it needs then.
	if by.Start(showing, ref.Path) == port.Queued {
		answer(w, began{Path: ref.Path, Answer: outcomeQueued, Why: says.queued})
		return
	}
	// The list of what is being done draws the run from the moment it begins,
	// under the work and the file it is over.
	answer(w, began{Path: ref.Path, Answer: outcomeStarted})
}

// about is the sentence a person reads for a source a run got no words out of.
// What the run said about bytes it could not open stands after it.
func (t telling) about(got reached) string {
	if got.gave == derived.Silent {
		return t.silent
	}
	if got.said == "" {
		return t.unopened
	}
	return t.unopened + " " + got.said
}

// reached is how far a run over one source has got: done is the whole of the
// text a model produced already standing, and under is a run holding this very
// source now.
//
// gave is what a run got out of a source it got no words out of, and said is
// what it wrote about it.
type reached struct {
	done  bool
	under bool
	gave  string
	said  string
}

// far says how far a run over the source at a path has got.
//
// A build that cannot say which model produced a text answers nothing, and the
// run itself then decides what is left to do.
func (a *API) far(
	ctx context.Context,
	v domain.Vault,
	path string,
	kind domain.SourceKind,
) (reached, error) {
	said, store, produced, err := a.heard(ctx, v, path)
	if err != nil {
		return reached{}, err
	}
	if produced {
		return farUnder(ctx, store, said.From, said.Hash)
	}
	return a.byBytes(ctx, v, path, unnamed(kind))
}

// farUnder says how far the run keeping its files under a name has got.
//
// What a run reaches is written down as it goes and the whole of it is written
// under its own name at the end, so a source stands on the text once that name
// is there. It holds the name it is still writing under for as long as it
// takes, so a claim on that name that is refused is a run over this source: the
// claim is taken and given straight back, and whether it was refused is the
// answer.
func farUnder(ctx context.Context, store port.DerivedStore, from, hash string) (reached, error) {
	var got reached
	_, err := store.Read(ctx, derived.Artifact(from, hash))
	switch {
	case err == nil:
		got.done = true
		return got, nil
	case !errors.Is(err, fs.ErrNotExist):
		return got, err
	}
	// A run that got no words out of a source wrote down what it got instead,
	// and asking again gets the same. Taking that record away is how a person
	// asks for the source to be tried afresh.
	switch held, err := store.Read(ctx, derived.Answer(from, hash)); {
	case err == nil:
		got.gave, got.said = derived.Answered(held)
		return got, nil
	case !errors.Is(err, fs.ErrNotExist):
		return got, err
	}

	got.under = !free(ctx, store, derived.Partial(from, hash))
	return got, nil
}

// byBytes says how far a run over a source the index names no producer for has
// got. A source a run got no words out of is one of those, and what the run
// wrote is kept under the fingerprint of the bytes.
//
// The file is read and fingerprinted here, which is what a run does before
// anything else.
func (a *API) byBytes(ctx context.Context, v domain.Vault, path, from string) (reached, error) {
	_, stores, ok := a.hearing()
	if !ok || from == "" {
		return reached{}, nil
	}
	reader, err := a.Readers.Open(v)
	if err != nil {
		return reached{}, err
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return reached{}, err
	}
	store, err := stores.Open(v)
	if err != nil {
		return reached{}, err
	}
	return farUnder(ctx, store, from, derived.Fingerprint(raw))
}

// unnamed is the producer whose files stand for a source the index names none
// for: a recording is listened to. Nothing produces a scan's text without the
// index saying what did.
func unnamed(kind domain.SourceKind) string {
	if kind == domain.KindRecording {
		return derived.ASR
	}
	return ""
}
