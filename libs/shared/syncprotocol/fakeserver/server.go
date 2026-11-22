package fakeserver

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"sync"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
)

const (
	challengeLife     = 60 * time.Second
	defaultSessionTTL = time.Hour
	failedSignIn      = "sign-in failed"
)

// Config sets what the server answers: the protocol version (zero is syncprotocol.Version), the address it
// tells itself by, the clock of its challenges and sessions (nil reads the clock of the machine) and how
// long a session lives (zero is an hour).
type Config struct {
	Protocol   uint32
	Audience   string
	Now        func() time.Time
	SessionTTL time.Duration
}

// Session is a session the server issued and has not ended.
type Session struct {
	Token              string
	DeviceSignKey      []byte
	AccountFingerprint string
	ExpiresAt          time.Time
}

type challenge struct {
	deviceSignKey []byte
	nonce         []byte
	expiresAt     int64
}

// Server serves SignInService and the admin paths /admin/sessions (GET lists, POST /admin/sessions/end ends
// every session, or the ones of the device in the query deviceSignKey, a hex string).
type Server struct {
	syncv1connect.UnimplementedSignInServiceHandler

	protocol uint32
	now      func() time.Time
	ttl      time.Duration
	handler  http.Handler

	mu         sync.Mutex
	audience   string
	challenges map[string]challenge
	accounts   map[string]string
	sessions   []Session
}

// New returns the server for the config.
func New(cfg Config) *Server {
	s := &Server{
		protocol: cfg.Protocol, now: cfg.Now, ttl: cfg.SessionTTL, audience: cfg.Audience,
		challenges: map[string]challenge{}, accounts: map[string]string{},
	}
	if s.protocol == 0 {
		s.protocol = syncprotocol.Version
	}
	if s.now == nil {
		s.now = time.Now
	}
	if s.ttl == 0 {
		s.ttl = defaultSessionTTL
	}
	mux := http.NewServeMux()
	mux.Handle(syncv1connect.NewSignInServiceHandler(s, connect.WithInterceptors(syncprotocol.NewVersionHandlerInterceptor(s.protocol))))
	mux.HandleFunc("/admin/sessions", s.serveSessions)
	mux.HandleFunc("/admin/sessions/end", s.serveEnd)
	s.handler = mux
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.handler.ServeHTTP(w, r) }

// SetAudience sets the address the server tells itself by, once the address it listens on is known.
func (s *Server) SetAudience(audience string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.audience = audience
}

// ListSessions returns the sessions that are not ended, oldest first.
func (s *Server) ListSessions() []Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.sessions)
}

// EndSessions ends the sessions of the device. A nil key ends the sessions of every device.
func (s *Server) EndSessions(deviceSignKey []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions = slices.DeleteFunc(s.sessions, func(one Session) bool {
		return deviceSignKey == nil || string(one.DeviceSignKey) == string(deviceSignKey)
	})
}

func (s *Server) GetChallenge(_ context.Context, req *connect.Request[syncv1.GetChallengeRequest]) (*connect.Response[syncv1.GetChallengeResponse], error) {
	if len(req.Msg.GetDeviceSignKey()) != 32 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("the device sign key must have 32 bytes"))
	}
	id, nonce := make([]byte, 16), make([]byte, 32)
	if _, err := rand.Read(id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if _, err := rand.Read(nonce); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	expiresAt := s.now().Add(challengeLife).Unix()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.challenges[string(id)] = challenge{deviceSignKey: slices.Clone(req.Msg.GetDeviceSignKey()), nonce: nonce, expiresAt: expiresAt}
	return connect.NewResponse(&syncv1.GetChallengeResponse{ChallengeId: id, Nonce: nonce, ExpiresAt: expiresAt, Audience: s.audience}), nil
}

func (s *Server) SignIn(_ context.Context, req *connect.Request[syncv1.SignInRequest]) (*connect.Response[syncv1.SignInResponse], error) {
	session, ok := s.openSession(req.Msg)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New(failedSignIn))
	}
	return connect.NewResponse(&syncv1.SignInResponse{
		SessionToken: session.Token, ExpiresAt: session.ExpiresAt.Unix(), AccountFingerprint: session.AccountFingerprint,
	}), nil
}

func (s *Server) openSession(req *syncv1.SignInRequest) (Session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pending, found := s.challenges[string(req.GetChallengeId())]
	delete(s.challenges, string(req.GetChallengeId()))
	key := req.GetDeviceSignKey()
	if !found || pending.expiresAt < s.now().Unix() || string(pending.deviceSignKey) != string(key) || len(req.GetSignature()) != 64 {
		return Session{}, false
	}
	message := syncprotocol.GetSignInMessage(&syncv1.GetChallengeResponse{Nonce: pending.nonce, ExpiresAt: pending.expiresAt, Audience: s.audience}, key, s.protocol)
	if !identity.VerifyForPlugin(key, syncprotocol.SignInPluginID, message, [64]byte(req.GetSignature())) {
		return Session{}, false
	}
	chain, err := identity.ParseChain(req.GetChain())
	if err != nil || !slices.ContainsFunc(chain.ListDevices(), func(d identity.Device) bool { return string(d.Public.Sign) == string(key) }) {
		return Session{}, false
	}
	account := chain.Root().Fingerprint().String()
	if known, isKnown := s.accounts[account]; isKnown && known != account {
		return Session{}, false
	}
	s.accounts[account] = account
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return Session{}, false
	}
	session := Session{
		Token: base64.RawURLEncoding.EncodeToString(token), DeviceSignKey: slices.Clone(key), AccountFingerprint: account, ExpiresAt: s.now().Add(s.ttl),
	}
	s.sessions = append(s.sessions, session)
	return session, true
}

type sessionJSON struct {
	DeviceSignKey      string `json:"deviceSignKey"`
	AccountFingerprint string `json:"accountFingerprint"`
	Token              string `json:"token"`
}

func (s *Server) serveSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	listed := []sessionJSON{}
	for _, one := range s.ListSessions() {
		listed = append(listed, sessionJSON{DeviceSignKey: hex.EncodeToString(one.DeviceSignKey), AccountFingerprint: one.AccountFingerprint, Token: one.Token})
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(listed); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) serveEnd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var device []byte
	if text := r.URL.Query().Get("deviceSignKey"); text != "" {
		decoded, err := hex.DecodeString(text)
		if err != nil {
			http.Error(w, "deviceSignKey is not hex", http.StatusBadRequest)
			return
		}
		device = decoded
	}
	s.EndSessions(device)
	w.WriteHeader(http.StatusNoContent)
}
