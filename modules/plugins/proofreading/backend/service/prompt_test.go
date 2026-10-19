package service

import (
	"strings"
	"testing"
)

// The digest stands before the first line of the question, and the lines are
// asked about as they always were.
func TestTheDigestStandsBeforeTheFirstLine(t *testing.T) {
	batch := Batch{Number: 0, IsJoinable: true, Context: "The speech opens: Ganaka spoke.", Lines: []Line{
		{Number: 0, Text: "Ganaka spoke"},
		{Number: 1, Text: "to the assembly."},
	}}

	asked := buildPrompt(batch)
	if !strings.HasPrefix(asked, "The speech opens: Ganaka spoke.\n\n") {
		t.Fatalf("the question is %q", asked)
	}
	if !strings.HasSuffix(asked, Opens+"0"+Closes+"Ganaka spoke "+
		Opens+"1"+Closes+"to the assembly.") {
		t.Errorf("the question is %q", asked)
	}
}

// A batch nothing was said about is the lines alone.
func TestABatchWithNoDigestIsTheLinesAlone(t *testing.T) {
	batch := Batch{Number: 0, Lines: []Line{{Number: 0, Text: "Ganaka spoke"}}}
	if asked := buildPrompt(batch); asked != Opens+"0"+Closes+"Ganaka spoke" {
		t.Errorf("the question is %q", asked)
	}
}
