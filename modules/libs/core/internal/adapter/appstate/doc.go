// Package appstate stores what this installation knows about itself, which today
// is the list of vaults. It is application state: not derivable from any vault,
// and never stored inside one.
//
// JSON, deliberately: it is a handful of entries whose most important moment
// is when the application will not start, and a person can open a file and fix
// it with nothing running.
package appstate
