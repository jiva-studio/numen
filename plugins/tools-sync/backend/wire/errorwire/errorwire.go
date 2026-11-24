// Package errorwire turns the errors of the methods into the codes and texts the frontend reads.
package errorwire

import (
	"errors"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	sessiondomain "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/session"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/fetchfile"
)

// GetCode returns the code of the error an API method reports.
func GetCode(err error) server.ErrorCode {
	switch {
	case errors.Is(err, syncprotocol.ErrUnreachable), errors.Is(err, sessiondomain.ErrUnreachable):
		return server.CodeUnavailable
	case errors.Is(err, syncprotocol.ErrAlreadySubscribed), errors.Is(err, syncprotocol.ErrSamePlan):
		return server.CodeAlreadyExists
	case errors.Is(err, syncprotocol.ErrUnknownPlan), errors.Is(err, share.ErrBadCode):
		return server.CodeInvalidArgument
	case errors.Is(err, share.ErrUnknownShare), errors.Is(err, fetchfile.ErrBlobMissing), errors.Is(err, host.ErrNotExist):
		return server.CodeNotFound
	case errors.Is(err, host.ErrInvalidRange):
		return server.CodeInvalidArgument
	case errors.Is(err, host.ErrChanged), errors.Is(err, fetchfile.ErrNotAbsent), errors.Is(err, fetchfile.ErrHeadChanged):
		return server.CodeFailedPrecondition
	case isRefusal(err):
		return server.CodeFailedPrecondition
	case isAccountDeleting(err):
		return server.CodeFailedPrecondition
	case errors.Is(err, sessiondomain.ErrNoIdentity), errors.Is(err, syncprotocol.ErrBillingUnavailable), errors.Is(err, syncprotocol.ErrNoActiveSubscription):
		return server.CodeFailedPrecondition
	}
	return server.CodeInternal
}

// GetText returns the text of the error an API method reports; a refusal of a share starts with a capital.
func GetText(err error) string {
	refusal := findRefusal(err)
	if refusal == nil {
		return err.Error()
	}
	text := refusal.Error()
	return strings.ToUpper(text[:1]) + text[1:]
}

func getRefusals() []error {
	return []error{
		share.ErrWholeVault, share.ErrInsideShare, share.ErrHoldsShare, share.ErrNotConnected, share.ErrReceivedVault,
		share.ErrReceivedNotSynced, share.ErrOwnIdentity, share.ErrAlreadyReader, share.ErrChangedIdentity, share.ErrReaderNotAdded,
		share.ErrNotForThisAccount, share.ErrAlreadyReceived, share.ErrOwnerChanged, share.ErrEnded, share.ErrDamagedLog, share.ErrNotReady,
		share.ErrNotReceived,
		share.ErrUnknownReader, share.ErrNotShared, share.ErrRolledBack, share.ErrBadLink, share.ErrLinkExpired, share.ErrLinkOtherServer,
		share.ErrLinkUsed, share.ErrUnknownRequest, share.ErrLinkDeclined, share.ErrLinkMismatch,
	}
}

func isAccountDeleting(err error) bool {
	return errors.Is(err, sessiondomain.AccountDeletingError{DeleteAt: time.Time{}})
}

func isRefusal(err error) bool { return findRefusal(err) != nil }

func findRefusal(err error) error {
	for _, refusal := range getRefusals() {
		if errors.Is(err, refusal) {
			return refusal
		}
	}
	return nil
}
