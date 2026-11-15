// Package random provides the entropy of the machine for the use cases.
package random

import "crypto/rand"

// Entropy is the cryptographically secure randomness of the machine.
type Entropy struct{}

// Fill fills b with random bytes. The runtime aborts the program when the system has no
// randomness, so filling never fails.
func (Entropy) Fill(b []byte) {
	rand.Read(b)
}
