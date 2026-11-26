package synctest

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
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
	// EnvBucketURL is the address of the bucket of that stack, which the signed addresses of the blobs lead to.
	EnvBucketURL = "NUMEN_SYNC_TEST_BUCKET_URL"
	// EnvRequired set to 1 makes a test of the real server fail, not skip, when the stack is not set. The job that runs the stack sets it; a job without the stack, such as the one of a pull request from a fork, does not.
	EnvRequired = "NUMEN_SYNC_TEST_REQUIRED"
)

type Stack struct {
	URL        string
	PaymentURL string
	BucketURL  string
	Roots      []ed25519.PublicKey
}

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
	return &Stack{URL: address, PaymentURL: os.Getenv(EnvPaymentURL), BucketURL: os.Getenv(EnvBucketURL), Roots: []ed25519.PublicKey{root}}
}

// NewRealServer puts a proxy of the test in front of the real sync server of the stack. The computers keep the public address
// of the server, which the server names as its own in a challenge, and their requests for it go to the proxy, which runs the
// switches, the lost responses, the hooks and the counters before and after the server. A link call the server refuses for its limit of new channels (6 a minute
// for one source, which every test of a run is) waits and goes again, and the channels a test opened are closed when it ends.
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
	s.URL, s.roots, s.origin, s.Stack = stack.URL, stack.Roots, target.Host, stack
	t.Cleanup(func() { s.closeChannels(t) })
	if bucket, err := url.Parse(stack.BucketURL); err == nil {
		s.bucketHost = bucket.Host
	}
	return s
}

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

func (s *Server) getRoots() []ed25519.PublicKey { return s.roots }

// NewServer puts the proxy in front of the real sync server of the stack, which the test skips without, and sets the clock to the time of the machine, as the
// server reads it.
func NewServer(t *testing.T, clock *Clock) *Server {
	t.Helper()
	server := NewRealServer(t, GetStack(t))
	clock.Set(GetNow())
	return server
}

// NewRealWorld returns the real server of the stack behind a proxy, a clock that reads the time of the machine, as the server does, and
// the recovery words of an account that only this test of this run has.
func NewRealWorld(t *testing.T) (*Server, *Clock, []string) {
	t.Helper()
	clock := NewClock(GetNow())
	return NewServer(t, clock), clock, NewWords(t)
}

type machineClock struct{}

func (machineClock) Now() time.Time { return time.Now() }

func GetNow() time.Time { return machineClock{}.Now() }

func GetConflictCopy(stem string, number int) string {
	suffix := ""
	if number > 1 {
		suffix = fmt.Sprintf(" %d", number)
	}
	return fmt.Sprintf("%s (conflict %s%s).md", stem, GetNow().Format(time.DateOnly), suffix)
}
