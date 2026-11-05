package chunking

// Tokens is the chunk size in tokens that the embedding plugin reports, shared by every copy of the
// configuration that holds it.
type Tokens struct{}

// Set holds the chunk size in tokens; zero is a size that is not known.
func (*Tokens) Set(int) {}

// Get is the chunk size in tokens, and zero where it is not known.
func (*Tokens) Get() int { return 0 }
