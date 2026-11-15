package transcript_test

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/domain/transcript"
)

func TestCuesBecomeSegmentsWithTheirSpanInSeconds(t *testing.T) {
	segments, err := transcript.GetSegments([]transcript.Cue{
		{Text: "Good morning.", From: 1500, To: 4200},
		{Text: "   ", From: 4200, To: 5000},
		{Text: "The river is high.", From: 5000, To: 9100},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []domain.TextSegment{
		{Text: "Good morning.\n", Selector: `{"t":[1.5,4.2]}`},
		{Text: "The river is high.\n", Selector: `{"t":[5,9.1]}`},
	}
	if !slices.Equal(segments, want) {
		t.Fatalf("got %+v, want %+v", segments, want)
	}
}

func TestFormatTimestampWritesHoursMinutesSecondsAndThousandths(t *testing.T) {
	for ms, want := range map[int]string{
		0:        "00:00:00.000",
		9100:     "00:00:09.100",
		60000:    "00:01:00.000",
		620000:   "00:10:20.000",
		3600000:  "01:00:00.000",
		3723456:  "01:02:03.456",
		32400000: "09:00:00.000",
		36000000: "10:00:00.000",
		86399999: "23:59:59.999",
		-5:       "00:00:00.000",
	} {
		if got := transcript.FormatTimestamp(ms); got != want {
			t.Errorf("%d ms is written %q, want %q", ms, got, want)
		}
	}
}
