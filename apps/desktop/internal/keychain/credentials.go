package keychain

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

// The fallback file is a flat JSON object keyed by the same entry names as the keychain.
const (
	credentialsDir  = "credentials"
	credentialsFile = "credentials.json"
)

// getCredentialsFile returns the path of the fallback file.
func (s *Store) getCredentialsFile() string {
	return filepath.Join(s.dir, credentialsDir, credentialsFile)
}

// readEntries reads every entry of the file. A file that is not there holds none.
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
		return nil, fmt.Errorf("read the credentials file: it is not one JSON object of texts: %s", describeSyntax(err))
	}
	return entries, nil
}

// describeSyntax names where a file is malformed without quoting any of it.
func describeSyntax(err error) string {
	var syntax *json.SyntaxError
	if errors.As(err, &syntax) {
		return fmt.Sprintf("invalid JSON at byte %d", syntax.Offset)
	}
	return "a value has the wrong type"
}

// readFromFile returns the key stored under the name in the file.
func (s *Store) readFromFile(name string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := s.readEntries()
	if err != nil {
		return "", err
	}
	key, found := entries[name]
	if !found {
		return "", fmt.Errorf("%w: %q", pluginport.ErrNoKey, name)
	}
	return key, nil
}

// writeToFile stores the key under the name in the file.
func (s *Store) writeToFile(name, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := s.readEntries()
	if err != nil {
		return err
	}
	entries[name] = key
	return s.writeEntries(entries)
}

// removeFromFile removes the key stored under the name from the file, or fails with ErrNoKey.
func (s *Store) removeFromFile(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := s.readEntries()
	if err != nil {
		return err
	}
	if _, found := entries[name]; !found {
		return fmt.Errorf("%w: %q", pluginport.ErrNoKey, name)
	}
	delete(entries, name)
	return s.writeEntries(entries)
}

// writeEntries replaces the file with the entries through a temporary file and a rename, in a
// folder only the user can enter. The file is readable by the user alone.
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
	if err := writeTemporary(tmp, content); err != nil {
		return fmt.Errorf("write the credentials file: %w", err)
	}
	if err := os.Rename(tmp.Name(), target); err != nil {
		return fmt.Errorf("write the credentials file: %w", err)
	}
	return nil
}

// writeTemporary writes the content to the file, flushes it, gives it the mode 0600 and closes it.
func writeTemporary(tmp *os.File, content []byte) error {
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
