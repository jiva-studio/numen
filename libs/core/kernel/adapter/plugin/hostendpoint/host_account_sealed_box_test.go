package hostendpoint

import (
	"bytes"
	"context"
	"testing"

	"connectrpc.com/connect"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

const maxSealedBox = 1 << 10

func (a *accountOperations) OpenSealedBox(_ context.Context, pluginID string, boxContext, sealed []byte) ([]byte, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pluginIDs = append(a.pluginIDs, pluginID)
	a.dataContexts = append(a.dataContexts, bytes.Clone(boxContext))
	a.plains = append(a.plains, bytes.Clone(sealed))
	return a.opened, a.err
}

func TestAPluginOpensASealedBoxWithTheContextItChoosesForItsOwnID(t *testing.T) {
	h := newAccountHost(t)
	h.operations.opened = []byte("epoch key")

	response, err := h.client("token-c").OpenSealedBox(t.Context(), connect.NewRequest(&v1.OpenSealedBoxRequest{Context: []byte("numen-sync/share-seal/v1"), Sealed: []byte("box bytes")}))
	if err != nil {
		t.Fatal(err)
	}

	if string(response.Msg.GetPlain()) != "epoch key" {
		t.Errorf("plain = %q, want what the operation returned", response.Msg.GetPlain())
	}
	if len(h.operations.pluginIDs) != 1 || h.operations.pluginIDs[0] != "md.numen.c" {
		t.Errorf("plugin ids = %v, want [md.numen.c]: the id of the token", h.operations.pluginIDs)
	}
	if string(h.operations.dataContexts[0]) != "numen-sync/share-seal/v1" || string(h.operations.plains[0]) != "box bytes" {
		t.Errorf("operation saw context %q and sealed %q", h.operations.dataContexts, h.operations.plains)
	}
}

func TestOpenSealedBoxNeedsThePermissionIdentity(t *testing.T) {
	h := newAccountHost(t)
	h.callers.granted = []string{manifest.PermissionVaultRead}

	_, err := h.client("token-a").OpenSealedBox(t.Context(), connect.NewRequest(&v1.OpenSealedBoxRequest{Sealed: []byte("m")}))

	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Errorf("error = %v, want permission denied", err)
	}
	if h.operations.callCount() != 0 {
		t.Error("an identity operation ran for a plugin without the permission")
	}
}

func TestOpenSealedBoxRefusesMoreThan1KiBAndAcceptsExactly1KiB(t *testing.T) {
	h := newAccountHost(t)
	client := h.client("token-a")

	_, over := client.OpenSealedBox(t.Context(), connect.NewRequest(&v1.OpenSealedBoxRequest{Sealed: bytes.Repeat([]byte{1}, maxSealedBox+1)}))
	_, exact := client.OpenSealedBox(t.Context(), connect.NewRequest(&v1.OpenSealedBoxRequest{Sealed: bytes.Repeat([]byte{1}, maxSealedBox)}))

	if connect.CodeOf(over) != connect.CodeInvalidArgument {
		t.Errorf("over the limit: error = %v, want invalid argument", over)
	}
	if exact != nil || h.operations.callCount() != 1 {
		t.Errorf("exact limit: error = %v, operations run %d times, want no error and one call", exact, h.operations.callCount())
	}
}

func TestABoxThatCannotBeOpenedAnswersAFailedPreconditionWithTheReasonCannotOpen(t *testing.T) {
	h := newAccountHost(t)
	h.operations.err = identitydomain.ErrCannotOpen

	_, err := h.client("token-a").OpenSealedBox(t.Context(), connect.NewRequest(&v1.OpenSealedBoxRequest{Sealed: []byte("box")}))

	if connect.CodeOf(err) != connect.CodeFailedPrecondition || readReason(err) != "CANNOT_OPEN" {
		t.Errorf("error = %v, reason %q, want failed precondition with CANNOT_OPEN", err, readReason(err))
	}
}

func TestAnUnknownTokenIsUnauthenticatedForOpenSealedBox(t *testing.T) {
	h := newAccountHost(t)

	_, err := h.client("token-of-nobody").OpenSealedBox(t.Context(), connect.NewRequest(&v1.OpenSealedBoxRequest{Sealed: []byte("box")}))

	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Errorf("error = %v, want unauthenticated", err)
	}
}
