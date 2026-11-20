// Package performance gives a plugin the limits of the performance profile of the person: how much
// of the machine background work may use. The host resolves the profile on the machine and passes
// the limits in every Configure call, and again when the person changes it; the plugin sizes its
// threads from them and the host sets the priority of each external process.
package performance

import "strconv"

// The entries of the Configure values that hold the limits.
const (
	// ProviderKey holds the requested hardware provider for local models.
	ProviderKey = "host.computeProvider"
	// ProfileKey holds the name of the profile.
	ProfileKey = "host.performanceProfile"
	// ThreadsKey holds the most CPU-bound threads the plugin may run at once.
	ThreadsKey = "host.performanceThreads"
	// NiceKey holds the niceness the plugin runs at.
	NiceKey = "host.performanceNice"
)

// Compute is the host policy for opening a model session. A zero thread limit leaves it unconstrained.
type Compute struct {
	Provider    string
	ThreadLimit int
}

// Profile is a workload preset of the settings page.
type Profile string

const (
	// Eco keeps background work unobtrusive: one thread, the lowest priority.
	Eco Profile = "eco"
	// Low gives background work a quarter of the performance threads.
	Low Profile = "low"
	// Balanced is the default: half of the performance threads.
	Balanced Profile = "balanced"
	// High gives background work three quarters of the performance threads.
	High Profile = "high"
	// Maximum gives background work every performance thread at the priority of the plugin.
	Maximum Profile = "maximum"
)

// Limits is what the profile lets one plugin occupy on the machine.
//
//numen:api
type Limits struct {
	// Profile is the preset the limits come from.
	Profile Profile
	// Threads is the most CPU-bound threads at once; at least 1.
	Threads int
	// Nice is the niceness of the process: 0 is the default priority, 19 the lowest.
	Nice int
}

const (
	fallbackThreads = 1
	fallbackNice    = 10
)

// Read returns the limits of a Configure call. A profile that is missing or unknown is Balanced, a
// thread count that is missing or below 1 is 1, and a missing niceness is the niceness of Balanced.
func Read(values map[string]string) Limits {
	return Limits{Profile: readProfile(values[ProfileKey]), Threads: readThreads(values[ThreadsKey]), Nice: readNice(values[NiceKey])}
}

func readProfile(value string) Profile {
	switch profile := Profile(value); profile {
	case Eco, Low, High, Maximum:
		return profile
	default:
		return Balanced
	}
}

func readThreads(value string) int {
	threads, err := strconv.Atoi(value)
	if err != nil || threads < 1 {
		return fallbackThreads
	}
	return threads
}

func readNice(value string) int {
	nice, err := strconv.Atoi(value)
	if err != nil {
		return fallbackNice
	}
	return nice
}
