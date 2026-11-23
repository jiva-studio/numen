package identity

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net/netip"
	"net/url"
	"slices"
	"strings"
)

const (
	linkVersion       = 1
	linkScheme        = "numen://link?"
	linkHello         = "hello"
	labelLinkTrans    = "numen/v1/link/transcript"
	labelLinkMessage  = "numen/v1/link/msg"
	labelLinkToDisp   = "numen/v1/link/s2d"
	labelLinkToScan   = "numen/v1/link/d2s"
	linkKeySize       = 32
	linkSecretSize    = 16
	linkCounterSize   = 8
	linkNoncePrefix   = 4
	linkMessageMinLen = linkCounterSize + 16
)

// LinkPayload is what the QR code of a link carries.
type LinkPayload struct {
	Version      uint8
	Channel      [16]byte
	DisplayerKey [32]byte
	Secret       [16]byte
	Server       string
}

// Text returns the text of the QR code.
func (p LinkPayload) Text() string {
	enc := base64.RawURLEncoding
	text := fmt.Sprintf("%sv=%d&c=%s&k=%s&s=%s", linkScheme, p.Version,
		enc.EncodeToString(p.Channel[:]), enc.EncodeToString(p.DisplayerKey[:]), enc.EncodeToString(p.Secret[:]))
	if p.Server != "" {
		text += "&h=" + url.QueryEscape(p.Server)
	}
	return text
}

// ParseLinkPayload reads the text of a QR code: version 1 with exactly the fields channel, key and secret, and an optional server origin.
func ParseLinkPayload(text string) (LinkPayload, error) {
	query, ok := strings.CutPrefix(text, linkScheme)
	if !ok {
		return LinkPayload{}, fmt.Errorf("%w: wrong scheme", ErrLinkPayload)
	}
	values, err := url.ParseQuery(query)
	if err != nil {
		return LinkPayload{}, fmt.Errorf("%w: %w", ErrLinkPayload, err)
	}
	for name, list := range values {
		if len(list) != 1 || !slices.Contains([]string{"v", "c", "k", "s", "h"}, name) {
			return LinkPayload{}, fmt.Errorf("%w: field %q", ErrLinkPayload, name)
		}
	}
	if values.Get("v") != "1" {
		return LinkPayload{}, fmt.Errorf("%w: version", ErrLinkPayload)
	}
	payload := LinkPayload{Version: linkVersion, Server: values.Get("h")}
	for name, into := range map[string][]byte{"c": payload.Channel[:], "k": payload.DisplayerKey[:], "s": payload.Secret[:]} {
		raw, err := base64.RawURLEncoding.DecodeString(values.Get(name))
		if err != nil || len(raw) != len(into) {
			return LinkPayload{}, fmt.Errorf("%w: field %q", ErrLinkPayload, name)
		}
		copy(into, raw)
	}
	if err := checkLinkServer(payload.Server); err != nil {
		return LinkPayload{}, err
	}
	return payload, nil
}

func checkLinkServer(server string) error {
	if server == "" {
		return nil
	}
	origin, err := url.Parse(server)
	if err != nil || !isLinkScheme(origin) || origin.Host == "" ||
		origin.User != nil || origin.Path != "" || origin.RawQuery != "" || origin.Fragment != "" || origin.Opaque != "" {
		return fmt.Errorf("%w: server", ErrLinkPayload)
	}
	return nil
}

// LinkDisplayer is the side of a link that shows the QR code.
type LinkDisplayer struct {
	private *ecdh.PrivateKey
	payload LinkPayload
	isBound bool
}

// NewLinkDisplayer makes the key and the secret of one link; the random reader gives 32 bytes for the key, then 16 for the secret.
func NewLinkDisplayer(random io.Reader, channel [16]byte, server string) (*LinkDisplayer, LinkPayload, error) {
	if err := checkLinkServer(server); err != nil {
		return nil, LinkPayload{}, err
	}
	private, err := readLinkKey(random)
	if err != nil {
		return nil, LinkPayload{}, err
	}
	payload := LinkPayload{Version: linkVersion, Channel: channel, Server: server}
	copy(payload.DisplayerKey[:], private.PublicKey().Bytes())
	if _, err := io.ReadFull(random, payload.Secret[:]); err != nil {
		return nil, LinkPayload{}, fmt.Errorf("identity: read link secret: %w", err)
	}
	return &LinkDisplayer{private: private, payload: payload}, payload, nil
}

func readLinkKey(random io.Reader) (*ecdh.PrivateKey, error) {
	seed := make([]byte, linkKeySize)
	defer clear(seed)
	if _, err := io.ReadFull(random, seed); err != nil {
		return nil, fmt.Errorf("identity: read link key: %w", err)
	}
	return ecdh.X25519().NewPrivateKey(seed)
}

// Accept takes the first message of a scanner. A message that does not prove the secret is ErrLinkAuth and the displayer keeps waiting;
// after one message was accepted every other one is refused.
func (d *LinkDisplayer) Accept(first []byte) ([]byte, *LinkChannel, error) {
	if d.isBound || len(first) < linkKeySize {
		return nil, nil, ErrLinkAuth
	}
	scannerKey, err := ecdh.X25519().NewPublicKey(first[:linkKeySize])
	if err != nil {
		return nil, nil, ErrLinkAuth
	}
	channel, err := newLinkChannel(d.private, scannerKey, d.payload, false)
	if err != nil {
		return nil, nil, ErrLinkAuth
	}
	if err := channel.openHello(first[linkKeySize:]); err != nil {
		return nil, nil, err
	}
	d.isBound = true
	return channel.Seal([]byte(linkHello)), channel, nil
}

// LinkScanner is the side of a link that reads the QR code.
type LinkScanner struct {
	channel *LinkChannel
}

// NewLinkScanner makes the key of the scanner and the first message, which proves that the scanner holds the secret of the QR code.
func NewLinkScanner(random io.Reader, p LinkPayload) (*LinkScanner, []byte, error) {
	if p.Version != linkVersion {
		return nil, nil, fmt.Errorf("%w: version", ErrLinkPayload)
	}
	private, err := readLinkKey(random)
	if err != nil {
		return nil, nil, err
	}
	displayerKey, err := ecdh.X25519().NewPublicKey(p.DisplayerKey[:])
	if err != nil {
		return nil, nil, fmt.Errorf("%w: key", ErrLinkPayload)
	}
	channel, err := newLinkChannel(private, displayerKey, p, true)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: key", ErrLinkPayload)
	}
	first := append(private.PublicKey().Bytes(), channel.Seal([]byte(linkHello))...)
	return &LinkScanner{channel: channel}, first, nil
}

// Accept takes the reply of the displayer, which proves that it holds the private key of the QR code.
func (s *LinkScanner) Accept(reply []byte) (*LinkChannel, error) {
	if err := s.channel.openHello(reply); err != nil {
		return nil, err
	}
	return s.channel, nil
}

// LinkChannel seals and opens the messages of one link in both directions.
// A message carries its number; the receiver accepts only the next one.
type LinkChannel struct {
	send, receive       cipher.AEAD
	transcript          [32]byte
	sendCount, recvNext uint64
}

func newLinkChannel(private *ecdh.PrivateKey, peer *ecdh.PublicKey, p LinkPayload, isScanner bool) (*LinkChannel, error) {
	shared, err := private.ECDH(peer)
	if err != nil {
		return nil, err
	}
	pseudo, err := hkdf.Extract(sha256.New, shared, p.Secret[:])
	if err != nil {
		return nil, err
	}
	scannerKey, displayerKey := private.PublicKey().Bytes(), p.DisplayerKey[:]
	if !isScanner {
		scannerKey = peer.Bytes()
	}
	hash := sha256.New()
	hash.Write([]byte(labelLinkTrans))
	hash.Write([]byte{0})
	hash.Write(p.Channel[:])
	hash.Write(displayerKey)
	hash.Write(scannerKey)
	var transcript [32]byte
	copy(transcript[:], hash.Sum(nil))
	toDisplayer, err := newLinkAEAD(pseudo, labelLinkToDisp, transcript)
	if err != nil {
		return nil, err
	}
	toScanner, err := newLinkAEAD(pseudo, labelLinkToScan, transcript)
	if err != nil {
		return nil, err
	}
	if isScanner {
		return &LinkChannel{send: toDisplayer, receive: toScanner, transcript: transcript}, nil
	}
	return &LinkChannel{send: toScanner, receive: toDisplayer, transcript: transcript}, nil
}

func newLinkAEAD(pseudo []byte, label string, transcript [32]byte) (cipher.AEAD, error) {
	key, err := hkdf.Expand(sha256.New, pseudo, label+string(transcript[:]), linkKeySize)
	if err != nil {
		return nil, err
	}
	defer clear(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func (c *LinkChannel) getNonceAndData(count []byte) (nonce, data []byte) {
	nonce = append(make([]byte, linkNoncePrefix), count...)
	data = append([]byte(labelLinkMessage), c.transcript[:]...)
	return nonce, append(data, count...)
}

// Seal returns the next message of this direction: its number, then the encrypted text.
func (c *LinkChannel) Seal(plain []byte) []byte {
	count := binary.BigEndian.AppendUint64(nil, c.sendCount)
	c.sendCount++
	nonce, data := c.getNonceAndData(count)
	return c.send.Seal(slices.Clone(count), nonce, plain, data)
}

// Open returns the text of the next message of the other direction.
// A message that does not authenticate is ErrLinkAuth; an authentic one that is not the next is ErrLinkReplay.
func (c *LinkChannel) Open(msg []byte) ([]byte, error) {
	if len(msg) < linkMessageMinLen {
		return nil, ErrLinkAuth
	}
	count := msg[:linkCounterSize]
	nonce, data := c.getNonceAndData(count)
	plain, err := c.receive.Open(nil, nonce, msg[linkCounterSize:], data)
	if err != nil {
		return nil, ErrLinkAuth
	}
	if binary.BigEndian.Uint64(count) != c.recvNext {
		return nil, ErrLinkReplay
	}
	c.recvNext++
	return plain, nil
}

func (c *LinkChannel) openHello(msg []byte) error {
	plain, err := c.Open(msg)
	if err != nil || subtle.ConstantTimeCompare(plain, []byte(linkHello)) != 1 {
		return ErrLinkAuth
	}
	return nil
}

func isLinkScheme(origin *url.URL) bool {
	if origin.Scheme == "https" {
		return true
	}
	host := origin.Hostname()
	ip, err := netip.ParseAddr(host)
	return origin.Scheme == "http" && (host == "localhost" || (err == nil && ip.IsLoopback()))
}
