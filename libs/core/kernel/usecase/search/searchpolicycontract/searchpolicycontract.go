// Package searchpolicycontract re-exports the search policy port contract.
package searchpolicycontract

import "github.com/jiva-studio/numen/libs/core/kernel/port/searchpolicyport"

type (
	FloorStore = searchpolicyport.FloorStore
	Nearest    = searchpolicyport.Nearest
	Passage    = searchpolicyport.Passage
	Samples    = searchpolicyport.Samples
)
