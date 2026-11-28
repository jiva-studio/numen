package identity_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/identity"
)

func TestChainMark_HasAtLeastSixteenBytesOfTheHash(t *testing.T) {
	t.Parallel()
	m := newTeam(t)

	text := m.chain.GetMark().String()

	_, head, _ := strings.Cut(text, "-")
	if len(head) < 32 {
		t.Errorf("the mark %q has %d hex digits of hash, want at least 32", text, len(head))
	}
}

func TestParseMark_RefusesAMarkWithAShortHead(t *testing.T) {
	t.Parallel()

	_, err := identity.ParseMark("3-3FA912C077B1")

	if !errors.Is(err, identity.ErrMalformed) {
		t.Errorf("ParseMark of a 6-byte head = %v, want ErrMalformed", err)
	}
}
