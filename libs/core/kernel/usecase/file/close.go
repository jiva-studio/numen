package file

import (
	"errors"
	"io"
)

func closeAndJoin(err *error, c io.Closer) {
	*err = errors.Join(*err, c.Close())
}
