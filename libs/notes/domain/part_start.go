package domain

// A PartStart is somewhere in a source's text that carries a name. Parts bound
// the divisions a text is cut inside, and need not arrive in order.
type PartStart struct {
	Title  string
	Offset int
}
