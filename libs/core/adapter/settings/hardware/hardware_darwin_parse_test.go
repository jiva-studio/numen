//go:build darwin

package hardware

import "testing"

func TestPickPerformanceLevel_ChoosesTheHighestRankedName(t *testing.T) {
	tests := []struct {
		name   string
		levels []perfLevel
		want   string
		cores  int
	}{
		{"an older chip names its levels Performance and Efficiency", []perfLevel{{"Performance", 4, 4}, {"Efficiency", 6, 6}}, "Performance", 4},
		{"a newer chip puts Super ahead of Performance", []perfLevel{{"Super", 6, 6}, {"Performance", 12, 12}, {"Efficiency", 4, 4}}, "Super", 6},
		{"the order of the levels does not decide", []perfLevel{{"Efficiency", 6, 6}, {"Performance", 4, 4}}, "Performance", 4},
		{"unnamed levels fall to the first", []perfLevel{{"", 8, 8}, {"", 4, 4}}, "", 8},
		{"names are compared without case or padding", []perfLevel{{"efficiency\n", 6, 6}, {" PERFORMANCE", 4, 4}}, " PERFORMANCE", 4},
		{"a level without cores is skipped", []perfLevel{{"Super", 0, 0}, {"Performance", 4, 4}}, "Performance", 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := pickPerformanceLevel(tt.levels)
			if !ok || got.Name != tt.want || got.PhysicalCPUs != tt.cores {
				t.Errorf("got %+v, %v; want %q with %d cores", got, ok, tt.want, tt.cores)
			}
		})
	}
}

func TestPickPerformanceLevel_ReportsNothingWithoutLevels(t *testing.T) {
	if _, ok := pickPerformanceLevel(nil); ok {
		t.Error("pickPerformanceLevel found a level in an empty list")
	}
}
