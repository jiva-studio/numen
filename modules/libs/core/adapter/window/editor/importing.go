package editor

import (
	"context"
	"fmt"
	pathpkg "path"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/task"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// importingFiles is what a drop is called in the list of what is being done.
const importingFiles = "importing files"

// namedInAnError is how many of the files that stayed outside are named before
// the rest are counted.
const namedInAnError = 3

// Imports copies files a person let go of over the window into a folder of the
// vault, the root being the empty path.
//
// It runs where the drop reached the application, which is off the thread the
// page is served on, and it answers nothing: what arrived is said to everyone
// drawing the vault, and what did not stands in the list of what is being done.
//
// The watcher reports a picture or an archive to nobody, so what arrived is
// named to the listeners here.
func (o *Installation) Imports(ctx context.Context, into string, paths []string) {
	api := o.API
	if api.Files.Import == nil || len(paths) == 0 {
		return
	}
	showing := api.GetShownVault()
	if showing.ID == "" {
		return
	}
	if !api.Writing.begin() {
		return
	}
	defer api.Writing.finish()

	at := task.Task{ID: importingFiles + " " + into, Doing: "Bringing files in", About: into}
	api.say(at)

	imported, err := api.Files.Import.Execute(ctx, showing, into, paths)
	if direct := getDirectChildren(into, imported.Landed); len(direct) > 0 {
		api.Listeners.tell(change{paths: direct})
	}
	if err != nil {
		at.Error = err.Error()
		api.say(at)
		return
	}
	if failureText := describeImportFailures(imported.Errors); failureText != "" {
		at.Error = failureText
		api.say(at)
		return
	}
	api.finishTask(at.ID)
}

// getDirectChildren returns items in the target folder.
func getDirectChildren(into string, landed []string) []string {
	shown := make([]string, 0, len(landed))
	for _, path := range landed {
		if pathpkg.Dir(path) == into || (into == "" && !strings.Contains(path, "/")) {
			shown = append(shown, path)
		}
	}
	return shown
}

// describeImportFailures formats import failures into a single descriptive message.
func describeImportFailures(errs []vaults.ImportFailure) string {
	if len(errs) == 0 {
		return ""
	}
	messages := make([]string, 0, namedInAnError)
	for _, one := range errs[:min(len(errs), namedInAnError)] {
		messages = append(messages, fmt.Sprintf("%s: %v", one.Name, one.Why))
	}
	if rest := len(errs) - len(messages); rest > 0 {
		messages = append(messages, fmt.Sprintf("and %d more", rest))
	}
	return strings.Join(messages, "; ")
}
