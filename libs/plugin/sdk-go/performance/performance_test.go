package performance

import "testing"

func TestReadTakesTheLimitsTheHostResolved(t *testing.T) {
	got := Read(map[string]string{ProfileKey: "high", ThreadsKey: "12", NiceKey: "5"})

	if want := (Limits{Profile: High, Threads: 12, Nice: 5}); got != want {
		t.Errorf("Read = %+v, want %+v", got, want)
	}
}

func TestReadKnowsEveryProfile(t *testing.T) {
	for _, want := range []Profile{Eco, Low, Balanced, High, Maximum} {
		if got := Read(map[string]string{ProfileKey: string(want)}).Profile; got != want {
			t.Errorf("Read(%q) = %s", want, got)
		}
	}
}

func TestReadWithoutValuesGivesOneThreadAtTheBalancedNiceness(t *testing.T) {
	if got, want := Read(nil), (Limits{Profile: Balanced, Threads: 1, Nice: 10}); got != want {
		t.Errorf("Read(nil) = %+v, want %+v", got, want)
	}
}

func TestReadReplacesValuesItCannotUse(t *testing.T) {
	got := Read(map[string]string{ProfileKey: "turbo", ThreadsKey: "0", NiceKey: "low"})

	if want := (Limits{Profile: Balanced, Threads: 1, Nice: 10}); got != want {
		t.Errorf("Read = %+v, want %+v", got, want)
	}
	if got := Read(map[string]string{ThreadsKey: "-3"}).Threads; got != 1 {
		t.Errorf("threads = %d, want 1", got)
	}
}
