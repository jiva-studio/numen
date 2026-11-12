package host_test

import (
	"math"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
)

func TestClientRejectsPassagePast32BitsWithoutCallingHost(t *testing.T) {
	for name, passage := range map[string]host.PassageSpan{
		"a start past the range":  {Start: 1<<32 + 5, Length: 4},
		"a length past the range": {Start: 5, Length: math.MaxInt32 + 1},
		"a start below the range": {Start: math.MinInt32 - 1, Length: 4},
	} {
		t.Run(name, func(t *testing.T) {
			h := &windowHost{}
			c := dialWindow(t, h)

			if err := c.FocusPassages(t.Context(), "library/Birds.pdf", []host.PassageSpan{passage}); err == nil {
				t.Error("the passage was sent")
			}
			if len(h.passages) != 0 {
				t.Errorf("the host received %v", h.passages)
			}
		})
	}
}
