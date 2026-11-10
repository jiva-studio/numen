package editor

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/protocol"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// What the services of notes, files and media ask of the window, as the methods their packages name.

// BeginWrite registers a write and reports false once the window is closing, when nothing may be
// written.
func (s *State) BeginWrite() bool { return s.Writing.begin() }

// FinishWrite ends a write that BeginWrite registered.
func (s *State) FinishWrite() { s.Writing.finish() }

// IsIndexOutdated reports whether err says the write reached the vault and the index did not
// follow, and shows the task that tells the user so.
func (s *State) IsIndexOutdated(err error) bool { return s.isIndexOutdated(err) }

// NoteWritten tells the embedder that a note holds new prose.
func (s *State) NoteWritten() {
	if s.NotifyNoteWritten != nil {
		s.NotifyNoteWritten()
	}
}

// SubscribeEdits returns the edits being made to notes and the function that ends the subscription.
func (s *State) SubscribeEdits() (<-chan domain.Edit, func()) { return s.Edits.subscribe() }

// PublishChange tells the views drawing the vault that the paths changed.
func (s *State) PublishChange(paths ...string) { s.ChangeEvents.publish(window.Change{Paths: paths}) }

// EmitFileEvent tells the plugins of the window that an event happened, when the window has plugins.
func (s *State) EmitFileEvent(event string) {
	if s.EmitEvent != nil {
		s.EmitEvent(event)
	}
}

// GetMediaAddress is where the file with this stat is loaded from, and nowhere while the socket
// does not answer.
func (s *State) GetMediaAddress(v domain.Vault, stat domain.FileStat) string {
	return s.Playing.Load().Address(v, stat)
}

// GetPluginFileAddress is where a file of a plugin's store of the vault is loaded from, and nowhere
// while the socket does not answer.
func (s *State) GetPluginFileAddress(v domain.Vault, address file.Address, size int64) string {
	return s.Playing.Load().PluginFileAddress(v, address, size)
}

// EmitVectorsReady tells the plugins of the window that every vector of the vault exists.
func (s *State) EmitVectorsReady(_ context.Context, v domain.Vault) {
	s.EmitFileEvent(protocol.GetVectorsReadyEvent(string(v.ID)))
}
