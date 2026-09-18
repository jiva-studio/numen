package main

import (
	"context"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// folderDialog is the machine's own folder dialog, put up by the toolkit that
// owns the window. It belongs to that window, so it opens over it.
//
// One dialog is up at a time: a person answers one at a time.
type folderDialog struct {
	window application.Window

	mu     sync.Mutex
	isOpen bool
}

// Choose answers with the folder the person chose, and with false where they
// closed the dialog. A dialog ends when the person answers it, and that wait is
// on a person and is not measured.
//
// It is called on a goroutine serving the page. The toolkit puts the dialog up
// on the thread that owns the window and hands the answer back here, so the
// wait happens off that thread.
func (d *folderDialog) Choose(_ context.Context, title, startingAt string) (string, bool, error) {
	// The library the dialog is built by ends the process where this machine
	// holds no settings for it to read, taking the window and whatever a person
	// had not written down with it.
	if !hasSchemas() {
		return "", false, port.ErrNoFolderDialog
	}
	if !d.tryTake() {
		return "", false, port.ErrChoosing
	}
	defer d.release()

	if title == "" {
		title = "Choose a folder"
	}
	asking := application.Get().Dialog.OpenFile().
		AttachToWindow(d.window).
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

// tryTake takes the dialog, and answers false where it is already up.
func (d *folderDialog) tryTake() bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.isOpen {
		return false
	}
	d.isOpen = true
	return true
}

func (d *folderDialog) release() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.isOpen = false
}
