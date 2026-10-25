package app

import (
	"math"
	"testing"
)

type fakeLimit struct {
	current int64
	set     []int64
}

func (f *fakeLimit) setLimit(n int64) int64 {
	previous := f.current
	if n >= 0 {
		f.current = n
		f.set = append(f.set, n)
	}
	return previous
}

func noEnv(string) (string, bool) { return "", false }

func TestApplyMemoryLimit_SetsTheLimitInBytes(t *testing.T) {
	f := &fakeLimit{current: math.MaxInt64}
	applyMemoryLimit(1024, f.setLimit, noEnv)
	if len(f.set) != 1 || f.set[0] != 1024<<20 {
		t.Fatalf("set = %v, want one call of %d", f.set, int64(1024<<20))
	}
}

func TestApplyMemoryLimit_KeepsAnExplicitEnvironmentLimit(t *testing.T) {
	f := &fakeLimit{current: math.MaxInt64}
	applyMemoryLimit(1024, f.setLimit, func(string) (string, bool) { return "2GiB", true })
	if len(f.set) != 0 {
		t.Fatalf("set = %v, want none", f.set)
	}
}

func TestApplyMemoryLimit_AppliesWhenTheEnvironmentLimitIsEmpty(t *testing.T) {
	f := &fakeLimit{current: math.MaxInt64}
	applyMemoryLimit(1024, f.setLimit, func(string) (string, bool) { return "", true })
	if len(f.set) != 1 {
		t.Fatalf("set = %v, want one call", f.set)
	}
}

func TestApplyMemoryLimit_DoesNothingWhenMemoryIsUnknown(t *testing.T) {
	f := &fakeLimit{current: math.MaxInt64}
	applyMemoryLimit(0, f.setLimit, noEnv)
	if len(f.set) != 0 {
		t.Fatalf("set = %v, want none", f.set)
	}
}

func TestApplyMemoryLimit_KeepsALowerExistingLimit(t *testing.T) {
	f := &fakeLimit{current: 256 << 20}
	applyMemoryLimit(1024, f.setLimit, noEnv)
	if len(f.set) != 0 || f.current != 256<<20 {
		t.Fatalf("set = %v, current = %d, want the existing limit kept", f.set, f.current)
	}
}
