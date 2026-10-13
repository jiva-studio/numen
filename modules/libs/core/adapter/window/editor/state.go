// Package editor answers what a client may ask about a vault.
//
// The questions and their answers are the schema in modules/libs/protocol; this
// is the half that answers them. The interface asks over HTTP either way, so
// the same handler serves the window the application opens and a browser during
// development.
package editor

import (
	"context"
	"errors"
	"sync/atomic"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/wire"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/task"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// State is the vault the window is looking at and what every handler of it
// reads: the vault shown, the door on every question, the writes and tasks in
// the air, and who is drawing what changes.
type State struct {
	// vault is the vault the window has open. It is replaced while requests are
	// being served, so every reader takes it through GetCurrentVault.
	vault atomic.Pointer[domain.Vault]

	// closed is the door on every question. It is closed before the index and the
	// embedder an answer reaches into are taken away.
	closed atomic.Bool

	// questions are the questions taken and not yet answered. A search, a note
	// and a link are read straight from the index, and Close waits here until
	// the last of them is off it.
	//
	// A stream is not counted: it lives as long as the page that opened it.
	questions inflight

	// session is the passes behind the vault the window is showing. It is
	// published as one after the vault, so a run, a cut and a drop reach the
	// vault the request was answered over. Nothing while the vault is being
	// changed.
	session atomic.Pointer[vaultSession]

	ChangeEvents broadcaster[change]

	// OpenVault puts another vault in the window: the agents are stopped, the vault
	// is swapped, and the agents are started again against the one that
	// arrived. It is bound by every build that serves the list of vaults.
	OpenVault func(context.Context, domain.Vault) error

	// agent is the agent the panel's tasks go to. A vault without one answers
	// that it has none, and the rest of the window works as it did. It is
	// replaced while requests are being served, so it is taken through
	// GetAgent.
	agent atomic.Pointer[port.Agent]

	// NotifyNoteWritten is what a save raises: the reading behind the window asks the index
	// what owes a vector, once the vault has been still. Nil for a build with
	// nothing reading behind it, and then a save changes no vectors.
	NotifyNoteWritten func()

	// Edits is everyone drawing this vault, for a change to a note being made
	// while they may be showing it.
	Edits broadcaster[domain.Edit]

	// Places is everyone drawing this vault, for when something asks that a
	// place be put in front of the person.
	Places broadcaster[domain.Place]

	// openTabs is what the person has open, as the window last said. It is
	// replaced while requests are being served, so every reader takes it
	// through GetOpenTabs.
	openTabs atomic.Pointer[domain.OpenTabs]

	// NotifyOpenTabs hears what the person has open each time the window says it,
	// once what it said stands. A build without one takes the report and tells
	// nobody.
	NotifyOpenTabs func(domain.OpenTabs)

	// Writing is the writes taken and not yet finished.
	Writing inflight

	// Ready is set when the scan finished, and Error says why it could not.
	Ready atomic.Bool
	Error wire.Reason
	// WatchError is why the vault is not being followed, when it is not.
	WatchError wire.Reason
	// Unreachable is why an agent cannot be reached, when one cannot.
	Unreachable wire.Reason

	// Window is this window itself: everything being done behind it, which
	// whatever does work puts itself into, and everyone drawing it for the
	// moment it goes.
	Window *wire.Window

	Indexing IndexState
}

// IndexState is how far the vault has been read for meaning.
type IndexState struct {
	// Progress answers how far cutting and embedding have got. Nil for a vault
	// nothing is reading for meaning, and the window then says nothing about it.
	Progress port.IndexProgress
	// Model is the identity vectors are being made under, which is what decides
	// whether a chunk already carries one. Empty for an installation with no
	// model, and then nothing is going to embed anything. It is named by the
	// goroutine reading the vault and asked for by every request.
	Model atomic.Value
	// Recipe is everything that decides what a vector is, which is what a
	// vector is found by.
	Recipe atomic.Value
}

// GetCurrentVault is the vault the window has open. A window standing on nothing
// answers with no vault at all.
func (s *State) GetCurrentVault() domain.Vault {
	if v := s.vault.Load(); v != nil {
		return *v
	}
	return domain.Vault{}
}

// setVault puts a vault in front of whoever asks from now on.
func (s *State) setVault(v domain.Vault) { s.vault.Store(&v) }

// setSession sets the session a run, a rechunk and a drop are taken through from now
// on. They arrive together, after the vault they belong to.
func (s *State) setSession(on *vaultSession) { s.session.Store(on) }

// getRecogniser reads a scanned document and getTranscriber hears a recording,
// each for whoever asks. They are the jobs an agent asks through too, so what a
// person started in the window is shown to both. Nothing where the window has
// no vault, and where this build does no such run.
func (s *State) getRecogniser() Runner {
	if on := s.session.Load(); on != nil {
		return on.recognitionRunner
	}
	return nil
}

func (s *State) getTranscriber() Runner {
	if on := s.session.Load(); on != nil {
		return on.transcriptionRunner
	}
	return nil
}

// getProofreader puts a recording's transcript right, for whoever asks.
func (s *State) getProofreader() Proofreader {
	if on := s.session.Load(); on != nil {
		return on.transcriptProofreader
	}
	return nil
}

// getRecognitionProofreader puts a document's recognition right, for whoever asks.
func (s *State) getRecognitionProofreader() RecognitionProofreader {
	if on := s.session.Load(); on != nil {
		return on.proofreadsRecognition
	}
	return nil
}

// getRechunker asks for a source to be cut again from whatever its text now says. A
// window that put a transcript right calls it, so search answers with the words
// as they now read. Nothing while the window has no vault, and then a
// correction is baseline in the tab alone.
func (s *State) getRechunker() func(context.Context, domain.Vault, string) error {
	if on := s.session.Load(); on != nil {
		return on.rechunk
	}
	return nil
}

// getRecordingForgetter takes a recording out of what the queue behind the vault has
// already had an answer about, so one that gave no words is offered again.
func (s *State) getRecordingForgetter() func(domain.Vault, string) {
	if on := s.session.Load(); on != nil {
		return on.forgetRecording
	}
	return nil
}

// reportTask puts one piece of work in the list of what is being done behind the
// window, for a build that keeps one.
func (s *State) reportTask(at task.Task) { s.Window.AddTask(at) }

// finishTask takes one piece of work out of that list.
func (s *State) finishTask(id string) { s.Window.RemoveTask(id) }

// isUnlevelled says whether a write reached the vault and the index did not
// follow.
// The answer goes on the wire, so the client that saved knows search has not
// caught up with what it saved, and it stands in the list of what is being done
// until a write levels the index again, so a person who is not looking at that
// note is told too.
func (s *State) isUnlevelled(err error) bool {
	if err == nil {
		s.finishTask(levellingTheIndex)
		return false
	}
	if !errors.Is(err, note.ErrUnlevelled) {
		return false
	}
	s.reportTask(task.Task{ID: levellingTheIndex, Title: "Bringing the index level", Error: err.Error()})
	return true
}

// Close refuses every question from now on, and there is no opening it again. It
// answers once the questions already taken have been answered, so everything an
// answer reaches into is still there for the whole of it.
func (s *State) Close() {
	s.closed.Store(true)
	<-s.questions.seal()
}

func (s *State) isClosed() bool { return s.closed.Load() }

// getCurrentVault is the vault a question is answered over. A window standing on nothing
// has none, and every question that would reach into a vault is refused there.
func (s *State) getCurrentVault() (domain.Vault, error) {
	v := s.GetCurrentVault()
	if v.ID == "" {
		return domain.Vault{}, connect.NewError(connect.CodeFailedPrecondition, errNoVault)
	}
	return v, nil
}

// GetAgent is the agent the panel's tasks go to, and nothing where the vault
// has none.
func (s *State) GetAgent() port.Agent {
	if taking := s.agent.Load(); taking != nil {
		return *taking
	}
	return nil
}

// SetAgent names who takes the panel's tasks from now on. Nil leaves the vault
// with no agent.
func (s *State) SetAgent(taking port.Agent) {
	s.agent.Store(&taking)
}

func text(v *atomic.Value) string {
	s, _ := v.Load().(string)
	return s
}

// getShownVault is the vault a document is read from. A window standing on
// nothing holds no file to say anything about.
func (s *State) getShownVault() (domain.Vault, error) {
	showing := s.GetCurrentVault()
	if showing.ID == "" {
		return domain.Vault{}, errNoVault
	}
	return showing, nil
}

// getBackgroundContext is what a drawing nobody asked for runs under: the
// context the passes behind the vault in the window run under. The vault going
// ends it, so a window that is closing is not held open by a page nobody has
// turned to. A window standing on no vault has no passes, and nothing to end.
func (s *State) getBackgroundContext() context.Context {
	if on := s.session.Load(); on != nil {
		return on.ctx
	}
	return context.Background()
}
