package identity_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	windowidentity "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/identity"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/identity"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

func getLegalWords() []string {
	return strings.Fields("legal winner thank year wave sausage worth useful legal winner thank yellow")
}

type keyStore struct {
	mu       sync.Mutex
	keys     map[string]string
	setFails error
}

func (s *keyStore) GetKey(_ context.Context, name string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if value, ok := s.keys[name]; ok {
		return value, nil
	}
	return "", plugindomain.ErrNoKey
}

func (s *keyStore) SetKey(_ context.Context, name, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.setFails != nil {
		return s.setFails
	}
	s.keys[name] = key
	return nil
}

func (s *keyStore) DeleteKey(_ context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.keys[name]; !ok {
		return plugindomain.ErrNoKey
	}
	delete(s.keys, name)
	return nil
}

type names struct{}

func (names) GetInstanceName() string { return "Laptop" }

func getNoon() *testsupport.ManualClock {
	return testsupport.NewClockAt(time.Date(2026, time.November, 22, 12, 0, 0, 0, time.UTC))
}

func newHandler(keys *keyStore) *windowidentity.Handler {
	return newHandlerOf(keys, &kitSaver{}, getNoon())
}

func newHandlerWith(keys *keyStore, saver identity.KitSaver) *windowidentity.Handler {
	return newHandlerOf(keys, saver, getNoon())
}

func newHandlerOf(keys *keyStore, saver identity.KitSaver, clock *testsupport.ManualClock) *windowidentity.Handler {
	deps := identity.NewDeps(keys, names{}, clock).WithChains(&chainFile{})
	return windowidentity.NewHandler(windowidentity.UseCases{
		Identity: windowidentity.IdentityUseCases{
			Get:           identity.NewGetIdentity(deps),
			GenerateWords: identity.NewGenerateRecoveryWords(deps),
			Create:        identity.NewCreateIdentity(deps),
			Delete:        identity.NewDeleteIdentity(deps),
			StartRestore:  identity.NewStartRestore(deps),
		},
		Kit: windowidentity.KitUseCases{
			Save:    identity.NewSaveRecoveryKit(deps, saver),
			Restore: identity.NewStartRestoreFromKit(deps),
		},
	})
}

func newKeys() *keyStore { return &keyStore{keys: map[string]string{}} }

func create(t *testing.T, h *windowidentity.Handler, words []string) *v1.CreateIdentityResponse {
	t.Helper()
	answer, err := h.CreateIdentity(t.Context(), connect.NewRequest(&v1.CreateIdentityRequest{Words: words}))
	if err != nil {
		t.Fatalf("CreateIdentity() error = %v", err)
	}
	return answer.Msg
}

func TestGetIdentity_AnswersNothingWithoutAnIdentityAndTheInfoAfterCreate(t *testing.T) {
	t.Parallel()
	h := newHandler(newKeys())
	none, err := h.GetIdentity(t.Context(), connect.NewRequest(&v1.GetIdentityRequest{}))
	if err != nil || none.Msg.GetIdentity() != nil {
		t.Fatalf("GetIdentity() = %v, %v; want no identity", none, err)
	}
	created := createGenerated(t, h)
	if created.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("CreateIdentity() refused: %v %q", created.GetError(), created.GetDetail())
	}
	got, err := h.GetIdentity(t.Context(), connect.NewRequest(&v1.GetIdentityRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	info := got.Msg.GetIdentity()
	if info.GetAccountFingerprint() == "" || info.GetDeviceFingerprint() == "" || info.GetDeviceName() != "Laptop" {
		t.Errorf("identity = %v", info)
	}
	if info.GetDeviceAddedAt() != "2026-11-22T12:00:00Z" {
		t.Errorf("device_added_at = %q, want RFC 3339 UTC", info.GetDeviceAddedAt())
	}
	if info.GetAccountFingerprint() != created.GetIdentity().GetAccountFingerprint() {
		t.Error("Get and Create answer different fingerprints")
	}
}

func TestGenerateRecoveryWords_AnswersTwelveWords(t *testing.T) {
	t.Parallel()
	keys := newKeys()
	answer, err := newHandler(keys).GenerateRecoveryWords(t.Context(), connect.NewRequest(&v1.GenerateRecoveryWordsRequest{}))
	if err != nil || len(answer.Msg.GetWords()) != 12 {
		t.Fatalf("answer = %v, %v", answer, err)
	}
	if len(keys.keys) != 0 {
		t.Error("generating words kept something")
	}
}

func TestCreateIdentity_RefusesWordsThatAreNotValidWithoutEchoingAWord(t *testing.T) {
	t.Parallel()
	h := newHandler(newKeys())
	words := getLegalWords()
	words[3] = "numen"
	answer := create(t, h, words)
	if answer.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_INVALID_WORDS || answer.GetIdentity() != nil {
		t.Fatalf("answer = %v", answer)
	}
	if !strings.Contains(answer.GetDetail(), "4") {
		t.Errorf("detail %q does not name the position", answer.GetDetail())
	}
	for _, word := range append(words, "legal") {
		if strings.Contains(answer.GetDetail(), word) {
			t.Errorf("detail %q echoes the word %q", answer.GetDetail(), word)
		}
	}
	abandon := create(t, h, strings.Fields(strings.Repeat("abandon ", 12)))
	if abandon.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_INVALID_WORDS || strings.Contains(abandon.GetDetail(), "abandon") {
		t.Errorf("checksum answer = %v", abandon)
	}
}

func TestCreateIdentity_RefusesASecondIdentity(t *testing.T) {
	t.Parallel()
	h := newHandler(newKeys())
	createGenerated(t, h)
	if answer := createGenerated(t, h); answer.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_IDENTITY_EXISTS {
		t.Errorf("answer = %v", answer)
	}
}

func TestCreateIdentity_AnswersAKeyStoreFailureAsACode(t *testing.T) {
	t.Parallel()
	keys := newKeys()
	keys.setFails = errors.New("the keychain refuses")
	answer := createGenerated(t, newHandler(keys))
	if answer.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_KEY_STORE_FAILED || len(keys.keys) != 0 {
		t.Errorf("answer = %v, keys = %v", answer, keys.keys)
	}
}

func TestDeleteIdentity_RemovesTheIdentityAndAnswersWithoutOne(t *testing.T) {
	t.Parallel()
	h := newHandler(newKeys())
	if _, err := h.DeleteIdentity(t.Context(), connect.NewRequest(&v1.DeleteIdentityRequest{})); err != nil {
		t.Fatalf("delete without identity = %v", err)
	}
	createGenerated(t, h)
	if _, err := h.DeleteIdentity(t.Context(), connect.NewRequest(&v1.DeleteIdentityRequest{})); err != nil {
		t.Fatal(err)
	}
	got, err := h.GetIdentity(t.Context(), connect.NewRequest(&v1.GetIdentityRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if got.Msg.GetIdentity() != nil {
		t.Error("the identity is still there")
	}
}

func TestNewService_AnswersUnimplementedWhereTheInstallationHasNoKeyStore(t *testing.T) {
	t.Parallel()
	service := windowidentity.NewService(nil)
	_, err := service.GetIdentity(t.Context(), connect.NewRequest(&v1.GetIdentityRequest{}))
	if connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Errorf("error = %v, want Unimplemented", err)
	}
	if path, handler := windowidentity.NewRoute(service, connect.WithInterceptors()); path == "" || handler == nil {
		t.Error("NewRoute returned no route")
	}
}

func createGenerated(t *testing.T, h *windowidentity.Handler) *v1.CreateIdentityResponse {
	t.Helper()
	generated, err := h.GenerateRecoveryWords(t.Context(), connect.NewRequest(&v1.GenerateRecoveryWordsRequest{}))
	if err != nil {
		t.Fatalf("GenerateRecoveryWords() error = %v", err)
	}
	return create(t, h, generated.Msg.GetWords())
}

func TestCreateIdentity_RefusesValidWordsTheCoreDidNotMake(t *testing.T) {
	t.Parallel()
	keys := newKeys()
	h := newHandler(keys)

	answer := create(t, h, getLegalWords())

	if answer.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_INVALID_WORDS || len(keys.keys) != 0 {
		t.Errorf("answer = %v with %d entries kept, want INVALID_WORDS and nothing kept", answer.GetError(), len(keys.keys))
	}
}

func TestCreateIdentity_TakesTheGeneratedWordsOnce(t *testing.T) {
	t.Parallel()
	h := newHandler(newKeys())
	generated, err := h.GenerateRecoveryWords(t.Context(), connect.NewRequest(&v1.GenerateRecoveryWordsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	words := generated.Msg.GetWords()
	if first := create(t, h, words); first.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("first create = %v, want success", first.GetError())
	}
	if _, err := h.DeleteIdentity(t.Context(), connect.NewRequest(&v1.DeleteIdentityRequest{})); err != nil {
		t.Fatal(err)
	}

	again := create(t, h, words)

	if again.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_INVALID_WORDS {
		t.Errorf("second create = %v, want INVALID_WORDS", again.GetError())
	}
}
