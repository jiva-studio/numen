package syncvault

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/healthkeytest"
)

// The host refuses a key that is not lowercase letters, digits and hyphens starting with a letter, or is
// longer than 32 characters (libs/core/kernel/domain/plugin/health.go, CheckHealthKey).
func TestEveryHealthKeyOfAVaultIsValid(t *testing.T) {
	const ulid = "01M7WEFFR86SKR8EG3SV5HF4XS"
	prefixes := []string{
		keyConflicts, keyUnverified, keyCaseClash, keyStorageFull, keyWentBack, keyEpochUntrusted, keyDiverged,
		keySharedConflicts, keyShareEnded, keySharedUnverified,
	}
	for _, prefix := range prefixes {
		if key := getKey(prefix, ulid); !healthkeytest.IsValid(key) {
			t.Errorf("the key %q of the prefix %q is not valid", key, prefix)
		}
	}
	for _, key := range []string{keyUnreachable, keyFailed} {
		if !healthkeytest.IsValid(key) {
			t.Errorf("the key %q is not valid", key)
		}
	}
}
