package fakeserver

import (
	"bytes"
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
	defaultQuota      = 10 << 30
	failedSignIn      = "sign-in failed"
)

// clock reads the time.
type clock interface{ Now() time.Time }

// systemClock reads the clock of the machine.
type systemClock struct{}

// Now returns the time of the machine.
func (systemClock) Now() time.Time { return time.Now() }

// Config sets what the server answers: the protocol version (zero is syncprotocol.Version), the address it
// tells itself by, the clock of its challenges and sessions (nil reads the clock of the machine), how long a
// session lives (zero is an hour), the bytes an account may store (zero is 10 GiB) and whether an account needs a
// subscription to write. Plans are the plans the server sells, ascending by quota; empty is Basic with 10 GiB for 5 USD, Plus with
// 100 GiB for 15 USD and Pro with 250 GiB for 30 USD (ids basic, plus, pro). BodyDir is the folder that keeps the parts and the
// objects of multipart uploads in files (empty keeps them in memory); Stop is called when an admin stop-after budget ends, so the
// host program can exit once it saved the data. Heartbeat is the silence after which a stream of change hints sends a heartbeat
// (zero is syncprotocol.WatchHeartbeat; the Started message announces it in whole seconds, rounded up).
type Config struct {
	Protocol   uint32
	Audience   string
	Now        func() time.Time
	SessionTTL time.Duration
	QuotaBytes uint64
	Plans      []syncprotocol.Plan

	NeedsSubscription bool

	BodyDir   string
	Stop      func()
	Heartbeat time.Duration
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

// Server serves the sign-in, account, catalog, revision, blob and share services from memory, the presigned blob
// URLs at /blobs/, and the admin paths /admin/sessions (GET lists, POST /admin/sessions/end ends every
// session, or the ones of the device in the query deviceSignKey, a hex string) and /admin/blobs (GET returns
// the id and the body of every stored blob, raw). The watch service streams change hints; /admin/streams (GET) returns the count of open streams as text and
// /admin/streams/end (POST) ends them all. The subscription service has the pay page at /pay/ and the admin
// path /admin/subscription (POST, phase=none|active|grace|read_only|deleted and an optional account). An account
// may write when its subscription allows it, and every account does unless the config sets NeedsSubscription. The admin paths
// /admin/trial and /admin/cancel-trial (POST, plan=id and an optional account) start and cancel a trial, and /admin/usage (POST, bytes=N and an
// optional account) sets the bytes the cloud holds. The restore service has the admin paths /admin/backup (POST keeps the state),
// /admin/restore (POST puts the kept state back and announces nothing), /admin/announce (POST issues a restore epoch),
// /admin/retention (POST deletes the vaults of every account) and /admin/blob-loss (POST drops the body of every blob).
type Server struct {
	syncv1connect.UnimplementedSignInServiceHandler

	protocol          uint32
	now               func() time.Time
	ttl               time.Duration
	quota             uint64
	plans             []syncprotocol.Plan
	needsSubscription bool
	handler           http.Handler

	mu            sync.Mutex
	audience      string
	streams       streamSet
	challenges    map[string]challenge
	chains        map[string]identity.Chain
	sessions      []Session
	catalogs      map[string][]*syncv1.CatalogEntry
	vaults        map[string]*vaultState
	blobs         map[string]*blobRow
	used          map[string]uint64
	calls         map[string]int
	subscriptions map[string]time.Time
	planOf        map[string]string
	checkouts     map[string]checkout
	shop          shop
	restore       restoreState
	store         *bodyStore
	partLog       []partLogEntry
	stopUpload    stopAfter
	stopDownload  stopAfter
	stop          func()
	links         map[string]*linkChannel
	shares        map[string]*shareState
}

// New returns the server for the config.
func New(cfg Config) *Server {
	s := &Server{
		protocol: cfg.Protocol, now: cfg.Now, ttl: cfg.SessionTTL, quota: cfg.QuotaBytes, plans: cfg.Plans, audience: cfg.Audience, needsSubscription: cfg.NeedsSubscription,
		challenges: map[string]challenge{}, chains: map[string]identity.Chain{}, catalogs: map[string][]*syncv1.CatalogEntry{},
		vaults: map[string]*vaultState{}, blobs: map[string]*blobRow{}, used: map[string]uint64{}, calls: map[string]int{},
		subscriptions: map[string]time.Time{}, planOf: map[string]string{}, checkouts: map[string]checkout{}, shop: newShop(),
		store: newBodyStore(cfg.BodyDir), stop: cfg.Stop, streams: streamSet{heartbeat: cfg.Heartbeat, watchers: map[*watcher]struct{}{}},
		links: map[string]*linkChannel{}, shares: map[string]*shareState{},
	}
	if s.protocol == 0 {
		s.protocol = syncprotocol.Version
	}
	if s.now == nil {
		var machine clock = systemClock{}
		s.now = machine.Now
	}
	if s.streams.heartbeat == 0 {
		s.streams.heartbeat = syncprotocol.WatchHeartbeat
	}
	if s.ttl == 0 {
		s.ttl = defaultSessionTTL
	}
	if s.quota == 0 {
		s.quota = defaultQuota
	}
	if len(s.plans) == 0 {
		s.plans = defaultPlans()
	}
	s.handler = s.newMux()
	return s
}

func (s *Server) newMux() http.Handler {
	options := connect.WithInterceptors(syncprotocol.NewVersionHandlerInterceptor(s.protocol), callCounter{server: s})
	mux := http.NewServeMux()
	mux.Handle(syncv1connect.NewSignInServiceHandler(s, options))
	mux.Handle(syncv1connect.NewAccountServiceHandler(s, options))
	mux.Handle(syncv1connect.NewLinkServiceHandler(s, options))
	mux.Handle(syncv1connect.NewCatalogServiceHandler(s, options))
	mux.Handle(syncv1connect.NewRevisionServiceHandler(s, options))
	mux.Handle(syncv1connect.NewBlobServiceHandler(s, options))
	mux.Handle(syncv1connect.NewSubscriptionServiceHandler(s, options))
	mux.Handle(syncv1connect.NewRestoreServiceHandler(s, options))
	mux.Handle(syncv1connect.NewWatchServiceHandler(s, options))
	mux.HandleFunc("/admin/streams", s.serveStreams)
	mux.HandleFunc("/admin/streams/end", s.serveStreamsEnd)
	mux.Handle(syncv1connect.NewShareServiceHandler(s, options))
	mux.HandleFunc(payFolder, s.servePay)
	mux.HandleFunc("/admin/subscription", s.serveSubscription)
	mux.HandleFunc("/admin/usage", s.serveUsage)
	mux.HandleFunc(portalFolder, s.servePortal)
	mux.HandleFunc("/admin/trial", s.serveTrial)
	mux.HandleFunc("/admin/cancel-trial", s.serveCancelTrial)
	mux.HandleFunc("/blobs/", s.serveBlob)
	mux.HandleFunc("/admin/sessions", s.serveSessions)
	mux.HandleFunc("/admin/sessions/end", s.serveEnd)
	mux.HandleFunc("/admin/blobs", s.serveStoredBlobs)
	mux.HandleFunc("/admin/backup", s.serveBackup)
	mux.HandleFunc("/admin/restore", s.serveRestore)
	mux.HandleFunc("/admin/announce", s.serveAnnounce)
	mux.HandleFunc("/admin/retention", s.serveRetention)
	mux.HandleFunc("/admin/blob-loss", s.serveBlobLoss)
	mux.HandleFunc("/admin/part-log", s.servePartLog)
	mux.HandleFunc("/admin/stop-after-bytes", s.serveStopAfter(&s.stopUpload))
	mux.HandleFunc("/admin/stop-after-download-bytes", s.serveStopAfter(&s.stopDownload))
	mux.HandleFunc("/admin/prune-tombstones", s.servePrune)
	return mux
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.handler.ServeHTTP(w, r) }

// SetAudience sets the address the server tells itself by, once the address it listens on is known. The
// presigned blob URLs are made from it.
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

// EndSessions drops the sessions of one device, or of all devices when deviceSignKey is nil.
func (s *Server) EndSessions(deviceSignKey []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions = slices.DeleteFunc(s.sessions, func(one Session) bool {
		return deviceSignKey == nil || bytes.Equal(one.DeviceSignKey, deviceSignKey)
	})
}

// CountCalls returns how many calls of the procedure the server has run, for example
// syncv1connect.BlobServiceReserveBlobsProcedure. A call of another protocol version is not counted.
func (s *Server) CountCalls(procedure string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls[procedure]
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
	session, err := s.openSession(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&syncv1.SignInResponse{
		SessionToken: session.Token, ExpiresAt: session.ExpiresAt.Unix(), AccountFingerprint: session.AccountFingerprint,
	}), nil
}

func (s *Server) openSession(req *syncv1.SignInRequest) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pending, found := s.popChallenge(req.GetChallengeId())
	if !found {
		return Session{}, newFailedSignIn()
	}
	chain, ok := s.checkProof(pending, req)
	if !ok {
		return Session{}, newFailedSignIn()
	}
	head, ok := s.mergeChain(chain)
	if !ok {
		return Session{}, newFailedSignIn()
	}
	key := req.GetDeviceSignKey()
	if !head.IsActiveDevice(key) {
		if head.IsKnownDevice(key) {
			return Session{}, newDeviceRevoked(head)
		}
		return Session{}, newFailedSignIn()
	}
	session, ok := s.addSession(key, head.Root().Fingerprint().String())
	if !ok {
		return Session{}, newFailedSignIn()
	}
	return session, nil
}

func newFailedSignIn() error {
	return connect.NewError(connect.CodeUnauthenticated, errors.New(failedSignIn))
}

func newDeviceRevoked(head identity.Chain) error {
	failure := newFailedSignIn()
	if detail, err := connect.NewErrorDetail(&syncv1.DeviceRevoked{Chain: head.Bytes()}); err == nil {
		if refusal, ok := errors.AsType[*connect.Error](failure); ok {
			refusal.AddDetail(detail)
		}
	}
	return failure
}

func (s *Server) popChallenge(id []byte) (challenge, bool) {
	pending, found := s.challenges[string(id)]
	delete(s.challenges, string(id))
	return pending, found && pending.expiresAt >= s.now().Unix()
}

func (s *Server) checkProof(pending challenge, req *syncv1.SignInRequest) (identity.Chain, bool) {
	key := req.GetDeviceSignKey()
	if !bytes.Equal(pending.deviceSignKey, key) || len(req.GetSignature()) != 64 {
		return identity.Chain{}, false
	}
	message := syncprotocol.GetSignInMessage(s.getChallengeResponse(pending), key, s.protocol)
	if !identity.VerifyForPlugin(key, syncprotocol.SignInPluginID, message, [64]byte(req.GetSignature())) {
		return identity.Chain{}, false
	}
	chain, err := identity.ParseChain(req.GetChain())
	if err != nil || !chain.IsKnownDevice(key) {
		return identity.Chain{}, false
	}
	return chain, true
}

func (s *Server) getChallengeResponse(pending challenge) *syncv1.GetChallengeResponse {
	return &syncv1.GetChallengeResponse{Nonce: pending.nonce, ExpiresAt: pending.expiresAt, Audience: s.audience}
}

func isDeviceOf(chain identity.Chain, deviceSignKey []byte) bool {
	return slices.ContainsFunc(chain.ListDevices(), func(d identity.Device) bool { return bytes.Equal(d.Public.Sign, deviceSignKey) })
}

func (s *Server) mergeChain(chain identity.Chain) (identity.Chain, bool) {
	account := chain.Root().Fingerprint().String()
	stored, known := s.chains[account]
	if !known || chain.IsExtensionOf(stored) {
		s.applyChain(account, chain)
		return chain, true
	}
	if stored.IsExtensionOf(chain) {
		return stored, true
	}
	return identity.Chain{}, false
}

func (s *Server) addSession(deviceSignKey []byte, account string) (Session, bool) {
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return Session{}, false
	}
	session := Session{
		Token: base64.RawURLEncoding.EncodeToString(token), DeviceSignKey: slices.Clone(deviceSignKey), AccountFingerprint: account, ExpiresAt: s.now().Add(s.ttl),
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
