package windowsession

import (
	"context"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// ChangeReporter tells the window, the index and the text versions what the passes behind a vault found
// changed.
type ChangeReporter struct {
	changes Changes
	wake    IndexWake
	texts   TextVersions
	diag    *Diagnostics
}

// NewChangeReporter makes the reporter over the feed of changes, the worker that indexes the vault, the
// text versions and the diagnostics of the window. wake and texts may be nil.
func NewChangeReporter(changes Changes, wake IndexWake, texts TextVersions, diag *Diagnostics) *ChangeReporter {
	return &ChangeReporter{changes: changes, wake: wake, texts: texts, diag: diag}
}

func (r *ChangeReporter) listenVersions(indexed IndexSession) {
	if r.texts != nil {
		r.texts.SetListener(versionChangeReporter{changes: r.changes, index: indexed})
	}
}

func (r *ChangeReporter) checkTexts(ctx context.Context, v domain.Vault, paths []string) {
	if r.texts == nil {
		return
	}
	if err := r.texts.ReportFilesChanged(ctx, v, paths); err != nil {
		r.diag.say("check the text versions of %s: %v\n", v.Name, err)
	}
}

func (r *ChangeReporter) newChangeReporter(
	triggers *triggerPasses, checkTexts func(paths []string),
) func(paths, assets []string, reload bool) {
	return func(paths, assets []string, reload bool) {
		// A client shows every file in the vault, so an asset is reported as a note is.
		changePaths := slices.Concat(paths, assets)
		if reload {
			triggers.requestAll()
		} else {
			triggers.requestPaths(changePaths)
		}
		r.changes.Publish(window.Change{Paths: changePaths, ShouldReload: reload})
		checkTexts(changePaths)
		if r.wake == nil {
			return
		}
		r.wake.ClearSkips(changePaths)
		if reload || len(assets) > 0 {
			// A book added to an open vault is read without a request.
			r.wake.WakeSources()
		}
		if reload || len(paths) > 0 {
			// A written note is chunked again, and its chunks need new vectors. The index
			// determines which. The rescan behind a reload may have saved or removed notes.
			r.wake.WakeNotes()
		}
	}
}

// versionChangeReporter asks the index session to cut a file whose text versions changed, and
// reports the file to the window.
type versionChangeReporter struct {
	changes Changes
	index   IndexSession
}

func (r versionChangeReporter) ReportVersionsChanged(v domain.Vault, path string) {
	r.index.ReportVersionsChanged(v, path)
	r.changes.Publish(window.Change{VersionPaths: []string{path}})
}
