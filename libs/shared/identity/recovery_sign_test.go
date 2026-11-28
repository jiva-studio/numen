package identity_test

import (
	"crypto/ed25519"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/identity"
)

func TestSignForRecovery_SignsTheDomainPrefixPluginIDAndMessage(t *testing.T) {
	t.Parallel()
	recovery := deriveRecovery(t, abandonWords(t))

	signature := recovery.SignForRecovery(syncPluginID, []byte("hello"))

	signed := append([]byte("numen/v1/recovery-sign/"+syncPluginID+"\x00"), []byte("hello")...)
	if !ed25519.Verify(recovery.Public().Sign, signed, signature[:]) {
		t.Error("the signature does not cover the prefix, the plugin id, a zero byte and the message")
	}
}

func TestVerifyForRecovery_AcceptsOnlyTheSamePluginMessageAndKey(t *testing.T) {
	t.Parallel()
	recovery := deriveRecovery(t, abandonWords(t))
	other := deriveRecovery(t, parseWords(t, bip39Vectors()[1].words))
	signature := recovery.SignForRecovery(syncPluginID, []byte("hello"))

	if !identity.VerifyForRecovery(recovery.Public().Sign, syncPluginID, []byte("hello"), signature) {
		t.Error("a signature of the recovery key was refused")
	}
	if identity.VerifyForRecovery(recovery.Public().Sign, "md.numen.tools.link", []byte("hello"), signature) {
		t.Error("a signature made for one plugin verified for another")
	}
	if identity.VerifyForRecovery(recovery.Public().Sign, syncPluginID, []byte("hellp"), signature) {
		t.Error("a changed message verified")
	}
	if identity.VerifyForRecovery(other.Public().Sign, syncPluginID, []byte("hello"), signature) {
		t.Error("the key of another account verified")
	}
	if identity.VerifyForRecovery(recovery.Public().Sign[:31], syncPluginID, []byte("hello"), signature) {
		t.Error("a key of the wrong length verified")
	}
}

func TestSignForRecovery_NeverVerifiesAsASignatureOfAPluginDomain(t *testing.T) {
	t.Parallel()
	recovery := deriveRecovery(t, abandonWords(t))
	signature := recovery.SignForRecovery(syncPluginID, []byte("hello"))

	if identity.VerifyForPlugin(recovery.Public().Sign, syncPluginID, []byte("hello"), signature) {
		t.Error("a recovery signature verified as a plugin signature of a device")
	}
}

func TestVerifyForRecovery_RefusesAPluginIDThatMovesBytesIntoTheMessage(t *testing.T) {
	t.Parallel()
	recovery := deriveRecovery(t, abandonWords(t))
	signature := recovery.SignForRecovery(syncPluginID, []byte("x\x00hello"))

	if identity.VerifyForRecovery(recovery.Public().Sign, syncPluginID+"\x00x", []byte("hello"), signature) {
		t.Error("the boundary between the plugin id and the message is ambiguous")
	}
}
