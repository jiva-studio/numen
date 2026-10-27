// Package filetext puts the text of the vault's files other than notes into the
// index: the live text version of each file is cut into the chunks a search runs
// over. The core reads no file format; a plugin makes the text versions.
//
// Discovery is what triggers it. Nobody asks for a file to be indexed: a search
// answers over a whole vault, so nothing in one may be waiting to be requested.
//
// Cutting is deterministic and either produces a file's chunks or leaves the
// file owing them. It keeps no record of its own progress: what owes work is
// read from the data, so a source with no small chunk owes a cut. Embedding the
// chunks is the use case of vectors.
package filetext
