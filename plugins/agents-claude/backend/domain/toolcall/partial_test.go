package toolcall_test

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/agents-claude/backend/domain/toolcall"
)

func TestTheTextOfAFieldIsReadAsFarAsItHasArrived(t *testing.T) {
	cases := []struct{ arguments, want string }{
		{`{"path":"Garden/Beds.md","text":"# Beds\n\nRaised beds of oak`, "# Beds\n\nRaised beds of oak"},
		{`{"path":"Garden/Beds.md","text":"Tomatoes \"cherry\""}`, `Tomatoes "cherry"`},
		{`{"path":"Garden/Beds.md","text":"Tomatoes \`, "Tomatoes "},
		{`{"path":"Garden/Beds.md","text":"Tomatoes \u00`, "Tomatoes "},
		{"{\"path\":\"Garden/Beds.md\",\"text\":\"Beet \xc3", "Beet "},
		{`{"path":"Garden/Beds.md","te`, ""},
		{`{"path":"Garden/Beds.md","text":4}`, ""},
	}
	for _, one := range cases {
		if got := toolcall.ReadPartialText(one.arguments, "text"); got != one.want {
			t.Errorf("ReadPartialText(%q) = %q, want %q", one.arguments, got, one.want)
		}
	}
}
