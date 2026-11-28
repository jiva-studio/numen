package slip39

import (
	"fmt"
	"strings"
)

// Share is one decoded mnemonic of a SLIP-39 set.
type Share struct {
	Identifier                             uint16
	Extendable                             bool
	IterationExponent                      uint8
	GroupIndex, GroupThreshold, GroupCount uint8
	MemberIndex, MemberThreshold           uint8
	Value                                  []byte
}

func (s Share) String() string { return "[slip39 share]" }

func (s Share) GoString() string { return s.String() }

func (s Share) Words() []string {
	list := listWords()
	indices := s.indices()
	out := make([]string, len(indices))
	for i, at := range indices {
		out[i] = list[at]
	}
	return out
}

func (s Share) indices() []int {
	extendable := 0
	if s.Extendable {
		extendable = 1
	}
	idExp := int(s.Identifier)<<(1+iterationExpBits) | extendable<<iterationExpBits | int(s.IterationExponent)
	params := int(s.GroupIndex)
	for _, field := range []int{int(s.GroupThreshold) - 1, int(s.GroupCount) - 1, int(s.MemberIndex), int(s.MemberThreshold) - 1} {
		params = params<<4 | field
	}
	data := append(toIndices(idExp, idExpWords), toIndices(params, paramWords)...)
	data = append(data, encodeValueIndices(s.Value)...)
	return append(data, createChecksum(getCustomization(s.Extendable), data)...)
}

func toIndices(number, count int) []int {
	out := make([]int, count)
	for i := range out {
		out[i] = number >> (radixBits * (count - 1 - i)) & (1<<radixBits - 1)
	}
	return out
}

func fromIndices(indices []int) int {
	number := 0
	for _, at := range indices {
		number = number<<radixBits | at
	}
	return number
}

func encodeValueIndices(value []byte) []int {
	count := (len(value)*8 + radixBits - 1) / radixBits
	padding := count*radixBits - len(value)*8
	out := make([]int, count)
	for pos := range count * radixBits {
		bit := 0
		if pos >= padding {
			at := pos - padding
			bit = int(value[at/8] >> (7 - at%8) & 1)
		}
		out[pos/radixBits] = out[pos/radixBits]<<1 | bit
	}
	return out
}

func computeValuePadding(wordCount int) int {
	return radixBits * (wordCount - metadataWords) % 16
}

func decodeValue(data []int, padding int) ([]byte, error) {
	total := len(data) * radixBits
	value := make([]byte, (total-padding)/8)
	for pos := range total {
		bit := data[pos/radixBits] >> (radixBits - 1 - pos%radixBits) & 1
		if pos < padding {
			if bit != 0 {
				return nil, ErrPadding
			}
			continue
		}
		at := pos - padding
		value[at/8] |= byte(bit) << (7 - at%8)
	}
	return value, nil
}

func lookupWords(words []string) ([]int, error) {
	index := indexWords()
	data := make([]int, len(words))
	for i, word := range words {
		at, ok := index[strings.ToLower(strings.TrimSpace(word))]
		if !ok {
			return nil, fmt.Errorf("word %d: %w", i+1, ErrUnknownWord)
		}
		data[i] = at
	}
	return data, nil
}

// ParseShare reads one mnemonic of whole words from the SLIP-39 list, each trimmed and lower-cased.
func ParseShare(words []string) (Share, error) {
	if len(words) < minMnemonicWords || computeValuePadding(len(words)) > 8 {
		return Share{}, fmt.Errorf("%w: got %d", ErrWordCount, len(words))
	}
	data, err := lookupWords(words)
	if err != nil {
		return Share{}, err
	}
	idExp := fromIndices(data[:idExpWords])
	share := Share{
		Identifier:        uint16(idExp >> (1 + iterationExpBits)),
		Extendable:        idExp>>iterationExpBits&1 == 1,
		IterationExponent: uint8(idExp & (1<<iterationExpBits - 1)),
	}
	if !verifyChecksum(getCustomization(share.Extendable), data) {
		return Share{}, ErrChecksum
	}
	params := fromIndices(data[idExpWords : idExpWords+paramWords])
	share.GroupIndex = uint8(params >> 16 & 15)
	share.GroupThreshold = uint8(params>>12&15) + 1
	share.GroupCount = uint8(params>>8&15) + 1
	share.MemberIndex = uint8(params >> 4 & 15)
	share.MemberThreshold = uint8(params&15) + 1
	if share.GroupCount < share.GroupThreshold {
		return Share{}, fmt.Errorf("%w: group threshold %d of %d groups", ErrInvalidShare, share.GroupThreshold, share.GroupCount)
	}
	share.Value, err = decodeValue(data[idExpWords+paramWords:len(data)-checksumWords], computeValuePadding(len(words)))
	if err != nil {
		return Share{}, err
	}
	if len(share.Value)%2 != 0 {
		return Share{}, fmt.Errorf("%w: odd secret length", ErrInvalidShare)
	}
	return share, nil
}
