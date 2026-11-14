package fingerprint_test

import (
	"errors"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/fingerprint"
)

func TestParseReadsWhatFormatWrote(t *testing.T) {
	for _, stat := range []domain.FileStat{
		{Path: "", Kind: "", Size: 482, ModTime: time.Unix(1700000000, 123)},
		{Path: "", Kind: "", Size: 0, ModTime: time.Time{}},
		{Path: "", Kind: "", Size: math.MaxInt64, ModTime: time.Unix(0, math.MaxInt64)},
		{Path: "", Kind: "", Size: 1, ModTime: time.Unix(0, 1)},
		{Path: "", Kind: "", Size: 1, ModTime: time.Unix(0, -1)},
	} {
		got, err := fingerprint.Parse(fingerprint.Format(stat))
		if err != nil {
			t.Fatal(err)
		}
		if got.Size != stat.Size || got.ModTime.IsZero() != stat.ModTime.IsZero() || !got.ModTime.Equal(stat.ModTime) {
			t.Errorf("got size %d and time %v (zero=%v), want %d and %v (zero=%v)",
				got.Size, got.ModTime, got.ModTime.IsZero(), stat.Size, stat.ModTime, stat.ModTime.IsZero())
		}
	}
}

func TestParseRejectsATokenThatIsNotASizeAndATime(t *testing.T) {
	for _, tc := range []struct {
		token   string
		wantMsg string
		wantErr error
	}{
		{token: "", wantMsg: "fingerprint is required", wantErr: nil},
		{token: "482", wantMsg: `invalid fingerprint format: "482"`, wantErr: nil},
		{token: "big-1", wantMsg: `invalid fingerprint size: strconv.ParseInt: parsing "big": invalid syntax`, wantErr: strconv.ErrSyntax},
		{token: "482-late", wantMsg: `invalid fingerprint timestamp: strconv.ParseInt: parsing "late": invalid syntax`, wantErr: strconv.ErrSyntax},
	} {
		_, err := fingerprint.Parse(tc.token)
		if err == nil {
			t.Fatalf("Parse(%q) returned no error", tc.token)
		}
		if err.Error() != tc.wantMsg {
			t.Errorf("Parse(%q) error = %q, want %q", tc.token, err.Error(), tc.wantMsg)
		}
		if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
			t.Errorf("Parse(%q) error = %v, want errors.Is %v", tc.token, err, tc.wantErr)
		}
	}
}
