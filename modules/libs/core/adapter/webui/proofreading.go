package webui

import (
	"context"
	"errors"
	"net/http"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	derived "github.com/jiva-studio/numen/modules/libs/core/text"
)

// A transcript a model heard is put right by a second model, asked for over the
// recording it belongs to. The window is answered at once and how far the run
// gets is in the list of what is being done.

// Proofreading puts the transcript of a recording right, for a person who asked
// for it.
type Proofreading interface {
	// ProofreaderReady says whether this installation has anything to put a
	// transcript right with.
	ProofreaderReady() bool
	// Proofread puts one transcript right, behind the caller.
	Proofread(v domain.Vault, path string)
}

// errNoProofreading is a build, or an installation, with nothing to put a
// transcript right with.
var errNoProofreading = errors.New("this build cannot proofread a transcript")

// The sentences a person reads for a proofreading they asked for.
const (
	notATranscript  = "Only a recording's transcript is proofread, and this file is not one."
	nothingHeard    = "Nothing has been transcribed here, so there is nothing to proofread."
	proofreadingNow = "This recording is being worked on now."
)

// Proofread begins putting the transcript of the recording at a path right.
//
// One run to a recording: a recording another run holds is said to be under way.
func (a *API) Proofread(w http.ResponseWriter, r *http.Request, path string) {
	if a.Proofreads == nil || !a.Proofreads.ProofreaderReady() {
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
	said, store, listened, err := a.heard(ctx, showing, ref.Path)
	if err != nil {
		refuse(w, err)
		return
	}
	if !listened {
		answer(w, began{Path: ref.Path, Answer: outcomeUnheard, Why: nothingHeard})
		return
	}

	// One run to a recording, by the name it writes under. A run listening to
	// this recording holds that name, and so does a proofreading of it.
	if !free(ctx, store, derived.Partial(said.From, said.Hash)) {
		answer(w, began{Path: ref.Path, Answer: outcomeRunning, Why: proofreadingNow})
		return
	}

	a.Proofreads.Proofread(showing, ref.Path)
	answer(w, began{Path: ref.Path, Answer: outcomeStarted})
}
