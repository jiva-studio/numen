package syncprotocol_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

func newChallenge() *syncv1.GetChallengeResponse {
	return &syncv1.GetChallengeResponse{
		ChallengeId: bytes.Repeat([]byte{1}, 16),
		Nonce:       bytes.Repeat([]byte{2}, 32),
		ExpiresAt:   1_795_000_000,
		Audience:    "https://sync.numen.md",
	}
}

func TestGetSignInMessage_JoinsTheLabelAudienceNonceExpiryKeyAndVersionInTheFrozenOrder(t *testing.T) {
	t.Parallel()
	challenge := newChallenge()
	key := bytes.Repeat([]byte{3}, 32)

	got := syncprotocol.GetSignInMessage(challenge, key, 1)

	var want []byte
	want = append(want, "numen-sync/sign-in/v1"...)
	want = append(want, 0)
	want = binary.AppendUvarint(want, uint64(len(challenge.GetAudience())))
	want = append(want, challenge.GetAudience()...)
	want = append(want, challenge.GetNonce()...)
	want = binary.BigEndian.AppendUint64(want, uint64(challenge.GetExpiresAt()))
	want = append(want, key...)
	want = binary.BigEndian.AppendUint32(want, 1)
	if !bytes.Equal(got, want) {
		t.Errorf("message = %x\nwant      %x", got, want)
	}
}

func TestGetSignInMessage_ChangesWithEveryFieldItCovers(t *testing.T) {
	t.Parallel()
	key := bytes.Repeat([]byte{3}, 32)
	base := syncprotocol.GetSignInMessage(newChallenge(), key, 1)

	for name, message := range map[string][]byte{
		"audience": syncprotocol.GetSignInMessage(withChallenge(func(c *syncv1.GetChallengeResponse) { c.Audience = "https://other.example" }), key, 1),
		"nonce":    syncprotocol.GetSignInMessage(withChallenge(func(c *syncv1.GetChallengeResponse) { c.Nonce[0] ^= 1 }), key, 1),
		"expiry":   syncprotocol.GetSignInMessage(withChallenge(func(c *syncv1.GetChallengeResponse) { c.ExpiresAt++ }), key, 1),
		"key":      syncprotocol.GetSignInMessage(newChallenge(), bytes.Repeat([]byte{4}, 32), 1),
		"version":  syncprotocol.GetSignInMessage(newChallenge(), key, 2),
	} {
		if bytes.Equal(message, base) {
			t.Errorf("a change of the %s leaves the message as it was", name)
		}
	}
}

func TestGetSignInMessage_IgnoresTheChallengeIDTheServerSendsBack(t *testing.T) {
	t.Parallel()
	key := bytes.Repeat([]byte{3}, 32)

	other := withChallenge(func(c *syncv1.GetChallengeResponse) { c.ChallengeId = bytes.Repeat([]byte{9}, 16) })

	if !bytes.Equal(syncprotocol.GetSignInMessage(newChallenge(), key, 1), syncprotocol.GetSignInMessage(other, key, 1)) {
		t.Error("the challenge id is part of the signed bytes, the spec leaves it out")
	}
}

func withChallenge(change func(*syncv1.GetChallengeResponse)) *syncv1.GetChallengeResponse {
	challenge := newChallenge()
	change(challenge)
	return challenge
}

func TestVersion_IsTwo(t *testing.T) {
	t.Parallel()
	if syncprotocol.Version != 2 {
		t.Errorf("Version = %d, want 2", syncprotocol.Version)
	}
	if syncprotocol.HeaderVersion != "Numen-Sync-Protocol" {
		t.Errorf("HeaderVersion = %q", syncprotocol.HeaderVersion)
	}
}
