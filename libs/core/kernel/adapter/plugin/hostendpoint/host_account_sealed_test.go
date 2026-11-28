package hostendpoint

import (
	"bytes"
	"errors"
	"testing"

	"connectrpc.com/connect"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

const maxSealedData = 32 << 10

func TestAPluginSealsDataWithTheContextItChoosesForItsOwnID(t *testing.T) {
	h := newAccountHost(t)
	h.operations.sealed = []byte("sealed bytes")

	response, err := h.client("token-c").SealData(t.Context(), connect.NewRequest(&v1.SealDataRequest{Context: []byte("numen-sync/catalog/v1"), Plain: []byte("Shopping list")}))
	if err != nil {
		t.Fatal(err)
	}

	if string(response.Msg.GetSealed()) != "sealed bytes" {
		t.Errorf("sealed = %q, want what the operation returned", response.Msg.GetSealed())
	}
	if len(h.operations.pluginIDs) != 1 || h.operations.pluginIDs[0] != "md.numen.c" {
		t.Errorf("plugin ids = %v, want [md.numen.c]: the id of the token", h.operations.pluginIDs)
	}
	if len(h.operations.dataContexts) != 1 || string(h.operations.dataContexts[0]) != "numen-sync/catalog/v1" || string(h.operations.plains[0]) != "Shopping list" {
		t.Errorf("operation saw context %q and plain %q", h.operations.dataContexts, h.operations.plains)
	}
}

func TestAPluginOpensDataWithTheContextItChoosesForItsOwnID(t *testing.T) {
	h := newAccountHost(t)
	h.operations.opened = []byte("Shopping list")

	response, err := h.client("token-c").OpenData(t.Context(), connect.NewRequest(&v1.OpenDataRequest{Context: []byte("numen-sync/catalog/v1"), Sealed: []byte("sealed bytes")}))
	if err != nil {
		t.Fatal(err)
	}

	if string(response.Msg.GetPlain()) != "Shopping list" {
		t.Errorf("plain = %q, want what the operation returned", response.Msg.GetPlain())
	}
	if len(h.operations.pluginIDs) != 1 || h.operations.pluginIDs[0] != "md.numen.c" {
		t.Errorf("plugin ids = %v, want [md.numen.c]: the id of the token", h.operations.pluginIDs)
	}
	if string(h.operations.dataContexts[0]) != "numen-sync/catalog/v1" || string(h.operations.plains[0]) != "sealed bytes" {
		t.Errorf("operation saw context %q and sealed %q", h.operations.dataContexts, h.operations.plains)
	}
}

func TestSealedDataCallsNeedThePermissionIdentity(t *testing.T) {
	h := newAccountHost(t)
	h.callers.granted = []string{manifest.PermissionVaultRead}
	client := h.client("token-a")

	_, sealErr := client.SealData(t.Context(), connect.NewRequest(&v1.SealDataRequest{Plain: []byte("m")}))
	_, openErr := client.OpenData(t.Context(), connect.NewRequest(&v1.OpenDataRequest{Sealed: []byte("m")}))

	for name, err := range map[string]error{"seal": sealErr, "open": openErr} {
		if connect.CodeOf(err) != connect.CodePermissionDenied {
			t.Errorf("%s: error = %v, want permission denied", name, err)
		}
	}
	if h.operations.callCount() != 0 {
		t.Error("an identity operation ran for a plugin without the permission")
	}
}

func TestAnUnknownTokenIsUnauthenticatedForTheSealedDataAndRestoreCalls(t *testing.T) {
	h := newAccountHost(t)
	client := h.client("token-of-nobody")

	_, sealErr := client.SealData(t.Context(), connect.NewRequest(&v1.SealDataRequest{Plain: []byte("m")}))
	_, completeErr := client.CompleteRestore(t.Context(), connect.NewRequest(&v1.CompleteRestoreRequest{}))

	if connect.CodeOf(sealErr) != connect.CodeUnauthenticated || connect.CodeOf(completeErr) != connect.CodeUnauthenticated {
		t.Errorf("seal = %v, complete = %v, want both unauthenticated", sealErr, completeErr)
	}
}

func TestSealDataAndOpenDataRefuseMoreThan32KiBAndAcceptExactly32KiB(t *testing.T) {
	h := newAccountHost(t)
	client := h.client("token-a")

	_, overSeal := client.SealData(t.Context(), connect.NewRequest(&v1.SealDataRequest{Plain: bytes.Repeat([]byte{1}, maxSealedData+1)}))
	_, overOpen := client.OpenData(t.Context(), connect.NewRequest(&v1.OpenDataRequest{Sealed: bytes.Repeat([]byte{1}, 2*maxSealedData)}))
	_, exactSeal := client.SealData(t.Context(), connect.NewRequest(&v1.SealDataRequest{Plain: bytes.Repeat([]byte{1}, maxSealedData)}))

	if connect.CodeOf(overSeal) != connect.CodeInvalidArgument || connect.CodeOf(overOpen) != connect.CodeInvalidArgument {
		t.Errorf("seal = %v, open = %v, want both invalid argument", overSeal, overOpen)
	}
	if exactSeal != nil {
		t.Errorf("32 KiB: %v", exactSeal)
	}
	if h.operations.callCount() != 1 {
		t.Errorf("operations ran %d times, want once for the call within the limit", h.operations.callCount())
	}
}

func TestADataThatCannotBeOpenedAnswersAFailedPreconditionWithTheReasonCannotOpen(t *testing.T) {
	h := newAccountHost(t)
	h.operations.err = identitydomain.ErrCannotOpen

	_, err := h.client("token-a").OpenData(t.Context(), connect.NewRequest(&v1.OpenDataRequest{Context: []byte("c"), Sealed: []byte("sealed")}))

	if connect.CodeOf(err) != connect.CodeFailedPrecondition || readReason(err) != "CANNOT_OPEN" {
		t.Errorf("error = %v, reason %q, want failed precondition with CANNOT_OPEN", err, readReason(err))
	}
}

func TestADeviceWithoutAnIdentityAnswersNoIdentityToSealAndOpen(t *testing.T) {
	h := newAccountHost(t)
	h.operations.err = identitydomain.ErrNoIdentity
	client := h.client("token-a")

	_, sealErr := client.SealData(t.Context(), connect.NewRequest(&v1.SealDataRequest{Plain: []byte("m")}))
	_, openErr := client.OpenData(t.Context(), connect.NewRequest(&v1.OpenDataRequest{Sealed: []byte("m")}))

	for name, err := range map[string]error{"seal": sealErr, "open": openErr} {
		if connect.CodeOf(err) != connect.CodeFailedPrecondition || readReason(err) != "NO_IDENTITY" {
			t.Errorf("%s: error = %v, reason %q, want failed precondition with NO_IDENTITY", name, err, readReason(err))
		}
	}
}

func TestAKeyStoreFailureOfSealedDataIsInternalAndNamesNoReason(t *testing.T) {
	h := newAccountHost(t)
	h.operations.err = errors.New("the keychain refuses")
	client := h.client("token-a")

	_, sealErr := client.SealData(t.Context(), connect.NewRequest(&v1.SealDataRequest{Plain: []byte("m")}))
	_, openErr := client.OpenData(t.Context(), connect.NewRequest(&v1.OpenDataRequest{Sealed: []byte("m")}))

	for name, err := range map[string]error{"seal": sealErr, "open": openErr} {
		if connect.CodeOf(err) != connect.CodeInternal || readReason(err) != "" {
			t.Errorf("%s: error = %v, reason %q, want internal with no reason", name, err, readReason(err))
		}
	}
}

func TestSealedDataResponsesHoldOnlyTheDataAndNothingSecret(t *testing.T) {
	h := newAccountHost(t)
	client := h.client("token-a")

	sealed, sealErr := client.SealData(t.Context(), connect.NewRequest(&v1.SealDataRequest{Plain: []byte("m")}))
	opened, openErr := client.OpenData(t.Context(), connect.NewRequest(&v1.OpenDataRequest{Sealed: []byte("m")}))

	if sealErr != nil || openErr != nil {
		t.Fatalf("seal = %v, open = %v", sealErr, openErr)
	}
	if got := sealed.Msg.ProtoReflect().Descriptor().Fields().Len(); got != 1 {
		t.Errorf("the seal response has %d fields, want 1", got)
	}
	if got := opened.Msg.ProtoReflect().Descriptor().Fields().Len(); got != 1 {
		t.Errorf("the open response has %d fields, want 1", got)
	}
}

func TestAPluginReadsThePendingRestoreWithPublicKeysOnly(t *testing.T) {
	h := newAccountHost(t)
	h.operations.pending = identitydomain.PendingRestore{IsPending: true, RootSignKey: bytes.Repeat([]byte{1}, 32), DeviceSignKey: bytes.Repeat([]byte{2}, 32)}

	response, err := h.client("token-a").GetPendingRestore(t.Context(), connect.NewRequest(&v1.GetPendingRestoreRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	got := response.Msg
	if !got.GetIsPending() || !bytes.Equal(got.GetRootSignKey(), h.operations.pending.RootSignKey) || !bytes.Equal(got.GetDeviceSignKey(), h.operations.pending.DeviceSignKey) {
		t.Errorf("response = %v, want the pending restore", got)
	}
	if fields := got.ProtoReflect().Descriptor().Fields().Len(); fields != 3 {
		t.Errorf("the response has %d fields, want the 3 public ones", fields)
	}
}

func TestAPluginReadsNothingPendingWhenNoRestoreIsPending(t *testing.T) {
	h := newAccountHost(t)

	response, err := h.client("token-a").GetPendingRestore(t.Context(), connect.NewRequest(&v1.GetPendingRestoreRequest{}))

	if err != nil || response.Msg.GetIsPending() || len(response.Msg.GetRootSignKey()) != 0 {
		t.Errorf("response = %v, error = %v, want nothing pending", response, err)
	}
}

func TestAPluginSignsAsRecoveryForItsOwnIDAndTheMessageItSent(t *testing.T) {
	h := newAccountHost(t)
	h.operations.recovered = bytes.Repeat([]byte{9}, 64)

	response, err := h.client("token-c").SignAsRecovery(t.Context(), connect.NewRequest(&v1.SignAsRecoveryRequest{Message: []byte("chain read")}))
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(response.Msg.GetSignature(), h.operations.recovered) {
		t.Errorf("signature = %x, want what the operation returned", response.Msg.GetSignature())
	}
	if len(h.operations.pluginIDs) != 1 || h.operations.pluginIDs[0] != "md.numen.c" || string(h.operations.messages[0]) != "chain read" {
		t.Errorf("operation saw plugin %v and message %q, want md.numen.c and the message", h.operations.pluginIDs, h.operations.messages)
	}
}

func TestASignAsRecoveryMessageOfNoBytesOrOverTheLimitIsAnInvalidArgument(t *testing.T) {
	h := newAccountHost(t)
	client := h.client("token-a")

	for name, message := range map[string][]byte{"empty": nil, "over the limit": bytes.Repeat([]byte{1}, 4097)} {
		_, err := client.SignAsRecovery(t.Context(), connect.NewRequest(&v1.SignAsRecoveryRequest{Message: message}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("%s: error = %v, want invalid argument", name, err)
		}
	}
	if _, err := client.SignAsRecovery(t.Context(), connect.NewRequest(&v1.SignAsRecoveryRequest{Message: bytes.Repeat([]byte{1}, 4096)})); err != nil {
		t.Errorf("a message of 4096 bytes: %v", err)
	}
	if h.operations.callCount() != 1 {
		t.Errorf("operations ran %d times, want once", h.operations.callCount())
	}
}

func TestSigningAsRecoveryWithNothingPendingAnswersFailedPreconditionWithTheReasonNoPendingRestore(t *testing.T) {
	h := newAccountHost(t)
	h.operations.err = identitydomain.ErrNoPendingRestore
	client := h.client("token-a")

	_, signErr := client.SignAsRecovery(t.Context(), connect.NewRequest(&v1.SignAsRecoveryRequest{Message: []byte("m")}))
	_, completeErr := client.CompleteRestore(t.Context(), connect.NewRequest(&v1.CompleteRestoreRequest{Chain: []byte("chain")}))

	for name, err := range map[string]error{"sign": signErr, "complete": completeErr} {
		if connect.CodeOf(err) != connect.CodeFailedPrecondition || readReason(err) != "NO_PENDING_RESTORE" {
			t.Errorf("%s: error = %v, reason %q, want failed precondition with NO_PENDING_RESTORE", name, err, readReason(err))
		}
	}
}

func TestAPluginCompletesTheRestoreWithTheChainOfTheServerForItsOwnID(t *testing.T) {
	h := newAccountHost(t)

	_, err := h.client("token-c").CompleteRestore(t.Context(), connect.NewRequest(&v1.CompleteRestoreRequest{Chain: []byte("chain of the server")}))
	if err != nil {
		t.Fatal(err)
	}

	if len(h.operations.chains) != 1 || string(h.operations.chains[0]) != "chain of the server" {
		t.Errorf("chains = %q, want the chain sent", h.operations.chains)
	}
	if len(h.operations.pluginIDs) != 1 || h.operations.pluginIDs[0] != "md.numen.c" {
		t.Errorf("plugin ids = %v, want [md.numen.c]", h.operations.pluginIDs)
	}
}

func TestACompleteRestoreWithNoChainReportsToTheCoreThatTheServerHasNoSuchAccount(t *testing.T) {
	h := newAccountHost(t)

	_, err := h.client("token-a").CompleteRestore(t.Context(), connect.NewRequest(&v1.CompleteRestoreRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	if len(h.operations.chains) != 1 || len(h.operations.chains[0]) != 0 {
		t.Errorf("chains = %q, want one call with an empty chain", h.operations.chains)
	}
}

func TestAChainThatDoesNotVerifyIsAnInvalidArgumentWithTheReasonInvalidChain(t *testing.T) {
	h := newAccountHost(t)
	h.operations.err = identitydomain.ErrInvalidChain

	_, err := h.client("token-a").CompleteRestore(t.Context(), connect.NewRequest(&v1.CompleteRestoreRequest{Chain: []byte("broken")}))

	if connect.CodeOf(err) != connect.CodeInvalidArgument || readReason(err) != "INVALID_CHAIN" {
		t.Errorf("error = %v, reason %q, want invalid argument with INVALID_CHAIN", err, readReason(err))
	}
}

func TestAnIdentityThatExistsRefusesTheRestoreAsAFailedPrecondition(t *testing.T) {
	h := newAccountHost(t)
	h.operations.err = identitydomain.ErrIdentityExists

	_, err := h.client("token-a").CompleteRestore(t.Context(), connect.NewRequest(&v1.CompleteRestoreRequest{}))

	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Errorf("error = %v, want failed precondition", err)
	}
}

func TestAKeyStoreFailureOfARestoreCallIsInternalAndNamesNoReason(t *testing.T) {
	h := newAccountHost(t)
	h.operations.err = errors.New("the keychain refuses")
	client := h.client("token-a")

	_, pendingErr := client.GetPendingRestore(t.Context(), connect.NewRequest(&v1.GetPendingRestoreRequest{}))
	_, signErr := client.SignAsRecovery(t.Context(), connect.NewRequest(&v1.SignAsRecoveryRequest{Message: []byte("m")}))
	_, completeErr := client.CompleteRestore(t.Context(), connect.NewRequest(&v1.CompleteRestoreRequest{}))

	for name, err := range map[string]error{"pending": pendingErr, "sign": signErr, "complete": completeErr} {
		if connect.CodeOf(err) != connect.CodeInternal || readReason(err) != "" {
			t.Errorf("%s: error = %v, reason %q, want internal with no reason", name, err, readReason(err))
		}
	}
}
