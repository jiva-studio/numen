package service

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/fakeopenai"
)

func TestAnAddressThatCannotCarryAKeyIsRefusedBeforeAnyRequest(t *testing.T) {
	fake, _ := startFakeServer(t, fakeopenai.Config{})
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
		_, err := newTestService(t).Complete(newContext(t), newCall("openai-compatible", address, testKey, "m"), oneUserMessage("hi"))
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
	_, err := newTestService(t).Complete(newContext(t), call, oneUserMessage("hi"))
	checkFailure(t, err, failureWant{reason: connection.ReasonInvalidRequest})
}

func TestAKeyIsSentOverPlainHTTPToTheLoopbackOnly(t *testing.T) {
	_, address := startFakeServer(t, fakeopenai.Config{APIKey: testKey})
	if _, err := newTestService(t).Complete(newContext(t), newCall("openai-compatible", address, testKey, "m"), oneUserMessage("hi")); err != nil {
		t.Errorf("a key to a loopback server over http: %v", err)
	}
}
