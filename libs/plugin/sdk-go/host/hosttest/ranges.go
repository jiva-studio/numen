package hosttest

import (
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
)

// maxReadLength is the maximum number of bytes one ranged read of a file returns, as in the host.
const maxReadLength = 16000

func getRange(text []byte, start, length int) (rangeStart, rangeEnd int) {
	rangeStart, rangeEnd = min(max(start, 0), len(text)), len(text)
	if length > 0 && length < len(text)-rangeStart {
		rangeEnd = rangeStart + length
	}
	for rangeStart > 0 && rangeStart < rangeEnd && !utf8.RuneStart(text[rangeStart]) {
		rangeStart++
	}
	if rangeEnd == len(text) {
		return rangeStart, rangeEnd
	}
	for pos := rangeEnd - 1; pos >= rangeStart && pos > rangeEnd-utf8.UTFMax; pos-- {
		if !utf8.RuneStart(text[pos]) {
			continue
		}
		if !utf8.FullRune(text[pos:rangeEnd]) {
			rangeEnd = pos
		}
		break
	}
	return rangeStart, rangeEnd
}

func (v *Vault) ReadRange(ctx context.Context, filePath string, start, length int) (host.FileRange, error) {
	content, stat, err := v.Read(ctx, filePath)
	if err != nil {
		return host.FileRange{}, err
	}
	if length <= 0 || length > maxReadLength {
		length = maxReadLength
	}
	rangeStart, rangeEnd := getRange(content, start, length)
	if !utf8.Valid(content[rangeStart:rangeEnd]) {
		return host.FileRange{}, fmt.Errorf("read %s from %d: %w", filePath, rangeStart, host.ErrNotText)
	}
	return host.FileRange{Text: string(content[rangeStart:rangeEnd]), Start: rangeStart, Stat: stat}, nil
}
