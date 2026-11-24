// Package stopafter counts the bytes of one direction against a budget.
package stopafter

// Limit counts the bytes of one direction since the admin call that armed it.
type Limit struct {
	isArmed bool
	left    int64
}

// Start arms the limit with a budget of bytes.
func (a *Limit) Start(bytes int64) {
	a.isArmed, a.left = true, bytes
}

// Take spends bytes of the budget and returns how many of them are allowed; isEnd is true when the budget ends.
func (a *Limit) Take(bytes int64) (allowed int64, isEnd bool) {
	if !a.isArmed {
		return bytes, false
	}
	if bytes < a.left {
		a.left -= bytes
		return bytes, false
	}
	allowed = a.left
	a.isArmed, a.left = false, 0
	return allowed, true
}
