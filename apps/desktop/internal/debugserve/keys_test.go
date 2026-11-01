//go:build numen_debug

package debugserve

import (
	"errors"
	"testing"

	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

func TestKeyStoreReturnsTheKeySetUnderAName(t *testing.T) {
	store := NewKeyStore()
	if err := store.SetKey(t.Context(), "plugin/apiKey", "first"); err != nil {
		t.Fatalf("set: %v", err)
	}
	if err := store.SetKey(t.Context(), "plugin/apiKey", "second"); err != nil {
		t.Fatalf("replace: %v", err)
	}
	key, err := store.GetKey(t.Context(), "plugin/apiKey")
	if err != nil || key != "second" {
		t.Errorf("get = %q, %v, want second", key, err)
	}
}

func TestKeyStoreHoldsNoKeyAfterItIsDeleted(t *testing.T) {
	store := NewKeyStore()
	if err := store.SetKey(t.Context(), "plugin/apiKey", "first"); err != nil {
		t.Fatalf("set: %v", err)
	}
	if err := store.DeleteKey(t.Context(), "plugin/apiKey"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := store.GetKey(t.Context(), "plugin/apiKey"); !errors.Is(err, pluginport.ErrNoKey) {
		t.Errorf("get after delete: error = %v, want ErrNoKey", err)
	}
	if err := store.DeleteKey(t.Context(), "plugin/apiKey"); !errors.Is(err, pluginport.ErrNoKey) {
		t.Errorf("second delete: error = %v, want ErrNoKey", err)
	}
}
