// Package api declares the JSON messages of the relevance service the plugin's backend serves to its
// own frontend. Keys are lowerCamelCase, and a failure is a server.APIError, not a message field.
package api

// Service is the service name the manifest declares.
const Service = "md.numen.tools.search.relevance"

// Method names of the service.
const (
	MethodGetFloor       = "GetFloor"
	MethodCalibrateFloor = "CalibrateFloor"
	MethodSetFloor       = "SetFloor"
	MethodProbeFloor     = "ProbeFloor"
)

// Source is where the floor in force comes from.
type Source string

const (
	// SourceManual: the person set the floor for the model in use.
	SourceManual Source = "manual"
	// SourceCalibrated: measured on the vault with the model in use.
	SourceCalibrated Source = "calibrated"
	// SourceModel: the starting floor the model's plugin reports.
	SourceModel Source = "model"
	// SourceDefault: no other source answered.
	SourceDefault Source = "default"
)

// Outcome is how a calibration ended when it did not fail.
type Outcome string

const (
	// OutcomeCalibrated: the floor was measured and kept.
	OutcomeCalibrated Outcome = "calibrated"
	// OutcomeTooFew: the vault holds too few embedded notes to measure.
	OutcomeTooFew Outcome = "too-few"
)

// GetFloorRequest requests the floor in force for the vault the window shows.
type GetFloorRequest struct{}

// Calibration is what one calibration measured.
type Calibration struct {
	Median       float64 `json:"median"`
	Spread       float64 `json:"spread"`
	Histogram    []int   `json:"histogram"`
	QueryCount   int     `json:"queryCount"`
	ChunkCount int     `json:"passageCount"`
	CalibratedAt int64   `json:"calibratedAt"`
}

// GetFloorResponse is the floor in force for a vault under the model in use.
type GetFloorResponse struct {
	Floor          float64      `json:"floor"`
	Source         Source       `json:"source"`
	Model          string       `json:"model"`
	AutomaticFloor float64      `json:"automaticFloor"`
	IsManual       bool         `json:"isManual"`
	ManualFloor    float64      `json:"manualFloor"`
	Calibration    *Calibration `json:"calibration,omitempty"`
}

// CalibrateFloorRequest measures the vault the window shows.
type CalibrateFloorRequest struct{}

// CalibrateFloorResponse is how the measurement ended and the floor in force after it.
type CalibrateFloorResponse struct {
	Outcome Outcome          `json:"outcome"`
	State   GetFloorResponse `json:"state"`
}

// SetFloorRequest chooses between the measured floor and one the person sets.
type SetFloorRequest struct {
	IsManual bool    `json:"isManual"`
	Floor    float64 `json:"floor"`
}

// SetFloorResponse is the floor in force after the choice.
type SetFloorResponse struct {
	State GetFloorResponse `json:"state"`
}

// ProbeFloorRequest asks how near the best passage of the vault stands to a query.
type ProbeFloorRequest struct {
	Query string `json:"query"`
}

// ProbeFloorResponse is what a probe measured.
type ProbeFloorResponse struct {
	BestSimilarity float64 `json:"bestSimilarity"`
	Floor          float64 `json:"floor"`
	DoesPass       bool    `json:"doesPass"`
}
