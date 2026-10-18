package domain_test

import (
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

func TestASpeechWindowAnswersForTheLinesFromItsFirstUpToItsLimit(t *testing.T) {
	lines := []domain.ProofreadLine{{Number: 0}, {Number: 2}, {Number: 3}, {Number: 5}, {Number: 6}}

	for name, one := range map[string]struct {
		window domain.SpeechWindow
		want   []int
	}{
		"every line":                 {domain.SpeechWindow{}, []int{0, 2, 3, 5, 6}},
		"from a line":                {domain.SpeechWindow{From: 3}, []int{3, 5, 6}},
		"from a number with no line": {domain.SpeechWindow{From: 4}, []int{5, 6}},
		"up to the limit":            {domain.SpeechWindow{From: 2, Limit: 2}, []int{2, 3}},
		"a limit past the end":       {domain.SpeechWindow{From: 5, Limit: 9}, []int{5, 6}},
		"past the last line":         {domain.SpeechWindow{From: 7}, nil},
	} {
		t.Run(name, func(t *testing.T) {
			var got []int
			for _, line := range one.window.GetLines(lines) {
				got = append(got, line.Number)
			}
			if !reflect.DeepEqual(got, one.want) {
				t.Errorf("got lines %v, want %v", got, one.want)
			}
		})
	}
}

func TestACorrectionOfALineNotAskedAboutIsFound(t *testing.T) {
	asked := []domain.ProofreadLine{{Number: 2}, {Number: 3}}

	if one, found := domain.FindUnaskedCorrection(asked, []domain.ProofreadCorrection{
		{Number: 2, Text: "asked"}, {Number: 4, Text: "unasked"}, {Number: 1, Text: "unasked too"},
	}); !found || one.Number != 4 {
		t.Errorf("got %+v, %v, want the correction of line 4", one, found)
	}
	if one, found := domain.FindUnaskedCorrection(asked, []domain.ProofreadCorrection{
		{Number: 2, Text: "asked"}, {Number: 3, Last: 5, Text: "a run past the lines asked"},
	}); !found || one.Number != 3 {
		t.Errorf("got %+v, %v, want the run from line 3", one, found)
	}
	if one, found := domain.FindUnaskedCorrection(asked, []domain.ProofreadCorrection{
		{Number: 3, Text: "asked"}, {Number: 2, Last: 3, Text: "asked as a run"},
	}); found {
		t.Errorf("found %+v among corrections of lines asked about", one)
	}
}
