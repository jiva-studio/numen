package identity

import (
	"fmt"
	"testing"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
)

func TestGetLinkError_OmitsTheTextOfTheError(t *testing.T) {
	t.Parallel()
	err := fmt.Errorf("%w: read /home/person/.config/numen/key: permission denied", identitydomain.ErrNoLink)

	code, detail, served := getLinkRefusal(err)

	if !served || code == 0 {
		t.Fatalf("code = %v, served = %v, want a served refusal", code, served)
	}
	if detail != "" {
		t.Errorf("detail = %q, want none", detail)
	}
}
