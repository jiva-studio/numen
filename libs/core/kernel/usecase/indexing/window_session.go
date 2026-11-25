package indexing

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// NewWindowSession creates the index session of the vault in the form the pass runner of a window takes.
func (w *Worker) NewWindowSession(v domain.Vault) windowsession.IndexSession {
	session := w.NewSession(v)
	return windowsession.IndexSession{
		Run: func(ctx context.Context, rebuild bool, read windowsession.ReadNotes) {
			session.Run(ctx, rebuild, ReadNotes(read))
		},
		ReportVersionsChanged: session.ReportVersionsChanged,
	}
}
