package flashcards_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/flashcards"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/review"
)

// A caller draws a fixed stretch of time, so it asks for one. Both ends are
// inside it, an end left empty is unbounded at that end, and the streak and the
// count are over the whole vault either way.
func TestGetReviewDaysKeepsTheDaysInsideTheStretch(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}
	now := time.Now()
	yesterday, twoDaysAgo := now.AddDate(0, 0, -1), now.AddDate(0, 0, -2)
	for _, at := range []time.Time{twoDaysAgo, yesterday, now} {
		if _, err := s.run(t, at).Answer(t.Context(), on, review.Good, 0); err != nil {
			t.Fatal(err)
		}
	}
	older, middle, newest := today.GetName(twoDaysAgo), today.GetName(yesterday), today.GetName(now)
	get := flashcards.NewGetReviewDays(s.counted)

	for name, one := range map[string]struct {
		from, to string
		want     []string
	}{
		"both ends":   {middle, middle, []string{middle}},
		"from alone":  {middle, "", []string{middle, newest}},
		"to alone":    {"", middle, []string{older, middle}},
		"neither end": {"", "", []string{older, middle, newest}},
		"nothing in":  {"2000-01-01", "2000-12-31", nil},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := get.Execute(t.Context(), s.vault, one.from, one.to)
			if err != nil {
				t.Fatal(err)
			}
			if len(got.Days) != len(one.want) {
				t.Fatalf("the days %v came back, want %v", got.Days, one.want)
			}
			for _, day := range one.want {
				if _, target := got.Days[day]; !target {
					t.Errorf("%s is missing from %v", day, got.Days)
				}
			}
			if got.Answered != 3 {
				t.Errorf("the vault holds %d answers, want 3", got.Answered)
			}
		})
	}
}
