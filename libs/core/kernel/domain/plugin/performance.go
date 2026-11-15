package plugin

import "strconv"

// The entries of the Configure values that hold the performance limits; plugin-sdk-go reads them in
// performance.Read.
const (
	ConfigPerformanceProfile = "host.performanceProfile"
	ConfigPerformanceThreads = "host.performanceThreads"
	ConfigPerformanceNice    = "host.performanceNice"
)

// Performance is the performance profile the host resolved on the machine: its name, the most
// CPU-bound threads one plugin may run at once and the niceness of the plugin process. The zero value
// is no profile yet.
type Performance struct {
	Profile string
	Threads int
	Nice    int
}

// GetConfigValues returns the Configure entries of the limits, none for the zero value.
func (p Performance) GetConfigValues() map[string]string {
	if p.Profile == "" {
		return map[string]string{}
	}
	return map[string]string{
		ConfigPerformanceProfile: p.Profile,
		ConfigPerformanceThreads: strconv.Itoa(p.Threads),
		ConfigPerformanceNice:    strconv.Itoa(p.Nice),
	}
}
