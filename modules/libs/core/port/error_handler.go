package port

// ErrorHandler is where the core says what went wrong in work it carries on
// past. A caller that sets none is told nothing.
type ErrorHandler func(error)
