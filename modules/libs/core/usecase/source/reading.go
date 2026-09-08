package source

import (
	"errors"
	"io"
)

// What a copy is read through on its way to disk: how much has gone past, and
// how much may.

// progressStep is how much has to arrive before it is said again. Finer than a
// megabyte is past what a bar shows, and every report crosses to the window.
const progressStep = 1 << 20

// passing is the bytes that went past. A copy the vault wrote says how large it
// is without being asked for again, and a copy still arriving says how far it
// has got, where anything is listening.
type passing struct {
	from   io.Reader
	total  int64
	report func(done, total int64)
	read   int64
	said   int64
}

func (p *passing) Read(into []byte) (int, error) {
	read, err := p.from.Read(into)
	p.read += int64(read)
	if p.report != nil && p.read-p.said >= progressStep {
		p.said = p.read
		p.report(p.read, p.total)
	}
	return read, err
}

// errTooLarge is a copy that ran past the size the settings name.
var errTooLarge = errors.New("over the size a copy may be")

// capped is what comes down, stopped at the size the settings name. A site that
// declares no size is held to it all the same, and a limit of nothing lets
// everything through.
func capped(from io.Reader, under int64) io.Reader {
	if under <= 0 {
		return from
	}
	return &limited{from: from, left: under + 1}
}

type limited struct {
	from io.Reader
	left int64
}

func (l *limited) Read(into []byte) (int, error) {
	if l.left <= 0 {
		return 0, errTooLarge
	}
	if int64(len(into)) > l.left {
		into = into[:l.left]
	}
	read, err := l.from.Read(into)
	l.left -= int64(read)
	if l.left <= 0 {
		return read, errTooLarge
	}
	return read, err
}
