package app

import (
	"context"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"

	machineport "github.com/jiva-studio/numen/libs/core/usecase/file/machinecontract"
)

// folderDialog is the system folder dialog, opened by the toolkit that owns the
// window. It belongs to that window and opens over it.
//
// One dialog is open at a time.
type folderDialog struct {
	window application.Window

	mu     sync.Mutex
	isOpen bool
}

// ChooseFolder returns the folder the user selected, and false when the user
// closed the dialog. The dialog ends on user input, and the wait is not
// bounded.
//
// It is called on a goroutine serving the page. The toolkit opens the dialog on
// the thread that owns the window and returns the result here, so the wait
// happens off that thread.
func (d *folderDialog) ChooseFolder(_ context.Context, title, startingAt string) (string, bool, error) {
	// The dialog library terminates the process when the machine has no settings
	// schemas to read, which closes the window and discards unsaved state.
	if !hasSchemas() {
		return "", false, machineport.ErrNoFolderDialog
	}
	if !d.tryAcquire() {
		return "", false, machineport.ErrChoosing
	}
	defer d.release()

	if title == "" {
		title = "Choose a folder"
	}
	dialog := application.Get().Dialog.OpenFile().
		AttachToWindow(d.window).
		CanChooseDirectories(true).
		CanChooseFiles(false).
		SetTitle(title)
	if startingAt != "" {
		dialog.SetDirectory(startingAt)
	}

	selected, err := dialog.PromptForSingleSelection()
	if err != nil {
		return "", false, err
	}
	return selected, selected != "", nil
}

// tryAcquire takes the dialog, and answers false where it is already up.
func (d *folderDialog) tryAcquire() bool {
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

// ChooseFile returns the file the user selected, and false when the user closed
// the dialog. The dialog shows the files whose extension is one of accept, or
// every file when accept is empty.
func (d *folderDialog) ChooseFile(_ context.Context, title, startingAt string, accept []string) (string, bool, error) {
	if !hasSchemas() {
		return "", false, machineport.ErrNoFolderDialog
	}
	if !d.tryAcquire() {
		return "", false, machineport.ErrChoosing
	}
	defer d.release()

	if title == "" {
		title = "Choose a file"
	}
	dialog := application.Get().Dialog.OpenFile().
		AttachToWindow(d.window).
		CanChooseDirectories(false).
		CanChooseFiles(true).
		SetTitle(title)
	if len(accept) > 0 {
		patterns := make([]string, 0, len(accept))
		for _, extension := range accept {
			patterns = append(patterns, "*"+extension)
		}
		dialog.AddFilter(strings.Join(accept, ", "), strings.Join(patterns, ";"))
	}
	if startingAt != "" {
		dialog.SetDirectory(startingAt)
	}

	selected, err := dialog.PromptForSingleSelection()
	if err != nil {
		return "", false, err
	}
	return selected, selected != "", nil
}
