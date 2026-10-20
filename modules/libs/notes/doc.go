// Package notes is the vocabulary of a vault and the way a note is written: what a note, a link, a
// card and a fingerprint are (domain), how a note's file is read and edited (markdown), how a note
// held in a vault is read and changed through the few calls on its files that takes (edit), and the
// identifiers it is given (ulid).
//
// The core and the plugins share it. It does no I/O of its own, reads no clock and names no port,
// scenario or adapter; the packages here reach only each other and the standard library.
package notes
