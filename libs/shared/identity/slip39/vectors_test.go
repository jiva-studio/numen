package slip39_test

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/identity/slip39"
)

const vectorPassphrase = "TREZOR"

type vector struct {
	Title     string
	Mnemonics []string
	Secret    string
}

func readVectors(t *testing.T) []vector {
	t.Helper()
	raw, err := os.ReadFile("testdata/vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows [][]json.RawMessage
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatal(err)
	}
	vectors := make([]vector, 0, len(rows))
	for i, row := range rows {
		if len(row) < 3 {
			t.Fatalf("row %d has %d columns, want at least 3", i, len(row))
		}
		var v vector
		for column, into := range []any{&v.Title, &v.Mnemonics, &v.Secret} {
			if err := json.Unmarshal(row[column], into); err != nil {
				t.Fatalf("row %d column %d: %v", i, column, err)
			}
		}
		vectors = append(vectors, v)
	}
	return vectors
}

func combineMnemonics(mnemonics []string, passphrase string) ([]byte, error) {
	shares := make([]slip39.Share, 0, len(mnemonics))
	for _, mnemonic := range mnemonics {
		share, err := slip39.ParseShare(strings.Fields(mnemonic))
		if err != nil {
			return nil, err
		}
		shares = append(shares, share)
	}
	return slip39.Combine(shares, []byte(passphrase))
}

func TestVectors_FileHas45Cases(t *testing.T) {
	t.Parallel()
	if got := len(readVectors(t)); got != 45 {
		t.Fatalf("testdata/vectors.json has %d cases, want 45", got)
	}
}

func TestCombine_PassesTheOfficialVectors(t *testing.T) {
	t.Parallel()
	for _, v := range readVectors(t) {
		t.Run(v.Title, func(t *testing.T) {
			t.Parallel()
			secret, err := combineMnemonics(v.Mnemonics, vectorPassphrase)
			if v.Secret == "" {
				if err == nil {
					t.Fatalf("the mnemonics restore %x, want a refusal", secret)
				}
				return
			}
			if err != nil {
				t.Fatalf("combine error = %v", err)
			}
			if got := hex.EncodeToString(secret); got != v.Secret {
				t.Errorf("secret = %s, want %s", got, v.Secret)
			}
		})
	}
}

func TestCombine_ReportsTheCauseOfOfficialVectorFailures(t *testing.T) {
	t.Parallel()
	vectors := readVectors(t)
	tests := []struct {
		number int
		want   error
	}{
		{2, slip39.ErrChecksum},
		{3, slip39.ErrPadding},
		{5, slip39.ErrNotEnoughShares},
		{6, slip39.ErrMixedSets},
		{7, slip39.ErrMixedSets},
		{8, slip39.ErrMixedSets},
		{9, slip39.ErrMixedSets},
		{13, slip39.ErrDigest},
		{21, slip39.ErrChecksum},
		{22, slip39.ErrPadding},
		{24, slip39.ErrNotEnoughShares},
		{25, slip39.ErrMixedSets},
		{26, slip39.ErrMixedSets},
		{32, slip39.ErrDigest},
		{39, slip39.ErrWordCount},
	}
	for _, tt := range tests {
		v := vectors[tt.number-1]
		t.Run(v.Title, func(t *testing.T) {
			t.Parallel()
			_, err := combineMnemonics(v.Mnemonics, vectorPassphrase)
			if !errors.Is(err, tt.want) {
				t.Errorf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestParseShare_ReadsTheFieldsOfAnExtendableVector(t *testing.T) {
	t.Parallel()
	v := readVectors(t)[41]
	share, err := slip39.ParseShare(strings.Fields(v.Mnemonics[0]))
	if err != nil {
		t.Fatal(err)
	}
	if !share.Extendable {
		t.Error("Extendable = false, want true")
	}
	if share.GroupThreshold != 1 || share.GroupCount != 1 || share.MemberThreshold != 1 {
		t.Errorf("thresholds = group %d of %d, member %d; want 1 of 1 and 1", share.GroupThreshold, share.GroupCount, share.MemberThreshold)
	}
	if len(share.Value) != 16 {
		t.Errorf("value has %d bytes, want 16", len(share.Value))
	}
}

func TestParseShare_ReadsTheFieldsOfAThresholdVector(t *testing.T) {
	t.Parallel()
	v := readVectors(t)[16]
	for _, mnemonic := range v.Mnemonics {
		share, err := slip39.ParseShare(strings.Fields(mnemonic))
		if err != nil {
			t.Fatal(err)
		}
		if share.GroupCount < 2 || share.GroupThreshold < 2 {
			t.Errorf("group threshold %d of %d, want a real group set", share.GroupThreshold, share.GroupCount)
		}
		if share.Identifier >= 1<<15 {
			t.Errorf("identifier %d does not fit 15 bits", share.Identifier)
		}
	}
}
