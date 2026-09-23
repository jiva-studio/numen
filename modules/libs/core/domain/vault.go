package domain

// VaultID names one vault. It is a ULID, and it is the only thing that says
// which vault a call is about: a name and a path are both about the same vault
// and neither identifies it.
type VaultID string

// Vault is a folder the user has added to the application. Its identity is a
// ULID carried inside the vault itself, not derived from its path, because
// folders move outside the application.
type Vault struct {
	ID   VaultID
	Name string
	Path string
}
