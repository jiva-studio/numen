// Module notes is the vocabulary of a vault and the way a note is written: what a note, a link, a
// card and a fingerprint are (domain), how a note's file is read and edited (markdown) and the
// identifiers it is given (ulid).
//
// The core and the plugins share it. It does no I/O, takes no clock and names no port, scenario or
// adapter; the packages here reach only each other and the standard library.
package notes
