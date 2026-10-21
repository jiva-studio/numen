package service

import (
	"strings"
	"testing"
)

// The digest stands before the first line of the question, and the lines are
// asked about as they always were.
func TestTheDigestExistsBeforeTheFirstLine(t *testing.T) {
	batch := Batch{Number: 0, IsJoinable: true, Context: "The speech opens: Ganaka spoke.", Lines: []Line{
		{Number: 0, Text: "Ganaka spoke"},
		{Number: 1, Text: "to the assembly."},
	}}

	request := buildPrompt(batch)
	if !strings.HasPrefix(request, "The speech opens: Ganaka spoke.\n\n") {
		t.Fatalf("the question is %q", request)
	}
	if !strings.HasSuffix(request, Opens+"0"+Closes+"Ganaka spoke "+
		Opens+"1"+Closes+"to the assembly.") {
		t.Errorf("the question is %q", request)
	}
}

// A batch nothing was said about is the lines alone.
func TestABatchWithNoDigestIsTheLinesAlone(t *testing.T) {
	batch := Batch{Number: 0, Lines: []Line{{Number: 0, Text: "Ganaka spoke"}}}
	if request := buildPrompt(batch); request != Opens+"0"+Closes+"Ganaka spoke" {
		t.Errorf("the question is %q", request)
	}
}
