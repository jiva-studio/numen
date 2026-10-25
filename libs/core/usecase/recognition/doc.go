// Package recognition reads a scanned document with a model and writes down what
// it saw, puts that reading right with a proofreader, and runs both behind
// whoever asked.
//
// What comes out is an artifact in the vault's derived store: a model made it
// and no machine here can make it again, so the source is cut from it
// afterwards.
package recognition
