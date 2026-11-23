package identity_test

import (
	"errors"
	"regexp"
	"testing"
	"time"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

var sixDigits = regexp.MustCompile(`^[0-9]{6}$`)

type scanned struct {
	host, joiner *linkDevice
	relay        *relay
	clock        *testsupport.ManualClock
}

func newScanPair(tb testing.TB) scanned {
	tb.Helper()
	clock := testsupport.NewClockAt(getNow())
	host, joiner := newLinkDevice("Laptop", clock), newLinkDevice("Phone", clock)
	host.createAccount(tb)
	r := newRelay(tb, host)
	if _, err := joiner.start().Execute(tb.Context(), identitydomain.LinkRoleJoiner, identitydomain.LinkModeShow, ""); err != nil {
		tb.Fatalf("joiner StartLink() = %v", err)
	}
	r.settle(tb, host, joiner)
	qr := joiner.getLink(tb).QRText
	if _, err := host.start().Execute(tb.Context(), identitydomain.LinkRoleHost, identitydomain.LinkModeScan, qr); err != nil {
		tb.Fatalf("host StartLink() = %v", err)
	}
	r.settle(tb, host, joiner)
	return scanned{host: host, joiner: joiner, relay: r, clock: clock}
}

func TestStartLink_ShowsAQRTextOnceTheChannelIsKnown(t *testing.T) {
	t.Parallel()
	clock := testsupport.NewClockAt(getNow())
	host, other := newLinkDevice("Laptop", clock), newLinkDevice("Phone", clock)
	host.createAccount(t)
	r := newRelay(t, host)
	started, err := host.start().Execute(t.Context(), identitydomain.LinkRoleHost, identitydomain.LinkModeShow, "")
	if err != nil {
		t.Fatal(err)
	}
	if started.Phase != identitydomain.LinkPhaseWaitingForChannel || started.QRText != "" {
		t.Errorf("before the channel: %+v, want WAITING_FOR_CHANNEL and no QR text", started)
	}
	r.settle(t, host, other)
	shown := host.getLink(t)
	if shown.Phase != identitydomain.LinkPhaseWaitingForPeer || shown.Role != identitydomain.LinkRoleHost || shown.Mode != identitydomain.LinkModeShow {
		t.Errorf("link = %+v, want a host in mode show, WAITING_FOR_PEER", shown)
	}
	payload, err := identitylib.ParseLinkPayload(shown.QRText)
	if err != nil || string(payload.Channel[:]) != string(r.channel) {
		t.Errorf("QR text %q: payload = %+v, error = %v, want the channel of the relay", shown.QRText, payload, err)
	}
	if shown.Code != "" {
		t.Error("a host holds a confirmation code before a new device asks")
	}
}

func TestLink_ByQRCodeOfHostSendsAccountToNewDevice(t *testing.T) {
	t.Parallel()
	l := newLinkPair(t)
	hostIdentity, err := l.host.get().Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	joined, err := l.joiner.get().Execute(t.Context())
	if err != nil {
		t.Fatalf("the new device has no identity: %v", err)
	}
	if joined.AccountFingerprint != hostIdentity.AccountFingerprint || joined.DeviceName != "Phone" {
		t.Errorf("joined = %+v, want the account of the host and the name Phone", joined)
	}
	for name, d := range map[string]*linkDevice{"host": l.host, "joiner": l.joiner} {
		if got := len(d.readChain(t).ListDevices()); got != 2 {
			t.Errorf("%s: the chain file lists %d devices, want 2", name, got)
		}
		if phase := d.getLink(t).Phase; phase != identitydomain.LinkPhaseLinked {
			t.Errorf("%s: phase = %s, want LINKED", name, phase)
		}
	}
	if l.relay.head.Len() != l.host.readChain(t).Len() {
		t.Error("the server holds another chain than the host")
	}
}

func TestLink_TheHostSeesTheNameAndTheFingerprintOfTheNewDevice(t *testing.T) {
	t.Parallel()
	clock := testsupport.NewClockAt(getNow())
	host, joiner := newLinkDevice("Laptop", clock), newLinkDevice("Phone", clock)
	host.createAccount(t)
	r := newRelay(t, host)
	if _, err := host.start().Execute(t.Context(), identitydomain.LinkRoleHost, identitydomain.LinkModeShow, ""); err != nil {
		t.Fatal(err)
	}
	r.settle(t, host, joiner)
	if _, err := joiner.start().Execute(t.Context(), identitydomain.LinkRoleJoiner, identitydomain.LinkModeScan, host.getLink(t).QRText); err != nil {
		t.Fatal(err)
	}
	r.settle(t, host, joiner)
	seen := host.getLink(t)
	if seen.Phase != identitydomain.LinkPhaseConfirmPeer || seen.PeerName != "Phone" || !fingerprint.MatchString(seen.PeerFingerprint) {
		t.Errorf("host link = %+v, want CONFIRM_PEER with the name Phone and a fingerprint", seen)
	}
	if !sixDigits.MatchString(seen.Code) {
		t.Errorf("the host shows code %q for a QR it displayed itself, want 6 digits", seen.Code)
	}
	if waiting := joiner.getLink(t); waiting.Phase != identitydomain.LinkPhaseWaitingForAccept {
		t.Errorf("joiner phase = %s, want WAITING_FOR_ACCEPT", waiting.Phase)
	}
}

func TestLink_NothingIsStoredByTheNewDeviceBeforeThePersonConfirmsTheAccount(t *testing.T) {
	t.Parallel()
	clock := testsupport.NewClockAt(getNow())
	host, joiner := newLinkDevice("Laptop", clock), newLinkDevice("Phone", clock)
	host.createAccount(t)
	r := newRelay(t, host)
	if _, err := host.start().Execute(t.Context(), identitydomain.LinkRoleHost, identitydomain.LinkModeShow, ""); err != nil {
		t.Fatal(err)
	}
	r.settle(t, host, joiner)
	if _, err := joiner.start().Execute(t.Context(), identitydomain.LinkRoleJoiner, identitydomain.LinkModeScan, host.getLink(t).QRText); err != nil {
		t.Fatal(err)
	}
	r.settle(t, host, joiner)
	approveAndEnterCode(t, r, host, joiner)

	asked := joiner.getLink(t)
	hostIdentity, err := host.get().Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if asked.Phase != identitydomain.LinkPhaseConfirmAccount || asked.AccountFingerprint != hostIdentity.AccountFingerprint {
		t.Errorf("joiner link = %+v, want CONFIRM_ACCOUNT with the account fingerprint of the host", asked)
	}
	if got := joiner.keys.snapshot(); len(got) != 0 {
		t.Errorf("the key store of the new device holds %v before the confirm", got)
	}
	if _, err := joiner.chains.Read(t.Context()); !errors.Is(err, identitydomain.ErrNoChain) {
		t.Errorf("the new device wrote a chain file before the confirm: %v", err)
	}
	if err := joiner.confirm().Execute(t.Context(), false); err != nil {
		t.Fatal(err)
	}
	r.settle(t, host, joiner)
	if len(joiner.keys.snapshot()) != 0 {
		t.Error("the new device stored keys after it declined the account")
	}
	if got := host.getLink(t).Phase; got != identitydomain.LinkPhaseFailed {
		t.Errorf("host phase = %s, want FAILED", got)
	}
}

func TestDecideLink_TheHostStoresTheLongerChainOnlyAfterTheServerAcceptedIt(t *testing.T) {
	t.Parallel()
	clock := testsupport.NewClockAt(getNow())
	host, joiner := newLinkDevice("Laptop", clock), newLinkDevice("Phone", clock)
	host.createAccount(t)
	r := newRelay(t, host)
	if _, err := host.start().Execute(t.Context(), identitydomain.LinkRoleHost, identitydomain.LinkModeShow, ""); err != nil {
		t.Fatal(err)
	}
	r.settle(t, host, joiner)
	if _, err := joiner.start().Execute(t.Context(), identitydomain.LinkRoleJoiner, identitydomain.LinkModeScan, host.getLink(t).QRText); err != nil {
		t.Fatal(err)
	}
	r.settle(t, host, joiner)
	approveAndEnterCode(t, r, host, joiner)

	pending, err := host.publish().Execute(t.Context())
	if err != nil || !pending.IsPending {
		t.Fatalf("GetChainPublish() = %+v, %v, want a pending chain", pending, err)
	}
	chain, err := identitylib.ParseChain(pending.Chain)
	if err != nil || len(chain.ListDevices()) != 2 {
		t.Fatalf("the pending chain: %v, devices = %d, want 2", err, len(chain.ListDevices()))
	}
	if got := len(host.readChain(t).ListDevices()); got != 1 {
		t.Errorf("the host stored a chain of %d devices before the server accepted it", got)
	}
	if got := host.getLink(t).Phase; got != identitydomain.LinkPhasePublishing {
		t.Errorf("phase = %s, want PUBLISHING", got)
	}
	grant, err := host.work().Execute(t.Context(), host.since, 0)
	if err != nil || len(grant.Outbound) != 0 {
		t.Errorf("the host sent %d blobs before the server accepted the chain: %v", len(grant.Outbound), err)
	}
	if err := host.answered().Execute(t.Context(), true, nil); err != nil {
		t.Fatal(err)
	}
	if got := len(host.readChain(t).ListDevices()); got != 2 {
		t.Errorf("the host chain lists %d devices after the accept, want 2", got)
	}
}

func TestLink_TheHostRebuildsOnTheHeadWhenAnotherDeviceAppendedFirst(t *testing.T) {
	t.Parallel()
	clock := testsupport.NewClockAt(getNow())
	host, joiner := newLinkDevice("Laptop", clock), newLinkDevice("Phone", clock)
	host.createAccount(t)
	r := newRelay(t, host)
	r.staleOnce = func(head identitylib.Chain) identitylib.Chain {
		next, err := head.AppendDevice(host.readRecord(t).Device, newDeviceKey(t).Public(), "Tablet", getNow().Add(time.Minute))
		if err != nil {
			t.Fatal(err)
		}
		return next
	}
	if _, err := host.start().Execute(t.Context(), identitydomain.LinkRoleHost, identitydomain.LinkModeShow, ""); err != nil {
		t.Fatal(err)
	}
	r.settle(t, host, joiner)
	if _, err := joiner.start().Execute(t.Context(), identitydomain.LinkRoleJoiner, identitydomain.LinkModeScan, host.getLink(t).QRText); err != nil {
		t.Fatal(err)
	}
	r.settle(t, host, joiner)
	approveAndEnterCode(t, r, host, joiner)
	if err := joiner.confirm().Execute(t.Context(), true); err != nil {
		t.Fatal(err)
	}
	r.settle(t, host, joiner)
	if got := len(r.head.ListDevices()); got != 3 {
		t.Errorf("the server holds %d devices, want the host, the tablet and the new device", got)
	}
	if got := joiner.getLink(t).Phase; got != identitydomain.LinkPhaseLinked {
		t.Errorf("joiner phase = %s, want LINKED after one rebuild", got)
	}
}

func TestLink_ADeclinedDeviceDoesNotJoinAndTheChainStaysTheSame(t *testing.T) {
	t.Parallel()
	clock := testsupport.NewClockAt(getNow())
	host, joiner := newLinkDevice("Laptop", clock), newLinkDevice("Phone", clock)
	host.createAccount(t)
	r := newRelay(t, host)
	before := host.readChain(t).Bytes()
	if _, err := host.start().Execute(t.Context(), identitydomain.LinkRoleHost, identitydomain.LinkModeShow, ""); err != nil {
		t.Fatal(err)
	}
	r.settle(t, host, joiner)
	if _, err := joiner.start().Execute(t.Context(), identitydomain.LinkRoleJoiner, identitydomain.LinkModeScan, host.getLink(t).QRText); err != nil {
		t.Fatal(err)
	}
	r.settle(t, host, joiner)
	if err := host.decide().Execute(t.Context(), false); err != nil {
		t.Fatal(err)
	}
	r.settle(t, host, joiner)

	declined := joiner.getLink(t)
	if declined.Phase != identitydomain.LinkPhaseFailed || declined.Reason != identitydomain.LinkReasonDeclined {
		t.Errorf("joiner link = %+v, want FAILED, declined", declined)
	}
	if string(host.readChain(t).Bytes()) != string(before) || r.head.Len() != host.readChain(t).Len() {
		t.Error("a declined device changed a chain")
	}
	if _, err := joiner.get().Execute(t.Context()); !errors.Is(err, identitydomain.ErrNoIdentity) {
		t.Errorf("the declined device has an identity: %v", err)
	}
}

func TestLink_TheHostScansTheQRCodeOfTheNewDeviceAndTheNewDeviceTypesTheCode(t *testing.T) {
	t.Parallel()
	s := newScanPair(t)
	seen := s.host.getLink(t)
	if seen.Phase != identitydomain.LinkPhaseConfirmPeer || seen.PeerName != "Phone" || !sixDigits.MatchString(seen.Code) {
		t.Fatalf("host link = %+v, want CONFIRM_PEER with the name Phone and a 6 digit code", seen)
	}
	if err := s.host.decide().Execute(t.Context(), true); err != nil {
		t.Fatal(err)
	}
	s.relay.settle(t, s.host, s.joiner)
	if got := s.joiner.getLink(t).Phase; got != identitydomain.LinkPhaseEnterCode {
		t.Fatalf("joiner phase = %s, want ENTER_CODE", got)
	}
	if got := len(s.host.readChain(t).ListDevices()); got != 1 {
		t.Errorf("the host chain lists %d devices before the code, want 1", got)
	}
	if err := s.joiner.code().Execute(t.Context(), seen.Code); err != nil {
		t.Fatal(err)
	}
	s.relay.settle(t, s.host, s.joiner)
	asked := s.joiner.getLink(t)
	if asked.Phase != identitydomain.LinkPhaseConfirmAccount {
		t.Fatalf("joiner phase = %s, want CONFIRM_ACCOUNT", asked.Phase)
	}
	if err := s.joiner.confirm().Execute(t.Context(), true); err != nil {
		t.Fatal(err)
	}
	s.relay.settle(t, s.host, s.joiner)
	if got := len(s.host.readChain(t).ListDevices()); got != 2 {
		t.Errorf("the host chain lists %d devices, want 2", got)
	}
	if _, err := s.joiner.get().Execute(t.Context()); err != nil {
		t.Errorf("the new device has no identity: %v", err)
	}
}

func TestLink_TheCodeIsNotTheSameForTwoLinks(t *testing.T) {
	t.Parallel()
	codes := make(map[string]bool)
	for range 5 {
		codes[newScanPair(t).host.getLink(t).Code] = true
	}
	if len(codes) < 2 {
		t.Errorf("five links showed %d different codes", len(codes))
	}
}

func wrongCode(code string) string {
	if code[0] == '0' {
		return "1" + code[1:]
	}
	return "0" + code[1:]
}

func TestEnterLinkCode_WrongCodeReportsRemainingTries(t *testing.T) {
	t.Parallel()
	s := newScanPair(t)
	code := s.host.getLink(t).Code
	if err := s.host.decide().Execute(t.Context(), true); err != nil {
		t.Fatal(err)
	}
	s.relay.settle(t, s.host, s.joiner)
	if err := s.joiner.code().Execute(t.Context(), wrongCode(code)); err != nil {
		t.Fatal(err)
	}
	s.relay.settle(t, s.host, s.joiner)
	asked := s.joiner.getLink(t)
	if asked.Phase != identitydomain.LinkPhaseEnterCode || asked.TriesLeft != 2 {
		t.Errorf("joiner link = %+v, want ENTER_CODE with 2 tries left", asked)
	}
	if got := len(s.host.readChain(t).ListDevices()); got != 1 {
		t.Errorf("a wrong code linked a device: the chain lists %d", got)
	}
}

func TestEnterLinkCode_ThreeWrongCodesCloseTheLinkEvenForTheRightCode(t *testing.T) {
	t.Parallel()
	s := newScanPair(t)
	code := s.host.getLink(t).Code
	if err := s.host.decide().Execute(t.Context(), true); err != nil {
		t.Fatal(err)
	}
	s.relay.settle(t, s.host, s.joiner)
	for range 3 {
		if err := s.joiner.code().Execute(t.Context(), wrongCode(code)); err != nil {
			t.Fatal(err)
		}
		s.relay.settle(t, s.host, s.joiner)
	}
	closed := s.joiner.getLink(t)
	if closed.Phase != identitydomain.LinkPhaseFailed || closed.Reason != identitydomain.LinkReasonTooManyWrongCodes {
		t.Fatalf("joiner link = %+v, want FAILED, too many wrong codes", closed)
	}
	if err := s.joiner.code().Execute(t.Context(), code); err == nil {
		t.Error("the right code was accepted after three wrong ones")
	}
	s.relay.settle(t, s.host, s.joiner)
	if !s.relay.isClosed {
		t.Error("the relay channel was not closed")
	}
	if got := len(s.host.readChain(t).ListDevices()); got != 1 {
		t.Errorf("the chain lists %d devices, want 1", got)
	}
	if _, err := s.joiner.get().Execute(t.Context()); !errors.Is(err, identitydomain.ErrNoIdentity) {
		t.Errorf("the new device has an identity: %v", err)
	}
}

func TestStartLink_ADeviceWithAnIdentityDoesNotJoin(t *testing.T) {
	t.Parallel()
	clock := testsupport.NewClockAt(getNow())
	device := newLinkDevice("Desktop", clock)
	device.createAccount(t)
	_, err := device.start().Execute(t.Context(), identitydomain.LinkRoleJoiner, identitydomain.LinkModeShow, "")
	if !errors.Is(err, identitydomain.ErrIdentityExists) {
		t.Errorf("error = %v, want ErrIdentityExists", err)
	}
	if _, err := device.view().Execute(t.Context()); !errors.Is(err, identitydomain.ErrNoLink) {
		t.Errorf("GetLink() = %v, want ErrNoLink: nothing was started", err)
	}
}

func TestStartLink_ADeviceWithoutAnIdentityCannotHostALink(t *testing.T) {
	t.Parallel()
	device := newLinkDevice("Phone", testsupport.NewClockAt(getNow()))
	if _, err := device.start().Execute(t.Context(), identitydomain.LinkRoleHost, identitydomain.LinkModeShow, ""); !errors.Is(err, identitydomain.ErrNoIdentity) {
		t.Errorf("error = %v, want ErrNoIdentity", err)
	}
}

func TestStartLink_RefusesASecondLinkAndATextThatIsNotAQRCode(t *testing.T) {
	t.Parallel()
	clock := testsupport.NewClockAt(getNow())
	host := newLinkDevice("Laptop", clock)
	host.createAccount(t)
	if _, err := host.start().Execute(t.Context(), identitydomain.LinkRoleHost, identitydomain.LinkModeScan, "https://example.org"); !errors.Is(err, identitylib.ErrLinkPayload) {
		t.Errorf("a text that is not a link: error = %v, want ErrLinkPayload", err)
	}
	if _, err := host.start().Execute(t.Context(), identitydomain.LinkRoleHost, identitydomain.LinkModeShow, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := host.start().Execute(t.Context(), identitydomain.LinkRoleHost, identitydomain.LinkModeShow, ""); !errors.Is(err, identitydomain.ErrLinkActive) {
		t.Errorf("a second link: error = %v, want ErrLinkActive", err)
	}
}

func TestCancelLink_EndsTheLinkOnBothDevices(t *testing.T) {
	t.Parallel()
	s := newScanPair(t)
	if err := s.host.cancel().Execute(t.Context()); err != nil {
		t.Fatal(err)
	}
	s.relay.settle(t, s.host, s.joiner)
	if _, err := s.host.view().Execute(t.Context()); !errors.Is(err, identitydomain.ErrNoLink) {
		t.Errorf("host GetLink() = %v, want ErrNoLink", err)
	}
	if got := s.joiner.getLink(t); got.Phase != identitydomain.LinkPhaseFailed || got.Reason != identitydomain.LinkReasonCanceled {
		t.Errorf("joiner link = %+v, want FAILED, canceled", got)
	}
	if err := s.host.cancel().Execute(t.Context()); !errors.Is(err, identitydomain.ErrNoLink) {
		t.Errorf("cancel without a link: error = %v, want ErrNoLink", err)
	}
}

func TestGetLink_ALinkEndsTenMinutesAfterItStarted(t *testing.T) {
	t.Parallel()
	clock := testsupport.NewClockAt(getNow())
	host := newLinkDevice("Laptop", clock)
	host.createAccount(t)
	if _, err := host.start().Execute(t.Context(), identitydomain.LinkRoleHost, identitydomain.LinkModeShow, ""); err != nil {
		t.Fatal(err)
	}
	clock.Advance(10*time.Minute - time.Second)
	if got := host.getLink(t); got.Phase == identitydomain.LinkPhaseFailed {
		t.Fatalf("the link ended after 9 minutes 59 seconds: %+v", got)
	}
	clock.Advance(2 * time.Second)
	expired := host.getLink(t)
	if expired.Phase != identitydomain.LinkPhaseFailed || expired.Reason != identitydomain.LinkReasonExpired {
		t.Errorf("link = %+v, want FAILED, expired", expired)
	}
	work, err := host.work().Execute(t.Context(), 0, 0)
	if err != nil || !work.IsClosed {
		t.Errorf("GetLinkWork() = %+v, %v, want the channel closed", work, err)
	}
}

func TestGetLinkWork_ReturnsNoWorkWhenNoLinkIsActive(t *testing.T) {
	t.Parallel()
	host := newLinkDevice("Laptop", testsupport.NewClockAt(getNow()))
	host.createAccount(t)
	work, err := host.work().Execute(t.Context(), 0, 0)
	if err != nil || work.IsActive || work.WantsChannel || len(work.Outbound) != 0 {
		t.Errorf("GetLinkWork() = %+v, %v, want an inactive answer", work, err)
	}
	if err := host.inbound().Execute(t.Context(), []byte("blob")); !errors.Is(err, identitydomain.ErrNoLink) {
		t.Errorf("PutLinkInbound() without a link = %v, want ErrNoLink", err)
	}
}

func TestGetLinkWork_ReturnsABlobUntilTheNextCallAcknowledgesIt(t *testing.T) {
	t.Parallel()
	clock := testsupport.NewClockAt(getNow())
	host, joiner := newLinkDevice("Laptop", clock), newLinkDevice("Phone", clock)
	host.createAccount(t)
	r := newRelay(t, host)
	if _, err := host.start().Execute(t.Context(), identitydomain.LinkRoleHost, identitydomain.LinkModeShow, ""); err != nil {
		t.Fatal(err)
	}
	r.settle(t, host, joiner)
	if _, err := joiner.start().Execute(t.Context(), identitydomain.LinkRoleJoiner, identitydomain.LinkModeScan, host.getLink(t).QRText); err != nil {
		t.Fatal(err)
	}
	first, err := joiner.work().Execute(t.Context(), 0, 0)
	if err != nil || len(first.Outbound) == 0 {
		t.Fatalf("GetLinkWork() = %+v, %v, want the first message of the scanner", first, err)
	}
	again, err := joiner.work().Execute(t.Context(), 0, 0)
	if err != nil || len(again.Outbound) != len(first.Outbound) {
		t.Errorf("an unacknowledged blob came back %d times, want %d", len(again.Outbound), len(first.Outbound))
	}
	acknowledged, err := joiner.work().Execute(t.Context(), first.Version, 0)
	if err != nil || len(acknowledged.Outbound) != 0 {
		t.Errorf("after the acknowledgment: %+v, %v, want no blobs", acknowledged, err)
	}
}

func newShownPair(tb testing.TB) scanned {
	tb.Helper()
	clock := testsupport.NewClockAt(getNow())
	host, joiner := newLinkDevice("Laptop", clock), newLinkDevice("Phone", clock)
	host.createAccount(tb)
	r := newRelay(tb, host)
	if _, err := host.start().Execute(tb.Context(), identitydomain.LinkRoleHost, identitydomain.LinkModeShow, ""); err != nil {
		tb.Fatal(err)
	}
	r.settle(tb, host, joiner)
	if _, err := joiner.start().Execute(tb.Context(), identitydomain.LinkRoleJoiner, identitydomain.LinkModeScan, host.getLink(tb).QRText); err != nil {
		tb.Fatal(err)
	}
	r.settle(tb, host, joiner)
	return scanned{host: host, joiner: joiner, relay: r, clock: clock}
}

func TestLink_ADeviceThatScannedTheQRCodeOfTheHostMustTypeTheCodeTheHostShows(t *testing.T) {
	t.Parallel()
	s := newShownPair(t)
	code := s.host.getLink(t).Code
	if err := s.host.decide().Execute(t.Context(), true); err != nil {
		t.Fatal(err)
	}
	s.relay.settle(t, s.host, s.joiner)

	if got := s.joiner.getLink(t).Phase; got != identitydomain.LinkPhaseEnterCode {
		t.Fatalf("joiner phase = %s, want ENTER_CODE after the approval", got)
	}
	if got := len(s.host.readChain(t).ListDevices()); got != 1 {
		t.Errorf("the host chain lists %d devices before the code, want 1", got)
	}
	if err := s.joiner.code().Execute(t.Context(), wrongCode(code)); err != nil {
		t.Fatal(err)
	}
	s.relay.settle(t, s.host, s.joiner)
	if got := len(s.host.readChain(t).ListDevices()); got != 1 {
		t.Errorf("the host chain lists %d devices after a wrong code, want 1", got)
	}
	if err := s.joiner.code().Execute(t.Context(), code); err != nil {
		t.Fatal(err)
	}
	s.relay.settle(t, s.host, s.joiner)
	if got := s.joiner.getLink(t).Phase; got != identitydomain.LinkPhaseConfirmAccount {
		t.Errorf("joiner phase = %s, want CONFIRM_ACCOUNT after the right code", got)
	}
}

func TestLink_ThreeWrongCodesCloseALinkTheHostShowed(t *testing.T) {
	t.Parallel()
	s := newShownPair(t)
	code := s.host.getLink(t).Code
	if err := s.host.decide().Execute(t.Context(), true); err != nil {
		t.Fatal(err)
	}
	s.relay.settle(t, s.host, s.joiner)

	for range 3 {
		if err := s.joiner.code().Execute(t.Context(), wrongCode(code)); err != nil {
			t.Fatal(err)
		}
		s.relay.settle(t, s.host, s.joiner)
	}

	if got := s.host.getLink(t); got.Phase != identitydomain.LinkPhaseFailed || got.Reason != identitydomain.LinkReasonTooManyWrongCodes {
		t.Errorf("host link = %+v, want FAILED, too many wrong codes", got)
	}
}
