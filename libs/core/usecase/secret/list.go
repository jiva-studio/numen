package secret

import (
	"context"
	"errors"
)

// Entry is a secret as the window sees it: whether a value is on this machine, never the value.
type Entry struct {
	Secret
	HasValue bool
}

// Listing is every secret in list order and where the values are kept when it is a file.
type Listing struct {
	Entries []Entry
	// CredentialsPath is empty while the system keychain is in use.
	CredentialsPath string
}

// List reads the secrets without their values.
type List struct {
	operation
	credentialsPath func() string
}

// Execute returns the valid secrets and whether each has a value on this machine.
func (u *List) Execute(ctx context.Context) (Listing, error) {
	list, err := u.ledger.GetSecrets()
	if err != nil {
		return Listing{}, err
	}
	listing := Listing{Entries: make([]Entry, 0, len(list))}
	for _, one := range list {
		listing.Entries = append(listing.Entries, Entry{Secret: one, HasValue: u.hasValue(ctx, one.ID)})
	}
	if u.credentialsPath != nil {
		listing.CredentialsPath = u.credentialsPath()
	}
	return listing, nil
}

// hasValue reports whether the key store holds a value for the secret. A store that fails counts
// as holding none.
func (u *List) hasValue(ctx context.Context, id string) bool {
	if u.keys == nil {
		return false
	}
	value, err := u.keys.GetKey(ctx, getEntryName(id))
	return err == nil && !errors.Is(err, context.Canceled) && value != ""
}
