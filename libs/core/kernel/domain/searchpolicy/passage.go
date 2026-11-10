// Package searchpolicy holds the vector sample a plugin measures a vault with.
package searchpolicy

// Passage is a small chunk that carries a vector, and the source it belongs to.
// Vector is the stored form of the vector.
type Passage struct {
	Source int64
	Path   string
	Vector []byte
}
