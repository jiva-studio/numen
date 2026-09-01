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

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
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

// listened is what the window is told a recording is: how far the words reach,
// and how much of it a run has written down, both in milliseconds.
//
// A recording nothing has listened to reaches nowhere, and the player it is
// loaded into is what then says how long it runs.
type listened struct {
	Path   string `json:"path"`
	Length int    `json:"length"`
	Heard  int    `json:"heard"`
	// Media is where the recording is played from. It is answered here and not
	// worked out by the window, because the socket it stands on is opened afresh
	// for every run.
	Media string `json:"media"`
}

// spoken is what was heard in a recording, in the order it was said.
type spoken struct {
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

	_, ref, err := a.held(ctx, path)
	if err != nil || ref.Kind != domain.KindRecording {
		a.Document(w, r, path)
		return
	}
	raw, err := a.transcript(ctx, ref.Path)
	if err != nil {
		refuse(w, err)
		return
	}
	heard, _ := transcript.Reached(raw)
	_, cues := transcript.Read(raw)
	told := listened{
		Path:   ref.Path,
		Length: heard,
		Heard:  heard,
		Media:  a.Playing.Address(a.Showing(), ref.Path),
	}
	if len(cues) > 0 {
		told.Length = max(told.Length, cues[len(cues)-1].To)
	}
	answer(w, told)
}

// Cues answers with the words heard in a recording, each against the
// milliseconds it was spoken in. A recording nothing has listened to holds no
// words, which is an answer.
func (a *API) Cues(w http.ResponseWriter, r *http.Request, path string) {
	if _, _, ok := a.hearing(); !ok {
		http.Error(w, errNoHearing.Error(), http.StatusNotImplemented)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), patience)
	defer cancel()

	_, ref, err := a.held(ctx, path)
	if err != nil {
		refuse(w, err)
		return
	}
	if ref.Kind != domain.KindRecording {
		http.Error(w, errNotARecording.Error(), http.StatusNotFound)
		return
	}
	raw, err := a.transcript(ctx, ref.Path)
	if err != nil {
		refuse(w, err)
		return
	}
	_, cues := transcript.Read(raw)
	cues, err = narrowed(r.URL.Query(), cues)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	told := spoken{Path: ref.Path, Cues: make([]cue, 0, len(cues))}
	for _, one := range cues {
		told.Cues = append(told.Cues, cue{Text: one.Text, From: one.From, To: one.To})
	}
	answer(w, told)
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

// held is the vault's reader and what it holds at a path. Everything from
// outside reaches the vault through a reader, so a path leaving it is refused
// there.
func (a *API) held(ctx context.Context, path string) (port.VaultReader, domain.FileRef, error) {
	showing := a.Showing()
	if showing.ID == "" || a.Readers == nil {
		return nil, domain.FileRef{}, errNoVault
	}
	reader, err := a.Readers.Open(showing)
	if err != nil {
		return nil, domain.FileRef{}, err
	}
	ref, err := reader.Stat(ctx, path)
	if err != nil {
		return nil, domain.FileRef{}, err
	}
	return reader, ref, nil
}

// hearing is what says which model listened to a recording and where what it
// wrote is kept. They are the index and the store a passage is placed from,
// which read the same artifacts.
func (a *API) hearing() (port.SourceQueries, port.DerivedStores, bool) {
	if a.Marking == nil || a.Marking.Sources == nil || a.Marking.Derived == nil {
		return nil, nil, false
	}
	return a.Marking.Sources, a.Marking.Derived, true
}

// transcript is what a model wrote down of the recording at a path, and nothing
// where nothing has listened to it. A run still going is read as far as it has
// got.
func (a *API) transcript(ctx context.Context, path string) ([]byte, error) {
	sources, stores, ok := a.hearing()
	if !ok {
		return nil, nil
	}
	said, held, err := sources.Reading(ctx, a.Showing().ID, path)
	if err != nil || !held || said.From == "" {
		return nil, err
	}
	store, err := stores.Open(a.Showing())
	if err != nil {
		return nil, err
	}
	for _, name := range []string{
		derived.Artifact(said.From, said.Hash),
		derived.Partial(said.From, said.Hash),
	} {
		raw, err := store.Read(ctx, name)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		return raw, nil
	}
	// The store is a folder on the person's disk and they may empty it.
	return nil, nil
}

// answer writes what the window is told, as the window reads it.
func answer(w http.ResponseWriter, told any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(told)
}
