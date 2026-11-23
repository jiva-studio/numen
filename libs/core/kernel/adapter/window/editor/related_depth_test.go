package editor_test

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

func noteWith(title string, links ...string) string {
	body := "---\ntitle: " + title + "\n"
	if len(links) > 0 {
		body += "links:\n"
		for _, l := range links {
			role, to, _ := strings.Cut(l, " ")
			body += fmt.Sprintf("  - to: %s\n    role: %s\n", to, role)
		}
	}
	return body + "---\n\n# " + title + "\n"
}

func relatedVault() map[string]string {
	return map[string]string{
		"Entropy.md": noteWith("Entropy", "parent Physics", "child Heat", "child Work", "jump Order"),
		"Heat.md":    noteWith("Heat", "child Engine", "parent Entropy"),
		"Work.md":    noteWith("Work", "child Engine"),
		"Engine.md":  noteWith("Engine"),
		"Order.md":   noteWith("Order"),
		"Physics.md": noteWith("Physics", "parent Science", "child Optics", "child Mirror", "jump Cosmos"),
		"Science.md": noteWith("Science"),
		"Cosmos.md":  noteWith("Cosmos"),
		"Optics.md":  noteWith("Optics", "child Lens"),
		"Mirror.md":  noteWith("Mirror"),
		"Lens.md":    noteWith("Lens"),
	}
}

func askRelated(t *testing.T, client questions, path string, depth uint32) (*v1.NoteGraph, error) {
	t.Helper()
	got, err := client.ListRelatedNotes(t.Context(), connect.NewRequest(&v1.ListRelatedNotesRequest{Paths: []string{path}, Depth: depth}))
	if err != nil {
		return nil, err
	}
	return graphOf(t, got.Msg), nil
}

func graphOf(t *testing.T, msg *v1.ListRelatedNotesResponse) *v1.NoteGraph {
	t.Helper()
	if len(msg.GetGraphs()) != 1 {
		t.Fatalf("%d graphs for one path, want 1", len(msg.GetGraphs()))
	}
	return msg.GetGraphs()[0]
}

func mustAskRelated(t *testing.T, client questions, path string, depth uint32) *v1.NoteGraph {
	t.Helper()
	got, err := askRelated(t, client, path, depth)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func itemsRelatedTo(msg *v1.NoteGraph, relatedTo string) []string {
	var out []string
	for _, r := range msg.GetRelatedNotes() {
		if r.GetRelatedTo() == relatedTo {
			out = append(out, r.GetRelation().String()+" "+r.GetNote().GetPath())
		}
	}
	sort.Strings(out)
	return out
}

func TestDepthZeroAndOneAnswerAlikeWithNoItemOfDepthTwo(t *testing.T) {
	client, _ := openVault(t, relatedVault())

	none := mustAskRelated(t, client, "Entropy.md", 0)
	one := mustAskRelated(t, client, "Entropy.md", 1)

	for name, got := range map[string]*v1.NoteGraph{"depth 0": none, "depth 1": one} {
		if len(got.GetRelatedNotes()) != 6 {
			t.Errorf("%s: %d related notes, want 6", name, len(got.GetRelatedNotes()))
		}
		for _, r := range got.GetRelatedNotes() {
			if r.GetRelatedTo() != "" {
				t.Errorf("%s: %s is related to %s", name, r.GetNote().GetPath(), r.GetRelatedTo())
			}
		}
	}
	if len(none.GetConnections()) != len(one.GetConnections()) {
		t.Errorf("connections differ: %d and %d", len(none.GetConnections()), len(one.GetConnections()))
	}
}

func TestDepthTwoAnswersTheNotesRelatedToEachRelatedNote(t *testing.T) {
	client, _ := openVault(t, relatedVault())

	got := mustAskRelated(t, client, "Entropy.md", 2)

	if want := []string{"RELATION_CHILD Engine.md"}; !slices.Equal(itemsRelatedTo(got, "Heat.md"), want) {
		t.Errorf("related to Heat: %v, want %v", itemsRelatedTo(got, "Heat.md"), want)
	}
	want := []string{"RELATION_CHILD Mirror.md", "RELATION_CHILD Optics.md", "RELATION_JUMP Cosmos.md", "RELATION_PARENT Science.md"}
	if g := itemsRelatedTo(got, "Physics.md"); !slices.Equal(g, want) {
		t.Errorf("related to Physics: %v, want %v", g, want)
	}
	if n := len(itemsRelatedTo(got, "")); n != 6 {
		t.Errorf("%d notes related to the focus, want the 6 of depth 1 only", n)
	}
}

func TestASiblingHasItsOwnRelatedNotesAndNoSiblings(t *testing.T) {
	client, _ := openVault(t, relatedVault())

	got := mustAskRelated(t, client, "Entropy.md", 2)

	if want := []string{"RELATION_CHILD Lens.md", "RELATION_PARENT Physics.md"}; !slices.Equal(itemsRelatedTo(got, "Optics.md"), want) {
		t.Errorf("related to Optics: %v, want %v", itemsRelatedTo(got, "Optics.md"), want)
	}
	for _, r := range got.GetRelatedNotes() {
		if r.GetRelatedTo() != "" && r.GetRelation() == v1.Relation_RELATION_SIBLING {
			t.Errorf("%s is a sibling item of %s", r.GetNote().GetPath(), r.GetRelatedTo())
		}
	}
}

func TestTheFocusIsNeverAnItemAndANoteBelongsToEachNoteItIsRelatedTo(t *testing.T) {
	client, _ := openVault(t, relatedVault())

	got := mustAskRelated(t, client, "Entropy.md", 2)

	for _, r := range got.GetRelatedNotes() {
		if r.GetNote().GetPath() == "Entropy.md" {
			t.Errorf("the focus is an item related to %q", r.GetRelatedTo())
		}
	}
	if !slices.Contains(itemsRelatedTo(got, "Heat.md"), "RELATION_CHILD Engine.md") ||
		!slices.Contains(itemsRelatedTo(got, "Work.md"), "RELATION_CHILD Engine.md") {
		t.Errorf("Engine is not an item of both Heat and Work: %v, %v", itemsRelatedTo(got, "Heat.md"), itemsRelatedTo(got, "Work.md"))
	}
}

func TestDepthTwoHasNoLimitAndTheConnectionsOfDepthOne(t *testing.T) {
	notes := map[string]string{
		"Entropy.md": noteWith("Entropy", "child Big"),
		"Big.md":     noteWith("Big"),
	}
	for i := range 60 {
		name := fmt.Sprintf("K%02d", i)
		notes[name+".md"] = noteWith(name, "parent Big")
	}
	client, _ := openVault(t, notes)

	two := mustAskRelated(t, client, "Entropy.md", 2)
	one := mustAskRelated(t, client, "Entropy.md", 1)

	if n := len(itemsRelatedTo(two, "Big.md")); n != 60 {
		t.Errorf("%d items of Big, want 60", n)
	}
	if len(one.GetConnections()) != 1 || len(two.GetConnections()) != 1 {
		t.Errorf("connections: %d at depth 1, %d at depth 2, want 1 and 1", len(one.GetConnections()), len(two.GetConnections()))
	}
}

func TestItemsOfDepthTwoFollowTheNotesRelatedToTheFocus(t *testing.T) {
	client, _ := openVault(t, relatedVault())

	got := mustAskRelated(t, client, "Entropy.md", 2)

	firstItem := -1
	for i, r := range got.GetRelatedNotes() {
		if r.GetRelatedTo() != "" && firstItem < 0 {
			firstItem = i
		}
		if r.GetRelatedTo() == "" && firstItem >= 0 {
			t.Fatalf("%s is related to the focus and comes after an item of depth 2", r.GetNote().GetPath())
		}
	}
	if firstItem < 0 {
		t.Fatal("no item of depth 2")
	}
	var groups []string
	for _, r := range got.GetRelatedNotes()[firstItem:] {
		if len(groups) == 0 || groups[len(groups)-1] != r.GetRelatedTo() {
			if slices.Contains(groups, r.GetRelatedTo()) {
				t.Errorf("the items of %s are not one group", r.GetRelatedTo())
			}
			groups = append(groups, r.GetRelatedTo())
		}
	}
	if want := []string{"Physics.md", "Heat.md", "Work.md", "Optics.md", "Mirror.md"}; !isSubsequence(groups, want) {
		t.Errorf("groups %v do not follow the order of the notes related to the focus", groups)
	}
}

func isSubsequence(got, order []string) bool {
	next := 0
	for _, g := range got {
		for next < len(order) && order[next] != g {
			next++
		}
		if next == len(order) {
			return false
		}
	}
	return true
}

func TestAnItemOfDepthTwoCarriesTheTypeAndTabKindItHasAsAFocus(t *testing.T) {
	client, _ := openVault(t, map[string]string{
		"Entropy.md": noteWith("Entropy", "child Heat"),
		"Heat.md":    noteWith("Heat", "child Gauge"),
		"Gauge.md":   "---\ntype: sample\ntitle: Gauge\n---\n\n# Gauge\n",
	})

	two := mustAskRelated(t, client, "Entropy.md", 2)
	asFocus := mustAskRelated(t, client, "Gauge.md", 1)

	var item *v1.RelatedNote
	for _, r := range two.GetRelatedNotes() {
		if r.GetNote().GetPath() == "Gauge.md" && r.GetRelatedTo() == "Heat.md" {
			item = r
		}
	}
	if item == nil {
		t.Fatal("Gauge is not an item of Heat")
	}
	if item.GetType() != "sample" || item.GetType() != asFocus.GetFocusType() || item.GetTabKind() != asFocus.GetFocusTabKind() {
		t.Errorf("type %q and tab kind %q, want %q and %q", item.GetType(), item.GetTabKind(), asFocus.GetFocusType(), asFocus.GetFocusTabKind())
	}
}

func TestADepthBeyondTwoIsAnError(t *testing.T) {
	client, _ := openVault(t, relatedVault())

	_, err := askRelated(t, client, "Entropy.md", 3)

	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("depth 3: %v, want invalid argument", err)
	}
	for _, depth := range []uint32{0, 1} {
		if _, err := askRelated(t, client, "Entropy.md", depth); err != nil {
			t.Errorf("depth %d: %v", depth, err)
		}
	}
}

func askManyRelated(t *testing.T, client questions, paths []string, depth uint32) (*v1.ListRelatedNotesResponse, error) {
	t.Helper()
	got, err := client.ListRelatedNotes(t.Context(), connect.NewRequest(&v1.ListRelatedNotesRequest{Paths: paths, Depth: depth}))
	if err != nil {
		return nil, err
	}
	return got.Msg, nil
}

func TestEachPathAnswersWithItsOwnGraphInTheOrderOfTheRequest(t *testing.T) {
	client, _ := openVault(t, relatedVault())
	paths := []string{"Heat.md", "Physics.md", "Entropy.md"}

	got, err := askManyRelated(t, client, paths, 1)
	if err != nil {
		t.Fatal(err)
	}

	if len(got.GetGraphs()) != len(paths) {
		t.Fatalf("%d graphs for %d paths", len(got.GetGraphs()), len(paths))
	}
	for i, path := range paths {
		alone := mustAskRelated(t, client, path, 1)
		if got.GetGraphs()[i].GetFocus().GetPath() != path {
			t.Errorf("graph %d is the graph of %q, want %q", i, got.GetGraphs()[i].GetFocus().GetPath(), path)
		}
		if !proto.Equal(got.GetGraphs()[i], alone) {
			t.Errorf("the graph of %s differs from the one asked alone", path)
		}
	}
}

func TestAnUnknownPathHasAnEmptyGraphAndTheOtherGraphsAreWhole(t *testing.T) {
	client, _ := openVault(t, relatedVault())

	got, err := askManyRelated(t, client, []string{"Entropy.md", "Nowhere.md", "Heat.md"}, 1)
	if err != nil {
		t.Fatal(err)
	}

	if len(got.GetGraphs()) != 3 {
		t.Fatalf("%d graphs for 3 paths", len(got.GetGraphs()))
	}
	if gone := got.GetGraphs()[1]; gone.GetFocus().GetPath() != "" || len(gone.GetRelatedNotes()) != 0 {
		t.Errorf("the graph of an unknown note is %v, want an empty graph", gone)
	}
	if got.GetGraphs()[0].GetFocus().GetPath() != "Entropy.md" || got.GetGraphs()[2].GetFocus().GetPath() != "Heat.md" {
		t.Errorf("the other graphs are %v and %v", got.GetGraphs()[0], got.GetGraphs()[2])
	}
}

func TestNoPathsAnswerNoGraphs(t *testing.T) {
	client, _ := openVault(t, relatedVault())

	got, err := askManyRelated(t, client, nil, 1)
	if err != nil {
		t.Fatal(err)
	}

	if len(got.GetGraphs()) != 0 {
		t.Errorf("%d graphs for no path", len(got.GetGraphs()))
	}
}

func TestMoreThanTheMostPathsFailTheCall(t *testing.T) {
	client, _ := openVault(t, relatedVault())
	paths := func(n int) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = "Heat.md"
		}
		return out
	}

	got, err := askManyRelated(t, client, paths(domain.MaxRelatedNotePaths), 1)
	if err != nil || len(got.GetGraphs()) != domain.MaxRelatedNotePaths {
		t.Errorf("the most paths: %d graphs (%v), want %d", len(got.GetGraphs()), err, domain.MaxRelatedNotePaths)
	}
	_, err = askManyRelated(t, client, paths(domain.MaxRelatedNotePaths+1), 1)
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("one path more than the most: %v, want invalid argument", err)
	}
}

func TestAnInvalidDepthFailsTheCallForEveryPath(t *testing.T) {
	client, _ := openVault(t, relatedVault())

	_, err := askManyRelated(t, client, []string{"Entropy.md", "Heat.md"}, 3)

	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("depth 3: %v, want invalid argument", err)
	}
}
