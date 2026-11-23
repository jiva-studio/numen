package plugin_test

import (
	"context"
	"net/http"
	"strconv"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

type signingAccount struct{}

func (signingAccount) SignAsDevice(context.Context, string, []byte) ([]byte, error) {
	return make([]byte, 64), nil
}

func (signingAccount) GetPublicIdentity(context.Context) (identitydomain.Public, error) {
	return identitydomain.Public{DeviceSignKey: make([]byte, 32)}, nil
}

func (signingAccount) SealData(_ context.Context, _ string, _, plain []byte) ([]byte, error) {
	return plain, nil
}

func (signingAccount) OpenData(_ context.Context, _ string, _, sealed []byte) ([]byte, error) {
	return sealed, nil
}

func (signingAccount) GetPendingRestore(context.Context) (identitydomain.PendingRestore, error) {
	return identitydomain.PendingRestore{}, nil
}

func (signingAccount) SignAsRecovery(context.Context, string, []byte) ([]byte, error) {
	return make([]byte, 64), nil
}

func (signingAccount) CompleteRestore(context.Context, string, []byte) error {
	return nil
}

// The core refuses an account call of a plugin whose manifest does not declare identity.
func TestAnAccountCallWithoutTheIdentityPermissionIsRefusedByTheCore(t *testing.T) {
	host := createManifestedHost(t, map[string]string{
		"token-identity": `{"id":"md.numen.signer","name":"Signer","version":"1.0.0","protocol":1,"permissions":["identity"]}`,
		"token-vault":    `{"id":"md.numen.vault","name":"Vault","version":"1.0.0","protocol":1,"permissions":["vault:read","vault:write"]}`,
		"token-none":     `{"id":"md.numen.none","name":"None","version":"1.0.0","protocol":1,"permissions":[]}`,
	})
	callers := lentCallers(host)
	callers.Account = signingAccount{}
	callers.Restore = signingAccount{}
	endpoint, err := hostendpoint.Open(t.Context(), hostendpoint.Address, callers)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := endpoint.Close(context.WithoutCancel(t.Context())); err != nil {
			t.Error(err)
		}
	})

	for _, one := range []struct {
		token string
		deny  bool
	}{{"token-identity", false}, {"token-vault", true}, {"token-none", true}} {
		client := numenv1connect.NewHostAccountServiceClient(
			&http.Client{Timeout: 5 * time.Second},
			"http://127.0.0.1:"+strconv.Itoa(endpoint.GetPort()),
			connect.WithInterceptors(transport.CreateAuthInterceptor(one.token)),
		)
		_, signErr := client.SignAsDevice(t.Context(), connect.NewRequest(&v1.SignAsDeviceRequest{Message: []byte("m")}))
		_, readErr := client.GetPublicIdentity(t.Context(), connect.NewRequest(&v1.GetPublicIdentityRequest{}))
		for name, err := range map[string]error{"sign": signErr, "read": readErr} {
			if got := connect.CodeOf(err) == connect.CodePermissionDenied; got != one.deny {
				t.Errorf("%s %s: permission denied is %v (%v), want %v", one.token, name, got, err, one.deny)
			}
		}
	}
}
