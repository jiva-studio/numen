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
	// Median and Spread describe the similarities between a query and passages that are not about it.
	Median float64 `json:"median"`
	Spread float64 `json:"spread"`
	// Histogram counts those similarities in equal bins from zero to one.
	Histogram []int `json:"histogram"`
	// QueryCount and PassageCount are how many queries were measured against how many passages.
	QueryCount   int `json:"queryCount"`
	PassageCount int `json:"passageCount"`
	// CalibratedAt is when it was measured, in seconds since the epoch.
	CalibratedAt int64 `json:"calibratedAt"`
}

// GetFloorResponse is the floor in force for a vault under the model in use.
type GetFloorResponse struct {
	// Floor is the floor in force, in cosine similarity, and Source is where it comes from.
	Floor  float64 `json:"floor"`
	Source Source  `json:"source"`
	// Model is the name of the model the floor is for, empty where there is none.
	Model string `json:"model"`
	// AutomaticFloor is the floor in force when the person has set none.
	AutomaticFloor float64 `json:"automaticFloor"`
	// IsManual is whether the person set the floor, and ManualFloor the value they set.
	IsManual    bool    `json:"isManual"`
	ManualFloor float64 `json:"manualFloor"`
	// Calibration is absent before the first measurement.
	Calibration *Calibration `json:"calibration,omitempty"`
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
	// IsManual is whether Floor is in force. False returns to the automatic floor.
	IsManual bool `json:"isManual"`
	// Floor is the floor, in cosine similarity above zero and at most one.
	Floor float64 `json:"floor"`
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
	// BestSimilarity is the similarity of the nearest passage, zero where the vault has none.
	BestSimilarity float64 `json:"bestSimilarity"`
	// Floor is the floor in force, and DoesPass whether the nearest passage reaches it.
	Floor    float64 `json:"floor"`
	DoesPass bool    `json:"doesPass"`
}
