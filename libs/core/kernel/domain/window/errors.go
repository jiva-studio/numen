package window

import "errors"

// ErrAnswerPending is what a vault switch fails with while a page holds work that waits for the
// user's confirmation.
var ErrAnswerPending = errors.New("pending review confirmation")

// ErrClosing is the error for a vault requested in a window that is closing.
var ErrClosing = errors.New("this window is closing")

// ErrFlushing is the error for a vault requested while pending writes are being flushed.
var ErrFlushing = errors.New("the window is flushing pending writes")
