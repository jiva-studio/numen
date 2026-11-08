package service

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/connection"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/fakeopenai"
)

func TestAnAddressThatCannotCarryAKeyIsRefusedBeforeAnyRequest(t *testing.T) {
	fake, _ := startFake(t, fakeopenai.Config{})
	addresses := map[string]string{
		"not a URL":                       "::not a url",
		"a scheme other than http":        "ftp://example.com/v1",
		"no scheme":                       "example.com/v1",
		"no host":                         "https:///v1",
		"a user name":                     "https://user:pass@example.com/v1",
		"a link-local address":            "http://169.254.169.254/latest",
		"the metadata name":               "https://metadata.google.internal/v1",
		"the unspecified host":            "http://0.0.0.0:8080/v1",
		"plain http to a host":            "http://example.com/v1",
		"plain http to a private address": "http://192.168.1.20/v1",
	}
	for name, address := range addresses {
		_, err := New().Complete(newContext(t), newCall("openai-compatible", address, testKey, "m"), oneUserMessage("hi"))
		if err == nil || !connection.IsStop(err) {
			t.Errorf("%s: err = %v, want a stop", name, err)
		}
	}
	if len(fake.Requests()) != 0 {
		t.Errorf("the service received %d requests", len(fake.Requests()))
	}
}

func TestAnAddressWithNoAddressAtAllIsRefused(t *testing.T) {
	call := newCall("openai-compatible", "", "", "m")
	_, err := New().Complete(newContext(t), call, oneUserMessage("hi"))
	checkFailure(t, err, failureWant{reason: connection.ReasonBadRequest})
}

func TestAKeyIsSentOverPlainHTTPToTheLoopbackOnly(t *testing.T) {
	_, address := startFake(t, fakeopenai.Config{APIKey: testKey})
	if _, err := New().Complete(newContext(t), newCall("openai-compatible", address, testKey, "m"), oneUserMessage("hi")); err != nil {
		t.Errorf("a key to a loopback server over http: %v", err)
	}
}

func TestAnAddressWithoutAKeyMayUsePlainHTTPOnAPrivateNetwork(t *testing.T) {
	if err := checkAddress("http://192.168.1.20:11434/v1", false); err != nil {
		t.Errorf("err = %v", err)
	}
	if err := checkAddress("http://ollama.lan:11434/v1", false); err != nil {
		t.Errorf("err = %v", err)
	}
}

func TestAResolvedAddressIsCheckedBeforeAConnectionIsMade(t *testing.T) {
	cases := []struct {
		address        string
		hasKey         bool
		isPlain        bool
		isAddressError bool
	}{
		{"127.0.0.1", true, true, false},
		{"::1", true, true, false},
		{"::ffff:127.0.0.1", true, true, false},
		{"93.184.216.34", true, false, false},
		{"93.184.216.34", true, true, true},
		{"93.184.216.34", false, true, false},
		{"10.0.0.8", false, true, false},
		{"169.254.169.254", false, false, true},
		{"fe80::1", false, false, true},
		{"0.0.0.0", false, false, true},
		{"::", false, false, true},
		{"224.0.0.1", false, false, true},
		{"100.100.100.200", false, false, true},
		{"fd00:ec2::254", false, false, true},
	}
	for _, one := range cases {
		err := checkResolved(netip.MustParseAddr(one.address), one.hasKey && one.isPlain)
		if (err != nil) != one.isAddressError {
			t.Errorf("%s (key over http %v): err = %v, refused %v", one.address, one.hasKey && one.isPlain, err, one.isAddressError)
		}
	}
}

func TestTheClientRefusesToConnectToAForbiddenAddress(t *testing.T) {
	ts := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(ts.Close)
	_, port, err := net.SplitHostPort(strings.TrimPrefix(ts.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequestWithContext(newContext(t), http.MethodGet, "http://0.0.0.0:"+port+"/", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := newClients().get(false).Do(request)
	if err == nil {
		if closeErr := response.Body.Close(); closeErr != nil {
			t.Error(closeErr)
		}
		t.Fatal("the client connected to 0.0.0.0")
	}
	if !isAddressError(err) {
		t.Errorf("err = %v, want a refusal", err)
	}
}

func TestTheClientConnectsToTheLoopbackWhenAKeyTravelsOverHTTP(t *testing.T) {
	ts := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(ts.Close)
	ctx := context.WithValue(newContext(t), loopbackOnlyKey{}, true)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := newClients().get(true).Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := response.Body.Close(); err != nil {
		t.Error(err)
	}
}
