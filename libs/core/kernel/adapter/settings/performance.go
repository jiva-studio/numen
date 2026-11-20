package settings

// Performance is the settings section for workload and resource profile.
type Performance struct {
	Provider string `json:"provider"`
	Profile  string `json:"profile"`
}
