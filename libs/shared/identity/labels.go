package identity

import (
	"crypto/hmac"
	"crypto/sha256"
)

const (
	labelRecovery           = "numen/v1/recovery"
	labelRecoverySign       = "numen/v1/recovery/sign"
	labelRecoveryBox        = "numen/v1/recovery/box"
	labelDeviceSign         = "numen/v1/device/sign"
	labelDeviceBox          = "numen/v1/device/box"
	labelAccountKeys        = "numen/v1/account-keys"
	labelChainEntry         = "numen/v1/chain/entry"
	labelFingerprintAccount = "numen/v1/fingerprint/account"
	labelFingerprintDevice  = "numen/v1/fingerprint/device"
)

func deriveKey(secret []byte, label string) [32]byte {
	extract := hmac.New(sha256.New, make([]byte, sha256.Size))
	extract.Write(secret)
	expand := hmac.New(sha256.New, extract.Sum(nil))
	expand.Write([]byte(label))
	expand.Write([]byte{1})
	return [32]byte(expand.Sum(nil))
}
