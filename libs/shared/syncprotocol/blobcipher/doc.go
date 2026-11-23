// Package blobcipher is the ciphertext format of a note revision stored on the sync server: a 56-byte header
// followed by the payload cut into chunks that are each sealed with XChaCha20-Poly1305.
//
// A chunk is bound to its position and to the last-chunk flag, so reordering, repeating and truncating chunks
// make Open fail. The header carries a commitment to the revision key. Every blob is padded to a step of a
// ladder so its size hides the exact size of the note. The package imports nothing of the repository, so the
// desktop plugin and the phone can share the format.
package blobcipher
