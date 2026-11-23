package identity_test

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/identity"
)

func fixedBytes(start byte, n int) *bytes.Reader {
	out := make([]byte, n)
	for i := range out {
		out[i] = start + byte(i)
	}
	return bytes.NewReader(out)
}

func linkChannelID() [16]byte {
	var id [16]byte
	for i := range id {
		id[i] = 0xA0 + byte(i)
	}
	return id
}

type pair struct {
	displayer *identity.LinkDisplayer
	payload   identity.LinkPayload
	scanner   *identity.LinkScanner
	first     []byte
}

func newPair(tb testing.TB) pair {
	tb.Helper()
	displayer, payload, err := identity.NewLinkDisplayer(rand.Reader, linkChannelID(), "")
	if err != nil {
		tb.Fatal(err)
	}
	scanner, first, err := identity.NewLinkScanner(rand.Reader, payload)
	if err != nil {
		tb.Fatal(err)
	}
	return pair{displayer: displayer, payload: payload, scanner: scanner, first: first}
}

func (p pair) connect(tb testing.TB) (displayer, scanner *identity.LinkChannel) {
	tb.Helper()
	reply, displayerChannel, err := p.displayer.Accept(p.first)
	if err != nil {
		tb.Fatalf("displayer Accept() = %v", err)
	}
	scannerChannel, err := p.scanner.Accept(reply)
	if err != nil {
		tb.Fatalf("scanner Accept() = %v", err)
	}
	return displayerChannel, scannerChannel
}

func TestLinkPayload_TextRoundTrips(t *testing.T) {
	t.Parallel()
	for _, server := range []string{"", "https://sync.example.org", "http://localhost:8080"} {
		_, payload, err := identity.NewLinkDisplayer(rand.Reader, linkChannelID(), server)
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := identity.ParseLinkPayload(payload.Text())
		if err != nil || parsed != payload {
			t.Errorf("server %q: parsed = %+v, error = %v, want %+v", server, parsed, err, payload)
		}
	}
}

func TestLinkPayload_TextHasTheFrozenShape(t *testing.T) {
	t.Parallel()
	payload := identity.LinkPayload{Version: 1, Channel: linkChannelID(), Server: "https://sync.example.org"}
	payload.DisplayerKey[0], payload.Secret[0] = 7, 9
	enc := base64.RawURLEncoding
	want := "numen://link?v=1&c=" + enc.EncodeToString(payload.Channel[:]) +
		"&k=" + enc.EncodeToString(payload.DisplayerKey[:]) +
		"&s=" + enc.EncodeToString(payload.Secret[:]) + "&h=https://sync.example.org"
	if got := payload.Text(); got != want && got != strings.Replace(want, "&h=https://sync.example.org", "&h=https%3A%2F%2Fsync.example.org", 1) {
		t.Errorf("Text() = %q, want %q", got, want)
	}
	payload.Server = ""
	if strings.Contains(payload.Text(), "&h=") {
		t.Error("a payload without a server carries an h field")
	}
}

func TestParseLinkPayload_RefusesWhatIsNotAVersionOneLink(t *testing.T) {
	t.Parallel()
	_, payload, err := identity.NewLinkDisplayer(rand.Reader, linkChannelID(), "")
	if err != nil {
		t.Fatal(err)
	}
	good := payload.Text()
	enc := base64.RawURLEncoding
	before := func(marker string) string {
		headOf, _, _ := strings.Cut(good, marker)
		return headOf
	}
	cases := map[string]string{
		"empty":                                  "",
		"another scheme":                         strings.Replace(good, "numen://", "https://", 1),
		"another version":                        strings.Replace(good, "v=1", "v=2", 1),
		"no version":                             strings.Replace(good, "v=1&", "", 1),
		"extra field":                            good + "&x=1",
		"short secret":                           before("&s=") + "&s=" + enc.EncodeToString(make([]byte, 15)),
		"long secret":                            before("&s=") + "&s=" + enc.EncodeToString(make([]byte, 17)),
		"short channel":                          strings.Replace(good, "c="+enc.EncodeToString(payload.Channel[:]), "c="+enc.EncodeToString(make([]byte, 15)), 1),
		"not base64":                             before("&k=") + "&k=!!!&s=" + enc.EncodeToString(payload.Secret[:]),
		"server not http":                        good + "&h=ftp://sync.example.org",
		"server with path":                       good + "&h=https://sync.example.org/path",
		"server plain http":                      good + "&h=http://sync.example.org",
		"server plain http on a private address": good + "&h=http://192.168.1.20:8080",
	}
	for name, text := range cases {
		if _, err := identity.ParseLinkPayload(text); !errors.Is(err, identity.ErrLinkPayload) {
			t.Errorf("%s: error = %v, want ErrLinkPayload", name, err)
		}
	}
}

func TestNewLinkDisplayer_MakesAKeyAndASecretForEachLink(t *testing.T) {
	t.Parallel()
	_, first, err := identity.NewLinkDisplayer(rand.Reader, linkChannelID(), "https://sync.example.org")
	if err != nil {
		t.Fatal(err)
	}
	_, second, err := identity.NewLinkDisplayer(rand.Reader, linkChannelID(), "https://sync.example.org")
	if err != nil {
		t.Fatal(err)
	}
	if first.Version != 1 || first.Channel != linkChannelID() || first.Server != "https://sync.example.org" {
		t.Errorf("payload = %+v", first)
	}
	if first.DisplayerKey == second.DisplayerKey || first.Secret == second.Secret {
		t.Error("two links share a key or a secret")
	}
}

func TestNewLinkEnds_FailWhenTheRandomSourceFails(t *testing.T) {
	t.Parallel()
	broken := errors.New("no entropy")
	if _, _, err := identity.NewLinkDisplayer(failingReader{err: broken}, linkChannelID(), ""); !errors.Is(err, broken) {
		t.Errorf("displayer: error = %v, want the reader failure", err)
	}
	_, payload, err := identity.NewLinkDisplayer(rand.Reader, linkChannelID(), "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := identity.NewLinkScanner(failingReader{err: broken}, payload); !errors.Is(err, broken) {
		t.Errorf("scanner: error = %v, want the reader failure", err)
	}
}

func TestLinkChannel_CarriesMessagesBothWays(t *testing.T) {
	t.Parallel()
	displayer, scanner := newPair(t).connect(t)
	for _, text := range []string{"one", "", "three"} {
		opened, err := displayer.Open(scanner.Seal([]byte(text)))
		if err != nil || string(opened) != text {
			t.Errorf("scanner to displayer: %q, %v, want %q", opened, err, text)
		}
		opened, err = scanner.Open(displayer.Seal([]byte(text)))
		if err != nil || string(opened) != text {
			t.Errorf("displayer to scanner: %q, %v, want %q", opened, err, text)
		}
	}
}

// linkVector is the key schedule written out with the standard library, as the spec freezes it.
type linkVector struct {
	scannerPublic []byte
	s2d, d2s, th  []byte
}

func readLinkVector(tb testing.TB, scannerPrivate []byte, p identity.LinkPayload) linkVector {
	tb.Helper()
	private, err := ecdh.X25519().NewPrivateKey(scannerPrivate)
	if err != nil {
		tb.Fatal(err)
	}
	public, err := ecdh.X25519().NewPublicKey(p.DisplayerKey[:])
	if err != nil {
		tb.Fatal(err)
	}
	shared, err := private.ECDH(public)
	if err != nil {
		tb.Fatal(err)
	}
	prk, err := hkdf.Extract(sha256.New, shared, p.Secret[:])
	if err != nil {
		tb.Fatal(err)
	}
	scannerKey := private.PublicKey().Bytes()
	transcript := append([]byte("numen/v1/link/transcript\x00"), p.Channel[:]...)
	transcript = append(append(transcript, p.DisplayerKey[:]...), scannerKey...)
	th := sha256.Sum256(transcript)
	expand := func(direction string) []byte {
		key, err := hkdf.Expand(sha256.New, prk, "numen/v1/link/"+direction+string(th[:]), 32)
		if err != nil {
			tb.Fatal(err)
		}
		return key
	}
	return linkVector{scannerPublic: scannerKey, s2d: expand("s2d"), d2s: expand("d2s"), th: th[:]}
}

func (v linkVector) seal(tb testing.TB, key []byte, n uint64, plain string) []byte {
	tb.Helper()
	block, err := aes.NewCipher(key)
	if err != nil {
		tb.Fatal(err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		tb.Fatal(err)
	}
	counter := binary.BigEndian.AppendUint64(nil, n)
	nonce := append(make([]byte, 4), counter...)
	associated := append(append([]byte("numen/v1/link/msg"), v.th...), counter...)
	return aead.Seal(counter, nonce, []byte(plain), associated)
}

func TestLinkKeyExchange_MatchesPinnedKeySchedule(t *testing.T) {
	t.Parallel()
	_, payload, err := identity.NewLinkDisplayer(fixedBytes(1, 64), linkChannelID(), "")
	if err != nil {
		t.Fatal(err)
	}
	scannerPrivate := make([]byte, 32)
	for i := range scannerPrivate {
		scannerPrivate[i] = byte(0x40 + i)
	}
	scanner, first, err := identity.NewLinkScanner(bytes.NewReader(scannerPrivate), payload)
	if err != nil {
		t.Fatal(err)
	}
	vector := readLinkVector(t, scannerPrivate, payload)
	want := append(bytes.Clone(vector.scannerPublic), vector.seal(t, vector.s2d, 0, "hello")...)
	if !bytes.Equal(first, want) {
		t.Errorf("first message = %x\nwant            %x", first, want)
	}

	displayer, _, err := identity.NewLinkDisplayer(fixedBytes(1, 64), linkChannelID(), "")
	if err != nil {
		t.Fatal(err)
	}
	reply, displayerChannel, err := displayer.Accept(first)
	if err != nil {
		t.Fatalf("Accept() = %v", err)
	}
	if want := vector.seal(t, vector.d2s, 0, "hello"); !bytes.Equal(reply, want) {
		t.Errorf("reply = %x\nwant  %x", reply, want)
	}
	scannerChannel, err := scanner.Accept(reply)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := scannerChannel.Seal([]byte("name")), vector.seal(t, vector.s2d, 1, "name"); !bytes.Equal(got, want) {
		t.Errorf("second message of the scanner = %x\nwant %x", got, want)
	}
	if got, want := displayerChannel.Seal([]byte("ok")), vector.seal(t, vector.d2s, 1, "ok"); !bytes.Equal(got, want) {
		t.Errorf("second message of the displayer = %x\nwant %x", got, want)
	}
}

func TestLinkDisplayer_DropsAFirstMessageMadeWithAnotherSecretAndKeepsWaiting(t *testing.T) {
	t.Parallel()
	p := newPair(t)
	forged := p.payload
	forged.Secret[0] ^= 0xFF
	_, wrongFirst, err := identity.NewLinkScanner(rand.Reader, forged)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := p.displayer.Accept(wrongFirst); !errors.Is(err, identity.ErrLinkAuth) {
		t.Fatalf("error = %v, want ErrLinkAuth", err)
	}
	p.connect(t)
}

func TestLinkDisplayer_RefusesAFirstMessageWithAnyChangedByte(t *testing.T) {
	t.Parallel()
	p := newPair(t)
	for i := range p.first {
		changed := bytes.Clone(p.first)
		changed[i] ^= 0x01
		if _, _, err := p.displayer.Accept(changed); !errors.Is(err, identity.ErrLinkAuth) {
			t.Fatalf("byte %d changed: error = %v, want ErrLinkAuth", i, err)
		}
	}
	for _, size := range []int{0, 1, 31, 32, 40, len(p.first) - 1} {
		if _, _, err := p.displayer.Accept(p.first[:size]); !errors.Is(err, identity.ErrLinkAuth) {
			t.Errorf("%d bytes: error = %v, want ErrLinkAuth", size, err)
		}
	}
	p.connect(t)
}

func TestLinkDisplayer_AcceptsOneFirstMessageOnly(t *testing.T) {
	t.Parallel()
	p := newPair(t)
	displayer, scanner := p.connect(t)
	if _, _, err := p.displayer.Accept(p.first); err == nil {
		t.Error("the same first message was accepted twice")
	}
	_, second, err := identity.NewLinkScanner(rand.Reader, p.payload)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := p.displayer.Accept(second); err == nil {
		t.Error("a second valid first message was accepted after the channel was bound")
	}
	opened, err := displayer.Open(scanner.Seal([]byte("still the first scanner")))
	if err != nil || string(opened) != "still the first scanner" {
		t.Errorf("the bound channel broke: %q, %v", opened, err)
	}
}

func TestLinkScanner_RefusesAReplyOfAnotherDisplayer(t *testing.T) {
	t.Parallel()
	p := newPair(t)
	other := newPair(t)
	reply, _, err := other.displayer.Accept(other.first)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.scanner.Accept(reply); !errors.Is(err, identity.ErrLinkAuth) {
		t.Errorf("error = %v, want ErrLinkAuth", err)
	}
	if _, err := p.scanner.Accept(nil); !errors.Is(err, identity.ErrLinkAuth) {
		t.Errorf("empty reply: error = %v, want ErrLinkAuth", err)
	}
}

func TestLinkChannel_RefusesAMessageWithAnyChangedByteAndStillOpensTheRightOne(t *testing.T) {
	t.Parallel()
	displayer, scanner := newPair(t).connect(t)
	sealed := scanner.Seal([]byte("milk and bread"))
	for i := range sealed {
		changed := bytes.Clone(sealed)
		changed[i] ^= 0x01
		if _, err := displayer.Open(changed); err == nil {
			t.Fatalf("byte %d changed: the message opened", i)
		}
	}
	if _, err := displayer.Open(sealed[:len(sealed)-1]); !errors.Is(err, identity.ErrLinkAuth) {
		t.Errorf("truncated: error = %v, want ErrLinkAuth", err)
	}
	opened, err := displayer.Open(sealed)
	if err != nil || string(opened) != "milk and bread" {
		t.Errorf("the right message: %q, %v", opened, err)
	}
}

func TestLinkChannel_RefusesAReplayAReorderAndAGap(t *testing.T) {
	t.Parallel()
	displayer, scanner := newPair(t).connect(t)
	one, two, three := scanner.Seal([]byte("one")), scanner.Seal([]byte("two")), scanner.Seal([]byte("three"))
	if _, err := displayer.Open(two); !errors.Is(err, identity.ErrLinkReplay) {
		t.Errorf("a message ahead of the counter: error = %v, want ErrLinkReplay", err)
	}
	if _, err := displayer.Open(one); err != nil {
		t.Fatal(err)
	}
	if _, err := displayer.Open(one); !errors.Is(err, identity.ErrLinkReplay) {
		t.Errorf("a replay: error = %v, want ErrLinkReplay", err)
	}
	if _, err := displayer.Open(three); !errors.Is(err, identity.ErrLinkReplay) {
		t.Errorf("a gap: error = %v, want ErrLinkReplay", err)
	}
	if _, err := displayer.Open(two); err != nil {
		t.Errorf("the message in order: %v", err)
	}
}

func TestLinkChannel_RefusesAMessageOfItsOwnDirection(t *testing.T) {
	t.Parallel()
	displayer, scanner := newPair(t).connect(t)
	if _, err := scanner.Open(scanner.Seal([]byte("echo"))); !errors.Is(err, identity.ErrLinkAuth) {
		t.Errorf("scanner reads its own message: error = %v, want ErrLinkAuth", err)
	}
	if _, err := displayer.Open(displayer.Seal([]byte("echo"))); !errors.Is(err, identity.ErrLinkAuth) {
		t.Errorf("displayer reads its own message: error = %v, want ErrLinkAuth", err)
	}
}

func TestLinkChannel_RefusesAMessageOfAnotherLink(t *testing.T) {
	t.Parallel()
	displayer, _ := newPair(t).connect(t)
	_, other := newPair(t).connect(t)
	if _, err := displayer.Open(other.Seal([]byte("hello"))); !errors.Is(err, identity.ErrLinkAuth) {
		t.Errorf("error = %v, want ErrLinkAuth", err)
	}
}
