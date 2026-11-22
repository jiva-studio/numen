package identity

import (
	"crypto/hkdf"
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
	key, err := hkdf.Key(sha256.New, secret, nil, label, 32)
	if err != nil {
		panic("identity: HKDF-SHA256 refused a 32-byte output: " + err.Error())
	}
	return [32]byte(key)
}
