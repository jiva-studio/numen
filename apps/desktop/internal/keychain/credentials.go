package keychain

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/shared/atomicfile"
)

// The fallback file is a flat JSON object keyed by the same entry names as the keychain.
const (
	credentialsDir     = "credentials"
	credentialsFile    = "credentials.json"
	credentialsDirPerm = 0o700
)

func (s *Store) getCredentialsFile() string {
	return filepath.Join(s.dir, credentialsDir, credentialsFile)
}

func (s *Store) readEntries(ctx context.Context) (map[string]string, error) {
	raw, err := os.ReadFile(s.getCredentialsFile())
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read the credentials file: %w", err)
	}
	if isEncryptedFile(raw) {
		return s.decryptFile(ctx, raw)
	}
	entries := map[string]string{}
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("read the credentials file: it is not one JSON object of texts: %s", describeJSONSyntax(err))
	}
	if len(entries) > 0 && s.getFileMode() == FileModePassphrase {
		return nil, ErrPlaintextCredentials
	}
	return entries, nil
}

func (s *Store) decryptFile(ctx context.Context, raw []byte) (map[string]string, error) {
	passphrase, err := s.getPassphrase(ctx)
	if err != nil {
		return nil, err
	}
	entries, err := decryptEntries(raw, passphrase)
	if errors.Is(err, ErrWrongPassphrase) {
		s.forgetPassphrase()
	}
	return entries, err
}

func describeJSONSyntax(err error) string {
	if syntax, ok := errors.AsType[*json.SyntaxError](err); ok {
		return fmt.Sprintf("invalid JSON at byte %d", syntax.Offset)
	}
	return "a value has the wrong type"
}

func (s *Store) readFromFile(ctx context.Context, name string) (string, error) {
	entries, err := s.readEntries(ctx)
	if err != nil {
		return "", err
	}
	key, found := entries[name]
	if !found {
		return "", fmt.Errorf("%w: %q", plugin.ErrNoKey, name)
	}
	return key, nil
}

func (s *Store) writeToFile(ctx context.Context, name, key string) error {
	mode := s.getFileMode()
	if mode == FileModeUnset {
		return ErrFileBackendNotConfirmed
	}
	if mode == FileModePlaintext {
		s.warnPlaintext()
	}
	return s.changeEntries(ctx, func() error {
		entries, err := s.readEntries(ctx)
		if err != nil {
			return err
		}
		entries[name] = key
		return s.writeEntries(ctx, entries, mode)
	})
}

func (s *Store) removeFromFile(ctx context.Context, name string) error {
	return s.changeEntries(ctx, func() error {
		entries, err := s.readEntries(ctx)
		if err != nil {
			return err
		}
		if _, found := entries[name]; !found {
			return fmt.Errorf("%w: %q", plugin.ErrNoKey, name)
		}
		delete(entries, name)
		mode := s.getFileMode()
		if mode == FileModeUnset {
			mode = FileModePlaintext
		}
		return s.writeEntries(ctx, entries, mode)
	})
}

func (s *Store) writeEntries(ctx context.Context, entries map[string]string, mode FileMode) error {
	content, err := s.encodeEntries(ctx, entries, mode)
	if err != nil {
		return fmt.Errorf("write the credentials file: %w", err)
	}
	target := s.getCredentialsFile()
	folder := filepath.Dir(target)
	if err := os.MkdirAll(folder, credentialsDirPerm); err != nil {
		return fmt.Errorf("write the credentials file: %w", err)
	}
	if err := os.Chmod(folder, credentialsDirPerm); err != nil {
		return fmt.Errorf("write the credentials file: %w", err)
	}
	if err := atomicfile.WriteFile(target, bytes.NewReader(content), 0o600); err != nil {
		return fmt.Errorf("write the credentials file: %w", err)
	}
	return nil
}

func (s *Store) encodeEntries(ctx context.Context, entries map[string]string, mode FileMode) ([]byte, error) {
	if mode != FileModePassphrase {
		return json.Marshal(entries)
	}
	passphrase, err := s.getPassphrase(ctx)
	if err != nil {
		return nil, err
	}
	return encryptEntries(entries, passphrase, s.getKDFParams())
}

func (s *Store) isFilePlaintext() bool {
	raw, err := os.ReadFile(s.getCredentialsFile())
	return err == nil && !isEncryptedFile(raw)
}
