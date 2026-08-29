package filesystem

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// OCRDir is where the text of a source that has none of its own is kept.
//
// The area is named for what made the files, because that is what is true of
// them: their shape, the fields recorded beside them and what a place in them
// is called all belong to the thing that wrote them, and another producer's
// would not be the same.
const OCRDir = "ocr"

// Derived is the application's own shelf inside one vault: where a file it
// made, and cannot make again, is kept.
//
// It is a type of its own and not a method on VaultWriter, because it writes to
// the one place VaultWriter refuses and refuses everywhere VaultWriter writes.
// Neither can be made to do the other's work, and that is why there are two.
//
// Every name it takes begins with the name of its own area, and it answers for
// no other, so the vault's identity — which is in the folder and in no area —
// is not a name this can express.
type Derived struct {
	root string // <vault>/<serviceDir>
	area string // the one folder inside it this store answers for
}

// DerivedStores opens the shelf of whichever vault a use case is working on.
type DerivedStores struct {
	Options Options
	// Area is the folder inside the service folder these files belong to.
	Area string
}

func (d DerivedStores) Open(v domain.Vault) (port.DerivedStore, error) {
	return OpenDerived(v.Path, d.Options, d.Area)
}

// OpenDerived opens one vault's store. Nothing is written: a store that created
// its folder on being opened would put one in every vault the application looks
// at.
func OpenDerived(vaultRoot string, opts Options, area string) (*Derived, error) {
	if area == "" {
		area = OCRDir
	}
	if strings.ContainsAny(area, `/\`) || area == "." || area == ".." {
		return nil, fmt.Errorf("%q is not one folder", area)
	}
	abs, err := filepath.Abs(vaultRoot)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(abs); err != nil {
		return nil, err
	}
	return &Derived{
		root: filepath.Join(abs, opts.serviceDir()),
		area: area,
	}, nil
}

// Area is the folder inside the service folder this store keeps, which is the
// first part of every name the index records.
func (d *Derived) Area() string { return d.area }

func (d *Derived) Read(_ context.Context, name string) ([]byte, error) {
	target, err := d.at(name)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(target)
}

func (d *Derived) Write(_ context.Context, name string, content []byte) error {
	target, err := d.at(name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	if _, err := replace(target, content, 0o644); err != nil {
		return err
	}
	return settle(filepath.Dir(target))
}

// Append adds to the end of what is there, in place.
//
// It is not atomic. A run that stopped partway leaves a torn tail, and what
// reads the file back takes the whole pages and drops what follows them.
func (d *Derived) Append(_ context.Context, name string, content []byte) error {
	target, err := d.at(name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	if _, err := file.Write(content); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

// claimSuffix names the file a claim on a name is held on. It outlives the
// name: a claim is held on an open file, and unlinking one lets a second caller
// make another at the same path and hold it too.
const claimSuffix = ".claim"

// Claim holds a name until the returned function is called, and refuses one
// another caller holds with port.ErrClaimed.
//
// What is held is a lock the kernel keeps on `<name>.claim` beside the
// artifact, so the claim crosses processes and a run that was killed leaves the
// name free. A claim file lying on disk with no lock on it is a name free to
// take.
func (d *Derived) Claim(_ context.Context, name string) (func() error, error) {
	target, err := d.at(name + claimSuffix)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return nil, err
	}
	return claim(target)
}

func (d *Derived) Remove(_ context.Context, name string) error {
	target, err := d.at(name)
	if err != nil {
		return err
	}
	if err := os.Remove(target); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// at is where one name lands on this machine.
//
// The name is joined under the store's own root and checked against it with
// every link on the way resolved. Without that check a name stored here could
// be a link to a note, and a recognition would be read back as what a person
// wrote — or would be written over it.
//
// The store's folder need not exist: as much of each path as does exist is
// resolved, which is the same rule a write into the vault is judged by.
func (d *Derived) at(name string) (string, error) {
	clean, err := cleaned(name)
	if err != nil {
		return "", err
	}
	// A name says which store it belongs to, and a store answers for its own
	// only. That is what keeps the vault's identity out of reach: `config.json`
	// is in the folder and in no store, so no name can express it.
	if clean != d.area && !strings.HasPrefix(clean, d.area+"/") {
		return "", fmt.Errorf("%s is not in the %s store: %w", name, d.area, ErrOutside)
	}
	target := filepath.Join(d.root, filepath.FromSlash(clean))

	real, err := deepest(target)
	if err != nil {
		return "", err
	}
	root, err := deepest(d.root)
	if err != nil {
		return "", err
	}
	if real != root && !strings.HasPrefix(real, root+string(filepath.Separator)) {
		return "", fmt.Errorf("%s: %w", name, ErrOutside)
	}
	return target, nil
}
