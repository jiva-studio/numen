// Package identityfile keeps the device chain of this device in a file: public keys and signatures, no secret.
package identityfile

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/shared/atomicfile"
)

const (
	folderName = "identity"
	fileName   = "chain.bin"
	folderMode = 0o700
	fileMode   = 0o600
)

// Chain is the chain file of one device.
type Chain struct{ path string }

// Open returns the chain file under the application data folder: identity/chain.bin.
func Open(folder string) *Chain { return &Chain{path: filepath.Join(folder, folderName, fileName)} }

// Read returns the bytes of the chain, or ErrNoChain when the file is not there.
func (c *Chain) Read(_ context.Context) ([]byte, error) {
	raw, err := os.ReadFile(c.path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, identitydomain.ErrNoChain
	}
	if err != nil {
		return nil, fmt.Errorf("read the chain file: %w", err)
	}
	return raw, nil
}

// Write replaces the file whole.
func (c *Chain) Write(_ context.Context, chain []byte) error {
	if err := os.MkdirAll(filepath.Dir(c.path), folderMode); err != nil {
		return fmt.Errorf("make the folder of the chain file: %w", err)
	}
	if err := atomicfile.WriteFile(c.path, bytes.NewReader(chain), fileMode); err != nil {
		return fmt.Errorf("write the chain file: %w", err)
	}
	return nil
}

// Delete removes the file; a file that is not there is not an error.
func (c *Chain) Delete(_ context.Context) error {
	if err := os.Remove(c.path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("delete the chain file: %w", err)
	}
	return nil
}
