package service_test

import (
	"strings"
	"testing"
)

func TestNoteReadAndFileCreateOutsideVaultOmitTheHostPrefix(t *testing.T) {
	_, registry := newGeologyVault(t)
	for name, arguments := range map[string]string{
		"note_read":   `{"path":"../outside.md"}`,
		"file_create": `{"path":"../outside.md","text":"Pumice floats."}`,
	} {
		text := getRejectionText(t, registry, geologyVault, name, arguments)
		if !strings.HasPrefix(text, name+": ") {
			t.Errorf("the error of %s is %q, want it to start with %q", name, text, name+": ")
		}
		if !strings.Contains(text, "the path is outside the vault") {
			t.Errorf("the error of %s is %q, want it to state that the path is outside the vault", name, text)
		}
		if strings.Contains(text, "host:") {
			t.Errorf("the error of %s is %q, which contains host:", name, text)
		}
	}
}
