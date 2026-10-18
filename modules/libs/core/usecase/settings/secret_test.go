package settings_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/settings"
)

func TestWriteSecretPassesTheKeyOn(t *testing.T) {
	t.Parallel()
	var passed [3]string
	write := settings.NewWriteSecret(func(_ context.Context, pluginID, name, key string) error {
		passed = [3]string{pluginID, name, key}
		return port.ErrNoKeyStore
	})

	err := write.Execute(t.Context(), "a.b", "apiKey", "k")

	if !errors.Is(err, port.ErrNoKeyStore) || passed != [3]string{"a.b", "apiKey", "k"} {
		t.Fatalf("error is %v after passing %v", err, passed)
	}
}

func TestDeleteSecretPassesTheSecretOn(t *testing.T) {
	t.Parallel()
	var passed [2]string
	remove := settings.NewDeleteSecret(func(_ context.Context, pluginID, name string) error {
		passed = [2]string{pluginID, name}
		return nil
	})

	if err := remove.Execute(t.Context(), "a.b", "apiKey"); err != nil || passed != [2]string{"a.b", "apiKey"} {
		t.Fatalf("error is %v after passing %v", err, passed)
	}
}
