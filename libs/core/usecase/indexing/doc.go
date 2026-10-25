// Package indexing keeps the index of a vault level with it while a window is open: the notes first, then the books, then the vectors both owe.
//
// A Worker schedules the passes, and a Session is the part of it that belongs to one vault. What a pass is doing is told to Events.
package indexing
