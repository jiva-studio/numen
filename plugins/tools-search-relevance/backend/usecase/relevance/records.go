package relevance

import "context"

// Record is what the plugin keeps for one vault under one recipe: the floor a person set, if any, and
// the last measurement. The statistics of a measurement are kept here and nowhere else.
type Record struct {
	Recipe      string
	IsManual    bool
	ManualFloor float64
	Calibration *Calibration
}

// Records keeps the record of each recipe of one vault.
type Records interface {
	// Read returns the record kept for the recipe, and false where none is kept.
	Read(ctx context.Context, recipe string) (Record, bool, error)
	// Write keeps the record, replacing the one kept for its recipe.
	Write(ctx context.Context, record Record) error
}
