package webui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/task"
	derived "github.com/jiva-studio/numen/modules/libs/core/text"
	"github.com/jiva-studio/numen/modules/libs/core/transcript"
)

// A recording crosses to the window twice: as the bytes a player is pointed at,
// and as the words a model heard in them.

// errNotARecording is what a facet of a recording answers for a file that is
// not one.
var errNotARecording = errors.New("not a recording this vault holds")

// errNoHearing is what a build with nothing to read a transcript with answers.
var errNoHearing = errors.New("this build cannot read what a recording says")

// errNotHeard is what an edit to a recording nothing has listened to gets.
var errNotHeard = errors.New("nothing has listened to this recording")

// errBeingHeard is what an edit to a recording a run holds gets. A run appends
// to the transcript, and what is being appended to is not edited underneath.
var errBeingHeard = errors.New("this recording is being listened to")

// listened is what the window is told a recording is: how far the words reach,
// and how much of it a run has written down, both in milliseconds.
//
// A recording nothing has listened to reaches nowhere, and the player it is
// loaded into is what then says how long it runs.
type listened struct {
	Path   string `json:"path"`
	Length int    `json:"length"`
	Heard  int    `json:"heard"`
	// Media is where the recording is played from, and Type is what it is
	// played as. Both are answered here: the socket is opened afresh for every
	// run, and what counts as a recording is this application's to say.
	Media string `json:"media"`
	Type  string `json:"type"`
}

// spoken is the transcript of a recording, in the order it was said.
//
// Editable says whether the words may be put right now. A run listening to the
// recording holds it, and the window draws what it reads and leaves it alone.
type spoken struct {
	Path     string `json:"path"`
	Cues     []cue  `json:"cues"`
	Editable bool   `json:"editable"`
}

// putRight is a transcript as a person left it in the window: the recording it
// belongs to, and the words against the milliseconds they were said in. The
// window does the arithmetic for lines it merged and split.
type putRight struct {
	Path string `json:"path"`
	Cues []cue  `json:"cues"`
}

// cue is one stretch of speech: what was said, and the milliseconds it spans.
type cue struct {
	Text string `json:"text"`
	From int    `json:"from"`
	To   int    `json:"to"`
}

// About answers what the file at a path is. A recording is how long it runs,
// and every other file is a document and is answered with its pages.
func (a *API) About(w http.ResponseWriter, r *http.Request, path string) {
	ctx, cancel := context.WithTimeout(r.Context(), patience)
	defer cancel()

	showing, ref, err := a.held(ctx, path)
	if err != nil || ref.Kind != domain.KindRecording {
		a.Document(w, r, path)
		return
	}
	raw, err := a.transcript(ctx, showing, ref.Path)
	if err != nil {
		refuse(w, err)
		return
	}
	heard, _ := transcript.Reached(raw)
	_, cues := transcript.Parse(raw)
	told := listened{
		Path:   ref.Path,
		Length: heard,
		Heard:  heard,
		Media:  a.Playing.Address(showing, ref.Path),
		Type:   domain.MediaType(ref.Path),
	}
	if len(cues) > 0 {
		told.Length = max(told.Length, cues[len(cues)-1].To)
	}
	answer(w, told)
}

// Cues answers with the transcript of a recording, each stretch of speech
// against the milliseconds it was spoken in. A recording nothing has listened to
// holds no words, which is an answer.
//
// A PUT puts the transcript right, and a DELETE takes it away.
func (a *API) Cues(w http.ResponseWriter, r *http.Request, path string) {
	switch r.Method {
	case http.MethodPut:
		a.PutRight(w, r, path)
		return
	case http.MethodDelete:
		a.Drop(w, r, path)
		return
	}
	if _, _, ok := a.hearing(); !ok {
		http.Error(w, errNoHearing.Error(), http.StatusNotImplemented)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), patience)
	defer cancel()

	showing, ref, err := a.held(ctx, path)
	if err != nil {
		refuse(w, err)
		return
	}
	if ref.Kind != domain.KindRecording {
		http.Error(w, errNotARecording.Error(), http.StatusNotFound)
		return
	}
	said, store, listened, err := a.heard(ctx, showing, ref.Path)
	if err != nil {
		refuse(w, err)
		return
	}
	told := spoken{Path: ref.Path}
	var raw []byte
	if listened {
		if raw, err = a.transcribed(ctx, store, said); err != nil {
			refuse(w, err)
			return
		}
		// Taken after the words, so a run that began while they were being read
		// is one the window is told about.
		told.Editable = free(ctx, store, derived.Partial(said.From, said.Hash))
	}
	_, cues := transcript.Parse(raw)
	cues, err = narrowed(r.URL.Query(), cues)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	told.Cues = make([]cue, 0, len(cues))
	for _, one := range cues {
		told.Cues = append(told.Cues, cue{Text: one.Text, From: one.From, To: one.To})
	}
	answer(w, told)
}

// PutRight writes the transcript of a recording as a person left it in the
// window.
//
// What the model heard stays under its own name and the words as they now stand
// go beside it, so a transcript edited into nonsense is a file that can be
// deleted and what was heard comes back.
func (a *API) PutRight(w http.ResponseWriter, r *http.Request, path string) {
	if _, _, ok := a.hearing(); !ok {
		http.Error(w, errNoHearing.Error(), http.StatusNotImplemented)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), patience)
	defer cancel()

	var put putRight
	if err := json.NewDecoder(r.Body).Decode(&put); err != nil {
		http.Error(w, "this is not a transcript: "+err.Error(), http.StatusBadRequest)
		return
	}
	if put.Path != path {
		http.Error(w, "the transcript names "+put.Path+", which is not the recording it was sent to", http.StatusBadRequest)
		return
	}
	cues, err := ordered(put.Cues)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	showing, ref, err := a.held(ctx, put.Path)
	if err != nil {
		refuse(w, err)
		return
	}
	if ref.Kind != domain.KindRecording {
		http.Error(w, errNotARecording.Error(), http.StatusNotFound)
		return
	}
	said, store, listened, err := a.heard(ctx, showing, ref.Path)
	if err != nil {
		refuse(w, err)
		return
	}
	if !listened {
		http.Error(w, errNotHeard.Error(), http.StatusNotFound)
		return
	}

	// A run holds the recording it is listening to for as long as it takes, by
	// the name it appends to.
	release, err := store.Claim(ctx, derived.Partial(said.From, said.Hash))
	if errors.Is(err, port.ErrClaimed) {
		http.Error(w, errBeingHeard.Error(), http.StatusConflict)
		return
	}
	if err != nil {
		refuse(w, err)
		return
	}
	defer release()

	// The words are a person's, and a proofreader leaves them alone.
	written := append(transcript.Marshal(cues), transcript.Hand()...)
	if err := store.Write(ctx, derived.Corrected(said.From, said.Hash), written); err != nil {
		refuse(w, err)
		return
	}

	// The chunks in the index hold the words as they were heard, so the source
	// is cut again from what it now says. A write that landed is not refused
	// for a cut that could not be asked for.
	if cut := a.cuts(); cut != nil {
		if err := cut(ctx, showing, ref.Path); err != nil {
			a.say(task.Task{ID: readingBooks, Doing: "Reading books", About: ref.Path, Failed: err.Error()})
		}
	}

	told := spoken{Path: ref.Path, Cues: make([]cue, 0, len(cues)), Editable: true}
	for _, one := range cues {
		told.Cues = append(told.Cues, cue{Text: one.Text, From: one.From, To: one.To})
	}
	answer(w, told)
}

// Drop takes the transcript of a recording away, with everything listening to
// it produced. It answers with the words the recording now has, which are none.
//
// A recording a run is listening to is refused, and one nothing has listened to
// is not found.
func (a *API) Drop(w http.ResponseWriter, r *http.Request, path string) {
	if a.Drops == nil {
		http.Error(w, errNoHearing.Error(), http.StatusNotImplemented)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), patience)
	defer cancel()

	showing, ref, err := a.held(ctx, path)
	if err != nil {
		refuse(w, err)
		return
	}
	if ref.Kind != domain.KindRecording {
		http.Error(w, errNotARecording.Error(), http.StatusNotFound)
		return
	}

	// The queue told about it is the one behind the vault the recording was
	// found in.
	drops := *a.Drops
	drops.Forgets = a.forgets()

	res, err := drops.Execute(ctx, showing, ref.Path)
	if err != nil {
		refuse(w, err)
		return
	}
	if res.Busy {
		http.Error(w, errBeingHeard.Error(), http.StatusConflict)
		return
	}
	if res.None {
		http.Error(w, errNotHeard.Error(), http.StatusNotFound)
		return
	}
	answer(w, spoken{Path: ref.Path, Cues: []cue{}, Editable: true})
}

// ordered is a transcript from the window as the cues it is written down as,
// and why it is not one where it cannot be.
//
// Speech runs forward: a cue ends no earlier than it begins, and begins after
// the one before it ends. A cue whose words trim away is dropped, and its
// timings still bound the cue after it.
//
// A transcript carrying no words at all is refused: what was heard comes back
// by deleting the file beside it, and writing nothing over the words leaves the
// recording saying nothing with nothing to edit.
func ordered(cues []cue) ([]transcript.Cue, error) {
	out := make([]transcript.Cue, 0, len(cues))
	last := cue{From: -1, To: -1}
	for at, one := range cues {
		switch {
		case one.From < 0 || one.To < one.From:
			return nil, fmt.Errorf("cue %d: %d to %d is not a stretch of a recording", at, one.From, one.To)
		case one.From < last.From:
			return nil, fmt.Errorf("cue %d: begins at %d, before the cue above it at %d", at, one.From, last.From)
		case one.From < last.To:
			return nil, fmt.Errorf("cue %d: begins at %d, inside the cue above it ending at %d", at, one.From, last.To)
		case strings.ContainsAny(one.Text, "\r\n"):
			// One cue is one line of the transcript, and the window edits it as
			// one. A cue broken over two lines is two the window would offer to
			// edit and one the recording would play.
			return nil, fmt.Errorf("cue %d: a cue stands on one line", at)
		}
		last = one
		if strings.TrimSpace(one.Text) == "" {
			continue
		}
		out = append(out, transcript.Cue{Text: one.Text, From: one.From, To: one.To})
	}
	if len(out) == 0 {
		return nil, errors.New("a transcript of no words is not one this recording was put right to")
	}
	return out, nil
}

// narrowed cuts the cues down to a run of the words, where the question named one.
// A question naming none is about the whole transcript.
//
// The run is a `start` and a `length` in the words, which is how a passage is
// addressed everywhere else, and what comes back is the speech those bytes were
// said in. A search hit is played from the first of them.
func narrowed(query url.Values, cues []transcript.Cue) ([]transcript.Cue, error) {
	at, wide := query.Get("start"), query.Get("length")
	if at == "" && wide == "" {
		return cues, nil
	}
	start, err := strconv.Atoi(at)
	if err != nil || start < 0 {
		return nil, fmt.Errorf("start: %q is not a place in the words", at)
	}
	length, err := strconv.Atoi(wide)
	if err != nil || length <= 0 {
		return nil, fmt.Errorf("length: %q is not a run of words", wide)
	}
	return transcript.At(cues, start, length), nil
}

// held is the vault the window is showing and what it holds at a path. It is
// the one reading of the vault a request gets, and everything the request goes
// on to do is done to that vault.
//
// Everything from outside reaches the vault through a reader, so a path leaving
// it is refused there.
func (a *API) held(ctx context.Context, path string) (domain.Vault, domain.FileRef, error) {
	showing := a.Showing()
	if showing.ID == "" || a.Readers == nil {
		return domain.Vault{}, domain.FileRef{}, errNoVault
	}
	reader, err := a.Readers.Open(showing)
	if err != nil {
		return domain.Vault{}, domain.FileRef{}, err
	}
	ref, err := reader.Stat(ctx, path)
	if err != nil {
		return domain.Vault{}, domain.FileRef{}, err
	}
	return showing, ref, nil
}

// hearing is what says which model listened to a recording and where what it
// wrote is kept. They are the index and the store a passage is placed from,
// which read the same artifacts.
func (a *API) hearing() (port.SourceQueries, port.DerivedStores, bool) {
	if a.Highlight == nil || a.Highlight.Sources == nil || a.Highlight.Derived == nil {
		return nil, nil, false
	}
	return a.Highlight.Sources, a.Highlight.Derived, true
}

// heard is what listened to the recording at a path and the store holding what
// it wrote. It answers false for a recording nothing has listened to.
func (a *API) heard(
	ctx context.Context,
	v domain.Vault,
	path string,
) (port.Recognised, port.DerivedStore, bool, error) {
	sources, stores, ok := a.hearing()
	if !ok {
		return port.Recognised{}, nil, false, nil
	}
	said, held, err := sources.Reading(ctx, v.ID, path)
	if err != nil || !held || said.From == "" {
		return port.Recognised{}, nil, false, err
	}
	store, err := stores.Open(v)
	if err != nil {
		return port.Recognised{}, nil, false, err
	}
	return said, store, true, nil
}

// transcript is what a model wrote down of the recording at a path, and nothing
// where nothing has listened to it. A run still going is read as far as it has
// got.
func (a *API) transcript(ctx context.Context, v domain.Vault, path string) ([]byte, error) {
	said, store, listened, err := a.heard(ctx, v, path)
	if err != nil || !listened {
		return nil, err
	}
	return written(ctx, store,
		derived.Artifact(said.From, said.Hash),
		derived.Partial(said.From, said.Hash),
	)
}

// transcribed is the transcript of a recording as it now stands: what it was put
// right to, and what was heard where nothing put it right.
func (a *API) transcribed(ctx context.Context, store port.DerivedStore, said port.Recognised) ([]byte, error) {
	return written(ctx, store,
		derived.Corrected(said.From, said.Hash),
		derived.Artifact(said.From, said.Hash),
		derived.Partial(said.From, said.Hash),
	)
}

// written is what the store holds under the first of these names, and nothing
// where it holds none of them. The store is a folder on the person's disk and
// they may empty it.
func written(ctx context.Context, store port.DerivedStore, names ...string) ([]byte, error) {
	for _, name := range names {
		raw, err := store.Read(ctx, name)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		return raw, nil
	}
	return nil, nil
}

// free says whether a name is one nobody holds. It is taken and let go, so what
// it answers is what stood a moment ago.
func free(ctx context.Context, store port.DerivedStore, name string) bool {
	release, err := store.Claim(ctx, name)
	if err != nil {
		return false
	}
	release()
	return true
}

// answer writes what the window is told, as the window reads it.
func answer(w http.ResponseWriter, told any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(told)
}
