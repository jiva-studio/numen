// Package indexing keeps the index of a vault up to date while a window is open: the notes first, then the books, then the vectors both still need.
//
// A Worker schedules the passes, and a Session is the part of it that belongs to one vault. Pass progress is reported to Events.
package indexing
