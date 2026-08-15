package note_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
	usecase "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/vault"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/testsupport"
)

// indexed writes a vault of the given notes, scans it, and hands back what is
// needed to ask questions about links.
func indexed(t *testing.T, notes map[string]string) (*container.Index, domain.Vault) {
	t.Helper()
	v := testsupport.NewVault(t, notes)
	db, err := container.Config{
		IndexPath: filepath.Join(t.TempDir(), "index.db"),
	}.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	scan := usecase.Scan{
		Readers: filesystem.Readers{},
		Vaults:  db.Vaults(),
		Notes:   db.Notes(),
		Known:   db.Queries(),
	}
	if _, err := scan.Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	return db, v
}

func connections(t *testing.T, db *container.Index, v domain.Vault, path string) note.Connections {
	t.Helper()
	c, err := note.ShowConnections{Links: db.Links()}.Execute(context.Background(), v, path)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestANameResolvesToTheNoteThatAnswersToIt(t *testing.T) {
	db, v := indexed(t, map[string]string{
		"source.md":        "Points at [[Entropy]].\n",
		"notes/Entropy.md": "# Entropy\n",
	})

	c := connections(t, db, v, "source.md")
	if len(c.Links) != 1 {
		t.Fatalf("got %+v", c.Links)
	}
	if c.Links[0].To != "notes/Entropy.md" {
		t.Errorf("resolved to %q", c.Links[0].To)
	}
	if c.Links[0].Ambiguous {
		t.Error("a single match was called ambiguous")
	}
}

func TestAPathFromTheRootWinsOverAName(t *testing.T) {
	db, v := indexed(t, map[string]string{
		"source.md":          "Points at [[archive/Entropy]].\n",
		"notes/Entropy.md":   "# The near one\n",
		"archive/Entropy.md": "# The one that was asked for\n",
	})

	c := connections(t, db, v, "source.md")
	if c.Links[0].To != "archive/Entropy.md" {
		t.Errorf("resolved to %q, want the path that was written", c.Links[0].To)
	}
}

func TestTheFolderTheLinkWasWrittenInDecidesIt(t *testing.T) {
	// Two notes answer to the name, and one of them is in the same folder as the
	// note that wrote the link. That is a determined answer rather than an
	// ambiguity: the rule picked it, not a tie-break.
	db, v := indexed(t, map[string]string{
		"projects/source.md":  "Points at [[Entropy]].\n",
		"projects/Entropy.md": "# The neighbour\n",
		"archive/Entropy.md":  "# The stranger\n",
	})

	c := connections(t, db, v, "projects/source.md")
	if c.Links[0].To != "projects/Entropy.md" {
		t.Errorf("resolved to %q", c.Links[0].To)
	}
	if c.Links[0].Ambiguous {
		t.Error("a rule decided it, so nothing should be called ambiguous")
	}
}

func TestSeveralNotesByOneNameAreAmbiguousAndStillResolve(t *testing.T) {
	// Neither an exact path nor the folder the link was written in picks one, so
	// only the name is left and it answers twice. The link still goes somewhere —
	// a dead link would be worse — and the vault has a question in it.
	db, v := indexed(t, map[string]string{
		"projects/source.md": "Points at [[Entropy]].\n",
		"notes/Entropy.md":   "# One\n",
		"archive/Entropy.md": "# The other\n",
	})

	c := connections(t, db, v, "projects/source.md")
	if !c.Links[0].Ambiguous {
		t.Error("two notes answer to that name and nothing said so")
	}
	if c.Links[0].To == "" {
		t.Error("an ambiguous link was left dangling")
	}
}

func TestALinkToNothingIsDanglingRatherThanAnError(t *testing.T) {
	db, v := indexed(t, map[string]string{
		"source.md": "Points at [[Nothing At All]].\n",
	})

	c := connections(t, db, v, "source.md")
	if len(c.Links) != 1 {
		t.Fatalf("got %+v", c.Links)
	}
	if c.Links[0].To != "" {
		t.Errorf("resolved to %q, want nothing", c.Links[0].To)
	}
}

func TestAnIdentifierResolvesWhateverTheFileIsCalled(t *testing.T) {
	// This is what the identifier form is for: the target was renamed, and the
	// link did not have to be.
	db, v := indexed(t, map[string]string{
		"source.md":        "---\nlinks:\n  - to: \"note://01M02ACGM0FYMSXNDP29C90JNR\"\n    role: parent\n---\n\nbody\n",
		"renamed-since.md": "---\nid: 01M02ACGM0FYMSXNDP29C90JNR\n---\n\n# Still the same note\n",
	})

	c := connections(t, db, v, "source.md")
	if len(c.Links) != 1 || c.Links[0].To != "renamed-since.md" {
		t.Errorf("got %+v", c.Links)
	}
}

func TestBacklinksFindBothFormsOfAddress(t *testing.T) {
	db, v := indexed(t, map[string]string{
		"target.md":    "---\nid: 01M02ACGM0FYMSXNDP29C90JNR\n---\n\n# Target\n",
		"by-name.md":   "Points at [[target]].\n",
		"by-id.md":     "---\nlinks:\n  - to: \"note://01M02ACGM0FYMSXNDP29C90JNR\"\n    role: jump\n---\n\nbody\n",
		"unrelated.md": "# Nothing to do with it\n",
	})

	c := connections(t, db, v, "target.md")
	if len(c.Backlinks) != 2 {
		t.Fatalf("backlinks = %+v", c.Backlinks)
	}
	found := map[string]bool{}
	for _, b := range c.Backlinks {
		found[b.From] = true
	}
	if !found["by-name.md"] || !found["by-id.md"] {
		t.Errorf("backlinks = %+v", c.Backlinks)
	}
}

func TestAnAttachmentIsALinkAndResolvesToNoNote(t *testing.T) {
	db, v := indexed(t, map[string]string{
		"source.md": "---\nlinks:\n  - to: \"https://example.org/paper\"\n    role: attachment\n---\n\nbody\n",
	})

	c := connections(t, db, v, "source.md")
	if len(c.Links) != 1 {
		t.Fatalf("got %+v", c.Links)
	}
	if c.Links[0].Role != domain.RoleAttachment {
		t.Errorf("role = %q", c.Links[0].Role)
	}
	if c.Links[0].To != "" {
		t.Errorf("an attachment resolved to a note: %q", c.Links[0].To)
	}
}

func TestLinksFollowTheVaultTheyAreWrittenIn(t *testing.T) {
	// A link resolves inside its own vault. Another vault holding a note by the
	// same name is not an answer, and there is no syntax for asking it.
	db, first := indexed(t, map[string]string{
		"source.md": "Points at [[Entropy]].\n",
	})
	second := testsupport.NewVault(t, map[string]string{"Entropy.md": "# Elsewhere\n"})
	scan := usecase.Scan{
		Readers: filesystem.Readers{},
		Vaults:  db.Vaults(),
		Notes:   db.Notes(),
		Known:   db.Queries(),
	}
	if _, err := scan.Execute(t.Context(), second); err != nil {
		t.Fatal(err)
	}

	c := connections(t, db, first, "source.md")
	if c.Links[0].To != "" {
		t.Errorf("a link resolved into another vault: %q", c.Links[0].To)
	}
}
