// Package source turns what a vault holds into what can be searched: a book's
// text is taken out of it and cut into the windows a search runs over, and the
// windows are embedded.
//
// Discovery is what triggers both. Nobody asks for a book to be indexed: a
// search answers over a whole vault, so nothing in one may be waiting to be
// requested.
//
// Cutting and embedding are separate scenarios. Cutting is deterministic, needs
// nothing but the file, and either produces a source's chunks or leaves the
// source owing them. Embedding needs a model, is interrupted at any point, and
// may never finish. Neither keeps a record of its own progress: what owes work is
// read from the data, so a source with no small window owes a cut and a chunk
// with no vector owes one.
package source
