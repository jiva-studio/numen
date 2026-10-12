package settings

import "github.com/jiva-studio/numen/modules/libs/core/port"

// Setting is one setting of the file, and what to put there. Model is a model a
// setting can name, and Presence is what its files are on this machine.
type (
	Setting  = port.Setting
	Model    = port.Model
	Presence = port.Presence
)

// What a model's files are on this machine.
const (
	NothingToFetch = port.NothingToFetch
	Present        = port.Present
	NotFetched     = port.NotFetched
)

// ErrNotASetting is a value the settings cannot be read out of again. Nothing is
// written, and the value is the caller's to correct.
var ErrNotASetting = port.ErrNotASetting

// ErrStale is a file that holds bytes the caller has not read. It is left alone.
var ErrStale = port.ErrStale
