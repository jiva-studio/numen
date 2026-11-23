package client_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
)

func serveStorage(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("sealed")) }))
	t.Cleanup(server.Close)
	return server
}

func TestNewBlobHTTPClient_ReachesLoopbackStorageOfALoopbackServer(t *testing.T) {
	t.Parallel()
	storage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("sealed")) }))
	t.Cleanup(storage.Close)

	got, err := syncclient.GetBlob(t.Context(), syncclient.NewBlobHTTPClient("http://127.0.0.1:8080", time.Minute), storage.URL+"/blob")

	if err != nil || string(got) != "sealed" {
		t.Errorf("got %q, error = %v, want the blob", got, err)
	}
}

func TestNewBlobHTTPClient_RefusesAddressesOnTheMachineOrTheNetworkOfAServerElsewhere(t *testing.T) {
	t.Parallel()
	storage := serveStorage(t)
	port := mustGetPort(t, storage.URL)
	web := syncclient.NewBlobHTTPClient("https://sync.example.org", time.Minute)

	for name, address := range map[string]string{
		"plain http":           "http://storage.example.org/blob",
		"loopback over https":  storage.URL + "/blob",
		"a name for loopback":  "https://localhost:" + port + "/blob",
		"private address":      "https://192.168.1.20/blob",
		"link-local address":   "https://169.254.169.254/latest/meta-data",
		"carrier-grade NAT":    "https://100.100.100.200/blob",
		"IPv6 loopback":        "https://[::1]:8384/rest",
		"IPv4-mapped loopback": "https://[::ffff:127.0.0.1]:8384/rest",
		"unspecified":          "https://0.0.0.0/blob",
		"a file":               "file:///etc/passwd",
	} {
		if _, err := syncclient.GetBlob(t.Context(), web, address); err == nil {
			t.Errorf("%s: GetBlob(%q) succeeded, want a refusal", name, address)
		}
		if err := syncclient.PutBlob(t.Context(), web, address, []byte("sealed")); err == nil {
			t.Errorf("%s: PutBlob(%q) succeeded, want a refusal", name, address)
		}
	}
}

func mustGetPort(t *testing.T, address string) string {
	t.Helper()
	parsed, err := url.Parse(address)
	if err != nil {
		t.Fatal(err)
	}
	return parsed.Port()
}

func TestBlobTransfers_FollowNoRedirect(t *testing.T) {
	t.Parallel()
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("internal")) }))
	t.Cleanup(target.Close)
	storage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/rest", http.StatusFound)
	}))
	t.Cleanup(storage.Close)

	if got, err := syncclient.GetBlob(t.Context(), http.DefaultClient, storage.URL+"/blob"); err == nil {
		t.Errorf("GetBlob followed a redirect and returned %q, want an error", got)
	}
	if err := syncclient.PutBlob(t.Context(), http.DefaultClient, storage.URL+"/blob", []byte("sealed")); err == nil {
		t.Error("PutBlob followed a redirect, want an error")
	}
	if err := syncclient.PutPart(t.Context(), http.DefaultClient, storage.URL+"/blob", []byte("sealed")); err == nil {
		t.Error("PutPart followed a redirect, want an error")
	}
	if body, err := syncclient.GetBlobRange(t.Context(), http.DefaultClient, storage.URL+"/blob", 0); err == nil {
		_ = body.Close()
		t.Error("GetBlobRange followed a redirect, want an error")
	}
}

func TestBlobTransfers_LeaveTheSignatureOutOfTheirErrors(t *testing.T) {
	t.Parallel()
	address := "http://127.0.0.1:1/blob?X-Amz-Signature=secret-signature"
	web := &http.Client{}

	_, getErr := syncclient.GetBlob(t.Context(), web, address)
	putErr := syncclient.PutBlob(t.Context(), web, address, []byte("sealed"))
	partErr := syncclient.PutPart(t.Context(), web, address, []byte("sealed"))
	_, rangeErr := syncclient.GetBlobRange(t.Context(), web, address, 0)

	for name, err := range map[string]error{"GetBlob": getErr, "PutBlob": putErr, "PutPart": partErr, "GetBlobRange": rangeErr} {
		if err == nil || strings.Contains(err.Error(), "secret-signature") || strings.Contains(err.Error(), "X-Amz") {
			t.Errorf("%s: error = %v, want a failure without the address", name, err)
		}
	}
}
