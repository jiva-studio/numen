package settings

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	setting "github.com/jiva-studio/numen/libs/core/usecase/settings"
)

var errNoKeychain = errors.New("no keychain on this machine")

// The key the window sends is handed on with the plugin and the secret, and a
func TestPluginSecret_ForwardsKeyAndReturnsError(t *testing.T) {
	var got [3]string
	handler := &Handler{
		WriteSecret: setting.NewWriteSecret(func(_ context.Context, pluginID, name, key string) error {
			got = [3]string{pluginID, name, key}
			return nil
		}),
		DeleteSecret: setting.NewDeleteSecret(func(context.Context, string, string) error { return errNoKeychain }),
	}
	_, err := handler.WritePluginSecret(t.Context(), connect.NewRequest(&v1.WritePluginSecretRequest{
		PluginId: "md.numen.sample", Name: "apiKey", Key: "made-up-key",
	}))
	if err != nil || got != [3]string{"md.numen.sample", "apiKey", "made-up-key"} {
		t.Fatalf("handed on %v, %v", got, err)
	}
	_, err = handler.DeletePluginSecret(t.Context(), connect.NewRequest(&v1.DeletePluginSecretRequest{
		PluginId: "md.numen.sample", Name: "apiKey",
	}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition || !errors.Is(err, errNoKeychain) {
		t.Errorf("an error answered %v", err)
	}
}

// A build that keeps no keys answers that it serves none.
func TestABuildWithoutKeysServesNone(t *testing.T) {
	handler := &Handler{}
	if _, err := handler.WritePluginSecret(t.Context(), connect.NewRequest(&v1.WritePluginSecretRequest{})); connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Errorf("write answered %v", err)
	}
	if _, err := handler.DeletePluginSecret(t.Context(), connect.NewRequest(&v1.DeletePluginSecretRequest{})); connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Errorf("delete answered %v", err)
	}
}

// The consent of a plugin travels to the window; a key never does.
func TestAPluginTravelsWithItsConsent(t *testing.T) {
	infos := NewWirePluginInfos([]setting.PluginInfo{{
		ID: "md.numen.sample", Secrets: []setting.PluginSecret{{Name: "apiKey", Title: "API key"}}, StoredSecrets: []string{"apiKey"},
		Consent: setting.ConsentRequested,
	}})
	one := infos[0]
	if one.GetCompletionConsent() != v1.CompletionConsent_COMPLETION_CONSENT_REQUESTED ||
		len(one.GetStoredSecrets()) != 1 || one.GetSecrets()[0].GetName() != "apiKey" {
		t.Errorf("travelled as %v", one)
	}
	for _, consent := range []setting.Consent{
		setting.ConsentNone, setting.ConsentPending, setting.ConsentRequested, setting.ConsentAllowed, setting.ConsentDenied,
	} {
		if _, found := Consents[consent]; !found {
			t.Errorf("consent %v has no wire value", consent)
		}
	}
}
