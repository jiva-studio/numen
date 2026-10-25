// Package testonly hands out the permission to use what only a test may use.
// It sits under internal, so the permission cannot leave this module.
package testonly

// Grant is the permission itself. It carries nothing; holding one is the point.
type Grant struct{}

// NewGrant makes a Grant.
func NewGrant() Grant { return Grant{} }
