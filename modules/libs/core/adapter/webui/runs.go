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
)

// The sentences the window shows, one for every outcome and told apart by what
// they say.
const (
	notAScan      = "Only a scan is read, and this file is not one."
	notARecording = "Only a recording is heard, and this file is not one."

	readAlready  = "This scan has already been read."
	heardAlready = "This recording has already been heard."

	readingNow = "This scan is being read now."
	hearingNow = "This recording is being heard now."

	readingQueued = "This scan is in line, behind the scan being read now."
	hearingQueued = "This recording is in line, behind the recording being heard now."

	readingBegun = "Reading this scan has begun."
	hearingBegun = "Listening to this recording has begun."
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
	// started is a run over this source, begun now.
	started string
}

// Recognise begins reading the scan at a path.
func (a *API) Recognise(w http.ResponseWriter, r *http.Request, path string) {
	if a.Recognises == nil {
		http.Error(w, errNoReading.Error(), http.StatusNotImplemented)
		return
	}
	a.begin(w, r, path, domain.KindBook, a.Recognises, telling{
		unfit:   notAScan,
		done:    readAlready,
		running: readingNow,
		queued:  readingQueued,
		started: readingBegun,
	})
}

// Transcribe begins listening to the recording at a path.
func (a *API) Transcribe(w http.ResponseWriter, r *http.Request, path string) {
	if a.Transcribes == nil {
		http.Error(w, errNoListening.Error(), http.StatusNotImplemented)
		return
	}
	a.begin(w, r, path, domain.KindRecording, a.Transcribes, telling{
		unfit:   notARecording,
		done:    heardAlready,
		running: hearingNow,
		queued:  hearingQueued,
		started: hearingBegun,
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
	got, err := a.far(ctx, showing, ref.Path)
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
	}

	// What this machine has fetched is not asked about. A run comes up in its
	// turn and fetches what it needs then.
	if by.Start(showing, ref.Path) == port.Queued {
		answer(w, began{Path: ref.Path, Answer: outcomeQueued, Why: says.queued})
		return
	}
	answer(w, began{Path: ref.Path, Answer: outcomeStarted, Why: says.started})
}

// reached is how far a run over one source has got: done is the whole of the
// text a model produced already standing, and under is a run holding this very
// source now.
type reached struct {
	done  bool
	under bool
}

// far says how far a run over the source at a path has got.
//
// What a run reaches is written down as it goes and the whole of it is written
// under its own name at the end, so a source stands on the text once that name
// is there. It holds the name it is still writing under for as long as it
// takes, so a claim on that name that is refused is a run over this source: the
// claim is taken and given straight back, and whether it was refused is the
// answer.
//
// A build that cannot say which model produced a text answers neither, and the
// run itself then decides what is left to do.
func (a *API) far(ctx context.Context, v domain.Vault, path string) (reached, error) {
	var got reached
	said, store, produced, err := a.heard(ctx, v, path)
	if err != nil || !produced {
		return got, err
	}
	_, err = store.Read(ctx, derived.Artifact(said.From, said.Hash))
	switch {
	case err == nil:
		got.done = true
		return got, nil
	case !errors.Is(err, fs.ErrNotExist):
		return got, err
	}
	got.under = !free(ctx, store, derived.Partial(said.From, said.Hash))
	return got, nil
}
