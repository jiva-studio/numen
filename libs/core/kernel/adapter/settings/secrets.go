package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"reflect"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// keySecrets is the key of the section of the file that lists the secrets. It is no plugin id,
// because an id has a dot.
const keySecrets = "secrets"

// secretEntry is a secret as the file lays it out. A value is never part of it.
type secretEntry struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Group   string            `json:"group"`
	Allowed []permissionEntry `json:"allowed"`
}

// permissionEntry is a declaration of a plugin as the file lays it out.
type permissionEntry struct {
	Plugin string `json:"plugin"`
	Name   string `json:"name"`
	Group  string `json:"group"`
}

func (f *File) readSecrets(raw json.RawMessage) {
	f.Secrets = nil
	if string(raw) == "null" {
		return
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil {
		f.AddNotice(fmt.Sprintf("numen.json: %s is not a list; its secrets are ignored", keySecrets))
		return
	}
	for index, one := range entries {
		var entry secretEntry
		if err := json.Unmarshal(one, &entry); err != nil {
			f.AddNotice(fmt.Sprintf("numen.json: %s[%d] has the wrong shape; the secret is ignored", keySecrets, index))
			continue
		}
		f.Secrets = append(f.Secrets, newSecret(entry))
	}
}

func newSecret(entry secretEntry) plugin.Secret {
	secret := plugin.Secret{ID: entry.ID, Name: entry.Name, Group: entry.Group}
	for _, one := range entry.Allowed {
		secret.Allowed = append(secret.Allowed, plugin.DeclarationKey(one))
	}
	return secret
}

func newSecretEntries(secrets []plugin.Secret) []secretEntry {
	entries := make([]secretEntry, 0, len(secrets))
	for _, secret := range secrets {
		entry := secretEntry{ID: secret.ID, Name: secret.Name, Group: secret.Group, Allowed: []permissionEntry{}}
		for _, one := range secret.Allowed {
			entry.Allowed = append(entry.Allowed, permissionEntry(one))
		}
		entries = append(entries, entry)
	}
	return entries
}

// DropInvalidSecrets leaves out of force every secret that is not valid and every permission whose
// declaration isDeclared does not know. A secret is invalid when its id, name or group has the
// wrong shape, its id repeats, or its name repeats inside its group. Each one adds a notice and the
// bytes of the file on disk are not touched.
func (f *File) DropInvalidSecrets(isDeclared func(plugin.DeclarationKey) bool) {
	var kept []plugin.Secret
	seenIDs := map[string]bool{}
	seenNames := map[[2]string]bool{}
	for _, secret := range f.Secrets {
		if reason := getSecretIssue(secret, seenIDs, seenNames); reason != "" {
			f.AddNotice(fmt.Sprintf("numen.json: secret %q: %s; the secret is ignored", secret.ID, reason))
			continue
		}
		seenIDs[secret.ID] = true
		seenNames[[2]string{secret.Group, secret.Name}] = true
		secret.Allowed = f.keepDeclared(secret, isDeclared)
		kept = append(kept, secret)
	}
	f.Secrets = kept
}

func getSecretIssue(secret plugin.Secret, seenIDs map[string]bool, seenNames map[[2]string]bool) string {
	switch {
	case !plugin.IsValidSecretID(secret.ID):
		return "the id has the wrong shape"
	case seenIDs[secret.ID]:
		return "the id is used twice"
	case !plugin.IsValidSecretName(secret.Name):
		return "the name has the wrong shape"
	case !plugin.IsValidSecretName(secret.Group):
		return "the group has the wrong shape"
	case seenNames[[2]string{secret.Group, secret.Name}]:
		return "the name is used twice in its group"
	}
	return ""
}

func (f *File) keepDeclared(secret plugin.Secret, isDeclared func(plugin.DeclarationKey) bool) []plugin.DeclarationKey {
	var kept []plugin.DeclarationKey
	for _, one := range secret.Allowed {
		switch {
		case !isDeclared(one):
			f.AddNotice(fmt.Sprintf("numen.json: secret %q: %s declares no %q in group %q; the permission is ignored",
				secret.ID, one.Plugin, one.Name, one.Group))
		case slices.Contains(kept, one):
			f.AddNotice(fmt.Sprintf("numen.json: secret %q: a permission is listed twice; the second is ignored", secret.ID))
		default:
			kept = append(kept, one)
		}
	}
	return kept
}

// ChangeSecrets reads the secrets section of the file, applies change and writes the section back,
// all under one hold of the lock. The rest of the file is kept byte for byte. Nothing is written
// where change fails or leaves the list as it was.
func (f *Files) ChangeSecrets(path string, change func([]plugin.Secret) ([]plugin.Secret, error)) error {
	return f.run(path, func(path string) error {
		raw, err := os.ReadFile(path)
		if errors.Is(err, fs.ErrNotExist) {
			raw = []byte("{}\n")
		} else if err != nil {
			return err
		}
		var root map[string]json.RawMessage
		if err := json.Unmarshal(raw, &root); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		var current File
		if section, found := root[keySecrets]; found {
			current.readSecrets(section)
		}
		updated, err := change(slices.Clone(current.Secrets))
		if err != nil {
			return err
		}
		if reflect.DeepEqual(normalize(updated), normalize(current.Secrets)) {
			return nil
		}
		patched, err := applySetting(raw, Setting{At: []string{keySecrets}, Written: newSecretEntries(updated)})
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		return writeFileAtomically(path, patched)
	})
}

func normalize(secrets []plugin.Secret) []plugin.Secret {
	out := make([]plugin.Secret, 0, len(secrets))
	for _, secret := range secrets {
		if secret.Allowed == nil {
			secret.Allowed = []plugin.DeclarationKey{}
		}
		out = append(out, secret)
	}
	return out
}

// WriteSecretsJSON is the secrets section as the file lays it out.
func (f *File) WriteSecretsJSON() (string, error) {
	content, err := json.Marshal(newSecretEntries(f.Secrets))
	if err != nil {
		return "", err
	}
	return string(content), nil
}
