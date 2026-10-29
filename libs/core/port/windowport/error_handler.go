package windowport

// ErrorHandler receives errors from work the core continues past. A caller that
// sets none receives no notification.
type ErrorHandler func(error)
