package webui_test

import (
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

const identified = "01M02ACGM0FYMSXNDP29C90JNR"

// reached is where each address landed, by what was asked about.
func reached(t *testing.T, answer *v1.ResolveAddressesResponse) map[string]*v1.ResolvedAddress {
	t.Helper()
	by := map[string]*v1.ResolvedAddress{}
	for _, one := range answer.GetResolved() {
		by[one.GetWritten()] = one
	}
	return by
}

// A window asks where a link written in an answer or in a note lands, and the
// answer is the one every link in the vault is answered with.
func TestAnAddressIsAnsweredWithTheNoteItReaches(t *testing.T) {
	client, _ := opened(t, map[string]string{
		"physics/Entropy.md": "---\ntitle: Entropy\nid: " + identified + "\n---\n\nheat\n",
		"Note.md":            "---\ntitle: Note\n---\n\nprose\n",
	})

	answer, err := client.ResolveAddresses(t.Context(), connect.NewRequest(&v1.ResolveAddressesRequest{
		From:    "Note.md",
		Written: []string{"name://Entropy", "note://" + identified, "name://Nowhere"},
	}))
	if err != nil {
		t.Fatal(err)
	}

	landed := reached(t, answer.Msg)
	for _, written := range []string{"name://Entropy", "note://" + identified} {
		one := landed[written]
		if one.GetPath() != "physics/Entropy.md" {
			t.Errorf("%q lands at %q", written, one.GetPath())
		}
		if one.GetVault() == "" || one.GetCrossed() {
			t.Errorf("%q lands in vault %q, crossed %v", written, one.GetVault(), one.GetCrossed())
		}
		if one.GetAmbiguous() {
			t.Errorf("%q is called ambiguous", written)
		}
	}
	if one, found := landed["name://Nowhere"]; found {
		t.Errorf("a name no note answers to lands at %q", one.GetPath())
	}
}

// A name resolves by a path relative to the note it is written in, so the same
// name written in two folders reaches two notes.
func TestANameReachesTheNoteBesideTheOneItIsWrittenIn(t *testing.T) {
	client, _ := opened(t, map[string]string{
		"heat/Entropy.md":  "---\ntitle: Entropy\n---\n\nheat\n",
		"heat/Note.md":     "---\ntitle: Note\n---\n\nprose\n",
		"order/Entropy.md": "---\ntitle: Entropy\n---\n\norder\n",
		"order/Note.md":    "---\ntitle: Note\n---\n\nprose\n",
	})

	for from, want := range map[string]string{
		"heat/Note.md":  "heat/Entropy.md",
		"order/Note.md": "order/Entropy.md",
	} {
		answer, err := client.ResolveAddresses(t.Context(), connect.NewRequest(&v1.ResolveAddressesRequest{
			From: from, Written: []string{"name://Entropy"},
		}))
		if err != nil {
			t.Fatal(err)
		}

		if got := reached(t, answer.Msg)["name://Entropy"].GetPath(); got != want {
			t.Errorf("written in %s it reaches %q, want %q", from, got, want)
		}
	}
}

// A name several notes answer to reaches the nearest of them, and says that it
// was more than one note's name.
func TestANameSeveralNotesAnswerToIsReported(t *testing.T) {
	client, _ := opened(t, map[string]string{
		"heat/Entropy.md":    "---\ntitle: Entropy\n---\n\nheat\n",
		"order/Entropy.md":   "---\ntitle: Entropy\n---\n\norder\n",
		"heat/steam/Note.md": "---\ntitle: Note\n---\n\nprose\n",
	})

	answer, err := client.ResolveAddresses(t.Context(), connect.NewRequest(&v1.ResolveAddressesRequest{
		From: "heat/steam/Note.md", Written: []string{"name://Entropy"},
	}))
	if err != nil {
		t.Fatal(err)
	}

	one := reached(t, answer.Msg)["name://Entropy"]
	if one.GetPath() != "heat/Entropy.md" {
		t.Errorf("it reaches %q, want the nearest in the tree", one.GetPath())
	}
	if !one.GetAmbiguous() {
		t.Error("a name two notes answer to is not called ambiguous")
	}
}

// An answer is written in no note, so it asks with no note to be relative to,
// and a name carrying dots is one name.
func TestANameWrittenInNoNoteReachesTheNoteItNames(t *testing.T) {
	const lecture = "Seminar 1.2–1.3 — Lisbon, 9 July 1973"
	client, _ := opened(t, map[string]string{
		"notes/" + lecture + ".md": "---\ntitle: " + lecture + "\n---\n\nprose\n",
	})

	answer, err := client.ResolveAddresses(t.Context(), connect.NewRequest(&v1.ResolveAddressesRequest{
		From: "", Written: []string{"name://" + lecture},
	}))
	if err != nil {
		t.Fatal(err)
	}

	one := reached(t, answer.Msg)["name://"+lecture]
	if one.GetPath() != "notes/"+lecture+".md" {
		t.Errorf("it reaches %q", one.GetPath())
	}
	if one.GetAmbiguous() {
		t.Error("one note answers to that name and it was called ambiguous")
	}
}

// An address asked about twice is one question, so a caller reading the answer
// by what it wrote finds one entry.
func TestAnAddressAskedTwiceIsAnsweredOnce(t *testing.T) {
	client, _ := opened(t, map[string]string{
		"Entropy.md": "---\ntitle: Entropy\n---\n\nheat\n",
	})

	answer, err := client.ResolveAddresses(t.Context(), connect.NewRequest(&v1.ResolveAddressesRequest{
		Written: []string{"name://Entropy", "name://Entropy"},
	}))
	if err != nil {
		t.Fatal(err)
	}

	if got := answer.Msg.GetResolved(); len(got) != 1 {
		t.Fatalf("reached = %+v, want the one address asked about", got)
	}
}
