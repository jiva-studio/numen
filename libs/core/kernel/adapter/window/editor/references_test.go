package editor_test

import (
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
)

func TestTheRelatedNotesOfANoteTellWhichWayAReferenceIsLinked(t *testing.T) {
	f := openWindow(t, nil, map[string]string{
		"Heat.md":    "---\ntitle: Heat\nlinks:\n  - to: \"[[Area]]\"\n    role: parent\n---\n\n# Heat\n\nSee [[Entropy]].\n",
		"Entropy.md": "---\ntitle: Entropy\n---\n\n# Entropy\n",
		"Work.md":    "---\ntitle: Work\n---\n\n# Work\n\nCompare [[Heat]].\n",
		"Area.md":    "---\ntitle: Area\n---\n\n# Area\n",
	})
	waitForScan(t, f)

	related, err := f.client.ListRelatedNotes(t.Context(), connect.NewRequest(&v1.ListRelatedNotesRequest{Path: "Heat.md"}))
	if err != nil {
		t.Fatal(err)
	}

	type item struct {
		relation  v1.Relation
		direction v1.LinkDirection
	}
	got := map[string]item{}
	for _, r := range related.Msg.GetRelated() {
		got[r.GetNote().GetPath()] = item{r.GetRelation(), r.GetDirection()}
	}
	want := map[string]item{
		"Area.md":    {v1.Relation_RELATION_PARENT, v1.LinkDirection_LINK_DIRECTION_UNSPECIFIED},
		"Entropy.md": {v1.Relation_RELATION_REFERENCE, v1.LinkDirection_LINK_DIRECTION_OUTGOING},
		"Work.md":    {v1.Relation_RELATION_REFERENCE, v1.LinkDirection_LINK_DIRECTION_INCOMING},
	}
	if len(got) != len(want) {
		t.Fatalf("related = %v", got)
	}
	for path, w := range want {
		if got[path] != w {
			t.Errorf("%s is %v, want %v", path, got[path], w)
		}
	}
	for _, c := range related.Msg.GetConnections() {
		if c.GetFirst() == "Entropy.md" || c.GetSecond() == "Entropy.md" || c.GetFirst() == "Work.md" || c.GetSecond() == "Work.md" {
			t.Errorf("a reference has a connection record: %v", c)
		}
	}
}
