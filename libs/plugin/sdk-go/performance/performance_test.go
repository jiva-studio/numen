package performance

import "testing"

func TestResolveFollowsTheProfileAndTheCores(t *testing.T) {
	table := []struct {
		profile Profile
		cpus    int
		threads int
		nice    int
	}{
		{Eco, 2, 1, 19}, {Eco, 4, 1, 19}, {Eco, 8, 1, 19}, {Eco, 16, 1, 19},
		{Balanced, 2, 1, 10}, {Balanced, 4, 1, 10}, {Balanced, 8, 2, 10}, {Balanced, 16, 4, 10},
		{Maximum, 2, 1, 0}, {Maximum, 4, 2, 0}, {Maximum, 8, 4, 0}, {Maximum, 16, 8, 0},
		{Balanced, 64, 4, 10}, {Maximum, 64, 8, 0}, {Balanced, 1, 1, 10}, {Maximum, 0, 1, 0},
	}
	for _, c := range table {
		got := Resolve(c.profile, c.cpus)
		if got.Threads != c.threads || got.Nice != c.nice {
			t.Errorf("Resolve(%s, %d) = %+v, want threads %d nice %d", c.profile, c.cpus, got, c.threads, c.nice)
		}
	}
}

func TestDefaultProfileUsesAtMostQuarterOfCores(t *testing.T) {
	for _, cpus := range []int{4, 8, 12, 16, 32, 64} {
		if got := Resolve(Read(nil), cpus).Threads; got*4 > cpus {
			t.Errorf("the default profile takes %d of %d cores", got, cpus)
		}
	}
}

func TestReadFallsBackToBalanced(t *testing.T) {
	for value, want := range map[string]Profile{
		"eco": Eco, "maximum": Maximum, "balanced": Balanced, "": Balanced, "turbo": Balanced,
	} {
		if got := Read(map[string]string{ConfigKey: value}); got != want {
			t.Errorf("Read(%q) = %s, want %s", value, got, want)
		}
	}
}
