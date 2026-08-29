package domain

// Vault is a folder the user has added to the application. Its identity is a
// ULID carried inside the vault itself, not derived from its path, because
// folders move outside the application.
type Vault struct {
	ID   string
	Name string
	Path string
}
