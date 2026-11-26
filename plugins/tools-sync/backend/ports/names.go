package ports

import "github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"

// Names puts the files of a vault under the names of its notes: the paths of the notes are in one normal form whatever
// the disk names are.
type Names interface {
	GetFiles(id domain.VaultID, inner Files) Files
	GetNotes(id domain.VaultID, files Files, inner Notes) Notes
	GetAbsent(id domain.VaultID, files Files, inner Absent) Absent
	GetTransfers(id domain.VaultID, files Files, inner Transfers) Transfers
}
