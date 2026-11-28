package identity

import (
	"bytes"
	"fmt"
	"io"

	"github.com/jiva-studio/numen/libs/shared/identity/slip39"
)

const (
	SheetCount     = 3
	SheetThreshold = 2
	SheetWordCount = 33
	secretSize     = 32
)

// RecoverySecret is the 32 bytes the recovery key comes from. It prints no byte.
type RecoverySecret struct{ bytes [secretSize]byte }

func (s RecoverySecret) String() string { return "[recovery secret]" }

func (s RecoverySecret) GoString() string { return s.String() }

func (s RecoverySecret) DeriveRecoveryKey() (RecoveryKey, error) {
	return newRecoveryKey(deriveKey(s.bytes[:], labelRecoverySign), deriveKey(s.bytes[:], labelRecoveryBox))
}

// Sheet is one printed sheet of a recovery kit. It prints no word.
type Sheet struct{ share slip39.Share }

func (s Sheet) String() string { return "[recovery sheet]" }

func (s Sheet) GoString() string { return s.String() }

func (s Sheet) Words() []string { return s.share.Words() }

func (s Sheet) Number() int { return int(s.share.MemberIndex) + 1 }

// ParseSheet reads the words of one sheet, each trimmed and lower-cased.
func ParseSheet(words []string) (Sheet, error) {
	share, err := slip39.ParseShare(words)
	if err != nil {
		return Sheet{}, fmt.Errorf("%w: %w", ErrSheetInvalid, err)
	}
	return Sheet{share: share}, nil
}

// Kit is three sheets of one split of a recovery secret: any two restore it.
type Kit struct {
	ID     string
	Sheets [SheetCount]Sheet
}

func NewKit(random io.Reader, secret RecoverySecret) (Kit, error) {
	shares, err := slip39.Split(random, secret.bytes[:], SheetThreshold, SheetCount)
	if err != nil {
		return Kit{}, fmt.Errorf("identity: split the recovery secret: %w", err)
	}
	kit := Kit{ID: fmt.Sprintf("%04X", shares[0].Identifier)}
	for i, share := range shares {
		kit.Sheets[i] = Sheet{share: share}
	}
	return kit, nil
}

func combinePair(sheets []Sheet, position int) ([]byte, error) {
	shares := []slip39.Share{sheets[0].share, sheets[position].share}
	secret, err := slip39.Combine(shares, nil)
	if err != nil {
		return nil, &SheetError{Position: position + 1, Err: fmt.Errorf("%w: %w", ErrSheetInvalid, err)}
	}
	if len(secret) != secretSize {
		return nil, &SheetError{Position: position + 1, Err: fmt.Errorf("%w: a secret of %d bytes", ErrSheetInvalid, len(secret))}
	}
	return secret, nil
}

// RestoreSecret takes two or more sheets of one kit and checks all of them against the first.
func RestoreSecret(sheets []Sheet) (RecoverySecret, error) {
	if len(sheets) < SheetThreshold {
		return RecoverySecret{}, fmt.Errorf("%w: got %d", ErrNotEnoughSheets, len(sheets))
	}
	var restored RecoverySecret
	for position := 1; position < len(sheets); position++ {
		secret, err := combinePair(sheets, position)
		if err != nil {
			return RecoverySecret{}, err
		}
		if position == 1 {
			copy(restored.bytes[:], secret)
		} else if !bytes.Equal(restored.bytes[:], secret) {
			return RecoverySecret{}, &SheetError{Position: position + 1, Err: fmt.Errorf("%w: %v: %w", ErrSheetInvalid, sheets[position], slip39.ErrMixedSets)}
		}
	}
	return restored, nil
}
