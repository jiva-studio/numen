package synctest

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/identity"
)

// ThirdVault is the vault of the third computer of a seeded account.
const ThirdVault = "Third"

// NewLabel returns a label of the account of this test that no other test and no other run has: the real server keeps every account
// it has seen, so a name that repeats would find the account of the last run.
func NewLabel(t *testing.T) string {
	t.Helper()
	run := make([]byte, 8)
	if _, err := rand.Read(run); err != nil {
		t.Fatal(err)
	}
	return t.Name() + " " + hex.EncodeToString(run)
}

type labelStream struct {
	label   string
	counter uint64
	block   []byte
}

func (s *labelStream) Read(p []byte) (int, error) {
	for n := 0; n < len(p); {
		if len(s.block) == 0 {
			var counter [8]byte
			binary.BigEndian.PutUint64(counter[:], s.counter)
			sum := sha256.Sum256(append([]byte("numen sync test account "+s.label), counter[:]...))
			s.block, s.counter = sum[:], s.counter+1
		}
		copied := copy(p[n:], s.block)
		s.block = s.block[copied:]
		n += copied
	}
	return len(p), nil
}

func NewWordsFor(t *testing.T, label string) []string {
	t.Helper()
	words, err := identity.NewRecoveryWords(&labelStream{label: label, counter: 0, block: nil})
	if err != nil {
		t.Fatal(err)
	}
	return words.List()
}

type SeededAccount struct {
	Words  []string
	First  *Computer
	Second *Computer
	Third  *Computer
}

// SeedAccount makes the account of the label on the server: the first computer signs in and the server makes the account, then
// the second and the third computer join it as a person does, with the recovery words. The account has no plan.
func SeedAccount(t *testing.T, server *Server, clock *Clock, label string) *SeededAccount {
	t.Helper()
	words := NewWordsFor(t, label)
	return &SeededAccount{
		Words:  words,
		First:  NewFirstComputer(t, server, clock, string(FirstVault), words),
		Second: NewSecondComputer(t, server, clock, string(SecondVault), words),
		Third:  NewSecondComputer(t, server, clock, ThirdVault, words),
	}
}
