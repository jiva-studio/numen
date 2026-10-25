// Package source turns what a vault holds into what can be searched: a book's
// text is taken out of it and cut into the chunks a search runs over.
//
// Discovery is what triggers it. Nobody asks for a book to be indexed: a
// search answers over a whole vault, so nothing in one may be waiting to be
// requested.
//
// Cutting is deterministic, needs nothing but the file, and either produces a
// source's chunks or leaves the source owing them. It keeps no record of its own
// progress: what owes work is read from the data, so a source with no small
// chunk owes a cut. Reading a scan, hearing a recording and embedding the
// chunks are the use cases of recognition, transcription and embedding.
package source
