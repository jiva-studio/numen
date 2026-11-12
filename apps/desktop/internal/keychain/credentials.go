package keychain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// The fallback file is a flat JSON object keyed by the same entry names as the keychain.
const (
	credentialsDir  = "credentials"
	credentialsFile = "credentials.json"
)

func (s *Store) getCredentialsFile() string {
	return filepath.Join(s.dir, credentialsDir, credentialsFile)
}

func (s *Store) readEntries() (map[string]string, error) {
	raw, err := os.ReadFile(s.getCredentialsFile())
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read the credentials file: %w", err)
	}
	entries := map[string]string{}
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("read the credentials file: it is not one JSON object of texts: %s", describeJSONSyntax(err))
	}
	return entries, nil
}

func describeJSONSyntax(err error) string {
	if syntax, ok := errors.AsType[*json.SyntaxError](err); ok {
		return fmt.Sprintf("invalid JSON at byte %d", syntax.Offset)
	}
	return "a value has the wrong type"
}

func (s *Store) readFromFile(name string) (string, error) {
	entries, err := s.readEntries()
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
	return s.changeEntries(ctx, func() error {
		entries, err := s.readEntries()
		if err != nil {
			return err
		}
		entries[name] = key
		return s.writeEntries(entries)
	})
}

func (s *Store) removeFromFile(ctx context.Context, name string) error {
	return s.changeEntries(ctx, func() error {
		entries, err := s.readEntries()
		if err != nil {
			return err
		}
		if _, found := entries[name]; !found {
			return fmt.Errorf("%w: %q", plugin.ErrNoKey, name)
		}
		delete(entries, name)
		return s.writeEntries(entries)
	})
}

func (s *Store) writeEntries(entries map[string]string) (err error) {
	content, err := json.Marshal(entries)
	if err != nil {
		return fmt.Errorf("write the credentials file: %w", err)
	}
	target := s.getCredentialsFile()
	folder := filepath.Dir(target)
	if err := os.MkdirAll(folder, 0o700); err != nil {
		return fmt.Errorf("write the credentials file: %w", err)
	}
	if err := os.Chmod(folder, 0o700); err != nil {
		return fmt.Errorf("write the credentials file: %w", err)
	}
	tmp, err := os.CreateTemp(folder, "."+credentialsFile+".*")
	if err != nil {
		return fmt.Errorf("write the credentials file: %w", err)
	}
	defer func() {
		if removeErr := os.Remove(tmp.Name()); removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
			err = errors.Join(err, removeErr)
		}
	}()
	if err := writeTemporaryFile(tmp, content); err != nil {
		return fmt.Errorf("write the credentials file: %w", err)
	}
	if err := os.Rename(tmp.Name(), target); err != nil {
		return fmt.Errorf("write the credentials file: %w", err)
	}
	return nil
}

func writeTemporaryFile(tmp *os.File, content []byte) error {
	if _, err := tmp.Write(content); err != nil {
		return errors.Join(err, tmp.Close())
	}
	if err := tmp.Sync(); err != nil {
		return errors.Join(err, tmp.Close())
	}
	if err := tmp.Chmod(0o600); err != nil {
		return errors.Join(err, tmp.Close())
	}
	return tmp.Close()
}
