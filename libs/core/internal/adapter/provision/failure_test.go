package provision

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The failure of a download is one line that names the model and the address it is fetched from.
func TestAFailedDownloadNamesTheModelAndItsAddress(t *testing.T) {
	data := []byte("the weights of an invented model")
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tampered.onnx":
			if _, err := w.Write([]byte("other bytes")); err != nil {
				t.Error(err)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cases := map[string]struct {
		address   string
		isOffline bool
		cause     error
	}{
		"offline":           {address: "https://models.example.invalid/weights.onnx", isOffline: true, cause: ErrOffline},
		"address refused":   {address: server.URL + "/missing.onnx"},
		"checksum mismatch": {address: server.URL + "/tampered.onnx", cause: ErrChecksum},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			provisioner := newTestProvisioner(t, tc.isOffline)
			provisioner.cfg.Client = server.Client()

			_, err := provisioner.Provision(t.Context(), newModelRequirement(tc.address, data), nil)
			if err == nil {
				t.Fatal("the download succeeded")
			}
			text := err.Error()
			if want := "download model whisper from " + tc.address + ": "; !strings.HasPrefix(text, want) {
				t.Errorf("the failure reads %q, want it to start with %q", text, want)
			}
			if count := strings.Count(text, tc.address); count != 1 {
				t.Errorf("the failure reads %q, which names the address %d times, want once", text, count)
			}
			if strings.Contains(text, "\n") {
				t.Errorf("the failure reads %q, want one line", text)
			}
			if tc.cause != nil && !errors.Is(err, tc.cause) {
				t.Errorf("the failure %v does not wrap %v", err, tc.cause)
			}
		})
	}
}
