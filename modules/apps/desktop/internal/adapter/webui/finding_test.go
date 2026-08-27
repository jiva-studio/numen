package webui_test

import (
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// The two questions the palette asks, over the wire.
//
// What a name or a passage is belongs to the index and the use case, and is
// asked there. What is asked here is what only the handler decides: which half
// the client's word for it runs, how many answers a number gets, and what
// reaches the client about a source that is not a note.

func TestANoteIsFoundByName(t *testing.T) {
	client, _ := opened(t, map[string]string{
		"Entropy.md": "---\ntitle: Entropy\n---\n\n# Entropy\n\n## Heat and work\n\nA reversible engine.\n",
		"Engines.md": "---\ntitle: Engines\n---\n\n# Engines\n",
	})

	answer, err := client.Names(t.Context(), connect.NewRequest(&v1.NamesRequest{Query: "entro"}))
	if err != nil {
		t.Fatal(err)
	}

	found := answer.Msg.GetFound()
	if len(found) == 0 {
		t.Fatal("a word still being typed found no name")
	}
	first := found[0]
	if first.GetNote().GetTitle() != "Entropy" {
		t.Errorf("the first answer is %q, want the note called that", first.GetNote().GetTitle())
	}
	if first.GetHeading() != nil {
		t.Errorf("a title matched, and the answer carries heading %+v", first.GetHeading())
	}
	if len(first.GetAt()) == 0 {
		t.Error("the answer says nothing about where it matched")
	}
}

func TestAHeadingIsFoundWithTheLineItStandsOn(t *testing.T) {
	client, _ := opened(t, map[string]string{
		"Entropy.md": "---\ntitle: Entropy\n---\n\n# Entropy\n\n## Heat and work\n\nA reversible engine.\n",
	})

	answer, err := client.Names(t.Context(), connect.NewRequest(&v1.NamesRequest{Query: "heat"}))
	if err != nil {
		t.Fatal(err)
	}

	found := answer.Msg.GetFound()
	if len(found) != 1 {
		t.Fatalf("%d answers, want the one heading: %+v", len(found), found)
	}
	heading := found[0].GetHeading()
	if heading.GetText() != "Heat and work" {
		t.Errorf("the answer is heading %q", heading.GetText())
	}
	if heading.GetLine() < 0 {
		t.Errorf("the heading stands on line %d, which is no line", heading.GetLine())
	}
}

func TestEachWayIsAskedByItself(t *testing.T) {
	client, _ := opened(t, map[string]string{
		"Engines.md": "---\ntitle: Engines\n---\n\n# Engines\n\nNo engine beats a reversible engine.\n",
	})

	// No model is set here, so the meaning half has nothing to answer with and
	// the words half answers on its own. Which is which is the handler's to get
	// right: both halves answer with the same shape.
	words, err := client.Search(t.Context(), connect.NewRequest(&v1.SearchRequest{
		Query: "reversible", Way: v1.Way_WAY_WORDS,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(words.Msg.GetFound()) == 0 {
		t.Fatal("the words half found nothing for a word that is written")
	}

	meaning, err := client.Search(t.Context(), connect.NewRequest(&v1.SearchRequest{
		Query: "reversible", Way: v1.Way_WAY_MEANING,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if got := len(meaning.Msg.GetFound()); got != 0 {
		t.Errorf("the meaning half answered %d times with no model set", got)
	}
}

func TestAPassageSaysWhichNoteItCameOutOf(t *testing.T) {
	client, _ := opened(t, map[string]string{
		"Engines.md": "---\ntitle: Engines\n---\n\n# Engines\n\nNo engine beats a reversible engine.\n",
	})

	answer, err := client.Search(t.Context(), connect.NewRequest(&v1.SearchRequest{
		Query: "reversible", Way: v1.Way_WAY_WORDS,
	}))
	if err != nil {
		t.Fatal(err)
	}

	found := answer.Msg.GetFound()
	if len(found) == 0 {
		t.Fatal("nothing found")
	}
	first := found[0]
	if first.GetNote().GetTitle() != "Engines" {
		t.Errorf("the passage says it came out of %q", first.GetNote().GetTitle())
	}
	if first.GetPath() == "" {
		t.Error("the passage says nothing about where it came from")
	}
	if len(first.GetAt()) == 0 {
		t.Error("the passage says nothing about where the word typed stands in it")
	}
}

// A hit is a place in a source, and the client is told which place: it is what
// opens the source there, and nothing the client holds says it.
func TestAPassageSaysWhereInItsSourceItStands(t *testing.T) {
	client, _ := opened(t, map[string]string{
		"Engines.md": "---\ntitle: Engines\n---\n\n# Engines\n\nNo engine beats a reversible engine.\n",
	})

	answer, err := client.Search(t.Context(), connect.NewRequest(&v1.SearchRequest{
		Query: "reversible", Way: v1.Way_WAY_WORDS,
	}))
	if err != nil {
		t.Fatal(err)
	}

	found := answer.Msg.GetFound()
	if len(found) == 0 {
		t.Fatal("nothing found")
	}
	first := found[0]
	if first.GetLength() == 0 {
		t.Errorf("the passage stands over no text: %d to %d",
			first.GetStart(), first.GetStart()+first.GetLength())
	}
	if first.GetStart() < 0 {
		t.Errorf("the passage begins at %d", first.GetStart())
	}
}

func TestAnAnswerIsCutToWhatWasAskedFor(t *testing.T) {
	client, _ := opened(t, map[string]string{
		"a.md": "# Entropy one\n",
		"b.md": "# Entropy two\n",
		"c.md": "# Entropy three\n",
	})

	answer, err := client.Names(t.Context(), connect.NewRequest(&v1.NamesRequest{
		Query: "entropy", Limit: 2,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if got := len(answer.Msg.GetFound()); got != 2 {
		t.Errorf("%d answers for a limit of 2", got)
	}
}

func TestNothingTypedIsAnsweredWithNothing(t *testing.T) {
	client, _ := opened(t, map[string]string{"a.md": "# Entropy\n"})

	for _, query := range []string{"", "   "} {
		answer, err := client.Search(t.Context(), connect.NewRequest(&v1.SearchRequest{Query: query}))
		if err != nil {
			t.Fatalf("%q: %v", query, err)
		}
		if got := len(answer.Msg.GetFound()); got != 0 {
			t.Errorf("%q was answered with %d passages", query, got)
		}
	}
}

// What a note is divided into, over the wire.

func TestANoteSaysWhatItIsDividedInto(t *testing.T) {
	client, _ := opened(t, map[string]string{
		"Entropy.md": "---\ntitle: Entropy\n---\n\n# Entropy\n\n## Heat and work\n\nA reversible engine.\n",
	})

	answer, err := client.Headings(t.Context(),
		connect.NewRequest(&v1.HeadingsRequest{Paths: []string{"Entropy.md"}}))
	if err != nil {
		t.Fatal(err)
	}

	found := answer.Msg.GetFound()
	if len(found) != 1 || found[0].GetPath() != "Entropy.md" {
		t.Fatalf("found = %+v, want the one note asked about", found)
	}

	headings := found[0].GetHeadings()
	if len(headings) != 2 {
		t.Fatalf("headings = %+v, want the two the note carries", headings)
	}
	if headings[0].GetText() != "Entropy" || headings[0].GetLevel() != 1 {
		t.Errorf("the first is %+v", headings[0])
	}
	if headings[1].GetText() != "Heat and work" || headings[1].GetLevel() != 2 {
		t.Errorf("the second is %+v", headings[1])
	}
	if !(headings[0].GetLine() < headings[1].GetLine()) {
		t.Errorf("lines are %d and %d, want them in the order they stand",
			headings[0].GetLine(), headings[1].GetLine())
	}
}

func TestAPathAskedTwiceIsAnsweredOnce(t *testing.T) {
	client, _ := opened(t, map[string]string{
		"Entropy.md": "---\ntitle: Entropy\n---\n\n# Entropy\n",
	})

	answer, err := client.Headings(t.Context(), connect.NewRequest(&v1.HeadingsRequest{
		Paths: []string{"Entropy.md", "Entropy.md", "Entropy.md"},
	}))
	if err != nil {
		t.Fatal(err)
	}

	if found := answer.Msg.GetFound(); len(found) != 1 {
		t.Errorf("found = %+v, want one entry for the one note", found)
	}
}

func TestANoteCarryingNoHeadingIsAbsent(t *testing.T) {
	client, _ := opened(t, map[string]string{
		"Plain.md": "---\ntitle: Plain\n---\n\nProse and nothing else.\n",
	})

	answer, err := client.Headings(t.Context(), connect.NewRequest(&v1.HeadingsRequest{
		Paths: []string{"Plain.md", "Gone.md"},
	}))
	if err != nil {
		t.Fatal(err)
	}

	if found := answer.Msg.GetFound(); len(found) != 0 {
		t.Errorf("found = %+v, want nothing for a note with no headings", found)
	}
}
