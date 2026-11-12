// Package filetext puts the text of the vault's files other than notes into the
// index: the live text version of each file is cut into the chunks a search runs
// over. The core reads no file format; a plugin makes the text versions.
//
// Discovery triggers it. No caller requests that a file be indexed: a search
// covers a whole vault, so no file may be waiting for a request.
//
// Cutting is deterministic and either produces a file's chunks or leaves the
// file without them. It keeps no record of its own progress: pending work is
// derived from the data, so a source with no small chunk needs a cut. Embedding the
// chunks is the use case of vectors.
package filetext
