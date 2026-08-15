// Package appstate stores what this installation knows about itself, which today
// is the list of vaults. It is application state: not derivable from any vault,
// and never stored inside one.
//
// JSON rather than a database, deliberately — it is a handful of entries whose
// most important moment is when the application will not start, and a file a
// human can open and fix beats a database that needs the application to read it.
package appstate
