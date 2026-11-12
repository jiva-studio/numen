package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
)

func TestReadSecrets(t *testing.T) {
	first := Secret{ID: "s0000000a", Name: "Work", Group: "openrouter", Value: "value-one"}
	second := Secret{ID: "s0000000b", Name: "Home", Group: "openai", Value: "value-two"}
	cases := []struct {
		name      string
		secrets   Secrets
		wantList  []Secret
		wantFound bool
	}{
		{"a declaration with two secrets", Secrets{SecretAPIKey: {first, second}}, []Secret{first, second}, true},
		{"a declaration with no secret", Secrets{SecretAPIKey: {}}, []Secret{}, true},
		{"a call that does not carry the declaration", Secrets{"other": {first}}, nil, false},
		{"no secrets at all", nil, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			list, found := ReadSecrets(tc.secrets, SecretAPIKey)
			if found != tc.wantFound || len(list) != len(tc.wantList) {
				t.Fatalf("ReadSecrets = %v, %v, want %v, %v", list, found, tc.wantList, tc.wantFound)
			}
			if tc.wantFound && !reflect.DeepEqual(list, tc.wantList) {
				t.Errorf("ReadSecrets list = %v, want %v", list, tc.wantList)
			}
			if !tc.wantFound && list != nil {
				t.Errorf("ReadSecrets list = %v, want nil", list)
			}
		})
	}
}

func TestReadSecretsKeepsTheOrderOfTheHost(t *testing.T) {
	secrets := Secrets{SecretAPIKey: {{ID: "s00000003"}, {ID: "s00000001"}, {ID: "s00000002"}}}
	list, _ := ReadSecrets(secrets, SecretAPIKey)
	var ids []string
	for _, one := range list {
		ids = append(ids, one.ID)
	}
	if want := []string{"s00000003", "s00000001", "s00000002"}; !reflect.DeepEqual(ids, want) {
		t.Errorf("ids = %v, want %v", ids, want)
	}
}

func newConfigureClient(t *testing.T, life *LifecycleServer) numenv1connect.PluginLifecycleServiceClient {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(lifecycleHandler{state: life}))
	web := httptest.NewServer(mux)
	t.Cleanup(web.Close)
	return numenv1connect.NewPluginLifecycleServiceClient(web.Client(), web.URL)
}

func TestAConfigureCallbackReceivesTheSecretsOfTheRequestByDeclarationName(t *testing.T) {
	var gotValues map[string]string
	var gotSecrets Secrets
	life := NewLifecycleServer(make(chan struct{}),
		WithLifecycleConfigure(func(_ context.Context, values map[string]string, secrets Secrets) error {
			gotValues, gotSecrets = values, secrets
			return nil
		}))
	client := newConfigureClient(t, life)

	_, err := client.Configure(t.Context(), connect.NewRequest(&v1.ConfigureRequest{
		Values: map[string]string{"model": "small"},
		Secrets: []*v1.ConfigureSecrets{
			{Name: "apiKey", Group: "openrouter", Values: []*v1.SecretValue{
				{Id: "s00000001", Name: "Work", Group: "openrouter", Value: "value-one"},
				{Id: "s00000002", Name: "Home", Group: "openai", Value: "value-two"},
			}},
			{Name: "orgId"},
		},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if gotValues["model"] != "small" || len(gotValues) != 1 {
		t.Errorf("values = %v, want only the model", gotValues)
	}
	list, found := ReadSecrets(gotSecrets, "apiKey")
	want := []Secret{
		{ID: "s00000001", Name: "Work", Group: "openrouter", Value: "value-one"},
		{ID: "s00000002", Name: "Home", Group: "openai", Value: "value-two"},
	}
	if !found || !reflect.DeepEqual(list, want) {
		t.Errorf("apiKey = %v, %v, want %v", list, found, want)
	}
	if empty, found := ReadSecrets(gotSecrets, "orgId"); !found || len(empty) != 0 {
		t.Errorf("orgId = %v, %v, want an empty list that is carried", empty, found)
	}
	if _, found := ReadSecrets(gotSecrets, "missing"); found {
		t.Error("a declaration the request does not carry was found")
	}
}

func TestTheLifecycleClientSendsSecretsOutsideTheValues(t *testing.T) {
	var gotSecrets Secrets
	var gotValues map[string]string
	life := NewLifecycleServer(make(chan struct{}),
		WithLifecycleUpdateConfig(func(values map[string]string, secrets Secrets) {
			gotValues, gotSecrets = values, secrets
		}))
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(lifecycleHandler{state: life}))
	web := httptest.NewServer(mux)
	t.Cleanup(web.Close)
	client := NewLifecycleClient(web.Client(), web.URL)

	secrets := Secrets{"apiKey": {{ID: "s00000001", Name: "Work", Group: "openrouter", Value: "value-one"}}}
	accepted, err := client.Configure(t.Context(), map[string]string{"model": "small"}, secrets)
	if err != nil || !accepted {
		t.Fatalf("Configure = %v, %v", accepted, err)
	}
	if !reflect.DeepEqual(gotSecrets, secrets) {
		t.Errorf("secrets = %v, want %v", gotSecrets, secrets)
	}
	for name, value := range gotValues {
		if value == "value-one" {
			t.Errorf("the value of a secret is in the values under %q", name)
		}
	}
}

func TestApplyConfigPassesSecretsToTheCallbackAndKeepsThemOutOfTheStoredConfig(t *testing.T) {
	var gotSecrets Secrets
	life := NewLifecycleServer(make(chan struct{}),
		WithLifecycleConfigure(func(_ context.Context, _ map[string]string, secrets Secrets) error {
			gotSecrets = secrets
			return nil
		}))
	secrets := Secrets{"apiKey": {{ID: "s00000001", Value: "value-one"}}}

	if ok, err := life.ApplyConfig(t.Context(), map[string]string{"model": "small"}, secrets); err != nil || !ok {
		t.Fatalf("ApplyConfig = %v, %v", ok, err)
	}
	if !reflect.DeepEqual(gotSecrets, secrets) {
		t.Errorf("secrets = %v, want %v", gotSecrets, secrets)
	}
	for key, value := range life.AllConfigs() {
		if value == "value-one" {
			t.Errorf("the stored configuration holds the value of a secret under %q", key)
		}
	}
}
