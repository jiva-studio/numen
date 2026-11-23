// Package slip39 is Shamir's secret sharing in mnemonic form as SLIP-0039 defines it. It knows
// nothing of Numen, uses only the standard library and does no I/O besides the random source it is given.
package slip39

import "errors"

const (
	radixBits        = 10
	idLengthBits     = 15
	iterationExpBits = 4
	idExpWords       = 2
	paramWords       = 2
	checksumWords    = 3
	metadataWords    = idExpWords + paramWords + checksumWords
	minMnemonicWords = metadataWords + 13
	digestLength     = 4
	baseIterations   = 10000
	roundCount       = 4
	maxShareCount    = 16
	secretIndex      = 255
	digestIndex      = 254
	minSecretBytes   = 16
	maxSecretBytes   = 32
	splitExponent    = 1
)

var (
	ErrWordCount       = errors.New("slip39: a share has an unsupported number of words")
	ErrUnknownWord     = errors.New("slip39: not a word of the list")
	ErrChecksum        = errors.New("slip39: the words do not match their checksum")
	ErrPadding         = errors.New("slip39: invalid padding")
	ErrInvalidShare    = errors.New("slip39: invalid share")
	ErrMixedSets       = errors.New("slip39: the shares belong to different sets")
	ErrDuplicateMember = errors.New("slip39: two shares have one member index")
	ErrNotEnoughShares = errors.New("slip39: not enough shares")
	ErrDigest          = errors.New("slip39: the digest of the secret does not match")
)
