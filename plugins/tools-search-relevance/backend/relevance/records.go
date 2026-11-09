package relevance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
)

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

// storeRecords keeps one JSON file for each recipe in the plugin's folder of the vault's store.
type storeRecords struct {
	store host.Store
}

// NewRecords returns the records kept in the store folder.
func NewRecords(store host.Store) Records {
	return storeRecords{store: store}
}

// recordFile is the shape of a record file. Time is in seconds since the epoch.
type recordFile struct {
	Recipe      string           `json:"recipe"`
	IsManual    bool             `json:"isManual"`
	ManualFloor float64          `json:"manualFloor"`
	Calibration *calibrationFile `json:"calibration,omitempty"`
}

type calibrationFile struct {
	Floor        float64 `json:"floor"`
	Median       float64 `json:"median"`
	Spread       float64 `json:"spread"`
	Histogram    []int   `json:"histogram"`
	QueryCount   int     `json:"queryCount"`
	PassageCount int     `json:"passageCount"`
	CalibratedAt int64   `json:"calibratedAt"`
}

func getRecordName(recipe string) string {
	sum := sha256.Sum256([]byte(recipe))
	return "floor-" + hex.EncodeToString(sum[:8]) + ".json"
}

func (r storeRecords) Read(ctx context.Context, recipe string) (Record, bool, error) {
	raw, err := r.store.Read(ctx, getRecordName(recipe))
	if errors.Is(err, host.ErrNotExist) {
		return Record{}, false, nil
	}
	if err != nil {
		return Record{}, false, fmt.Errorf("read the record of the floor: %w", err)
	}
	var file recordFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return Record{}, false, fmt.Errorf("decode the record of the floor: %w", err)
	}
	if file.Recipe != recipe {
		return Record{}, false, nil
	}
	record := Record{Recipe: file.Recipe, IsManual: file.IsManual, ManualFloor: file.ManualFloor, Calibration: nil}
	if c := file.Calibration; c != nil {
		record.Calibration = &Calibration{
			Floor: c.Floor, Median: c.Median, Spread: c.Spread, Histogram: c.Histogram,
			QueryCount: c.QueryCount, PassageCount: c.PassageCount,
			CalibratedTime: time.Unix(c.CalibratedAt, 0).UTC(),
		}
	}
	return record, true, nil
}

func (r storeRecords) Write(ctx context.Context, record Record) error {
	file := recordFile{Recipe: record.Recipe, IsManual: record.IsManual, ManualFloor: record.ManualFloor, Calibration: nil}
	if c := record.Calibration; c != nil {
		file.Calibration = &calibrationFile{
			Floor: c.Floor, Median: c.Median, Spread: c.Spread, Histogram: c.Histogram,
			QueryCount: c.QueryCount, PassageCount: c.PassageCount, CalibratedAt: c.CalibratedTime.Unix(),
		}
	}
	raw, err := json.Marshal(file)
	if err != nil {
		return fmt.Errorf("encode the record of the floor: %w", err)
	}
	if err := r.store.Write(ctx, getRecordName(record.Recipe), raw); err != nil {
		return fmt.Errorf("write the record of the floor: %w", err)
	}
	return nil
}
