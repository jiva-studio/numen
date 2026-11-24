package healthkey_test

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/healthkey"
)

func TestCreateGivesValidDistinctKeys(t *testing.T) {
	prefix := "restore-epoch-untrusted"
	one := healthkey.Create(prefix, "01M7WEFFR86SKR8EG3SV5HF4XS")
	other := healthkey.Create(prefix, "01M7WEFFR86SKR8EG3SV5HF4XT")
	if !healthkey.IsValid(one) || len(one) > healthkey.MaxLength {
		t.Fatalf("key %q is not valid", one)
	}
	if one == other {
		t.Fatalf("two ids gave the key %q", one)
	}
	if one != healthkey.Create(prefix, "01M7WEFFR86SKR8EG3SV5HF4XS") {
		t.Fatal("the key of one id changed")
	}
}

func TestIsValidRefusesWhatTheHostRefuses(t *testing.T) {
	for _, key := range []string{"restore.wentBack/01M7WEFFR86SKR8EG3SV5HF4XS", "tools-sync/epoch-root", "share.requestWaiting", "1a", "", "a-very-long-key-of-more-than-32-chars"} {
		if healthkey.IsValid(key) {
			t.Errorf("key %q taken", key)
		}
	}
}
