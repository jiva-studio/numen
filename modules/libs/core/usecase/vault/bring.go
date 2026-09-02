package vault

import (
	"context"
	"errors"
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Refusal is one file that stayed outside the vault, by the name it carries on
// this machine and by what stopped it.
type Refusal struct {
	Name string
	Why  error
}

// Brought is what a drop came to: what the vault now holds, and what it does
// not.
type Brought struct {
	// Landed is each file and folder that arrived, by the path the vault files
	// it under.
	Landed []string
	// Refused is each file that stayed where it was.
	Refused []Refusal
}

// Bring copies files from this machine into a folder of the vault.
//
// A person hands the window a file by letting go of it over the tree, and every
// file that arrives comes through here. The bytes are copied: what the person
// dropped stays where it was.
type Bring struct {
	Writers port.VaultWriters
}

// Execute brings each path into the folder, under the name it already carries.
// A folder arrives with everything under it.
//
// One file refused leaves the rest to arrive: a drop of twenty pictures is
// nineteen pictures and a sentence. A name the folder already carries is one of
// those refusals: what a person meant by a second file of that name is theirs
// to say.
func (u Bring) Execute(
	ctx context.Context,
	v domain.Vault,
	into string,
	paths []string,
) (Brought, error) {
	var brought Brought
	if len(paths) == 0 {
		return brought, nil
	}
	writer, err := u.Writers.Open(v)
	if err != nil {
		return brought, err
	}

	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return brought, err
		}
		name := filepath.Base(path)
		if err := u.bring(ctx, writer, v, path, filed(into, name), &brought); err != nil {
			brought.Refused = append(brought.Refused, Refusal{Name: name, Why: err})
		}
	}
	return brought, nil
}

// bring copies one file or one whole folder to a path in the vault.
func (u Bring) bring(
	ctx context.Context,
	writer port.VaultWriter,
	v domain.Vault,
	from, to string,
	brought *Brought,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	info, err := os.Lstat(from)
	if err != nil {
		return err
	}

	switch {
	case info.IsDir():
		// A folder the vault sits inside does not come in: the vault is where it
		// would be copied to.
		if holds(from, v.Path) {
			return errAround
		}
		if err := writer.MakeFolder(ctx, to); err != nil {
			return err
		}
		brought.Landed = append(brought.Landed, to)
		held, err := os.ReadDir(from)
		if err != nil {
			return err
		}
		for _, one := range held {
			inside := filepath.Join(from, one.Name())
			if err := u.bring(ctx, writer, v, inside, filed(to, one.Name()), brought); err != nil {
				brought.Refused = append(brought.Refused, Refusal{Name: one.Name(), Why: err})
			}
		}
		return nil

	case info.Mode().IsRegular():
		file, err := os.Open(from)
		if err != nil {
			return err
		}
		defer file.Close()
		if err := writer.Bring(ctx, to, file); err != nil {
			return err
		}
		brought.Landed = append(brought.Landed, to)
		return nil

	default:
		return errNotAFile
	}
}

// errAround is a folder handed to the window that the vault itself sits inside.
var errAround = errors.New("the vault is inside it")

// errNotAFile is a device, a socket or a link handed to the window. The vault
// holds files and folders.
var errNotAFile = errors.New("it is neither a file nor a folder")

// filed is where a name goes in a folder of the vault. The root is the empty
// path, and a name at the root is the whole of it.
func filed(folder, name string) string {
	if folder == "" {
		return name
	}
	return pathpkg.Join(folder, name)
}

// holds is whether a folder on this machine is one that inside sits under.
func holds(folder, inside string) bool {
	from, err := filepath.Abs(folder)
	if err != nil {
		return false
	}
	under, err := filepath.Abs(inside)
	if err != nil {
		return false
	}
	return under == from || strings.HasPrefix(under, from+string(filepath.Separator))
}
