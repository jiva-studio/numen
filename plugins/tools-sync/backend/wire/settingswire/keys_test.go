package settingswire

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/healthkey"
)

func TestHealthKeysAreValid(t *testing.T) {
	for _, key := range []string{keyEpochRoot, keyInsecureServer} {
		if !healthkey.IsValid(key) {
			t.Errorf("the key %q is not valid", key)
		}
	}
}
