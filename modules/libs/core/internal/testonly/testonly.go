// Package testonly hands out the permission to use what only a test may use.
// It sits under internal, so the permission cannot leave this module.
package testonly

// Grant is the permission itself. It carries nothing; holding one is the point.
type Grant struct{}

// Granted makes a Grant.
func Granted() Grant { return Grant{} }
