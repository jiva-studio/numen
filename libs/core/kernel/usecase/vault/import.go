package vault

import (
	"context"
	"errors"
	"path"

	"github.com/jiva-studio/numen/libs/note/domain"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// Import copies files from this machine into a folder of the vault.
//
// A file dropped onto the tree is imported through this use case. The bytes are
// copied, and the source file is left in place.
type Import struct {
	writers vaultport.FolderWriters
	files   vaultport.ImportedFiles
}

// NewImport creates an Import from the writers of the target vault and the port
// that reads imported files.
func NewImport(writers vaultport.FolderWriters, files vaultport.ImportedFiles) Import {
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
) (vaultdomain.ImportResult, error) {
	if len(handles) == 0 {
		return vaultdomain.ImportResult{}, nil
	}
	writer, err := u.writers.OpenFolderWriter(v)
	if err != nil {
		return vaultdomain.ImportResult{}, err
	}

	var result vaultdomain.ImportResult

	for _, handle := range handles {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		name := u.files.GetName(handle)
		if err := u.bring(ctx, writer, v, handle, path.Join(into, name), &result); err != nil {
			result.Errors = append(result.Errors, vaultdomain.ImportFailure{Name: name, Why: err})
		}
	}
	return result, nil
}

func (u Import) bring(
	ctx context.Context,
	writer vaultport.FolderWriter,
	v domain.Vault,
	from, to string,
	result *vaultdomain.ImportResult,
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

func (u Import) importFolder(
	ctx context.Context,
	writer vaultport.FolderWriter,
	v domain.Vault,
	from, to string,
	result *vaultdomain.ImportResult,
) error {
	// would be copied to.
	if u.files.HasHandle(from, v.Path) {
		return errHoldsTheVault
	}
	if err := writer.MakeFolder(ctx, to); err != nil {
		return err
	}
	result.Imported = append(result.Imported, to)
	entries, err := u.files.List(ctx, from)
	if err != nil {
		return err
	}
	for _, one := range entries {
		if err := u.bring(ctx, writer, v, one.Handle, path.Join(to, one.Name), result); err != nil {
			result.Errors = append(result.Errors, vaultdomain.ImportFailure{Name: one.Name, Why: err})
		}
	}
	return nil
}

func (u Import) importFile(
	ctx context.Context, writer vaultport.FolderWriter, from, to string, result *vaultdomain.ImportResult,
) (err error) {
	file, err := u.files.Open(ctx, from)
	if err != nil {
		return err
	}
	defer closeAndJoin(&err, file)
	if err := writer.Import(ctx, to, file); err != nil {
		return err
	}
	result.Imported = append(result.Imported, to)
	return nil
}

// errHoldsTheVault marks a dropped folder that contains the vault itself.
var errHoldsTheVault = errors.New("the vault is inside it")

// errNotAFile is a device, a socket or a link handed to the window. The vault
// holds files and folders.
var errNotAFile = errors.New("it is neither a file nor a folder")
