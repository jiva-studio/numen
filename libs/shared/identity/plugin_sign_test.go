package identity_test

import (
	"crypto/ed25519"
	"math/big"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/identity"
)

const syncPluginID = "md.numen.tools.sync"

func TestSignForPlugin_SignsTheDomainPrefixPluginIDAndMessage(t *testing.T) {
	t.Parallel()
	device := newDevice(t)
	signature := device.SignForPlugin(syncPluginID, []byte("hello"))

	signed := append([]byte("numen/v1/plugin-sign/"+syncPluginID+"\x00"), []byte("hello")...)
	if !ed25519.Verify(device.Public().Sign, signed, signature[:]) {
		t.Error("the signature does not cover the prefix, the plugin id, a zero byte and the message")
	}
}

func TestVerifyForPlugin_AcceptsASignatureOfTheSamePluginAndMessage(t *testing.T) {
	t.Parallel()
	device := newDevice(t)
	signature := device.SignForPlugin(syncPluginID, []byte("hello"))

	if !identity.VerifyForPlugin(device.Public().Sign, syncPluginID, []byte("hello"), signature) {
		t.Error("a signature made for the plugin was refused")
	}
}

func TestVerifyForPlugin_RefusesASignatureOfAnotherPlugin(t *testing.T) {
	t.Parallel()
	device := newDevice(t)
	signature := device.SignForPlugin("md.numen.tools.link", []byte("hello"))

	if identity.VerifyForPlugin(device.Public().Sign, syncPluginID, []byte("hello"), signature) {
		t.Error("a signature made for another plugin verified as the plugin of the server")
	}
}

func TestVerifyForPlugin_RefusesAPluginIDThatMovesBytesIntoTheMessage(t *testing.T) {
	t.Parallel()
	device := newDevice(t)
	signature := device.SignForPlugin("md.numen.tools.sync", []byte("x\x00hello"))

	if identity.VerifyForPlugin(device.Public().Sign, "md.numen.tools.sync\x00x", []byte("hello"), signature) {
		t.Error("the boundary between the plugin id and the message is ambiguous")
	}
}

func TestVerifyForPlugin_RefusesAChangedMessageKeyOrBit(t *testing.T) {
	t.Parallel()
	device := newDevice(t)
	other := newDevice(t)
	signature := device.SignForPlugin(syncPluginID, []byte("hello"))

	if identity.VerifyForPlugin(device.Public().Sign, syncPluginID, []byte("hellp"), signature) {
		t.Error("a changed message verified")
	}
	if identity.VerifyForPlugin(other.Public().Sign, syncPluginID, []byte("hello"), signature) {
		t.Error("the key of another device verified")
	}
	for i := range signature {
		flipped := signature
		flipped[i] ^= 0x01
		if identity.VerifyForPlugin(device.Public().Sign, syncPluginID, []byte("hello"), flipped) {
			t.Fatalf("a signature with bit 0 of byte %d flipped verified", i)
		}
	}
}

func TestVerifyForPlugin_RefusesAKeyOfTheWrongLength(t *testing.T) {
	t.Parallel()
	device := newDevice(t)
	signature := device.SignForPlugin(syncPluginID, []byte("hello"))

	for name, key := range map[string]ed25519.PublicKey{"nil": nil, "short": device.Public().Sign[:31], "long": append(device.Public().Sign[:32:32], 0)} {
		if identity.VerifyForPlugin(key, syncPluginID, []byte("hello"), signature) {
			t.Errorf("%s key verified", name)
		}
	}
}

func TestVerifyForPlugin_RefusesANonCanonicalS(t *testing.T) {
	t.Parallel()
	device := newDevice(t)
	signature := device.SignForPlugin(syncPluginID, []byte("hello"))
	malleable := addGroupOrder(signature)

	if identity.VerifyForPlugin(device.Public().Sign, syncPluginID, []byte("hello"), malleable) {
		t.Error("the signature with S + L verified")
	}
}

func addGroupOrder(signature [64]byte) [64]byte {
	order, _ := new(big.Int).SetString("7237005577332262213973186563042994240857116359379907606001950938285454250989", 10)
	s := littleEndian(signature[32:])
	s.Add(s, order)
	out := signature
	encoded := s.FillBytes(make([]byte, 32))
	for i := range 32 {
		out[32+i] = encoded[31-i]
	}
	return out
}

func littleEndian(b []byte) *big.Int {
	reversed := make([]byte, len(b))
	for i := range b {
		reversed[len(b)-1-i] = b[i]
	}
	return new(big.Int).SetBytes(reversed)
}
