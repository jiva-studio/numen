package vault

import (
	"context"
	"errors"
	"path"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Refusal is one file that stayed outside the vault, by the name it carries on
// this machine and by what stopped it.
type Refusal struct {
	Name string
	Why  error
}

// ImportResult is what a drop came to: what the vault now holds, and what it
// does not.
type ImportResult struct {
	// Landed is each file and folder that arrived, by the path the vault files
	// it under.
	Landed []string
	// Refused is each file that stayed where it was.
	Refused []Refusal
}

// Import copies files from this machine into a folder of the vault.
//
// A person hands the window a file by letting go of it over the tree, and every
// file that arrives comes through here. The bytes are copied: what the person
// dropped stays where it was.
type Import struct {
	Writers port.VaultWriters
	// Files is what the person handed over, read where it stands. Nothing here
	// reaches the machine itself: what arrives is a path on a desktop and a
	// content URI on a phone, and this is what tells them apart.
	Files port.ImportedFiles
}

// NewImport is what a file a person handed over arrives through: the vault it
// is copied into, and what reads it where it stands.
func NewImport(writers port.VaultWriters, files port.ImportedFiles) Import {
	return Import{Writers: writers, Files: files}
}

// Execute brings each handle into the folder, under the name it already
// carries. A folder arrives with everything under it.
//
// One file refused leaves the rest to arrive: a drop of twenty pictures is
// nineteen pictures and a sentence. A name the folder already carries is one of
// those refusals: what a person meant by a second file of that name is theirs
// to say.
func (u Import) Execute(
	ctx context.Context,
	v domain.Vault,
	into string,
	handles []string,
) (ImportResult, error) {
	var brought ImportResult
	if len(handles) == 0 {
		return brought, nil
	}
	writer, err := u.Writers.Open(v)
	if err != nil {
		return brought, err
	}

	for _, handle := range handles {
		if err := ctx.Err(); err != nil {
			return brought, err
		}
		name := u.Files.Named(handle)
		if err := u.bring(ctx, writer, v, handle, filed(into, name), &brought); err != nil {
			brought.Refused = append(brought.Refused, Refusal{Name: name, Why: err})
		}
	}
	return brought, nil
}

// bring copies one file or one whole folder to a path in the vault.
func (u Import) bring(
	ctx context.Context,
	writer port.VaultWriter,
	v domain.Vault,
	from, to string,
	brought *ImportResult,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	info, err := u.Files.Stat(ctx, from)
	if err != nil {
		return err
	}

	switch {
	case info.Folder:
		// A folder the vault sits inside does not come in: the vault is where it
		// would be copied to.
		if u.Files.Holds(from, v.Path) {
			return errHoldsTheVault
		}
		if err := writer.MakeFolder(ctx, to); err != nil {
			return err
		}
		brought.Landed = append(brought.Landed, to)
		held, err := u.Files.List(ctx, from)
		if err != nil {
			return err
		}
		for _, one := range held {
			if err := u.bring(ctx, writer, v, one.Handle, filed(to, one.Name), brought); err != nil {
				brought.Refused = append(brought.Refused, Refusal{Name: one.Name, Why: err})
			}
		}
		return nil

	case info.File:
		file, err := u.Files.Open(ctx, from)
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

// errHoldsTheVault is a folder handed to the window that the vault itself sits
// inside.
var errHoldsTheVault = errors.New("the vault is inside it")

// errNotAFile is a device, a socket or a link handed to the window. The vault
// holds files and folders.
var errNotAFile = errors.New("it is neither a file nor a folder")

// filed is where a name goes in a folder of the vault. The root is the empty
// path, and a name at the root is the whole of it.
func filed(folder, name string) string {
	if folder == "" {
		return name
	}
	return path.Join(folder, name)
}
