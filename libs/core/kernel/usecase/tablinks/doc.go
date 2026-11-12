// Package tablinks keeps the link groups of the tabs of all windows of a vault, moves tabs between windows
// and tells each change to the watching windows.
//
// The three use cases Links, Windows and Transfers share one state under one lock.
package tablinks
