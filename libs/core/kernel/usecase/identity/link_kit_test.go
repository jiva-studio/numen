package identity_test

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"sync"
	"testing"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

// fakeChainStore is the chain file of one device kept in memory.
type fakeChainStore struct {
	mu     sync.Mutex
	chain  []byte
	writes int
}

func (s *fakeChainStore) Read(context.Context) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.chain == nil {
		return nil, identitydomain.ErrNoChain
	}
	return append([]byte(nil), s.chain...), nil
}

func (s *fakeChainStore) Write(_ context.Context, chain []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.chain = append([]byte(nil), chain...)
	s.writes++
	return nil
}

func (s *fakeChainStore) Delete(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.chain = nil
	return nil
}

// linkDevice is one device: its key store and its chain file. The use cases are made over its deps at each call.
type linkDevice struct {
	keys   *fakeKeyStore
	chains *fakeChainStore
	deps   identity.Deps
	since  uint64
}

func newLinkDevice(name string, clock *testsupport.ManualClock) *linkDevice {
	keys, chains := newFakeKeyStore(), &fakeChainStore{}
	deps := identity.NewDeps(keys, fakeNames{name: name}, clock).WithChains(chains)
	return &linkDevice{keys: keys, chains: chains, deps: deps}
}

func (d *linkDevice) create() *identity.CreateIdentity   { return identity.NewCreateIdentity(d.deps) }
func (d *linkDevice) get() *identity.GetIdentity         { return identity.NewGetIdentity(d.deps) }
func (d *linkDevice) start() *identity.StartLink         { return identity.NewStartLink(d.deps) }
func (d *linkDevice) view() *identity.GetLink            { return identity.NewGetLink(d.deps) }
func (d *linkDevice) decide() *identity.DecideLink       { return identity.NewDecideLink(d.deps) }
func (d *linkDevice) code() *identity.EnterLinkCode      { return identity.NewEnterLinkCode(d.deps) }
func (d *linkDevice) confirm() *identity.ConfirmJoin     { return identity.NewConfirmJoin(d.deps) }
func (d *linkDevice) cancel() *identity.CancelLink       { return identity.NewCancelLink(d.deps) }
func (d *linkDevice) devices() *identity.ListDevices     { return identity.NewListDevices(d.deps) }
func (d *linkDevice) revoke() *identity.RevokeDevice     { return identity.NewRevokeDevice(d.deps) }
func (d *linkDevice) work() *identity.GetLinkWork        { return identity.NewGetLinkWork(d.deps) }
func (d *linkDevice) channel() *identity.SetLinkChannel  { return identity.NewSetLinkChannel(d.deps) }
func (d *linkDevice) inbound() *identity.PutLinkInbound  { return identity.NewPutLinkInbound(d.deps) }
func (d *linkDevice) publish() *identity.GetChainPublish { return identity.NewGetChainPublish(d.deps) }
func (d *linkDevice) answered() *identity.PutChainPublished {
	return identity.NewPutChainPublished(d.deps)
}
func (d *linkDevice) adopt() *identity.AdoptChain { return identity.NewAdoptChain(d.deps) }
func (d *linkDevice) public() *identity.GetPublicIdentity {
	return identity.NewGetPublicIdentity(d.deps)
}
func (d *linkDevice) seal() *identity.SealData         { return identity.NewSealData(d.deps) }
func (d *linkDevice) remove() *identity.DeleteIdentity { return identity.NewDeleteIdentity(d.deps) }

func (d *linkDevice) createAccount(tb testing.TB) identitydomain.Identity {
	tb.Helper()
	made, err := d.create().Execute(tb.Context(), getLegalWords())
	if err != nil {
		tb.Fatalf("create identity: %v", err)
	}
	return made
}

func (d *linkDevice) readChain(tb testing.TB) identitylib.Chain {
	tb.Helper()
	raw, err := d.chains.Read(tb.Context())
	if err != nil {
		tb.Fatalf("read the chain file: %v", err)
	}
	chain, err := identitylib.ParseChain(raw)
	if err != nil {
		tb.Fatal(err)
	}
	return chain
}

func (d *linkDevice) readRecord(tb testing.TB) identitylib.Record {
	tb.Helper()
	raw, err := base64.RawURLEncoding.DecodeString(d.keys.snapshot()[keyName])
	if err != nil {
		tb.Fatal(err)
	}
	record, err := identitylib.ParseRecord(raw)
	if err != nil {
		tb.Fatal(err)
	}
	return record
}

func (d *linkDevice) getLink(tb testing.TB) identitydomain.Link {
	tb.Helper()
	link, err := d.view().Execute(tb.Context())
	if err != nil {
		tb.Fatalf("GetLink() = %v", err)
	}
	return link
}

// relay is the sync server as the courier sees it: it carries the blobs of a link between two devices and
// holds the chain the devices append to.
type relay struct {
	head      identitylib.Chain
	channel   []byte
	isClosed  bool
	published []error
	staleOnce func(head identitylib.Chain) identitylib.Chain
}

func newRelay(tb testing.TB, host *linkDevice) *relay {
	tb.Helper()
	return &relay{head: host.readChain(tb), channel: nil}
}

func (r *relay) settle(tb testing.TB, left, right *linkDevice) {
	tb.Helper()
	for range 30 {
		moved := false
		for _, side := range [][2]*linkDevice{{left, right}, {right, left}} {
			moved = r.carry(tb, side[0], side[1]) || moved
		}
		for _, device := range []*linkDevice{left, right} {
			moved = r.publish(tb, device) || moved
		}
		if !moved {
			return
		}
	}
	tb.Fatal("the link did not settle in 30 passes")
}

func (r *relay) carry(tb testing.TB, from, to *linkDevice) bool {
	tb.Helper()
	work, err := from.work().Execute(tb.Context(), from.since, 0)
	if err != nil {
		tb.Fatalf("GetLinkWork() = %v", err)
	}
	from.since = work.Version
	moved := false
	if work.WantsChannel && r.channel == nil {
		r.channel = []byte("0123456789abcdef")
		if err := from.channel().Execute(tb.Context(), r.channel); err != nil {
			tb.Fatalf("SetLinkChannel() = %v", err)
		}
		moved = true
	}
	for _, blob := range work.Outbound {
		moved = true
		if err := to.inbound().Execute(tb.Context(), blob); err != nil && !r.isClosed {
			tb.Fatalf("PutLinkInbound() = %v", err)
		}
	}
	if work.IsClosed {
		r.isClosed = true
	}
	return moved
}

func (r *relay) publish(tb testing.TB, device *linkDevice) bool {
	tb.Helper()
	pending, err := device.publish().Execute(tb.Context())
	if err != nil {
		tb.Fatalf("GetChainPublish() = %v", err)
	}
	if !pending.IsPending {
		return false
	}
	chain, err := identitylib.ParseChain(pending.Chain)
	if err != nil {
		tb.Fatalf("the core published a chain that does not parse: %v", err)
	}
	if r.staleOnce != nil {
		r.head, r.staleOnce = r.staleOnce(r.head), nil
	}
	if chain.Len() > r.head.Len() && chain.IsExtensionOf(r.head) {
		r.head = chain
		r.published = append(r.published, device.answered().Execute(tb.Context(), true, nil))
		return true
	}
	r.published = append(r.published, device.answered().Execute(tb.Context(), false, r.head.Bytes()))
	return true
}

// linked is two devices of one account, made by linking through a QR code shown by the first.
type linked struct {
	host, joiner *linkDevice
	relay        *relay
	clock        *testsupport.ManualClock
}

func newLinkPair(tb testing.TB) linked {
	tb.Helper()
	clock := testsupport.NewClockAt(getNow())
	host, joiner := newLinkDevice("Laptop", clock), newLinkDevice("Phone", clock)
	host.createAccount(tb)
	r := newRelay(tb, host)
	shown, err := host.start().Execute(tb.Context(), identitydomain.LinkRoleHost, identitydomain.LinkModeShow, "")
	if err != nil {
		tb.Fatalf("host StartLink() = %v", err)
	}
	r.settle(tb, host, joiner)
	qr := host.getLink(tb).QRText
	if qr == "" || shown.Phase == "" {
		tb.Fatalf("the host shows no QR text: %+v", host.getLink(tb))
	}
	if _, err := joiner.start().Execute(tb.Context(), identitydomain.LinkRoleJoiner, identitydomain.LinkModeScan, qr); err != nil {
		tb.Fatalf("joiner StartLink() = %v", err)
	}
	r.settle(tb, host, joiner)
	approveAndEnterCode(tb, r, host, joiner)
	if err := joiner.confirm().Execute(tb.Context(), true); err != nil {
		tb.Fatalf("ConfirmJoin() = %v", err)
	}
	r.settle(tb, host, joiner)
	r.published = nil
	return linked{host: host, joiner: joiner, relay: r, clock: clock}
}

func newDeviceKey(tb testing.TB) identitylib.DeviceKey {
	tb.Helper()
	key, err := identitylib.NewDeviceKey(rand.Reader)
	if err != nil {
		tb.Fatal(err)
	}
	return key
}

func approveAndEnterCode(tb testing.TB, r *relay, host, joiner *linkDevice) {
	tb.Helper()
	code := host.getLink(tb).Code
	if err := host.decide().Execute(tb.Context(), true); err != nil {
		tb.Fatalf("DecideLink() = %v", err)
	}
	r.settle(tb, host, joiner)
	if err := joiner.code().Execute(tb.Context(), code); err != nil {
		tb.Fatalf("EnterLinkCode() = %v", err)
	}
	r.settle(tb, host, joiner)
}
