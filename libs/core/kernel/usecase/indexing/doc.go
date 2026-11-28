// Package indexing keeps the index of a vault up to date while a window is open: the notes first, then the books, then the vectors both still need.
//
// A Worker schedules the passes, and a Session is the part of it that belongs to one vault. Pass progress is reported to Events.
//
// The passes are in this package: the text of the files other than notes is cut into the chunks a search runs over (Index). The chunks
// are given their missing vectors by usecase/embedchunks, which the worker reaches through ReadWholeVault. The core reads no file format; a plugin makes
// the text versions. Scan reads the notes of a vault into the index, and VaultOpener and OpenVaults open the vaults of a window around it. Cutting is deterministic and keeps no record of its progress: pending work is derived from the data.
package indexing
