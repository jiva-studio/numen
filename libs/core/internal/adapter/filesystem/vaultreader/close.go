package vaultreader

import (
	"errors"
	"io"
)

// closeAndJoin closes c and joins the error of closing into *err.
func closeAndJoin(err *error, c io.Closer) {
	*err = errors.Join(*err, c.Close())
}
