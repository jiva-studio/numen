// Package errorwire turns the errors of the methods into the codes and texts the frontend reads. The person reads a
// plain sentence only: an error that is not in the tables gets one generic sentence, and its chain goes to the log.
package errorwire

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	sessiondomain "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/session"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/fetchfile"
)

// GenericText is what the person reads for a failure that has no entry in the tables.
const GenericText = "Something went wrong in sync. Try again; if it fails again, look at the log of the plugin."

const (
	textUnreachable = "Can't connect right now. Check your internet connection and try again."
	textNoPlan      = "Your plan does not allow syncing: choose a plan."
	textStalled     = "The sync server stopped sending. Try again."
	textStorageFull = "The cloud storage is full: changes stay on this computer. Free some space or choose a bigger plan."
	textSignIn      = "The server did not accept the sign-in of this computer. Try again later."
	textUpdate      = "The sync server needs a newer version of Numen. Update Numen."
)

// Describe returns the code and the plain text of the failure of an operation. It writes the full error to the log
// when the failure has no plain text of its own.
func (f *Failures) Describe(ctx context.Context, operation string, err error) (server.ErrorCode, string) {
	text, isKnown := findText(err)
	if !isKnown {
		if f.isNew(operation, err) {
			slog.WarnContext(ctx, "a method of Numen Sync failed", "operation", operation, "error", err)
		}
		text = GenericText
	}
	return GetCode(err), text
}

// GetText returns the plain text of the error: its entry of the tables, or GenericText.
func GetText(err error) string {
	if text, isKnown := findText(err); isKnown {
		return text
	}
	return GenericText
}

func findText(err error) (string, bool) {
	var nesting share.NestingError
	var missing vaultlink.FolderMissingError
	var quota syncprotocol.QuotaExceededError
	switch {
	case errors.As(err, &nesting):
		return convertFirstToUpper(nesting.Error()), true
	case errors.As(err, &missing):
		return "The folder of the vault «" + missing.Name + "» is missing: " + missing.Path + ". Remove the vault from the list or restore the folder.", true
	case errors.As(err, &quota):
		return textStorageFull, true
	case errors.As(err, new(sessiondomain.DeviceRevokedError)):
		return "This computer was removed from the identity. Sign in again in the settings.", true
	}
	for _, one := range getEntries() {
		if errors.Is(err, one.target) {
			return one.text, true
		}
	}
	if refusal := findRefusal(err); refusal != nil {
		return convertFirstToUpper(refusal.Error()), true
	}
	return "", false
}

// GetCode returns the code of the error an API method reports.
func GetCode(err error) server.ErrorCode {
	var quota syncprotocol.QuotaExceededError
	var missing vaultlink.FolderMissingError
	if errors.As(err, &quota) || errors.As(err, &missing) {
		return server.CodeFailedPrecondition
	}
	for _, one := range getEntries() {
		if errors.Is(err, one.target) {
			return one.code
		}
	}
	if findRefusal(err) != nil {
		return server.CodeFailedPrecondition
	}
	return server.CodeInternal
}

func findRefusal(err error) error {
	for _, refusal := range getRefusals() {
		if errors.Is(err, refusal) {
			return refusal
		}
	}
	return nil
}

func convertFirstToUpper(text string) string { return strings.ToUpper(text[:1]) + text[1:] }

type entry struct {
	target error
	code   server.ErrorCode
	text   string
}

func getEntries() []entry {
	return []entry{
		{syncprotocol.ErrUnreachable, server.CodeUnavailable, textUnreachable},
		{syncclient.ErrTransferStalled, server.CodeUnavailable, textStalled},
		{io.ErrUnexpectedEOF, server.CodeUnavailable, textStalled},
		{context.Canceled, server.CodeUnavailable, "The transfer was canceled."},
		{context.DeadlineExceeded, server.CodeUnavailable, textStalled},
		{sessiondomain.ErrUnreachable, server.CodeUnavailable, textUnreachable},
		{sessiondomain.ErrNoServer, server.CodeUnavailable, "No sync server is set. Enter the server address in the settings."},
		{syncprotocol.ErrInsecureAddress, server.CodeInvalidArgument, "The server address must start with https://, or be an http address on this computer. Check the server address in the settings."},
		{syncprotocol.ErrAlreadySubscribed, server.CodeAlreadyExists, "The account has a subscription already."},
		{syncprotocol.ErrSamePlan, server.CodeAlreadyExists, "The account has this plan already."},
		{syncprotocol.ErrUnknownPlan, server.CodeInvalidArgument, "The server does not sell this plan. Choose another plan."},
		{share.ErrBadCode, server.CodeInvalidArgument, "This is not a reader code or an invitation."},
		{share.ErrUnknownShare, server.CodeNotFound, "This share does not exist or is not meant for you."},
		{fetchfile.ErrBlobMissing, server.CodeNotFound, "The sync server does not hold this file any more."},
		{host.ErrNotExist, server.CodeNotFound, "A file or a folder of the vault is gone. Check that the folder of the vault is there."},
		{host.ErrInvalidRange, server.CodeInvalidArgument, GenericText},
		{host.ErrChanged, server.CodeFailedPrecondition, "The file changed while it was read. Try again."},
		{fetchfile.ErrHeadChanged, server.CodeFailedPrecondition, "The file changed while it was fetched. Try again."},
		{fetchfile.ErrNotAbsent, server.CodeFailedPrecondition, "This file is on this computer already."},
		{sessiondomain.ErrNoIdentity, server.CodeFailedPrecondition, "Sync is not set up on this computer. Create an identity or restore one in the settings."},
		{syncprotocol.ErrBillingUnavailable, server.CodeFailedPrecondition, "This sync server does not sell subscriptions."},
		{syncprotocol.ErrNoActiveSubscription, server.CodeFailedPrecondition, textNoPlan},
		{syncprotocol.ErrWriteNotAllowed, server.CodeFailedPrecondition, textNoPlan},
		{sessiondomain.ErrUnauthenticated, server.CodeFailedPrecondition, textSignIn},
		{syncprotocol.ErrUnauthenticated, server.CodeFailedPrecondition, textSignIn},
		{sessiondomain.ErrUpdateRequired, server.CodeFailedPrecondition, textUpdate},
		{syncprotocol.ErrUpdateRequired, server.CodeFailedPrecondition, textUpdate},
		{syncprotocol.ErrRateLimited, server.CodeUnavailable, "The sync server is busy. Try again in a minute."},
		{syncprotocol.ErrNoVault, server.CodeNotFound, "The sync server has no such vault. Choose the vault again."},
		{vaultlink.ErrNoVaults, server.CodeFailedPrecondition, "There is no vault with a folder to sync. Add a vault, or restore the folder of one."},
	}
}

func getRefusals() []error {
	return []error{
		share.ErrInsideShare, share.ErrHoldsShare, share.ErrNotConnected, share.ErrReceivedVault,
		share.ErrReceivedNotSynced, share.ErrOwnIdentity, share.ErrAlreadyReader, share.ErrChangedIdentity, share.ErrReaderNotAdded,
		share.ErrNotForThisAccount, share.ErrAlreadyReceived, share.ErrOwnerChanged, share.ErrEnded, share.ErrDamagedLog, share.ErrNotReady,
		share.ErrReceivedStop, share.ErrNotReceived,
		share.ErrUnknownReader, share.ErrNotShared, share.ErrRolledBack, share.ErrBadLink, share.ErrLinkExpired, share.ErrLinkOtherServer,
		share.ErrLinkUsed, share.ErrUnknownRequest, share.ErrFingerprintNotConfirmed, share.ErrLinkDeclined, share.ErrLinkMismatch,
	}
}

// Failures remembers the last error logged for each operation.
type Failures struct {
	mu   sync.Mutex
	last map[string]string
}

// NewFailures returns an empty Failures.
func NewFailures() *Failures { return &Failures{mu: sync.Mutex{}, last: map[string]string{}} }

func (f *Failures) isNew(operation string, err error) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.last[operation] == err.Error() {
		return false
	}
	f.last[operation] = err.Error()
	return true
}
