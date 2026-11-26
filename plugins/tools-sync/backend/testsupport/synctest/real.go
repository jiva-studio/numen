package synctest

import (
	"crypto/ed25519"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"testing"
	"time"
)

const (
	// EnvURL is the public address of the real sync server of the test stack (tests/server).
	EnvURL = "NUMEN_SYNC_TEST_URL"
	// EnvRootKey is the root public key of that stack, 64 hex digits, as `make sync_stack_env` prints it.
	EnvRootKey = "NUMEN_SYNC_TEST_ROOT_KEY"
	// EnvPaymentURL is the control API of the payment stand-in of that stack; a test that buys a plan needs it.
	EnvPaymentURL = "NUMEN_SYNC_TEST_PAYMENT_URL"
	// EnvRequired set to 1 makes a test of the real server fail, not skip, when the stack is not set. The job that runs the stack sets it; a job without the stack, such as the one of a pull request from a fork, does not.
	EnvRequired = "NUMEN_SYNC_TEST_REQUIRED"
)

// Stack is where the real sync server of the test stack is and the key its epochs are signed under.
type Stack struct {
	URL        string
	PaymentURL string
	Roots      []ed25519.PublicKey
}

// GetStack reads the stack from the environment. The test skips when the stack is not set, and fails then when EnvRequired is 1.
func GetStack(t *testing.T) *Stack {
	t.Helper()
	address, rootHex := os.Getenv(EnvURL), os.Getenv(EnvRootKey)
	if address == "" || rootHex == "" {
		if os.Getenv(EnvRequired) != "1" {
			t.Skip("no real sync server: make sync_stack_up, then eval \"$(make -s sync_stack_env)\"")
		}
		t.Fatalf("no real sync server: run `make sync_stack_up` and then eval \"$(make -s sync_stack_env)\" to set %s and %s", EnvURL, EnvRootKey)
	}
	root, err := hex.DecodeString(rootHex)
	if err != nil || len(root) != ed25519.PublicKeySize {
		t.Fatalf("%s is not 64 hex digits of an ed25519 public key", EnvRootKey)
	}
	return &Stack{URL: address, PaymentURL: os.Getenv(EnvPaymentURL), Roots: []ed25519.PublicKey{root}}
}

// NewRealServer puts a proxy of the test in front of the real sync server of the stack. The computers keep the public address
// of the server, which the server names as its own in a challenge, and their requests for it go to the proxy, which runs the
// switches, the lost responses, the hooks and the counters before and after the server.
func NewRealServer(t *testing.T, stack *Stack) *Server {
	t.Helper()
	target, err := url.Parse(stack.URL)
	if err != nil {
		t.Fatal(err)
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.FlushInterval = -1
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, _ error) { w.WriteHeader(http.StatusBadGateway) }
	s := NewProxy(t, proxy)
	s.URL, s.roots, s.origin = stack.URL, stack.Roots, target.Host
	return s
}

// NewProxy returns a server of the test that runs the switches, the hooks and the counters over the backend, on a loopback port that closes with the test.
func NewProxy(t *testing.T, backend http.Handler) *Server {
	t.Helper()
	s := newServer()
	s.backend = backend
	served := httptest.NewServer(http.HandlerFunc(s.serve))
	t.Cleanup(served.Close)
	s.URL = served.URL
	if parsed, err := url.Parse(served.URL); err == nil {
		s.dial, s.origin = parsed.Host, parsed.Host
	}
	return s
}

// CountCalls returns how many requests for the path the server has taken, answered or not, for example
// syncv1connect.BlobServiceReserveBlobsProcedure.
func (s *Server) CountCalls(path string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls[path]
}

func (s *Server) getRoots() []ed25519.PublicKey {
	if s.roots != nil {
		return s.roots
	}
	return fakeRoots()
}

// NewRealWorld returns the real server of the stack behind a proxy, a clock that reads the time of the machine, as the server does, and
// the recovery words of an account that only this test of this run has.
func NewRealWorld(t *testing.T) (*Server, *Clock, []string) {
	t.Helper()
	return NewRealServer(t, GetStack(t)), NewClock(GetNow()), NewWordsFor(t, NewLabel(t))
}

// machineClock reads the clock of the machine, which the real server also reads.
type machineClock struct{}

func (machineClock) Now() time.Time { return time.Now() }

// GetNow returns the time of the machine, the start of a clock that a real server runs with.
func GetNow() time.Time { return machineClock{}.Now() }
