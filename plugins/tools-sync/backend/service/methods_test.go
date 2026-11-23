package service

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	sessiondomain "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/session"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
)

func TestGetCode_MapsTheErrorsOfTheSubscriptionMethodsToTheCodesTheFrontendReads(t *testing.T) {
	for name, one := range map[string]struct {
		err  error
		want server.ErrorCode
	}{
		"server away":            {err: syncprotocol.ErrUnreachable, want: server.CodeUnavailable},
		"session unreachable":    {err: sessiondomain.ErrUnreachable, want: server.CodeUnavailable},
		"already subscribed":     {err: fmt.Errorf("checkout: %w", syncprotocol.ErrAlreadySubscribed), want: server.CodeAlreadyExists},
		"billing unavailable":    {err: syncprotocol.ErrBillingUnavailable, want: server.CodeFailedPrecondition},
		"no identity":            {err: sessiondomain.ErrNoIdentity, want: server.CodeFailedPrecondition},
		"no active subscription": {err: syncprotocol.ErrNoActiveSubscription, want: server.CodeFailedPrecondition},
		"account deleting":       {err: sessiondomain.AccountDeletingError{}, want: server.CodeFailedPrecondition},
		"same plan":              {err: fmt.Errorf("change: %w", syncprotocol.ErrSamePlan), want: server.CodeAlreadyExists},
		"unknown plan":           {err: syncprotocol.ErrUnknownPlan, want: server.CodeInvalidArgument},
		"another error":          {err: errors.New("broken"), want: server.CodeInternal},
	} {
		if got := getCode(one.err); got != one.want {
			t.Errorf("%s: code = %q, want %q", name, got, one.want)
		}
	}
}

func TestGetCodeAndGetText_DistinguishShareErrorsFromOtherErrors(t *testing.T) {
	for name, one := range map[string]struct {
		err      error
		wantCode server.ErrorCode
		wantText string
	}{
		"own identity":          {err: fmt.Errorf("add: %w", share.ErrOwnIdentity), wantCode: server.CodeFailedPrecondition, wantText: "That is your own identity"},
		"inside a share":        {err: share.ErrInsideShare, wantCode: server.CodeFailedPrecondition, wantText: "A folder inside a shared folder cannot be shared"},
		"share ended":           {err: share.ErrEnded, wantCode: server.CodeFailedPrecondition, wantText: "The owner ended this share"},
		"not a code":            {err: fmt.Errorf("%w: base64", share.ErrBadCode), wantCode: server.CodeInvalidArgument, wantText: "this is not a reader code or an invitation: base64"},
		"no such share":         {err: share.ErrUnknownShare, wantCode: server.CodeNotFound, wantText: share.ErrUnknownShare.Error()},
		"another error":         {err: errors.New("broken"), wantCode: server.CodeInternal, wantText: "broken"},
		"changed identity":      {err: share.ErrChangedIdentity, wantCode: server.CodeFailedPrecondition, wantText: "This person has another identity than before"},
		"refused by the server": {err: share.ErrReaderNotAdded, wantCode: server.CodeFailedPrecondition, wantText: "The reader could not be added; the reader may not have signed in to Numen Sync yet"},
	} {
		if got := getCode(one.err); got != one.wantCode {
			t.Errorf("%s: code = %q, want %q", name, got, one.wantCode)
		}
		if got := getText(one.err); got != one.wantText {
			t.Errorf("%s: text = %q, want %q", name, got, one.wantText)
		}
	}
}
