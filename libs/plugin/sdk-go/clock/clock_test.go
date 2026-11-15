package clock_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
)

func TestSystemReadsMachineTime(t *testing.T) {
	var system clock.System
	first := system.Now()
	second := system.Now()
	if second.Before(first) {
		t.Fatalf("second reading %v precedes first %v", second, first)
	}
	if first.Year() < 2025 {
		t.Fatalf("reading %v is not machine time", first)
	}
}
