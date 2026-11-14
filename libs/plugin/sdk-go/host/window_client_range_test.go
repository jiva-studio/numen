package host_test

import (
	"math"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
)

func TestClientRejectsSpanPast32BitsWithoutCallingHost(t *testing.T) {
	for name, span := range map[string]host.ByteSpan{
		"a start past the range":  {Start: 1<<32 + 5, Length: 4},
		"a length past the range": {Start: 5, Length: math.MaxInt32 + 1},
		"a start below the range": {Start: math.MinInt32 - 1, Length: 4},
	} {
		t.Run(name, func(t *testing.T) {
			h := &windowHost{}
			c := dialWindow(t, h)

			if err := c.FocusFile(t.Context(), "library/Birds.pdf", []host.ByteSpan{span}); err == nil {
				t.Error("the passage was sent")
			}
			if len(h.focusRequests) != 0 {
				t.Errorf("the host received %v", h.focusRequests)
			}
		})
	}
}
