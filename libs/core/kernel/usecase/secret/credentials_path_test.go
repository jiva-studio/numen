package secret_test

import (
	"context"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/secret"
)

type plainStore struct{}

func (plainStore) GetKey(context.Context, string) (string, error) { return "", nil }
func (plainStore) SetKey(context.Context, string, string) error   { return nil }
func (plainStore) DeleteKey(context.Context, string) error        { return nil }

type fileStore struct{ plainStore }

func (fileStore) GetCredentialsPath(context.Context) string { return "/keys/credentials" }

func TestGetCredentialsPathAnswersOnlyForAStoreThatKeepsItsValuesInAFile(t *testing.T) {
	if got := secret.GetCredentialsPath(fileStore{})(t.Context()); got != "/keys/credentials" {
		t.Errorf("a store in a file answered %q", got)
	}
	if got := secret.GetCredentialsPath(plainStore{})(t.Context()); got != "" {
		t.Errorf("a store in the keychain answered %q", got)
	}
}
