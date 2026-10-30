package vault

import (
	"context"
	"errors"
	"path"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
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
// A file dropped onto the tree is imported through this use case. The bytes are
// copied, and the source file is left in place.
type Import struct {
	writers vaultport.VaultWriters
	// files reads the imported handles. A handle is a path on a desktop and a
	// content URI on a phone; this port resolves both, and nothing else in this
	// type accesses the machine directly.
	files sourceport.ImportedFiles
}

// NewImport creates an Import from the writers of the target vault and the port
// that reads imported files.
func NewImport(writers vaultport.VaultWriters, files sourceport.ImportedFiles) Import {
	return Import{writers: writers, files: files}
}

// Execute copies each handle into the folder under its own name. A folder is
// copied with everything under it. Failures are collected per handle in the
// result; a name the folder already contains is one such error.
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
	writer vaultport.VaultWriter,
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
	writer vaultport.VaultWriter,
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
	ctx context.Context, writer vaultport.VaultWriter, from, to string, result *ImportResult,
) (err error) {
	file, err := u.files.Open(ctx, from)
	if err != nil {
		return err
	}
	defer closeAndJoin(&err, file)
	if err := writer.Import(ctx, to, file); err != nil {
		return err
	}
	result.Landed = append(result.Landed, to)
	return nil
}

// errHoldsTheVault marks a dropped folder that contains the vault itself.
var errHoldsTheVault = errors.New("the vault is inside it")

// errNotAFile is a device, a socket or a link handed to the window. The vault
// holds files and folders.
var errNotAFile = errors.New("it is neither a file nor a folder")
