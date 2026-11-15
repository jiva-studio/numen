// Package floorrecords keeps the records of the floors as JSON files in the plugin's folder of the
// store of a vault.
package floorrecords

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/usecase/relevance"
)

// storeRecords keeps one JSON file for each recipe in the plugin's folder of the vault's store.
type storeRecords struct {
	store relevance.Folder
}

// New returns the records kept in the store folder.
func New(store relevance.Folder) relevance.Records {
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
	ChunkCount   int     `json:"chunkCount"`
	CalibratedAt int64   `json:"calibratedAt"`
}

func getRecordName(recipe string) string {
	sum := sha256.Sum256([]byte(recipe))
	return "floor-" + hex.EncodeToString(sum[:8]) + ".json"
}

func (r storeRecords) Read(ctx context.Context, recipe string) (relevance.Record, bool, error) {
	raw, err := r.store.Read(ctx, getRecordName(recipe))
	if errors.Is(err, host.ErrNotExist) {
		return relevance.Record{}, false, nil
	}
	if err != nil {
		return relevance.Record{}, false, fmt.Errorf("read the record of the floor: %w", err)
	}
	var file recordFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return relevance.Record{}, false, fmt.Errorf("decode the record of the floor: %w", err)
	}
	if file.Recipe != recipe {
		return relevance.Record{}, false, nil
	}
	record := relevance.Record{Recipe: file.Recipe, IsManual: file.IsManual, ManualFloor: file.ManualFloor, Calibration: nil}
	if c := file.Calibration; c != nil {
		record.Calibration = &relevance.Calibration{
			Floor: c.Floor, Median: c.Median, Spread: c.Spread, Histogram: c.Histogram,
			QueryCount: c.QueryCount, ChunkCount: c.ChunkCount,
			CalibratedTime: time.Unix(c.CalibratedAt, 0).UTC(),
		}
	}
	return record, true, nil
}

func (r storeRecords) Write(ctx context.Context, record relevance.Record) error {
	file := recordFile{Recipe: record.Recipe, IsManual: record.IsManual, ManualFloor: record.ManualFloor, Calibration: nil}
	if c := record.Calibration; c != nil {
		file.Calibration = &calibrationFile{
			Floor: c.Floor, Median: c.Median, Spread: c.Spread, Histogram: c.Histogram,
			QueryCount: c.QueryCount, ChunkCount: c.ChunkCount, CalibratedAt: c.CalibratedTime.Unix(),
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
