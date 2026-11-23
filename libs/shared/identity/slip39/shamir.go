package slip39

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"io"
	"slices"
)

type rawShare struct {
	x    byte
	data []byte
}

func gfMul(a, b byte) byte {
	var product byte
	for range 8 {
		if b&1 == 1 {
			product ^= a
		}
		carry := a & 0x80
		a <<= 1
		if carry != 0 {
			a ^= 0x1B
		}
		b >>= 1
	}
	return product
}

func gfInverse(a byte) byte {
	result := byte(1)
	for range 254 {
		result = gfMul(result, a)
	}
	return result
}

func interpolate(shares []rawShare, x byte) []byte {
	for _, share := range shares {
		if share.x == x {
			return slices.Clone(share.data)
		}
	}
	out := make([]byte, len(shares[0].data))
	for i, share := range shares {
		basis := byte(1)
		for j, other := range shares {
			if i != j {
				basis = gfMul(basis, gfMul(x^other.x, gfInverse(share.x^other.x)))
			}
		}
		for k := range out {
			out[k] ^= gfMul(share.data[k], basis)
		}
	}
	return out
}

func createDigest(random, secret []byte) []byte {
	mac := hmac.New(sha256.New, random)
	mac.Write(secret)
	return mac.Sum(nil)[:digestLength]
}

func readRandom(random io.Reader, size int) ([]byte, error) {
	out := make([]byte, size)
	if _, err := io.ReadFull(random, out); err != nil {
		return nil, fmt.Errorf("slip39: read random bytes: %w", err)
	}
	return out, nil
}

func splitSecret(random io.Reader, threshold, count int, secret []byte) ([]rawShare, error) {
	shares := make([]rawShare, 0, count)
	if threshold == 1 {
		for i := range count {
			shares = append(shares, rawShare{x: byte(i), data: slices.Clone(secret)})
		}
		return shares, nil
	}
	for i := range threshold - 2 {
		data, err := readRandom(random, len(secret))
		if err != nil {
			return nil, err
		}
		shares = append(shares, rawShare{x: byte(i), data: data})
	}
	randomPart, err := readRandom(random, len(secret)-digestLength)
	if err != nil {
		return nil, err
	}
	digest := slices.Concat(createDigest(randomPart, secret), randomPart)
	base := append(slices.Clone(shares), rawShare{x: digestIndex, data: digest}, rawShare{x: secretIndex, data: secret})
	for i := threshold - 2; i < count; i++ {
		shares = append(shares, rawShare{x: byte(i), data: interpolate(base, byte(i))})
	}
	return shares, nil
}

func recoverSecret(threshold int, shares []rawShare) ([]byte, error) {
	if threshold == 1 {
		return slices.Clone(shares[0].data), nil
	}
	secret := interpolate(shares, secretIndex)
	digestShare := interpolate(shares, digestIndex)
	if !hmac.Equal(digestShare[:digestLength], createDigest(digestShare[digestLength:], secret)) {
		return nil, ErrDigest
	}
	return secret, nil
}
