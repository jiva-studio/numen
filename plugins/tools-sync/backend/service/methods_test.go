package service

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	sessiondomain "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/session"
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
		"same plan":              {err: fmt.Errorf("change: %w", syncprotocol.ErrSamePlan), want: server.CodeAlreadyExists},
		"unknown plan":           {err: syncprotocol.ErrUnknownPlan, want: server.CodeInvalidArgument},
		"another error":          {err: errors.New("broken"), want: server.CodeInternal},
	} {
		if got := getCode(one.err); got != one.want {
			t.Errorf("%s: code = %q, want %q", name, got, one.want)
		}
	}
}
