package searchrun_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchrun"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func getEpoch() time.Time { return time.Date(2026, time.March, 2, 12, 0, 0, 0, time.UTC) }

func TestSelectExpiredDropsSearchesNotReadForTheTTL(t *testing.T) {
	records := []searchrun.Record{
		{ID: "old", AccessedAt: getEpoch()},
		{ID: "recent", AccessedAt: getEpoch().Add(time.Minute)},
		{ID: "running", IsRunning: true, AccessedAt: getEpoch()},
	}
	got := searchrun.SelectExpired(records, getEpoch().Add(10*time.Minute), searchrun.DefaultPolicy(), 0)
	if !slices.Equal(got, []searchrun.ID{"old", "running"}) {
		t.Fatalf("expired %v, want [old running]", got)
	}
}

func TestSelectExpiredMakesRoomByDroppingTheStoppedSearchReadLeastRecently(t *testing.T) {
	var records []searchrun.Record
	for i := range 20 {
		records = append(records, searchrun.Record{
			ID: searchrun.ID(rune('a' + i)), AccessedAt: getEpoch().Add(time.Duration(20-i) * time.Second), IsRunning: i == 19,
		})
	}
	got := searchrun.SelectExpired(records, getEpoch().Add(time.Minute), searchrun.DefaultPolicy(), 1)
	if !slices.Equal(got, []searchrun.ID{"s"}) {
		t.Fatalf("expired %v, want [s]", got)
	}
}

func TestSelectExpiredKeepsRunningSearchesWhenRoomIsShort(t *testing.T) {
	var records []searchrun.Record
	for i := range 20 {
		records = append(records, searchrun.Record{ID: searchrun.ID(rune('a' + i)), AccessedAt: getEpoch(), IsRunning: true})
	}
	if got := searchrun.SelectExpired(records, getEpoch().Add(time.Minute), searchrun.DefaultPolicy(), 1); len(got) != 0 {
		t.Fatalf("expired %v, want none", got)
	}
}

func TestIsSameSearchComparesOwnerVaultQueryAndBackground(t *testing.T) {
	base := searchrun.Start{Owner: "p", Vault: domain.Vault{ID: "v"}, Query: "words(\"tide\")"}
	if !searchrun.IsSameSearch(base, searchrun.Start{Owner: "p", Vault: domain.Vault{ID: "v"}, Query: "  words(\"tide\")\n"}) {
		t.Error("the same query with surrounding space is another search")
	}
	changes := map[string]func(*searchrun.Start){
		"owner":      func(s *searchrun.Start) { s.Owner = "q" },
		"vault":      func(s *searchrun.Start) { s.Vault.ID = "w" },
		"query":      func(s *searchrun.Start) { s.Query = "words(\"sea\")" },
		"background": func(s *searchrun.Start) { s.IsBackground = true },
	}
	for name, change := range changes {
		other := base
		change(&other)
		if searchrun.IsSameSearch(base, other) {
			t.Errorf("another %s is the same search", name)
		}
	}
}

func TestParsePageTokenReadsTheOffsetOfItsSearchOnly(t *testing.T) {
	token := searchrun.EncodePageToken("s1", 20)
	if offset, err := searchrun.ParsePageToken(token, "s1", 45); err != nil || offset != 20 {
		t.Fatalf("offset %d (%v), want 20", offset, err)
	}
	for name, read := range map[string]func() (int, error){
		"another search":   func() (int, error) { return searchrun.ParsePageToken(token, "s2", 45) },
		"beyond the total": func() (int, error) { return searchrun.ParsePageToken(token, "s1", 10) },
		"malformed":        func() (int, error) { return searchrun.ParsePageToken("%%%", "s1", 45) },
	} {
		if _, err := read(); !errors.Is(err, searchrun.ErrInvalidPageToken) {
			t.Errorf("%s: error %v, want ErrInvalidPageToken", name, err)
		}
	}
}

func TestReadProblemsKeepsThePositionsOfAParseError(t *testing.T) {
	err := &searchquery.ParseError{Problems: []searchquery.Problem{
		{Position: searchquery.Position{Line: 2, Column: 5}, Message: "unknown function mening", Suggestion: "meaning"},
	}}
	got := searchrun.ReadProblems(err)
	want := []searchrun.Problem{{Line: 2, Column: 5, Message: "unknown function mening", Suggestion: "meaning"}}
	if !slices.Equal(got, want) {
		t.Fatalf("problems %v, want %v", got, want)
	}
	other := searchrun.ReadProblems(errors.New("no reranker"))
	if !slices.Equal(other, []searchrun.Problem{{Message: "no reranker"}}) {
		t.Fatalf("problems %v", other)
	}
}
