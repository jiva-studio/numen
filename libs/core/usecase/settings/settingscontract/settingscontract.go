// Package settingscontract re-exports the settings port contract.
package settingscontract

import "github.com/jiva-studio/numen/libs/core/port/settingsport"

type (
	Model = settingsport.Model
	Presence = settingsport.Presence
	Setting = settingsport.Setting
)

const (
	NotFetched = settingsport.NotFetched
	NothingToFetch = settingsport.NothingToFetch
	Present = settingsport.Present
)

var (
	ErrNotASetting = settingsport.ErrNotASetting
)
