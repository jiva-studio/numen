// Package performance gives a plugin the performance profile of the person: how much of the
// machine background work may use. The host passes the profile in every Configure call under
// ConfigKey, and again when the person changes it; the plugin sizes its threads and batches from
// it and lowers its own priority with SetNice.
package performance

// ConfigKey is the entry of the Configure values that holds the profile.
const ConfigKey = "host.performanceProfile"

// Profile is a workload preset of the settings page.
type Profile string

const (
	// Eco keeps background work unobtrusive: one thread, the lowest priority.
	Eco Profile = "eco"
	// Balanced is the default: a quarter of the cores, at most four, at a lowered priority.
	Balanced Profile = "balanced"
	// Maximum gives background work half the cores, at most eight, at the priority of the plugin.
	Maximum Profile = "maximum"
)

// Read returns the profile of a Configure call. A call that carries none, or a name this package
// does not know, gives Balanced.
func Read(values map[string]string) Profile {
	switch Profile(values[ConfigKey]) {
	case Eco:
		return Eco
	case Maximum:
		return Maximum
	default:
		return Balanced
	}
}

// Limits is what a profile lets one plugin occupy on a machine.
//
//numen:api
type Limits struct {
	// Threads is the most CPU-bound threads at once; at least 1.
	Threads int
	// Nice is the niceness of the process: 0 is the default priority, 19 the lowest.
	Nice int
}

const (
	balancedCoreDivisor = 4
	balancedMaxThreads  = 4
	maximumCoreDivisor  = 2
	maximumMaxThreads   = 8

	ecoNice      = 19
	balancedNice = 10
	maximumNice  = 0
)

// Resolve returns the limits of a profile on a machine with cpus logical CPUs. Threads never
// exceed the cap of the profile and are at least 1.
func Resolve(profile Profile, cpus int) Limits {
	switch profile {
	case Eco:
		return Limits{Threads: 1, Nice: ecoNice}
	case Maximum:
		return Limits{Threads: clamp(cpus/maximumCoreDivisor, maximumMaxThreads), Nice: maximumNice}
	default:
		return Limits{Threads: clamp(cpus/balancedCoreDivisor, balancedMaxThreads), Nice: balancedNice}
	}
}

func clamp(threads, most int) int {
	return max(1, min(threads, most))
}
