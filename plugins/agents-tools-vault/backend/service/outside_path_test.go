package service_test

import (
	"strings"
	"testing"
)

func TestNoteReadAndFileCreateOfAPathOutsideTheVaultAnswerWithoutTheHostPrefix(t *testing.T) {
	_, registry := newGeologyVault(t)
	for name, arguments := range map[string]string{
		"note_read":   `{"path":"../outside.md"}`,
		"file_create": `{"path":"../outside.md","content":"Pumice floats."}`,
	} {
		text := getRefusalText(t, registry, geologyVault, name, arguments)
		if !strings.HasPrefix(text, name+": ") {
			t.Errorf("the refusal of %s is %q, want it to start with %q", name, text, name+": ")
		}
		if !strings.Contains(text, "the path is outside the vault") {
			t.Errorf("the refusal of %s is %q, want it to say the path is outside the vault", name, text)
		}
		if strings.Contains(text, "host:") {
			t.Errorf("the refusal of %s is %q, which contains host:", name, text)
		}
	}
}
