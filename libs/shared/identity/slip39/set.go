package slip39

import (
	"fmt"
	"io"
	"maps"
	"slices"
)

// Split writes one group of count shares, any threshold of them restore secret. The secret has 16 to
// 32 bytes, an even count. A threshold of 1 needs a count of 1. The shares are extendable, use
// iteration exponent 1 and an empty passphrase.
func Split(random io.Reader, secret []byte, threshold, count int) ([]Share, error) {
	if err := checkSplit(len(secret), threshold, count); err != nil {
		return nil, err
	}
	idBytes, err := readRandom(random, 2)
	if err != nil {
		return nil, err
	}
	template := Share{
		Identifier:        (uint16(idBytes[0])<<8 | uint16(idBytes[1])) % (1 << idLengthBits),
		Extendable:        true,
		IterationExponent: splitExponent,
		GroupThreshold:    1,
		GroupCount:        1,
		MemberThreshold:   uint8(threshold),
	}
	encrypted, err := applyFeistel(secret, nil, template, false)
	if err != nil {
		return nil, err
	}
	raw, err := splitSecret(random, threshold, count, encrypted)
	if err != nil {
		return nil, err
	}
	shares := make([]Share, len(raw))
	for i, part := range raw {
		shares[i] = template
		shares[i].MemberIndex = part.x
		shares[i].Value = part.data
	}
	return shares, nil
}

func checkSplit(size, threshold, count int) error {
	switch {
	case size < minSecretBytes || size > maxSecretBytes || size%2 != 0:
		return fmt.Errorf("%w: a secret of %d bytes", ErrInvalidShare, size)
	case count < 1 || count > maxShareCount || threshold < 1 || threshold > count:
		return fmt.Errorf("%w: threshold %d of %d shares", ErrInvalidShare, threshold, count)
	case threshold == 1 && count != 1:
		return fmt.Errorf("%w: threshold 1 needs one share, got %d", ErrInvalidShare, count)
	}
	return nil
}

func isSameSet(a, b Share) bool {
	return a.Identifier == b.Identifier && a.Extendable == b.Extendable &&
		a.IterationExponent == b.IterationExponent && a.GroupThreshold == b.GroupThreshold &&
		a.GroupCount == b.GroupCount && len(a.Value) == len(b.Value)
}

func groupByIndex(shares []Share) map[uint8][]Share {
	groups := make(map[uint8][]Share)
	for _, share := range shares {
		groups[share.GroupIndex] = append(groups[share.GroupIndex], share)
	}
	return groups
}

func recoverGroup(members []Share) (rawShare, error) {
	threshold := int(members[0].MemberThreshold)
	raw := make([]rawShare, len(members))
	seen := make(map[uint8]bool, len(members))
	for i, member := range members {
		if int(member.MemberThreshold) != threshold {
			return rawShare{}, fmt.Errorf("%w: member thresholds differ", ErrMixedSets)
		}
		if seen[member.MemberIndex] {
			return rawShare{}, fmt.Errorf("%v: %w", member, ErrDuplicateMember)
		}
		seen[member.MemberIndex] = true
		raw[i] = rawShare{x: member.MemberIndex, data: member.Value}
	}
	if len(members) < threshold {
		return rawShare{}, fmt.Errorf("%w: %d of %d shares of group %d", ErrNotEnoughShares, len(members), threshold, members[0].GroupIndex)
	}
	if len(members) > threshold {
		return rawShare{}, fmt.Errorf("%w: %d shares of a group with threshold %d", ErrInvalidShare, len(members), threshold)
	}
	secret, err := recoverSecret(threshold, raw)
	if err != nil {
		return rawShare{}, err
	}
	return rawShare{x: members[0].GroupIndex, data: secret}, nil
}

func Combine(shares []Share, passphrase []byte) ([]byte, error) {
	if len(shares) == 0 {
		return nil, ErrNotEnoughShares
	}
	for _, share := range shares[1:] {
		if !isSameSet(shares[0], share) {
			return nil, ErrMixedSets
		}
	}
	groups := groupByIndex(shares)
	threshold := int(shares[0].GroupThreshold)
	if len(groups) < threshold {
		return nil, fmt.Errorf("%w: %d of %d groups", ErrNotEnoughShares, len(groups), threshold)
	}
	if len(groups) > threshold {
		return nil, fmt.Errorf("%w: %d groups, want %d", ErrInvalidShare, len(groups), threshold)
	}
	raw := make([]rawShare, 0, len(groups))
	for _, index := range slices.Sorted(maps.Keys(groups)) {
		part, err := recoverGroup(groups[index])
		if err != nil {
			return nil, err
		}
		raw = append(raw, part)
	}
	encrypted, err := recoverSecret(threshold, raw)
	if err != nil {
		return nil, err
	}
	return applyFeistel(encrypted, passphrase, shares[0], true)
}
