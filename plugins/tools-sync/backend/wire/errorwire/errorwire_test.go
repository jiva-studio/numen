package errorwire

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	sessiondomain "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/session"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
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
		if got := GetCode(one.err); got != one.want {
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
		"own identity":   {err: fmt.Errorf("add: %w", share.ErrOwnIdentity), wantCode: server.CodeFailedPrecondition, wantText: "That is your own identity"},
		"inside a share": {err: share.ErrInsideShare, wantCode: server.CodeFailedPrecondition, wantText: "A folder inside a shared folder cannot be shared"},
		"a share in the way": {err: fmt.Errorf("share: %w", share.NestingError{Kind: share.ErrHoldsShare, Folder: "", Shared: "Recipes"}),
			wantCode: server.CodeFailedPrecondition, wantText: "The vault cannot be shared while its folder «Recipes» is shared"},
		"share ended":           {err: share.ErrEnded, wantCode: server.CodeFailedPrecondition, wantText: "The owner ended this share"},
		"not a code":            {err: fmt.Errorf("%w: base64", share.ErrBadCode), wantCode: server.CodeInvalidArgument, wantText: "This is not a reader code or an invitation."},
		"no such share":         {err: share.ErrUnknownShare, wantCode: server.CodeNotFound, wantText: "This share does not exist or is not meant for you."},
		"another error":         {err: errors.New("broken"), wantCode: server.CodeInternal, wantText: GenericText},
		"changed identity":      {err: share.ErrChangedIdentity, wantCode: server.CodeFailedPrecondition, wantText: "This person has another identity than before"},
		"refused by the server": {err: share.ErrReaderNotAdded, wantCode: server.CodeFailedPrecondition, wantText: "The reader could not be added; the reader may not have signed in to Numen Sync yet"},
	} {
		if got := GetCode(one.err); got != one.wantCode {
			t.Errorf("%s: code = %q, want %q", name, got, one.wantCode)
		}
		if got := GetText(one.err); got != one.wantText {
			t.Errorf("%s: text = %q, want %q", name, got, one.wantText)
		}
	}
}

func TestDescribe_ReplacesAnUnknownErrorWithTheGenericTextAndLogsTheChain(t *testing.T) {
	var logged bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logged, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	for name, err := range map[string]error{
		"a file name":      fmt.Errorf("write journal-catalog.json: %w", errors.New("open /home/anna/.numen/x: permission denied")),
		"a refused socket": errors.New("dial tcp 127.0.0.1:8080: connect: connection refused"),
		"a Go chain":       fmt.Errorf("sync vault 3f9a: %w", fmt.Errorf("pass: %w", errors.New("internal error"))),
	} {
		logged.Reset()
		code, text := NewFailures().Describe(t.Context(), "connect-vault", err)
		if text != GenericText || code != server.CodeInternal {
			t.Errorf("%s: got %q %q, want the generic text", name, code, text)
		}
		if line := logged.String(); !strings.Contains(line, "connect-vault") || !strings.Contains(line, "level=WARN") || !strings.Contains(line, "error=") {
			t.Errorf("%s: log = %q, want a warning with the operation and the chain", name, line)
		}
	}
}

func TestDescribe_HidesFileNames(t *testing.T) {
	wrapped := fmt.Errorf("write journal-catalog.json: %w", host.ErrNotExist)
	_, text := NewFailures().Describe(t.Context(), "connect-vault", wrapped)
	if strings.Contains(text, "journal-catalog.json") || strings.Contains(text, "file does not exist") {
		t.Errorf("text = %q, want no file name and no Go text", text)
	}
}

func TestDescribe_ExplainsFirstSyncFailures(t *testing.T) {
	for name, one := range map[string]struct {
		err  error
		want string
	}{
		"folder gone": {
			err:  fmt.Errorf("connect: %w", vaultlink.FolderMissingError{Name: "Recipes", Path: "/home/anna/Recipes"}),
			want: "The folder of the vault «Recipes» is missing: /home/anna/Recipes. Remove the vault from the list or restore the folder.",
		},
		"server away": {err: fmt.Errorf("get catalog: %w", syncprotocol.ErrUnreachable), want: "Can't connect right now. Check your internet connection and try again."},
		"no plan":     {err: fmt.Errorf("append: %w", syncprotocol.ErrWriteNotAllowed), want: "Your plan does not allow syncing: choose a plan."},
		"quota":       {err: fmt.Errorf("append: %w", syncprotocol.QuotaExceededError{Used: 5, Limit: 4}), want: textStorageFull},
		"no vaults":   {err: fmt.Errorf("write: %w", vaultlink.ErrNoVaults), want: "There is no vault with a folder to sync. Add a vault, or restore the folder of one."},
		"no identity": {err: sessiondomain.ErrNoIdentity, want: "Sync is not set up on this computer. Create an identity or restore one in the settings."},
		"no server":   {err: sessiondomain.ErrNoServer, want: "No sync server is set. Enter the server address in the settings."},
	} {
		code, text := NewFailures().Describe(t.Context(), "connect-vault", one.err)
		if text != one.want {
			t.Errorf("%s: text = %q, want %q", name, text, one.want)
		}
		if code == server.CodeInternal {
			t.Errorf("%s: code = %q, want a code of a known failure", name, code)
		}
	}
}

func TestDescribe_LogsTheSameFailureOfAnOperationOnce(t *testing.T) {
	var logged bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logged, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	failure := errors.New("the identity of this device could not be read")
	failures := NewFailures()

	failures.Describe(t.Context(), "get-status-once", failure)
	failures.Describe(t.Context(), "get-status-once", failure)
	failures.Describe(t.Context(), "get-status-once", failure)
	failures.Describe(t.Context(), "get-status-once", errors.New("another failure"))

	if lines := strings.Count(logged.String(), "level=WARN"); lines != 2 {
		t.Errorf("logged %d warnings, want 2: one for each distinct failure\n%s", lines, logged.String())
	}
}

func TestGetCodeAndGetText_AnswerAStoppedOrCanceledTransferWithAPlainSentence(t *testing.T) {
	for name, one := range map[string]struct {
		err      error
		wantText string
	}{
		"stalled":   {err: fmt.Errorf("download a range: %w", syncclient.ErrTransferStalled), wantText: "The sync server stopped sending. Try again."},
		"cut short": {err: fmt.Errorf("stopped: %w", io.ErrUnexpectedEOF), wantText: "The sync server stopped sending. Try again."},
		"canceled": {err: fmt.Errorf("download: %w", context.Canceled), wantText: "The transfer was canceled."},
		"too late":  {err: context.DeadlineExceeded, wantText: "The sync server stopped sending. Try again."},
	} {
		if got := GetCode(one.err); got != server.CodeUnavailable {
			t.Errorf("%s: code = %q, want %q", name, got, server.CodeUnavailable)
		}
		if got := GetText(one.err); got != one.wantText {
			t.Errorf("%s: text = %q, want %q", name, got, one.wantText)
		}
	}
}
