package note_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport/indexfile"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	usecase "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// indexed writes a vault of the given notes, scans it, and hands back what is
// needed to ask questions about links.
func indexed(t *testing.T, notes map[string]string) (*container.Index, domain.Vault) {
	t.Helper()
	v := testsupport.NewVault(t, notes)
	db, err := container.Config{IndexPath: indexfile.Path(t)}.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	scan := usecase.Scan{
		Readers:     filesystem.VaultReaders{},
		Vaults:      db.Vaults(),
		Notes:       db.Notes(),
		Known:       db.Queries(),
		Maintenance: db.Maintenance(),
	}
	if _, err := scan.Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	return db, v
}

func links(t *testing.T, db *container.Index, v domain.Vault, path string) note.NoteLinks {
	t.Helper()
	c, err := note.ShowLinks{Links: db.Links()}.Execute(t.Context(), v, path)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestANameResolvesToTheNoteThatAnswersToIt(t *testing.T) {
	t.Parallel()
	db, v := indexed(t, map[string]string{
		"source.md":        "Points at [[Entropy]].\n",
		"notes/Entropy.md": "# Entropy\n",
	})

	c := links(t, db, v, "source.md")
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
	t.Parallel()
	db, v := indexed(t, map[string]string{
		"source.md":          "Points at [[archive/Entropy]].\n",
		"notes/Entropy.md":   "# The near one\n",
		"archive/Entropy.md": "# The one that was asked for\n",
	})

	c := links(t, db, v, "source.md")
	if c.Links[0].To != "archive/Entropy.md" {
		t.Errorf("resolved to %q, want the path that was written", c.Links[0].To)
	}
}

func TestTheFolderTheLinkWasWrittenInDecidesIt(t *testing.T) {
	t.Parallel()
	// Two notes answer to the name, and one of them is in the same folder as the
	// note that wrote the link. That is a determined answer rather than an
	// ambiguity: the rule picked it, not a tie-break.
	db, v := indexed(t, map[string]string{
		"projects/source.md":  "Points at [[Entropy]].\n",
		"projects/Entropy.md": "# The neighbour\n",
		"archive/Entropy.md":  "# The stranger\n",
	})

	c := links(t, db, v, "projects/source.md")
	if c.Links[0].To != "projects/Entropy.md" {
		t.Errorf("resolved to %q", c.Links[0].To)
	}
	if c.Links[0].Ambiguous {
		t.Error("a rule decided it, so nothing should be called ambiguous")
	}
}

func TestSeveralNotesByOneNameAreAmbiguousAndStillResolve(t *testing.T) {
	t.Parallel()
	// Neither an exact path nor the folder the link was written in picks one, so
	// only the name is left and it answers twice. The link still goes somewhere —
	// a dead link would be worse — and the vault has a question in it.
	db, v := indexed(t, map[string]string{
		"projects/source.md": "Points at [[Entropy]].\n",
		"notes/Entropy.md":   "# One\n",
		"archive/Entropy.md": "# The other\n",
	})

	c := links(t, db, v, "projects/source.md")
	if !c.Links[0].Ambiguous {
		t.Error("two notes answer to that name and nothing said so")
	}
	if c.Links[0].To == "" {
		t.Error("an ambiguous link was left dangling")
	}
}

// A dot in a name is part of the name, and the whole of it is what the note is
// filed under. Both directions are asked here: the link and the backlink are
// answered through the same stored name.
func TestANameCarryingDotsResolvesWholeAndIsABacklink(t *testing.T) {
	t.Parallel()
	const lecture = "Seminar 1.2–1.3 — Lisbon, 9 July 1973"
	db, v := indexed(t, map[string]string{
		"source.md":                "Points at [[" + lecture + "]].\n",
		"notes/" + lecture + ".md": "# " + lecture + "\n",
	})

	c := links(t, db, v, "source.md")
	if len(c.Links) != 1 {
		t.Fatalf("got %+v", c.Links)
	}
	if c.Links[0].To != "notes/"+lecture+".md" {
		t.Errorf("resolved to %q", c.Links[0].To)
	}

	back := links(t, db, v, "notes/"+lecture+".md")
	if len(back.Backlinks) != 1 || back.Backlinks[0].From != "source.md" {
		t.Errorf("backlinks = %+v", back.Backlinks)
	}
}

func TestALinkToNothingIsDanglingRatherThanAnError(t *testing.T) {
	t.Parallel()
	db, v := indexed(t, map[string]string{
		"source.md": "Points at [[Nothing At All]].\n",
	})

	c := links(t, db, v, "source.md")
	if len(c.Links) != 1 {
		t.Fatalf("got %+v", c.Links)
	}
	if c.Links[0].To != "" {
		t.Errorf("resolved to %q, want nothing", c.Links[0].To)
	}
}

func TestAnIdentifierResolvesWhateverTheFileIsCalled(t *testing.T) {
	t.Parallel()
	// This is what the identifier form is for: the target was renamed, and the
	// link did not have to be.
	db, v := indexed(t, map[string]string{
		"source.md":        "---\nlinks:\n  - to: \"note://01M02ACGM0FYMSXNDP29C90JNR\"\n    role: parent\n---\n\nbody\n",
		"renamed-since.md": "---\nid: 01M02ACGM0FYMSXNDP29C90JNR\n---\n\n# Still the same note\n",
	})

	c := links(t, db, v, "source.md")
	if len(c.Links) != 1 || c.Links[0].To != "renamed-since.md" {
		t.Errorf("got %+v", c.Links)
	}
}

func TestBacklinksFindBothFormsOfAddress(t *testing.T) {
	t.Parallel()
	db, v := indexed(t, map[string]string{
		"target.md":    "---\nid: 01M02ACGM0FYMSXNDP29C90JNR\n---\n\n# Target\n",
		"by-name.md":   "Points at [[target]].\n",
		"by-id.md":     "---\nlinks:\n  - to: \"note://01M02ACGM0FYMSXNDP29C90JNR\"\n    role: jump\n---\n\nbody\n",
		"unrelated.md": "# Nothing to do with it\n",
	})

	c := links(t, db, v, "target.md")
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
	t.Parallel()
	db, v := indexed(t, map[string]string{
		"source.md": "---\nlinks:\n  - to: \"https://example.org/paper\"\n    role: attachment\n---\n\nbody\n",
	})

	c := links(t, db, v, "source.md")
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

func TestANameNeverLeavesItsVault(t *testing.T) {
	t.Parallel()
	// A name means something only inside one vault. Another vault holding a note
	// by the same name is not an answer, and there is no way to ask it for one.
	db, first := indexed(t, map[string]string{
		"source.md": "Points at [[Entropy]].\n",
	})
	addVault(t, db, testsupport.NewVault(t, map[string]string{"Entropy.md": "# Elsewhere\n"}))

	c := links(t, db, first, "source.md")
	if c.Links[0].To != "" {
		t.Errorf("a name resolved into another vault: %q", c.Links[0].To)
	}
}

func TestAnIdentifierCrossesIntoAConnectedVault(t *testing.T) {
	t.Parallel()
	// The seam the user put there on purpose: a link written by identifier finds
	// its note wherever that note is, and says which vault that turned out to be.
	const id = "01M02DTC80PABQQW3XS3XWDVHW"
	db, first := indexed(t, map[string]string{
		"source.md": "---\nlinks:\n  - to: \"note://" + id + "\"\n    role: jump\n---\n\nbody\n",
	})
	other := addVault(t, db, testsupport.NewVault(t, map[string]string{
		"elsewhere.md": "---\nid: " + id + "\n---\n\n# In the other vault\n",
	}))

	c := links(t, db, first, "source.md")
	if c.Links[0].To != "elsewhere.md" {
		t.Fatalf("resolved to %q", c.Links[0].To)
	}
	vault, crossed := c.Links[0].InVault(first.ID)
	if !crossed || vault != other.ID {
		t.Errorf("landed in vault %q, crossed=%v", vault, crossed)
	}
}

func TestAnIdentifierInAVaultThatIsNotConnectedIsNeitherResolvedNorBroken(t *testing.T) {
	t.Parallel()
	// Nothing here can tell a deleted note from one in a vault the user has not
	// added, and calling it broken would report a link that is fine on the
	// machine where both vaults are open.
	db, v := indexed(t, map[string]string{
		"source.md": "---\nlinks:\n  - to: \"note://01M02DTC80PABQQW3XS3XWDVHW\"\n    role: jump\n---\n\nbody\n",
	})

	c := links(t, db, v, "source.md")
	if len(c.Links) != 1 {
		t.Fatalf("got %+v", c.Links)
	}
	if c.Links[0].To != "" || c.Links[0].ToVault != "" {
		t.Errorf("something resolved: %+v", c.Links[0])
	}
	if c.Links[0].Ambiguous {
		t.Error("an unreachable note is not an ambiguity")
	}
}

// addVault indexes a second vault into the same database, which is what makes
// a link across vaults possible at all.
func addVault(t *testing.T, db *container.Index, v domain.Vault) domain.Vault {
	t.Helper()
	scan := usecase.Scan{
		Readers:     filesystem.VaultReaders{},
		Vaults:      db.Vaults(),
		Notes:       db.Notes(),
		Known:       db.Queries(),
		Maintenance: db.Maintenance(),
	}
	if _, err := scan.Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestALinkWrittenAsAPathIsStillABacklink(t *testing.T) {
	t.Parallel()
	// A backlink is a link that resolves here, not one whose text looks like
	// this note. Written as a path, it never matches by name.
	db, v := indexed(t, map[string]string{
		"source.md":        "---\nlinks:\n  - to: \"[[notes/Entropy]]\"\n    role: child\n---\n\nbody\n",
		"notes/Entropy.md": "# Entropy\n",
	})

	c := links(t, db, v, "notes/Entropy.md")
	if len(c.Backlinks) != 1 {
		t.Fatalf("backlinks = %+v", c.Backlinks)
	}
	if c.Backlinks[0].Role != domain.RoleChild {
		t.Errorf("role = %q, want child — the edge that makes the hierarchy", c.Backlinks[0].Role)
	}
}

func TestALinkThatResolvesElsewhereIsNotABacklink(t *testing.T) {
	t.Parallel()
	// Two notes answer to the name, and the link resolves to the near one. The
	// far one must not claim it.
	db, v := indexed(t, map[string]string{
		"projects/source.md":  "Points at [[Entropy]].\n",
		"projects/Entropy.md": "# The one it means\n",
		"archive/Entropy.md":  "# The one it does not\n",
	})

	near := links(t, db, v, "projects/Entropy.md")
	if len(near.Backlinks) != 1 {
		t.Errorf("the note the link resolves to has %d backlinks", len(near.Backlinks))
	}
	far := links(t, db, v, "archive/Entropy.md")
	if len(far.Backlinks) != 0 {
		t.Errorf("a note claimed a link that resolves elsewhere: %+v", far.Backlinks)
	}
}

func TestBacklinksNeverCrossVaults(t *testing.T) {
	t.Parallel()
	// One database for every vault makes a query that forgets its vault
	// invisible by construction. Asked here of the direction that has to look
	// at every link in the vault.
	db, first := indexed(t, map[string]string{
		"target.md": "# Target\n",
		"source.md": "Points at [[target]].\n",
	})
	second := addVault(t, db, testsupport.NewVault(t, map[string]string{
		"target.md":    "# A different target\n",
		"elsewhere.md": "Points at [[target]].\n",
	}))

	c := links(t, db, first, "target.md")
	if len(c.Backlinks) != 1 || c.Backlinks[0].From != "source.md" {
		t.Errorf("first vault backlinks = %+v", c.Backlinks)
	}
	other := links(t, db, second, "target.md")
	if len(other.Backlinks) != 1 || other.Backlinks[0].From != "elsewhere.md" {
		t.Errorf("second vault backlinks = %+v", other.Backlinks)
	}
}

func TestOneNoteWrittenTwoWaysIsOneLink(t *testing.T) {
	t.Parallel()
	// The links block names it by path, the prose names it by name. Both mean
	// the same note, so there is one link, and the described one wins.
	db, v := indexed(t, map[string]string{
		"source.md":        "---\nlinks:\n  - to: \"[[notes/Entropy]]\"\n    role: child\n---\n\nAlso mentioned as [[Entropy]].\n",
		"notes/Entropy.md": "# Entropy\n",
	})

	c := links(t, db, v, "source.md")
	if len(c.Links) != 1 {
		t.Fatalf("got %d links, want one: %+v", len(c.Links), c.Links)
	}
	if c.Links[0].Role != domain.RoleChild {
		t.Errorf("role = %q, want the described one", c.Links[0].Role)
	}
}

func TestTwoUnresolvedLinksAreOnlyTheSameWhenWrittenTheSame(t *testing.T) {
	t.Parallel()
	// A name that answers to nothing is matched as it is written, so two of
	// them stay two.
	db, v := indexed(t, map[string]string{
		"source.md": "Points at [[Nowhere]] and [[Elsewhere]].\n",
	})

	c := links(t, db, v, "source.md")
	if len(c.Links) != 2 {
		t.Errorf("got %+v", c.Links)
	}
}

func TestANameMatchesWhateverCaseItWasTypedIn(t *testing.T) {
	t.Parallel()
	db, v := indexed(t, map[string]string{
		"source.md":  "Points at [[entropy]].\n",
		"Entropy.md": "# Entropy\n",
	})

	c := links(t, db, v, "source.md")
	if c.Links[0].To != "Entropy.md" {
		t.Errorf("resolved to %q — people type lowercase", c.Links[0].To)
	}
}
