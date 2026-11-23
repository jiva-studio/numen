package syncprotocol_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
)

func TestLimits_HaveTheValuesOfTheMultipartContract(t *testing.T) {
	t.Parallel()

	for name, one := range map[string]struct{ got, want int64 }{
		"MinPartSize":      {syncprotocol.MinPartSize, 5242880},
		"MaxPartSize":      {syncprotocol.MaxPartSize, 5368709120},
		"MaxParts":         {syncprotocol.MaxParts, 10000},
		"DefaultPartSize":  {syncprotocol.DefaultPartSize, 16 << 20},
		"MaxPartURLs":      {syncprotocol.MaxPartURLs, 100},
		"MaxMultipartSize": {syncprotocol.MaxMultipartSize, 5497558138880 - 5242880},
	} {
		if one.got != one.want {
			t.Errorf("%s = %d, want %d", name, one.got, one.want)
		}
	}
}

func TestCheckReserveMultipart_AllowsOnlySizesAndPartSizesInsideTheLimits(t *testing.T) {
	t.Parallel()
	const mib = 1 << 20

	for _, one := range []struct {
		name           string
		size, partSize uint64
		isValid        bool
	}{
		{"one byte above the single upload", syncprotocol.MaxBlobSize + 1, syncprotocol.MinPartSize, true},
		{"the single upload size", syncprotocol.MaxBlobSize, syncprotocol.MinPartSize, false},
		{"zero", 0, syncprotocol.MinPartSize, false},
		{"the largest object", syncprotocol.MaxMultipartSize, syncprotocol.MaxPartSize, true},
		{"one byte over the largest object", syncprotocol.MaxMultipartSize + 1, syncprotocol.MaxPartSize, false},
		{"default part of 70 MiB", 70 * mib, syncprotocol.DefaultPartSize, true},
		{"part one byte under the minimum", 70 * mib, syncprotocol.MinPartSize - 1, false},
		{"part at the minimum", 70 * mib, syncprotocol.MinPartSize, true},
		{"part at the maximum", 70 * mib, syncprotocol.MaxPartSize, true},
		{"part one byte over the maximum", 70 * mib, syncprotocol.MaxPartSize + 1, false},
		{"10000 parts", 10000 * syncprotocol.MinPartSize, syncprotocol.MinPartSize, true},
		{"10001 parts", 10000*syncprotocol.MinPartSize + 1, syncprotocol.MinPartSize, false},
		{"zero part size", 70 * mib, 0, false},
	} {
		err := syncprotocol.CheckReserveMultipart(one.size, one.partSize)

		if one.isValid && err != nil {
			t.Errorf("%s: error = %v, want none", one.name, err)
		}
		if !one.isValid && !errors.Is(err, syncprotocol.ErrInvalidPartSize) {
			t.Errorf("%s: error = %v, want ErrInvalidPartSize", one.name, err)
		}
	}
}

func TestGetPartSize_KeepsTheDefaultUpToItsLimitAndGrowsInMebibytesAbove(t *testing.T) {
	t.Parallel()
	cases := map[string]int64{
		"just above a single body": syncprotocol.MaxBlobSize + 1,
		"at the default limit":     int64(syncprotocol.DefaultPartSize) * syncprotocol.MaxParts,
		"one byte above it":        int64(syncprotocol.DefaultPartSize)*syncprotocol.MaxParts + 1,
		"one tebibyte":             1 << 40,
		"the largest object":       syncprotocol.MaxMultipartSize,
	}
	for name, size := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			part := syncprotocol.GetPartSize(size)
			if part%(1<<20) != 0 || part < syncprotocol.DefaultPartSize || part > syncprotocol.MaxPartSize {
				t.Errorf("GetPartSize(%d) = %d, want a multiple of 1 MiB from the default to the maximum", size, part)
			}
			if err := syncprotocol.CheckReserveMultipart(uint64(size), uint64(part)); err != nil {
				t.Errorf("GetPartSize(%d) = %d is refused: %v", size, part, err)
			}
		})
	}
	if got := syncprotocol.GetPartSize(int64(syncprotocol.DefaultPartSize) * syncprotocol.MaxParts); got != syncprotocol.DefaultPartSize {
		t.Errorf("at the default limit the part is %d, want the default", got)
	}
	if got := syncprotocol.GetPartSize(int64(syncprotocol.DefaultPartSize)*syncprotocol.MaxParts + 1); got != syncprotocol.DefaultPartSize+1<<20 {
		t.Errorf("above the default limit the part is %d, want the default plus 1 MiB", got)
	}
}
