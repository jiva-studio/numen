package transcript_test

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/transcript"
)

func TestCuesBecomeSegmentsWithTheirSpanInSeconds(t *testing.T) {
	segments := transcript.GetSegments([]transcript.Cue{
		{Text: "Good morning.", From: 1500, To: 4200},
		{Text: "   ", From: 4200, To: 5000},
		{Text: "The river is high.", From: 5000, To: 9100},
	})
	want := []domain.TextSegment{
		{Text: "Good morning.\n", Selector: `{"t":[1.5,4.2]}`},
		{Text: "The river is high.\n", Selector: `{"t":[5,9.1]}`},
	}
	if !slices.Equal(segments, want) {
		t.Fatalf("got %+v, want %+v", segments, want)
	}
}

func TestSegmentsReadBackAsTheCuesTheyWere(t *testing.T) {
	cues := []transcript.Cue{{Text: "Good morning.", From: 1500, To: 4200}, {Text: "Again.", From: 62500, To: 63001}}
	if got := transcript.GetCues(transcript.GetSegments(cues)); !slices.Equal(got, cues) {
		t.Fatalf("got %+v, want %+v", got, cues)
	}
}

func TestFormatTimestampWritesHoursMinutesSecondsAndThousandths(t *testing.T) {
	for ms, want := range map[int]string{
		0:        "00:00:00.000",
		9100:     "00:00:09.100",
		3723456:  "01:02:03.456",
		86399999: "23:59:59.999",
		-5:       "00:00:00.000",
	} {
		if got := transcript.FormatTimestamp(ms); got != want {
			t.Errorf("%d ms is written %q, want %q", ms, got, want)
		}
	}
}
