package models

// Pins maps the address of a model file to the sha256 its bytes must hold.
type Pins map[string]string

// GetSum returns the sha256 pinned for an address, or "" for an address a
// person named.
func (p Pins) GetSum(address string) string {
	return p[address]
}
