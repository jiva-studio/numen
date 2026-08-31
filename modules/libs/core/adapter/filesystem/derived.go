package filesystem

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
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

// FlashcardsDir is where the answers a person gave their cards are kept. They
// are the one thing here nobody can produce a second time: the notes are the
// person's own writing, and a year of answers to them is not.
const FlashcardsDir = "flashcards"

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
	vault   string // the vault folder
	service string // the application's folder inside it
	id      string // the identity that folder carried when this store was opened
	root    string // <vault>/<serviceDir>
	area    string // the one folder inside it this store answers for
}

// ErrNotThisVault is what a name gets when the folder underneath it no longer
// carries the identity the store was opened on.
var ErrNotThisVault = errors.New("the folder is not the vault this store was opened on")

// DerivedStores opens the shelf of whichever vault a use case is working on.
type DerivedStores struct {
	Options Options
	// Area is the folder inside the service folder these files belong to.
	Area string
}

func (d DerivedStores) Open(v domain.Vault) (port.DerivedStore, error) {
	return OpenDerived(v.Path, d.Options, d.Area)
}

// OpenDerived opens one vault's store, and holds on to the identity that vault
// carries. Nothing is written: a store that created its folder on being opened
// would put one in every vault the application looks at.
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
	id, err := carried(abs, opts.serviceDir())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", abs, err)
	}
	return &Derived{
		vault:   abs,
		service: opts.serviceDir(),
		id:      id,
		root:    filepath.Join(abs, opts.serviceDir()),
		area:    area,
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

// Append adds to the end of what is there, in place. What it is given lands
// whole or does not land at all: a write that stopped partway is cut back to
// the length the file had, so the next append begins where this one found it.
//
// A machine that stopped mid-write leaves a torn tail all the same, and what
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
	had, err := file.Stat()
	if err != nil {
		file.Close()
		return err
	}
	if n, err := file.Write(content); err != nil || n != len(content) {
		return errors.Join(short(name, n, len(content), err), back(file, had.Size()))
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

// short is what an append that did not land says.
func short(name string, wrote, asked int, why error) error {
	if why == nil {
		why = io.ErrShortWrite
	}
	return fmt.Errorf("%s: %d of %d bytes: %w", name, wrote, asked, why)
}

// back cuts a file to the length it had and closes it.
func back(file *os.File, to int64) error {
	err := errors.Join(file.Truncate(to), file.Sync())
	return errors.Join(err, file.Close())
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

// List reports the files directly under a name, as names of this store, sorted.
// A folder among them is not one: what is kept here is files, and a caller
// after them would have to be told which entries it may read.
func (d *Derived) List(_ context.Context, name string) ([]port.Stored, error) {
	target, err := d.at(name)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(target)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	clean, err := cleaned(name)
	if err != nil {
		return nil, err
	}
	out := make([]port.Stored, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		// A file that went between the listing and the asking is one another
		// machine's synchroniser took away, and it is left out here rather than
		// listed at a length nothing has.
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, port.Stored{Name: clean + "/" + e.Name(), Size: int(info.Size())})
	}
	slices.SortFunc(out, func(a, b port.Stored) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
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

// still confirms the folder is the vault this store was opened on. A vault
// carries its identity inside itself, and a folder that has lost the identity
// it had is somewhere else: an unmounted disk, a synchroniser's stub, a folder
// this store's own creation of its parents would otherwise make.
func (d *Derived) still() error {
	id, err := carried(d.vault, d.service)
	if err != nil {
		return fmt.Errorf("%s: %w", d.vault, err)
	}
	if id != d.id {
		return fmt.Errorf("%s carries %q and not %q: %w", d.vault, id, d.id, ErrNotThisVault)
	}
	return nil
}

// carried is the identity a folder holds, and nothing where it holds none.
func carried(root, serviceDir string) (string, error) {
	cfg, err := ReadConfig(root, serviceDir)
	if errors.Is(err, ErrNotAVault) || errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return cfg.ID, nil
}

// at is where one name lands on this machine. Every name is answered where the
// vault still is, so nothing here reads or writes a folder that is no longer
// the one this store was opened on.
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
	if err := d.still(); err != nil {
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
