package vault

import (
	"context"
	"errors"
	"path"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// ImportFailure is one file that stayed outside the vault, by the name it
// carries on this machine and by what stopped it.
type ImportFailure struct {
	Name string
	Why  error
}

// ImportResult is what a drop came to: what the vault now holds, and what it
// does not.
type ImportResult struct {
	// Landed is each file and folder that arrived, by the path the vault files
	// it under.
	Landed []string
	// Errors is each file that stayed where it was.
	Errors []ImportFailure
}

// Import copies files from this machine into a folder of the vault.
//
// A person hands the window a file by letting go of it over the tree, and every
// file that arrives comes through here. The bytes are copied: what the person
// dropped stays where it was.
type Import struct {
	writers port.VaultWriters
	// files is what the person handed over, read at its current position. Nothing here
	// reaches the machine itself: what arrives is a path on a desktop and a
	// content URI on a phone, and this is what tells them apart.
	files port.ImportedFiles
}

// NewImport is what a file a person handed over arrives through: the vault it
// is copied into, and what reads it at its current position.
func NewImport(writers port.VaultWriters, files port.ImportedFiles) Import {
	return Import{writers: writers, files: files}
}

// Execute brings each handle into the folder, under the name it already
// carries. A folder arrives with everything under it.
//
// nineteen pictures and a sentence. A name the folder already carries is one of
// those errors: what a person meant by a second file of that name is theirs to
// say.
func (u Import) Execute(
	ctx context.Context,
	v domain.Vault,
	into string,
	handles []string,
) (ImportResult, error) {
	if len(handles) == 0 {
		return ImportResult{}, nil
	}
	writer, err := u.writers.Open(v)
	if err != nil {
		return ImportResult{}, err
	}

	var result ImportResult

	for _, handle := range handles {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		name := u.files.GetName(handle)
		if err := u.bring(ctx, writer, v, handle, path.Join(into, name), &result); err != nil {
			result.Errors = append(result.Errors, ImportFailure{Name: name, Why: err})
		}
	}
	return result, nil
}

// bring copies one file or one whole folder to a path in the vault.
func (u Import) bring(
	ctx context.Context,
	writer port.VaultWriter,
	v domain.Vault,
	from, to string,
	result *ImportResult,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	info, err := u.files.GetImportedFile(ctx, from)
	if err != nil {
		return err
	}

	switch {
	case info.IsFolder:
		return u.importFolder(ctx, writer, v, from, to, result)
	case info.IsFile:
		return u.importFile(ctx, writer, from, to, result)
	default:
		return errNotAFile
	}
}

// importFolder makes the folder at a path in the vault and brings each entry of
// leaves the rest to arrive.
func (u Import) importFolder(
	ctx context.Context,
	writer port.VaultWriter,
	v domain.Vault,
	from, to string,
	result *ImportResult,
) error {
	// would be copied to.
	if u.files.HasHandle(from, v.Path) {
		return errHoldsTheVault
	}
	if err := writer.MakeFolder(ctx, to); err != nil {
		return err
	}
	result.Landed = append(result.Landed, to)
	entries, err := u.files.List(ctx, from)
	if err != nil {
		return err
	}
	for _, one := range entries {
		if err := u.bring(ctx, writer, v, one.Handle, path.Join(to, one.Name), result); err != nil {
			result.Errors = append(result.Errors, ImportFailure{Name: one.Name, Why: err})
		}
	}
	return nil
}

// importFile copies the file at from to a path in the vault.
func (u Import) importFile(
	ctx context.Context, writer port.VaultWriter, from, to string, result *ImportResult,
) error {
	file, err := u.files.Open(ctx, from)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := writer.Import(ctx, to, file); err != nil {
		return err
	}
	result.Landed = append(result.Landed, to)
	return nil
}

// errHoldsTheVault is a folder handed to the window that the vault itself sits
// inside.
var errHoldsTheVault = errors.New("the vault is inside it")

// errNotAFile is a device, a socket or a link handed to the window. The vault
// holds files and folders.
var errNotAFile = errors.New("it is neither a file nor a folder")
