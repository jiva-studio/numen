package accounttest

import (
	"bytes"
	"crypto/rand"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/identity"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

func getWords(t *testing.T) []string {
	t.Helper()
	words, err := identity.NewRecoveryWords(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return words.List()
}

func TestStartRestore_RemovesTheIdentityTheFakeHad(t *testing.T) {
	fake := NewFake(t, "md.numen.a")

	fake.StartRestore(getWords(t))

	if fake.Public().AccountFingerprint != "" {
		t.Errorf("public = %+v, want no identity while the restore is pending", fake.Public())
	}
}

func TestSealData_RefusesPlainOver32KiBAsTheHostDoes(t *testing.T) {
	fake := NewFake(t, "md.numen.a")
	s := &service{fake: fake}
	sealRequest := func(size int) *connect.Request[v1.SealDataRequest] {
		return connect.NewRequest(&v1.SealDataRequest{Context: []byte("c"), Plain: bytes.Repeat([]byte{1}, size)})
	}

	_, over := s.SealData(t.Context(), sealRequest(maxSealPlain+1))
	_, exact := s.SealData(t.Context(), sealRequest(maxSealPlain))

	if connect.CodeOf(over) != connect.CodeInvalidArgument || exact != nil {
		t.Errorf("over = %v, exact = %v, want INVALID_ARGUMENT for 32 KiB + 1 and success for 32 KiB", over, exact)
	}
}

func TestCompleteRestore_RefusesWhenItsRestoreWasReplacedAndKeepsTheNewOne(t *testing.T) {
	fake := NewFake(t, "md.numen.a")
	fake.StartRestore(getWords(t))
	second := getWords(t)
	fake.completeHook = func() { fake.StartRestore(second) }
	s := &service{fake: fake}

	_, err := s.CompleteRestore(t.Context(), connect.NewRequest(&v1.CompleteRestoreRequest{}))

	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Errorf("error = %v, want FAILED_PRECONDITION", err)
	}
	if fake.Public().AccountFingerprint != "" || fake.pending == nil {
		t.Errorf("public = %+v, pending = %v, want no identity and the second restore pending", fake.Public(), fake.pending)
	}
}
