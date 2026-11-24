package syncprotocol

import (
	"encoding/binary"
	"fmt"
	"time"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

const (
	signInLabel         = "numen-sync/sign-in/v1"
	chainReadLabel      = "numen-sync/chain-read/v1"
	accountDeleteLabel  = "numen-sync/account-delete/v1"
	challengeIDSize     = 16
	challengeNonceSize  = 32
	maxChallengeSkewSec = 120
)

// GetSignInMessage returns the bytes a device signs to sign in: the label of the protocol, the audience,
// the nonce, the expiry, the device key and the protocol version. The id of the challenge is not signed.
func GetSignInMessage(challenge *syncv1.GetChallengeResponse, deviceSignKey []byte, version uint32) []byte {
	return getChallengeMessage(signInLabel, challenge, deviceSignKey, version)
}

// GetChainReadMessage returns the bytes the recovery key signs to read the chain of its account: the layout of
// GetSignInMessage under the label numen-sync/chain-read/v1, with the root key in place of the device key.
func GetChainReadMessage(challenge *syncv1.GetChallengeResponse, rootSignKey []byte, version uint32) []byte {
	return getChallengeMessage(chainReadLabel, challenge, rootSignKey, version)
}

// GetAccountDeleteMessage returns the bytes a device signs to delete its account: the layout of GetSignInMessage under
// the label numen-sync/account-delete/v1, so a signature made to sign in never deletes.
func GetAccountDeleteMessage(challenge *syncv1.GetChallengeResponse, deviceSignKey []byte, version uint32) []byte {
	return getChallengeMessage(accountDeleteLabel, challenge, deviceSignKey, version)
}

func getChallengeMessage(label string, challenge *syncv1.GetChallengeResponse, deviceSignKey []byte, version uint32) []byte {
	audience := challenge.GetAudience()
	message := make([]byte, 0, len(label)+1+binary.MaxVarintLen64+len(audience)+len(challenge.GetNonce())+8+len(deviceSignKey)+4)
	message = append(message, label...)
	message = append(message, 0)
	message = binary.AppendUvarint(message, uint64(len(audience)))
	message = append(message, audience...)
	message = append(message, challenge.GetNonce()...)
	message = binary.BigEndian.AppendUint64(message, uint64(challenge.GetExpiresAt()))
	message = append(message, deviceSignKey...)
	return binary.BigEndian.AppendUint32(message, version)
}

// CheckChallenge checks a challenge before the client signs it: its id and nonce have their sizes, its
// audience is the address the client dialed, and it expires within 120 seconds of now, before or after.
func CheckChallenge(challenge *syncv1.GetChallengeResponse, address string, now time.Time) error {
	if len(challenge.GetChallengeId()) != challengeIDSize || len(challenge.GetNonce()) != challengeNonceSize {
		return ErrMalformedChallenge
	}
	if !IsSameAddress(challenge.GetAudience(), address) {
		return fmt.Errorf("%w: %q, not %q", ErrAudienceMismatch, challenge.GetAudience(), address)
	}
	if skew := challenge.GetExpiresAt() - now.Unix(); skew > maxChallengeSkewSec || skew < -maxChallengeSkewSec {
		return ErrChallengeOutOfRange
	}
	return nil
}
