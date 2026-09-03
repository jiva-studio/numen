package main

import (
	"context"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// picker is the machine's own folder picker, put up by the toolkit that owns
// the window. It belongs to that window, so it opens over it.
//
// One picker is up at a time: a person answers one at a time.
type picker struct {
	window application.Window

	mu sync.Mutex
	up bool
}

var _ port.FolderDialog = (*picker)(nil)

// Choose answers with the folder the person chose, and with false where they
// closed the picker. A picker ends when the person answers it, and that wait is
// on a person and is not measured.
//
// It is called on a goroutine serving the page. The toolkit puts the picker up
// on the thread that owns the window and hands the answer back here, so the
// wait happens off that thread.
func (p *picker) Choose(_ context.Context, title, startingAt string) (string, bool, error) {
	// The library the picker is built by ends the process where this machine
	// holds no settings for it to read, taking the window and whatever a person
	// had not written down with it.
	if !settled() {
		return "", false, port.ErrNoFolderDialog
	}
	if !p.alone() {
		return "", false, port.ErrChoosing
	}
	defer p.done()

	if title == "" {
		title = "Choose a folder"
	}
	asking := application.Get().Dialog.OpenFile().
		AttachToWindow(p.window).
		CanChooseDirectories(true).
		CanChooseFiles(false).
		SetTitle(title)
	if startingAt != "" {
		asking.SetDirectory(startingAt)
	}

	chosen, err := asking.PromptForSingleSelection()
	if err != nil {
		return "", false, err
	}
	return chosen, chosen != "", nil
}

// alone takes the picker, and answers false where it is already up.
func (p *picker) alone() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.up {
		return false
	}
	p.up = true
	return true
}

func (p *picker) done() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.up = false
}
