// Package windowsession runs the vault a window shows: it switches the window to another vault,
// starts and stops the passes behind a vault, flushes what the pages hold before either, follows the
// model of the embedder, reports the changes of the vault, and shuts the window down.
//
// The window's pages, its clients and its toolkit stay outside: the scenarios reach them through the
// ports of ports.go.
package windowsession
