package api

// Goal is what a preset schedules toward.
type Goal string

const (
	GoalUnspecified Goal = ""
	GoalMinutesADay Goal = "minutes_a_day"
	GoalRetention   Goal = "retention"
	GoalByDate      Goal = "by_date"
)

// Rule is how the interval of a learned card is set.
type Rule string

const (
	RuleUnspecified Rule = ""
	RuleInterval    Rule = "interval"
	RuleRetention   Rule = "retention"
)

// BudgetUnit is what the daily budget counts.
type BudgetUnit string

const (
	BudgetUnitUnspecified BudgetUnit = ""
	BudgetUnitCards       BudgetUnit = "cards"
	BudgetUnitShows       BudgetUnit = "shows"
)

// StopReason is why a preset schedules no card for the day.
type StopReason string

const (
	StopReasonUnspecified StopReason = ""
	StopReasonNothing     StopReason = "nothing"
	StopReasonNoMinutes   StopReason = "no_minutes"
	StopReasonNoCards     StopReason = "no_cards"
	StopReasonNoDay       StopReason = "no_day"
	StopReasonPastDay     StopReason = "past_day"
	StopReasonNoLoad      StopReason = "no_load"
	StopReasonNoWeek      StopReason = "no_week"
)

// BudgetName is a limit that closes a day of review.
type BudgetName string

const (
	BudgetNameUnspecified BudgetName = ""
	BudgetNameMinutesADay BudgetName = "minutes_a_day"
	BudgetNameNewADay     BudgetName = "new_a_day"
	BudgetNameReviewsADay BudgetName = "reviews_a_day"
	BudgetNameByDate      BudgetName = "by_date"
	BudgetNameBacklog     BudgetName = "backlog"
	BudgetNamePaused      BudgetName = "paused"
)

// Settings is the scheduling settings of a preset. Load maps a weekday name to the cards due that day.
type Settings struct {
	Goal        Goal             `json:"goal"`
	ByDate      string           `json:"byDate"`
	MinutesADay int32            `json:"minutesADay"`
	NewADay     int32            `json:"newADay"`
	ReviewsADay int32            `json:"reviewsADay"`
	Retention   float64          `json:"retention"`
	HasEvenLoad bool             `json:"hasEvenLoad"`
	Counts      BudgetUnit       `json:"counts"`
	Backlog     int32            `json:"backlog"`
	Load        map[string]int32 `json:"load"`
	Learned     Rule             `json:"learned"`
	Interval    int32            `json:"interval"`
}

// Preset is a preset note with its settings and the reasons it schedules nothing.
type Preset struct {
	Path     string     `json:"path"`
	Title    string     `json:"title"`
	Settings *Settings  `json:"settings,omitempty"`
	Problems []string   `json:"problems"`
	Stops    StopReason `json:"stops"`
	StopsOn  StopReason `json:"stopsOn"`
}

// Curve is the projection of a preset over the range of its goal.
type Curve struct {
	Goal      Goal        `json:"goal"`
	Grid      []float64   `json:"grid"`
	Days      []string    `json:"days"`
	At        []*Point    `json:"at"`
	Now       *CurvePoint `json:"now,omitempty"`
	Suggested *CurvePoint `json:"suggested,omitempty"`
	Decks     int32       `json:"decks"`
	Cards     int32       `json:"cards"`
	Overdue   int32       `json:"overdue"`
	Unbegun   int32       `json:"unbegun"`
}

// Point is one sample of a projection curve. Learns is absent when the learned rule has no count.
type Point struct {
	Reviews  float64      `json:"reviews"`
	Minutes  float64      `json:"minutes"`
	Retained float64      `json:"retained"`
	Owed     int32        `json:"owed"`
	Through  float64      `json:"through"`
	IsEnough bool         `json:"isEnough"`
	Closed   []BudgetName `json:"closed"`
	Clears   int32        `json:"clears"`
	Backlog  []int32      `json:"backlog"`
	Learned  int32        `json:"learned"`
	Learns   *int32       `json:"learns,omitempty"`
	Short    int32        `json:"short"`
}

// CurvePoint is a marked position on a projection curve.
type CurvePoint struct {
	At    int32   `json:"at"`
	Value float64 `json:"value"`
	Day   string  `json:"day"`
}

// GetDeckPresetRequest is the request of the GetDeckPreset method.
type GetDeckPresetRequest struct {
	Deck string `json:"deck"`
}

// GetDeckPresetResponse is the response of the GetDeckPreset method.
type GetDeckPresetResponse struct {
	Preset *Preset         `json:"preset,omitempty"`
	Error  ErrorCode       `json:"error,omitempty"`
	At     *FileStat       `json:"at,omitempty"`
	Bounds *SettingsBounds `json:"bounds,omitempty"`
}

// ListPresetsRequest is the request of the ListPresets method.
type ListPresetsRequest struct {
}

// ListPresetsResponse is the response of the ListPresets method.
type ListPresetsResponse struct {
	Presets []*PresetSummary `json:"presets"`
}

// PresetSummary is the listing entry of a preset.
type PresetSummary struct {
	Path  string `json:"path"`
	Title string `json:"title"`
}

// CreatePresetRequest is the request of the CreatePreset method.
type CreatePresetRequest struct {
	Title string `json:"title"`
	Path  string `json:"path"`
}

// CreatePresetResponse is the response of the CreatePreset method.
type CreatePresetResponse struct {
	Path         string    `json:"path"`
	Error        ErrorCode `json:"error,omitempty"`
	IsUnlevelled bool      `json:"isUnlevelled"`
}

// ScheduleDeckRequest is the request of the ScheduleDeck method.
type ScheduleDeckRequest struct {
	Deck     string    `json:"deck"`
	Preset   string    `json:"preset"`
	Baseline *FileStat `json:"baseline,omitempty"`
}

// ScheduleDeckResponse is the response of the ScheduleDeck method.
type ScheduleDeckResponse struct {
	Error        ErrorCode `json:"error,omitempty"`
	At           *FileStat `json:"at,omitempty"`
	IsUnlevelled bool      `json:"isUnlevelled"`
}

// ReadPresetRequest is the request of the ReadPreset method.
type ReadPresetRequest struct {
	Path string `json:"path"`
}

// ReadPresetResponse is the response of the ReadPreset method.
type ReadPresetResponse struct {
	Preset *Preset         `json:"preset,omitempty"`
	Error  ErrorCode       `json:"error,omitempty"`
	At     *FileStat       `json:"at,omitempty"`
	Bounds *SettingsBounds `json:"bounds,omitempty"`
}

// SettingsBounds is the range each numeric setting of a preset may take.
type SettingsBounds struct {
	MinutesADay *Bounds `json:"minutesADay,omitempty"`
	NewADay     *Bounds `json:"newADay,omitempty"`
	ReviewsADay *Bounds `json:"reviewsADay,omitempty"`
	Retention   *Bounds `json:"retention,omitempty"`
	Backlog     *Bounds `json:"backlog,omitempty"`
	Interval    *Bounds `json:"interval,omitempty"`
	Load        *Bounds `json:"load,omitempty"`
}

// WritePresetRequest is the request of the WritePreset method.
type WritePresetRequest struct {
	Path     string    `json:"path"`
	Settings *Settings `json:"settings,omitempty"`
	Baseline *FileStat `json:"baseline,omitempty"`
}

// WritePresetResponse is the response of the WritePreset method.
type WritePresetResponse struct {
	Error        ErrorCode `json:"error,omitempty"`
	At           *FileStat `json:"at,omitempty"`
	IsUnlevelled bool      `json:"isUnlevelled"`
}

// ComputeCurveRequest is the request of the ComputeCurve method.
type ComputeCurveRequest struct {
	Path     string    `json:"path"`
	Settings *Settings `json:"settings,omitempty"`
}

// ComputeCurveResponse is the response of the ComputeCurve method.
type ComputeCurveResponse struct {
	Curve *Curve `json:"curve,omitempty"`
}
