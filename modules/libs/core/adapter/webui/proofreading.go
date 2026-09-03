package webui

import (
	"context"
	"errors"
	"net/http"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// A transcript a model heard is put right by a second model, asked for over the
// recording it belongs to. The work carries on behind the answer, and the answer
// says what came of asking.

// Proofreading puts the transcript of a recording right, for a person who asked
// for it.
type Proofreading interface {
	// ProofreaderReady says whether this installation has anything to put a
	// transcript right with.
	ProofreaderReady() bool
	// Proofread puts one transcript right and says what came of asking. It
	// answers before the work is over.
	Proofread(ctx context.Context, v domain.Vault, path string) (source.PutRightResult, error)
}

// errNoProofreading is a build, or an installation, with nothing to put a
// transcript right with.
var errNoProofreading = errors.New("this build cannot proofread a transcript")

// The sentences a person reads for a proofreading they asked for.
const (
	notATranscript    = "Only a recording's transcript is proofread, and this file is not one."
	nothingHeard      = "Nothing has been transcribed here, so there is nothing to proofread."
	proofreadingNow   = "This recording is being worked on now."
	proofreadAlready  = "This transcript has already been put right."
	proofreadByHand   = "These words were written by hand, and a model does not correct them."
	proofreadingBegun = "Putting this transcript right has begun."
)

// Proofread begins putting the transcript of the recording at a path right, and
// says what came of asking.
func (a *API) Proofread(w http.ResponseWriter, r *http.Request, path string) {
	// Taken once, so the whole answer is the work of the vault the window was
	// showing when it was asked.
	puts := a.proofreads()
	if puts == nil || !puts.ProofreaderReady() {
		http.Error(w, errNoProofreading.Error(), http.StatusNotImplemented)
		return
	}
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
	if ref.Kind != domain.KindRecording {
		answer(w, began{Path: ref.Path, Answer: outcomeUnfit, Why: notATranscript})
		return
	}
	_, _, listened, err := a.heard(ctx, showing, ref.Path)
	if err != nil {
		refuse(w, err)
		return
	}
	if !listened {
		answer(w, began{Path: ref.Path, Answer: outcomeUnheard, Why: nothingHeard})
		return
	}

	res, err := puts.Proofread(ctx, showing, ref.Path)
	if err != nil {
		refuse(w, err)
		return
	}
	answer(w, began{Path: ref.Path, Answer: came(res), Why: says(res)})
}

// came is what asking for a transcript to be put right came to, and says is the
// sentence a person reads for it.
//
// One run to a recording, by the name it writes under: a run listening to this
// recording holds that name, and so does a proofreading of it.
func came(res source.PutRightResult) string {
	switch {
	case res.Busy:
		return outcomeRunning
	case res.None:
		return outcomeUnheard
	case res.Edited:
		return outcomeByHand
	case res.Already:
		return outcomeDone
	}
	return outcomeStarted
}

func says(res source.PutRightResult) string {
	switch {
	case res.Busy:
		return proofreadingNow
	case res.None:
		return nothingHeard
	case res.Edited:
		return proofreadByHand
	case res.Already:
		return proofreadAlready
	}
	return proofreadingBegun
}
