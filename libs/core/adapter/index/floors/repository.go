// Package floors stores the similarity floor of each vault under each embedding
// recipe in the index database, and reads the samples a calibration measures.
package floors

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/core/adapter/index/sqlfile"
	"github.com/jiva-studio/numen/libs/core/adapter/index/sqlwrite"
	"github.com/jiva-studio/numen/libs/core/port/floorport"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

//go:embed sql/*.sql
var files embed.FS

var stmt = sqlfile.Load(files, "sql")

// Repository is the floors of every vault. Reads go through read and writes
// through write, the pools the index database opens.
type Repository struct {
	read  *sql.DB
	write *sql.DB
}

// NewRepository is the floors over the index database's two pools.
func NewRepository(read, write *sql.DB) *Repository {
	return &Repository{read: read, write: write}
}

// ReadFloor returns the setting kept for the vault under the recipe, and false
// where nothing is kept.
func (r *Repository) ReadFloor(ctx context.Context, v domain.Vault, recipe string) (floorport.Setting, bool, error) {
	var (
		setting                    floorport.Setting
		floor, median, spread      sql.NullFloat64
		histogram                  sql.NullString
		queries, passages, instant sql.NullInt64
	)
	err := r.read.QueryRowContext(ctx, stmt.Get("read"), string(v.ID), recipe).Scan(
		&setting.IsManual, &setting.ManualFloor, &floor, &median, &spread, &histogram, &queries, &passages, &instant)
	if errors.Is(err, sql.ErrNoRows) {
		return floorport.Setting{}, false, nil
	}
	if err != nil {
		return floorport.Setting{}, false, fmt.Errorf("read the floor of %s: %w", v.ID, err)
	}
	if !floor.Valid {
		return setting, true, nil
	}
	calibration := floorport.Calibration{
		Floor:          floor.Float64,
		Median:         median.Float64,
		Spread:         spread.Float64,
		QueryCount:     int(queries.Int64),
		PassageCount:   int(passages.Int64),
		CalibratedTime: time.Unix(instant.Int64, 0).UTC(),
	}
	if histogram.Valid && histogram.String != "" {
		if err := json.Unmarshal([]byte(histogram.String), &calibration.Histogram); err != nil {
			return floorport.Setting{}, false, fmt.Errorf("read the histogram of the floor of %s: %w", v.ID, err)
		}
	}
	setting.Calibration = &calibration
	return setting, true, nil
}

// WriteCalibration keeps a measurement and leaves the manual floor as it is.
func (r *Repository) WriteCalibration(ctx context.Context, v domain.Vault, recipe string, c floorport.Calibration) error {
	histogram, err := json.Marshal(c.Histogram)
	if err != nil {
		return fmt.Errorf("write the floor of %s: %w", v.ID, err)
	}
	if _, err := sqlwrite.Exec(ctx, r.write, stmt.Get("save_calibration"),
		string(v.ID), recipe, c.Floor, c.Median, c.Spread, string(histogram),
		c.QueryCount, c.PassageCount, c.CalibratedTime.Unix()); err != nil {
		return fmt.Errorf("write the floor of %s: %w", v.ID, err)
	}
	return nil
}

// WriteManualFloor sets or clears the manual floor and leaves the measurement as
// it is.
func (r *Repository) WriteManualFloor(ctx context.Context, v domain.Vault, recipe string, isManual bool, floor float64) error {
	if _, err := sqlwrite.Exec(ctx, r.write, stmt.Get("save_manual"), string(v.ID), recipe, isManual, floor); err != nil {
		return fmt.Errorf("write the manual floor of %s: %w", v.ID, err)
	}
	return nil
}
