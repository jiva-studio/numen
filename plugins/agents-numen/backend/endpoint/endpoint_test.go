package endpoint_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/jiva-studio/numen/plugins/agents-numen/backend/endpoint"
)

func TestAKeyIsNotFollowedToPlainHTTP(t *testing.T) {
	err := endpoint.RefuseDowngrade(httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://192.0.2.1/v1/models", nil), nil)
	if err == nil {
		t.Error("a redirect to plain http was followed")
	}
	if err := endpoint.RefuseDowngrade(httptest.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.com/v1/models", nil), nil); err != nil {
		t.Errorf("a redirect to https failed: %v", err)
	}
}

func TestARedirectChainOfTenIsStopped(t *testing.T) {
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.com/v1/models", nil)

	if err := endpoint.RefuseDowngrade(req, make([]*http.Request, 10)); err == nil {
		t.Error("the tenth redirect was followed")
	}
}

func TestOnlyThisMachineIsLocal(t *testing.T) {
	for _, host := range []string{"localhost", "127.0.0.1", "::1"} {
		if !endpoint.IsLocalHost(host) {
			t.Errorf("%s is not local", host)
		}
	}
	if endpoint.IsLocalHost("api.example.com") {
		t.Error("a remote host is local")
	}
}

func TestHTTPSAnywhereAndHTTPOnlyToThisMachineMayReceiveACredential(t *testing.T) {
	cases := map[string]bool{
		"https://api.example.com/v1": true,
		"http://localhost:8080/v1":   true,
		"http://192.0.2.1/v1":        false,
		"ftp://localhost/v1":         false,
	}
	for address, want := range cases {
		parsed, err := url.Parse(address)
		if err != nil {
			t.Fatal(err)
		}
		if got := endpoint.IsPrivate(parsed); got != want {
			t.Errorf("%s is private: %v, want %v", address, got, want)
		}
	}
}
