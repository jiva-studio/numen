package identity

import (
	"context"
	"errors"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
)

// ErrCodeFormat means the confirmation code is not six digits.
var ErrCodeFormat = errors.New("the confirmation code has six digits")

// EnterLinkCode sends the confirmation code the person read on the host.
type EnterLinkCode struct {
	links linkHold
	clock Clock
}

// NewEnterLinkCode returns the use case that sends the confirmation code.
func NewEnterLinkCode(d Deps) *EnterLinkCode {
	return &EnterLinkCode{links: d.progress.links, clock: d.clock}
}

// Execute sends the code of six digits to the host. It returns ErrNoLink when this device does not ask for a code.
func (u *EnterLinkCode) Execute(_ context.Context, code string) error {
	store, release := u.links.hold()
	defer release()
	store.expireLocked(u.clock.Now())
	link := store.current
	if link == nil || link.progress.Role != identitydomain.LinkRoleJoiner || link.progress.Phase != identitydomain.LinkPhaseEnterCode {
		return identitydomain.ErrNoLink
	}
	if !isCode(code) {
		return ErrCodeFormat
	}
	store.sendLocked(link, encodeCode(code))
	store.setPhaseLocked(link, identitydomain.LinkPhaseWaitingForAccept)
	return nil
}

func isCode(code string) bool {
	if len(code) != codeDigits {
		return false
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
