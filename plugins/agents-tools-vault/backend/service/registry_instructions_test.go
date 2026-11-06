package service_test

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/service"
)

func TestTheVaultToolsListOneParagraphOfInstructions(t *testing.T) {
	_, svc := setupTestVault()
	registry, err := svc.Registry()
	if err != nil {
		t.Fatal(err)
	}

	listing, err := registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	instructions := listing.Instructions
	if strings.Contains(instructions, "\n\n") {
		t.Errorf("the instructions are more than one paragraph:\n%s", instructions)
	}
	for _, want := range []string{
		"[[",
		"[[note://",
		"file://",
		"`at`",
		"numen:",
		"start=",
		"length=",
		"library_search",
		"note_search",
		"library_read",
		"mid-sentence",
		"kinds",
		`"file"`,
		"note_list",
	} {
		if !strings.Contains(instructions, want) {
			t.Errorf("the instructions do not say %q:\n%s", want, instructions)
		}
	}
	if !strings.Contains(instructions, service.ArrangeInstructions) {
		t.Errorf("the instructions leave out the arranging of notes:\n%s", instructions)
	}
}
