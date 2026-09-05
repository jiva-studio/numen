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

// namedInARefusal is how many of the files that stayed outside are named before
// the rest are counted.
const namedInARefusal = 3

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
	showing := api.Showing()
	if showing.ID == "" {
		return
	}
	if !api.Writing.begin() {
		return
	}
	defer api.Writing.done()

	at := task.Task{ID: importingFiles + " " + into, Doing: "Bringing files in", About: into}
	api.say(at)

	brought, err := api.Files.Import.Execute(ctx, showing, into, paths)
	if landed := directlyIn(into, brought.Landed); len(landed) > 0 {
		api.Listeners.tell(change{paths: landed})
	}
	if err != nil {
		at.Failed = err.Error()
		api.say(at)
		return
	}
	if said := refusedIn(brought.Refused); said != "" {
		at.Failed = said
		api.say(at)
		return
	}
	api.finished(at.ID)
}

// directlyIn is what of a drop sits in the folder it was let go over. What
// arrived deeper sits in folders that arrived with it, and those are read when
// a person opens them.
func directlyIn(into string, landed []string) []string {
	shown := make([]string, 0, len(landed))
	for _, path := range landed {
		if pathpkg.Dir(path) == into || (into == "" && !strings.Contains(path, "/")) {
			shown = append(shown, path)
		}
	}
	return shown
}

// refusedIn is what a drop could not bring in, in one sentence. Nothing is said
// where every file arrived.
func refusedIn(refused []vaults.Refusal) string {
	if len(refused) == 0 {
		return ""
	}
	said := make([]string, 0, namedInARefusal)
	for _, one := range refused[:min(len(refused), namedInARefusal)] {
		said = append(said, fmt.Sprintf("%s: %v", one.Name, one.Why))
	}
	if rest := len(refused) - len(said); rest > 0 {
		said = append(said, fmt.Sprintf("and %d more", rest))
	}
	return strings.Join(said, "; ")
}
