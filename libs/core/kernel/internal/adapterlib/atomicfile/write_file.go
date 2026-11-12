package atomicfile

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// WriteFile writes the content of from to the file at path, whole or not at all: a reader of the
// path sees the old file or the new one, and a failed read of from or a failed write leaves the old
// file and no temporary file. The folder of the path must exist.
//
// The new file has the mode, whatever the old one had. A from that returns an error before its
// end, as one does that checks what it read, stops the write. A rename that fails because another
// process holds the target is not tried again; Replace waits that out for the files of a vault.
func WriteFile(path string, from io.Reader, mode fs.FileMode) (err error) {
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	_, err = replace(root, filepath.Base(path), from, mode, root.Rename)
	return err
}
