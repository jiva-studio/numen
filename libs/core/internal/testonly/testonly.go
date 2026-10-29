// Package testonly provides the Grant that permits use of APIs only tests may
// use. It sits under internal, so the Grant cannot leave this module.
package testonly

// Grant is the permission itself. It carries nothing; holding one is the point.
type Grant struct{}

// NewGrant makes a Grant.
func NewGrant() Grant { return Grant{} }
