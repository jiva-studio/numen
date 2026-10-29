package review_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

// makeDueByDay is a table of the days of review, loaded with as many review cards
// on each of these many days past an instant.
func makeDueByDay(d review.Day, pos time.Time, on map[int]int) *review.DueByDay {
	out := review.NewDueByDay(d)
	for day, cards := range on {
		for range cards {
			out.Add(pos.AddDate(0, 0, day))
		}
	}
	return out
}

// A card goes on the heaviest day of the window around the interval it was sent
// away for: the day whose share of the load stands highest over what already
// falls on it. A day at none of the load weighs nothing and takes no card.
func TestACardGoesOnTheHeaviestDayOfItsWindow(t *testing.T) {
	day := review.Day{Starts: review.DayStartTime, In: time.UTC}
	pos := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	if pos.Weekday() != time.Sunday {
		t.Fatalf("%s is a %s", pos.Format(review.DayLayout), pos.Weekday())
	}
	// Seven days away, so the window runs from five days off to nine: the
	// Friday, the Saturday, the Sunday it was sent to, the Monday and the
	// Tuesday.
	due := pos.AddDate(0, 0, 7)
	p := review.Preset{
		Goal: review.GoalMinutes, MinutesADay: 20, IsEvenLoad: true,
		Load: map[time.Weekday]int{time.Friday: 50, time.Saturday: 0},
	}

	// The Tuesday carries the whole load and nothing yet, which is more than
	// the half day on the Friday, the three already on the Sunday and the one
	// on the Monday.
	on := makeDueByDay(day, pos, map[int]int{5: 0, 6: 0, 7: 3, 8: 1, 9: 0})
	if got, want := p.AssignDueDay(on, pos, due), pos.AddDate(0, 0, 9); !got.Equal(want) {
		t.Errorf("the card was put on %s, want %s",
			got.Format(time.RFC3339), want.Format(time.RFC3339))
	}
	// And the day it landed on is counted against that day.
	if got := on.CountOn(pos.AddDate(0, 0, 9)); got != 1 {
		t.Errorf("the day the card landed on carries %d review cards, want 1", got)
	}

	// The day the scheduler named weighs as much as the Monday beside it, and
	// keeps the card.
	tied := makeDueByDay(day, pos, map[int]int{5: 3, 6: 0, 7: 0, 8: 0, 9: 1})
	if got := p.AssignDueDay(tied, pos, due); !got.Equal(due) {
		t.Errorf("a card tied between its own day and the next was put on %s, want %s",
			got.Format(time.RFC3339), due.Format(time.RFC3339))
	}
}

// A table of the days of review says how many review cards fall on the day
// holding an instant.
func TestHowLoadedADayOfReviewIs(t *testing.T) {
	day := review.Day{Starts: review.DayStartTime, In: time.UTC}
	pos := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	on := makeDueByDay(day, pos, map[int]int{0: 2, 1: 1})

	if got := on.CountOn(pos.Add(6 * time.Hour)); got != 2 {
		t.Errorf("the day holding the two review cards carries %d, want 2", got)
	}
	// A day of review runs to the hour it opens at, so an instant before that
	// hour belongs to the day before it.
	if got := on.CountOn(pos.AddDate(0, 0, 1).Add(-8 * time.Hour)); got != 2 {
		t.Errorf("the small hours carry %d review cards, want the 2 of the day before", got)
	}
	if got := on.CountOn(pos.AddDate(0, 0, 2)); got != 0 {
		t.Errorf("a day nothing falls on carries %d review cards", got)
	}
}

// A session works out the day from a local now, and a replay from the stamp its
// log carries, which is read back in UTC. One card and one answer, over the
// night the clock goes back: both land on one moment, and not merely on one
// date.
func TestTheSessionAndTheReplayLandOnOneMomentAcrossAClockChange(t *testing.T) {
	in, err := time.LoadLocation("Europe/Warsaw")
	if err != nil {
		t.Skipf("this machine holds no zone whose clock changes: %v", err)
	}
	day := review.Day{Starts: review.DayStartTime, In: in}
	delta := review.NewFSRS()
	p := review.Defaults()

	reviewCard := review.ReviewCardID{Card: "k7m2xq9fzp", Template: "Recognise"}
	status := review.OrderAnswers([]review.Answer{
		makeAnswer("01A", reviewCard.Card, reviewCard.Template, "2026-10-10T08:00:00Z", review.Good),
		makeAnswer("01B", reviewCard.Card, reviewCard.Template, "2026-10-13T08:00:00Z", review.Good),
	}).Replay(day, review.ScheduleBy(delta))[reviewCard]

	// The answer is given ten days before the night the clock goes back, so the
	// day the scheduler names falls the far side of it and the window the card
	// may be moved within reaches back over it.
	local := time.Date(2026, 10, 15, 10, 0, 0, 0, in)
	changes := time.Date(2026, 10, 25, 3, 0, 0, 0, in)
	due := delta.Next(status, local, review.Good).Due
	if !due.After(changes) {
		t.Fatalf("the card comes back on %v, which is not past the clock change", due.In(in))
	}

	// The stamp that answer stands as in the log, read back as a replay reads it.
	stamp, err := review.ParseTimestamp(local.UTC().Format(review.TimestampLayout))
	if err != nil {
		t.Fatal(err)
	}
	if !stamp.Equal(local) {
		t.Fatalf("the stamp reads back as %v, and the answer was given at %v", stamp, local)
	}

	// The day the scheduler named already carries cards, so the placement moves
	// the card and the arithmetic that adds days is reached.
	loaded := func(on time.Time) *review.DueByDay {
		s := review.NewDueByDay(day)
		for range 9 {
			s.Add(on)
		}
		return s
	}

	button := p.GetDueDay(loaded(due), local, due)
	replayed := p.AssignDueDay(loaded(due.UTC()), stamp, delta.Next(status, stamp, review.Good).Due)
	if button.Equal(due) {
		t.Fatalf("the placement left the card on %v, where the scheduler put it", due.In(in))
	}
	if !button.Equal(replayed) {
		t.Errorf("the button said %v and the replay put the card on %v",
			button.In(in), replayed.In(in))
	}
}

// scheduling is every arithmetic that places a card's day: the session that hands
// the cards out, and the projection drawn beside a control. Both go through
// AssignDueDay, and that is the whole of the list.
//
// A function is one arithmetic however many times its body asks — the
// projection places a card it has begun and a card it has answered again, and
// they are one step. A third function here is a third arithmetic, kept in step
// with these two by tests and drifting the day one of them is touched.
var scheduling = []string{"History.Replay", "Simulation.step"}

// A day is chosen in one function, and a caller wanting one evaluates to it. The
// walk is over this package's own files, because AssignDueDay is where a day is
// decided and the deciding is all here.
func TestTheSessionAndTheProjectionAreWhatScheduleADay(t *testing.T) {
	var calls []string
	read, declared := 0, 0
	for _, dir := range []string{".", "projection"} {
		target, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, one := range target {
			if one.IsDir() || !strings.HasSuffix(one.Name(), ".go") ||
				strings.HasSuffix(one.Name(), "_test.go") {
				continue
			}
			path := filepath.Join(dir, one.Name())
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			read++
			declared += countDeclarations(file, "AssignDueDay")
			calls = append(calls, getCallers(file, "AssignDueDay")...)
		}
	}

	// A walk that read none of the package, or one that did not find the
	// function the rule is about, is a rule checked against nothing.
	if read < 10 {
		t.Fatalf("%d files of this package read: the walk is not reading it", read)
	}
	if declared != 1 {
		t.Fatalf("%d functions named AssignDueDay are declared here: the walk is reading something else", declared)
	}

	slices.Sort(calls)
	calls = slices.Compact(calls)
	for _, one := range calls {
		if !slices.Contains(scheduling, one) {
			t.Errorf("%s places a card's day: one function places it and everything else comes to that one", one)
		}
	}
	for _, one := range scheduling {
		if !slices.Contains(calls, one) {
			t.Errorf("%s is named here as an arithmetic scheduling a day and places none", one)
		}
	}
}

// Put over sources written for it, the rule has to name the function a call
func TestGetCallers_MatchesOnlyCallsToAssignDueDay(t *testing.T) {
	for _, one := range []struct {
		why, src string
		calls    []string
	}{{
		why:   "a method calling it is named by its receiver's type",
		src:   "package review\nfunc (s Simulation) step() { p.AssignDueDay(on, at, due) }\n",
		calls: []string{"Simulation.step"},
	}, {
		why:   "a function calling it is named on its own",
		src:   "package review\nfunc drawn() { p.AssignDueDay(on, at, due) }\n",
		calls: []string{"drawn"},
	}, {
		why:   "a body asking twice is one arithmetic",
		src:   "package review\nfunc (s Simulation) step() { p.AssignDueDay(a, b, c); p.AssignDueDay(d, e, f) }\n",
		calls: []string{"Simulation.step"},
	}, {
		why:   "a call inside a literal is the function holding the literal",
		src:   "package review\nfunc (h History) Replay() { each(func() { one.Preset.AssignDueDay(on, at, due) }) }\n",
		calls: []string{"History.Replay"},
	}, {
		why:   "a pointer receiver is named by the type it is of",
		src:   "package review\nfunc (s *Simulation) step() { p.AssignDueDay(on, at, due) }\n",
		calls: []string{"Simulation.step"},
	}, {
		why: "the declaration of AssignDueDay is not a call of it",
		src: "package review\nfunc (p Preset) AssignDueDay(s *DueByDay) { p.computeDueDay(s) }\n",
	}, {
		why: "a comment naming it is not a call",
		src: "package review\n// AssignDueDay is where the day is chosen.\nfunc drawn() { p.computeDueDay(on) }\n",
	}, {
		why: "another name is another function",
		src: "package review\nfunc drawn() { p.Placed(on, at, due) }\n",
	}, {
		why: "naming it without calling it is not a call",
		src: "package review\nfunc drawn() { each(p.AssignDueDay) }\n",
	}} {
		file, err := parser.ParseFile(token.NewFileSet(), "one.go", one.src, 0)
		if err != nil {
			t.Fatalf("%s: %v", one.why, err)
		}
		if got := getCallers(file, "AssignDueDay"); !slices.Equal(got, one.calls) {
			t.Errorf("%s: %v, and the rule reads %v", one.why, one.calls, got)
		}
	}
}

// getCallers is every function of a file whose body calls this one, named by
// the type it is a method of and by its own name. A function calling it more
// than once is named once: what the rule counts is arithmetics and not call
// sites.
func getCallers(file *ast.File, name string) []string {
	var out []string
	for _, decl := range file.Decls {
		pos, is := decl.(*ast.FuncDecl)
		if !is || pos.Body == nil {
			continue
		}
		var calls bool
		ast.Inspect(pos.Body, func(node ast.Node) bool {
			call, is := node.(*ast.CallExpr)
			if !is {
				return true
			}
			if to, is := call.Fun.(*ast.SelectorExpr); is && to.Sel.Name == name {
				calls = true
			}
			return true
		})
		if calls {
			out = append(out, getFuncName(pos))
		}
	}
	return out
}

// countDeclarations is how many functions of a file are declared under this
// name.
func countDeclarations(file *ast.File, name string) int {
	out := 0
	for _, decl := range file.Decls {
		if pos, is := decl.(*ast.FuncDecl); is && pos.Name.Name == name {
			out++
		}
	}
	return out
}

// getFuncName is what a function is called, a method by the type it is of.
func getFuncName(pos *ast.FuncDecl) string {
	if pos.Recv == nil || len(pos.Recv.List) == 0 {
		return pos.Name.Name
	}
	of := pos.Recv.List[0].Type
	if star, is := of.(*ast.StarExpr); is {
		of = star.X
	}
	if target, is := of.(*ast.Ident); is {
		return target.Name + "." + pos.Name.Name
	}
	return pos.Name.Name
}
