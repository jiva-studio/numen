package process

import "errors"

// ErrNiceDenied means the operating system refused to lower the niceness of a plugin process: the
// user holds no right to raise its priority. The process keeps the niceness it has.
var ErrNiceDenied = errors.New("priority change denied")
