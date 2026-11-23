package slip39

import (
	"crypto/pbkdf2"
	"crypto/sha256"
	"fmt"
	"slices"
)

func saltOf(identifier uint16, extendable bool) []byte {
	if extendable {
		return nil
	}
	return append([]byte("shamir"), byte(identifier>>8), byte(identifier))
}

func roundFunction(round int, passphrase, salt, right []byte, exponent uint8) ([]byte, error) {
	password := append([]byte{byte(round)}, passphrase...)
	iterations := (baseIterations << exponent) / roundCount
	key, err := pbkdf2.Key(sha256.New, string(password), slices.Concat(salt, right), iterations, len(right))
	if err != nil {
		return nil, fmt.Errorf("slip39: derive round key: %w", err)
	}
	return key, nil
}

func feistel(data, passphrase []byte, share Share, decrypt bool) ([]byte, error) {
	half := len(data) / 2
	left, right := slices.Clone(data[:half]), slices.Clone(data[half:])
	salt := saltOf(share.Identifier, share.Extendable)
	for step := range roundCount {
		round := step
		if decrypt {
			round = roundCount - 1 - step
		}
		mask, err := roundFunction(round, passphrase, salt, right, share.IterationExponent)
		if err != nil {
			return nil, err
		}
		for i := range left {
			left[i] ^= mask[i]
		}
		left, right = right, left
	}
	return slices.Concat(right, left), nil
}
