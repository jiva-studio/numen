package webui_test

import (
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

const identified = "01M02ACGM0FYMSXNDP29C90JNR"

// A window asks where a link written in an answer or in a note lands, and the
// answer is the one every link in the vault is answered with.
func TestAnAddressIsAnsweredWithTheNoteItReaches(t *testing.T) {
	client, _ := opened(t, map[string]string{
		"physics/Entropy.md": "---\ntitle: Entropy\nid: " + identified + "\n---\n\nheat\n",
		"Note.md":            "---\ntitle: Note\n---\n\nprose\n",
	})

	answer, err := client.Resolve(t.Context(), connect.NewRequest(&v1.ResolveRequest{
		From:    "Note.md",
		Written: []string{"Entropy", "note://" + identified, "Nowhere"},
	}))
	if err != nil {
		t.Fatal(err)
	}

	landed := map[string]string{}
	for _, one := range answer.Msg.GetLanded() {
		landed[one.GetWritten()] = one.GetPath()
	}
	want := map[string]string{
		"Entropy":              "physics/Entropy.md",
		"note://" + identified: "physics/Entropy.md",
	}
	for written, path := range want {
		if landed[written] != path {
			t.Errorf("%q lands at %q, want %q", written, landed[written], path)
		}
	}
	if _, reached := landed["Nowhere"]; reached {
		t.Errorf("a name no note answers to lands at %q", landed["Nowhere"])
	}
}

// An address asked about twice is one question, so a caller reading the answer
// by what it wrote finds one entry.
func TestAnAddressAskedTwiceIsAnsweredOnce(t *testing.T) {
	client, _ := opened(t, map[string]string{
		"Entropy.md": "---\ntitle: Entropy\n---\n\nheat\n",
	})

	answer, err := client.Resolve(t.Context(), connect.NewRequest(&v1.ResolveRequest{
		Written: []string{"Entropy", "Entropy"},
	}))
	if err != nil {
		t.Fatal(err)
	}

	if got := answer.Msg.GetLanded(); len(got) != 1 {
		t.Fatalf("landed = %+v, want the one address asked about", got)
	}
}
