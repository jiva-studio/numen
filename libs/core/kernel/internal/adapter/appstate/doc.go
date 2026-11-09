// Package appstate stores what this installation records about itself, currently
// the list of vaults. It is application state: not derivable from any vault,
// and never stored inside one.
//
// The format is JSON: it is a handful of entries whose most important moment
// is when the application will not start, and a file can be opened and fixed
// with nothing running.
package appstate
