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
	"sync/atomic"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/text"
)

// OCRDir is where the text of a source that has none of its own is kept.
//
// The area is named for what made the files, because that is what is true of
// them: their shape, the fields recorded beside them and what a place in them
// is called all belong to the thing that wrote them, and another producer's
// would not be the same.
const OCRDir = "ocr"

// SpeechDir is where the words a model heard in a recording are kept.
const SpeechDir = text.ASR

// FlashcardsDir is where the answers a person gave their cards are kept. They
// are the one thing here nobody can produce a second time: the notes are the
// person's own writing, and a year of answers to them is not.
const FlashcardsDir = "flashcards"

// DerivedStore is the application's own shelf inside one vault: where a file
// it made, and cannot make again, is kept.
//
// It is a type of its own and not a method on VaultWriter, because it writes to
// the one place VaultWriter refuses and refuses everywhere VaultWriter writes.
// Neither can be made to do the other's work, and that is why there are two.
//
// Every name it takes begins with the name of one of its areas, and it answers
// for no other, so the vault's identity — which is in the folder and in no area
// — is not a name this can express.
type DerivedStore struct {
	vault   string   // the vault folder
	service string   // the application's folder inside it
	id      string   // the identity that folder carried when this store was opened
	root    string   // <vault>/<serviceDir>
	areas   []string // the folders inside it this store answers for
	// last is the configuration file as it stood when the identity was last
	// read out of it. Every name checks the identity, and a file that has not
	// moved carries the identity already read.
	last atomic.Pointer[fileInfo]
	// where the store's own folder is, with every link on the way to it
	// resolved. It is what a name is judged inside, and it is asked again
	// whenever the store's folder is no longer there.
	where atomic.Pointer[places]
}

// places is the store's folder as it is on this machine, and its areas as the
// folders directly inside it.
type places struct {
	root  string
	areas map[string]string
}

// holds says a resolved name is inside the area it claims. An area is one
// folder of the store's own, so a link standing where an area should be holds
// nothing.
func (p *places) holds(real, area string) bool {
	bound, named := p.areas[area]
	return named && under(real, bound)
}

// current says the store's folder still resolves to the place found.
func (d *DerivedStore) current(p *places) bool {
	found, err := os.Lstat(p.root)
	if err != nil {
		return false
	}
	now, err := os.Stat(d.root)
	return err == nil && os.SameFile(found, now)
}

// locate resolves the store's folder and places each area inside it.
func (d *DerivedStore) locate() (*places, error) {
	root, err := deepest(d.root)
	if err != nil {
		return nil, err
	}
	found := &places{root: root, areas: make(map[string]string, len(d.areas))}
	for _, area := range d.areas {
		found.areas[area] = filepath.Join(root, area)
	}
	return found, nil
}

// fileInfo is a file as it stood: what says whether it is still the one read.
type fileInfo struct{ stat os.FileInfo }

// holds reports whether a file is the one a fileInfo was taken of.
func (s *fileInfo) holds(now os.FileInfo) bool {
	return s != nil && os.SameFile(s.stat, now) &&
		s.stat.Size() == now.Size() && s.stat.ModTime().Equal(now.ModTime())
}

// ErrNotThisVault is what a name gets when the folder underneath it no longer
// carries the identity the store was opened on.
var ErrNotThisVault = errors.New("the folder is not the vault this store was opened on")

// DerivedStores opens the shelf of whichever vault a use case is working on.
type DerivedStores struct {
	Options Options
	// Area is the folder inside the service folder these files belong to, and
	// Areas are the further folders the same store answers for. A use case
	// reading what two producers wrote names both.
	Area  string
	Areas []string
}

func (d DerivedStores) Open(v domain.Vault) (port.DerivedStore, error) {
	return OpenDerived(v.Path, d.Options, append([]string{d.Area}, d.Areas...)...)
}

// OpenDerived opens one vault's store, and holds on to the identity that vault
// carries. Nothing is written: a store that created its folder on being opened
// would put one in every vault the application looks at.
//
// The store answers for every area named and the first of them is what it is
// called. Naming none is the default area alone.
func OpenDerived(vaultRoot string, opts Options, areas ...string) (*DerivedStore, error) {
	kept := make([]string, 0, len(areas))
	for _, area := range areas {
		if area == "" {
			continue
		}
		if strings.ContainsAny(area, `/\`) || area == "." || area == ".." {
			return nil, fmt.Errorf("%q is not one folder", area)
		}
		kept = append(kept, area)
	}
	if len(kept) == 0 {
		kept = []string{OCRDir}
	}
	abs, err := filepath.Abs(vaultRoot)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(abs); err != nil {
		return nil, err
	}
	id, was, err := carried(abs, opts.serviceDir())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", abs, err)
	}
	d := &DerivedStore{
		vault:   abs,
		service: opts.serviceDir(),
		id:      id,
		root:    filepath.Join(abs, opts.serviceDir()),
		areas:   kept,
	}
	d.last.Store(was)
	if found, err := d.locate(); err == nil {
		d.where.Store(found)
	}
	return d, nil
}

// Area is the folder inside the service folder this store is called by, which
// is the first part of every name the index records.
func (d *DerivedStore) Area() string { return d.areas[0] }

func (d *DerivedStore) Read(_ context.Context, name string) ([]byte, error) {
	target, err := d.at(name)
	if err != nil {
		return nil, err
	}
	root, at, err := d.beneath(target)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return root.ReadFile(at)
}

func (d *DerivedStore) Write(_ context.Context, name string, content []byte) error {
	target, err := d.at(name)
	if err != nil {
		return err
	}
	root, at, err := d.making(target)
	if err != nil {
		return err
	}
	defer root.Close()
	if err := root.MkdirAll(filepath.Dir(at), 0o755); err != nil {
		return err
	}
	if _, err := replace(root, at, content, 0o644); err != nil {
		return err
	}
	return settle(root, filepath.Dir(at))
}

// Append adds to the end of what is there, in place. What it is given lands
// whole or does not land at all: a write that stopped partway is cut back to
// where it began, so the next append begins where this one found it.
//
// A name two callers append to is held under a claim, because the cut reaches
// whatever was written after this append's own bytes.
//
// A machine that stopped mid-write leaves a torn tail all the same, and what
// reads the file back takes the whole pages and drops what follows them.
func (d *DerivedStore) Append(_ context.Context, name string, content []byte) error {
	target, err := d.at(name)
	if err != nil {
		return err
	}
	root, at, err := d.making(target)
	if err != nil {
		return err
	}
	defer root.Close()
	if err := root.MkdirAll(filepath.Dir(at), 0o755); err != nil {
		return err
	}
	file, err := root.OpenFile(at, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	if n, err := file.Write(content); err != nil || n != len(content) {
		return errors.Join(short(name, n, len(content), err), back(root, file, at, n))
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

// short is what an append that did not land says.
func short(name string, written, wanted int, why error) error {
	if why == nil {
		why = io.ErrShortWrite
	}
	return fmt.Errorf("%s: %d of %d bytes: %w", name, written, wanted, why)
}

// back cuts the bytes an append left behind and closes the file. What it wrote
// ends where the offset now stands, so the cut is that offset less what
// landed, and a write that landed nothing leaves the file as it found it.
//
// The cut is made by name, once the file is closed: a file opened to append
// carries the right to add to the end and not the right to move it. The name is
// the store's own, so the cut lands where the write did.
func back(root *os.Root, file *os.File, name string, wrote int) error {
	if wrote <= 0 {
		return file.Close()
	}
	at, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return errors.Join(err, file.Close())
	}
	if err := file.Close(); err != nil {
		return err
	}
	return cut(root, name, at-int64(wrote))
}

// cut takes a file back to a length and flushes it there.
func cut(root *os.Root, name string, to int64) error {
	file, err := root.OpenFile(name, os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if err := file.Truncate(to); err != nil {
		return errors.Join(err, file.Close())
	}
	if err := file.Sync(); err != nil {
		return errors.Join(err, file.Close())
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
func (d *DerivedStore) Claim(_ context.Context, name string) (func() error, error) {
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
func (d *DerivedStore) List(_ context.Context, name string) ([]port.Entry, error) {
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
	out := make([]port.Entry, 0, len(entries))
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
		out = append(out, port.Entry{Name: clean + "/" + e.Name(), Size: int(info.Size())})
	}
	slices.SortFunc(out, func(a, b port.Entry) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
}

// Remove takes a name out of the store, along with the file a claim on it is
// held on. A caller works in names and knows of no claim, so a name it takes
// away leaves none behind.
func (d *DerivedStore) Remove(_ context.Context, name string) error {
	for _, one := range []string{name, name + claimSuffix} {
		target, err := d.at(one)
		if err != nil {
			return err
		}
		root, at, err := d.beneath(target)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		err = root.Remove(at)
		root.Close()
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	return nil
}

// still confirms the folder is the vault this store was opened on. A vault
// carries its identity inside itself, and a folder that has lost the identity
// it had is somewhere else: an unmounted disk, a synchroniser's stub, an empty
// folder this store made on its way to a name.
//
// Every name is checked, so the check is a stat of the file the identity is
// written in: the same file, of the same length and the same age, carries the
// identity already read out of it. Anything else is read again.
func (d *DerivedStore) still() error {
	if now, err := os.Stat(configAt(d.vault, d.service)); err == nil && d.last.Load().holds(now) {
		return nil
	}
	id, was, err := carried(d.vault, d.service)
	if err != nil {
		return fmt.Errorf("%s: %w", d.vault, err)
	}
	if id != d.id {
		return fmt.Errorf("%s carries %q and not %q: %w", d.vault, id, d.id, ErrNotThisVault)
	}
	// A folder that carried no identity carries none when it is gone, so the
	// folder itself is what says this one is there.
	if id == "" {
		if _, err := os.Stat(d.vault); err != nil {
			return fmt.Errorf("%s: %w", d.vault, err)
		}
	}
	d.last.Store(was)
	// The file the identity is written in is not the one that was read, so the
	// folders around it are asked where they are again.
	if found, err := d.locate(); err == nil {
		d.where.Store(found)
	}
	return nil
}

// carried is the identity a folder holds, and nothing where it holds none. The
// file it was read from comes back with it, stamped before the reading, so a
// file that changed under the reading is read again at the next asking.
func carried(root, serviceDir string) (string, *fileInfo, error) {
	if serviceDir == "" {
		serviceDir = DefaultServiceDir
	}
	var was *fileInfo
	if info, err := os.Stat(configAt(root, serviceDir)); err == nil {
		was = &fileInfo{stat: info}
	}
	cfg, err := ReadConfig(root, serviceDir)
	if errors.Is(err, ErrNotAVault) || errors.Is(err, fs.ErrNotExist) {
		return "", nil, nil
	}
	if err != nil {
		return "", nil, err
	}
	return cfg.ID, was, nil
}

// beneath is the store's folder as a handle, and a place in it as a name that
// handle takes. Every step of a write is made through the handle, so a folder
// swapped for a link while the write is on its way is refused by the machine
// itself and not by a rule read a moment before. The caller closes the handle.
func (d *DerivedStore) beneath(target string) (*os.Root, string, error) {
	name, err := filepath.Rel(d.root, target)
	if err != nil || name == ".." || strings.HasPrefix(name, ".."+string(filepath.Separator)) {
		return nil, "", fmt.Errorf("%s: %w", target, ErrOutside)
	}
	root, err := os.OpenRoot(d.root)
	if err != nil {
		return nil, "", err
	}
	return root, name, nil
}

// making is the same, with the store's own folder put there if it is not. It is
// what a write does that a read does not.
func (d *DerivedStore) making(target string) (*os.Root, string, error) {
	if err := os.MkdirAll(d.root, 0o755); err != nil {
		return nil, "", err
	}
	return d.beneath(target)
}

// area is the one of this store's areas a cleaned name is in, and whether it is
// in any of them.
func (d *DerivedStore) area(clean string) (string, bool) {
	for _, area := range d.areas {
		if clean == area || strings.HasPrefix(clean, area+"/") {
			return area, true
		}
	}
	return "", false
}

// at is where one name lands on this machine. Every name is answered where the
// vault still is, so nothing here reads or writes a folder that is no longer
// the one this store was opened on.
//
// The name is joined under the area it names and checked against it with every
// link on the way resolved. Without that check a name stored here could be a
// link to a note, and a reading would be read back as what a person wrote —
// or would be written over it.
//
// The store's folder need not exist: as much of each path as does exist is
// resolved, which is the same rule a write into the vault is judged by.
func (d *DerivedStore) at(name string) (string, error) {
	clean, err := cleaned(name)
	if err != nil {
		return "", err
	}
	if err := d.still(); err != nil {
		return "", err
	}
	// A name says which area it belongs to, and a store answers for its own
	// only.
	area, held := d.area(clean)
	if !held {
		return "", fmt.Errorf("%s is not in the %s store: %w", name, d.Area(), ErrOutside)
	}
	target := filepath.Join(d.root, filepath.FromSlash(clean))

	real, err := deepest(target)
	if err != nil {
		return "", err
	}
	enclosed, err := d.encloses(real, area)
	if err != nil {
		return "", err
	}
	if !enclosed {
		return "", fmt.Errorf("%s: %w", name, ErrOutside)
	}
	return target, nil
}

// encloses says whether the store's area holds a name where it lands. A link
// stays inside the area it was written into, so what no name expresses no link
// expresses either.
//
// The folder it is judged against is the one resolved last, and an answer
// stands only while the store's folder is still that one. A name it refuses,
// and a folder that has moved, are judged against where the folder is now.
func (d *DerivedStore) encloses(real, area string) (bool, error) {
	if where := d.where.Load(); where != nil && where.holds(real, area) && d.current(where) {
		return true, nil
	}
	where, err := d.locate()
	if err != nil {
		return false, err
	}
	d.where.Store(where)
	return where.holds(real, area), nil
}
