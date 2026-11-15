package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

var instanceNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

type instanceState struct {
	Name         string `json:"name"`
	VM           string `json:"vm"`
	ControllerID string `json:"controller_id"`
	PoolID       string `json:"pool_id"`
	Image        string `json:"image"`
	OSType       string `json:"os_type"`
	OSArch       string `json:"os_arch"`
	IsReady      bool   `json:"is_ready"`
}

type stateStore struct {
	dir string
}

func isValidInstanceName(name string) bool {
	return instanceNamePattern.MatchString(name)
}

func (s stateStore) getInstanceDir(name string) string {
	return filepath.Join(s.dir, "instances", name)
}

func (s stateStore) getSharedDir(name string) string {
	return filepath.Join(s.getInstanceDir(name), "shared")
}

func (s stateStore) getLogPath(name string) string {
	return filepath.Join(s.getInstanceDir(name), "tart-run.log")
}

func (s stateStore) hasInstance(name string) bool {
	_, err := os.Stat(filepath.Join(s.getInstanceDir(name), "instance.json"))
	return err == nil
}

func (s stateStore) writeInstance(state instanceState) error {
	dir := s.getInstanceDir(state.Name)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state of %s: %w", state.Name, err)
	}
	return writeFileAtomic(filepath.Join(dir, "instance.json"), data, 0o600)
}

func (s stateStore) readInstance(name string) (instanceState, error) {
	if !isValidInstanceName(name) {
		return instanceState{}, fmt.Errorf("instance %q: %w", name, errNotFound)
	}
	data, err := os.ReadFile(filepath.Join(s.getInstanceDir(name), "instance.json"))
	if errors.Is(err, fs.ErrNotExist) {
		return instanceState{}, fmt.Errorf("instance %s: %w", name, errNotFound)
	}
	if err != nil {
		return instanceState{}, fmt.Errorf("read state of %s: %w", name, err)
	}
	var state instanceState
	if err := json.Unmarshal(data, &state); err != nil {
		return instanceState{}, fmt.Errorf("parse state of %s: %w", name, err)
	}
	return state, nil
}

func (s stateStore) listInstances() ([]instanceState, error) {
	entries, err := os.ReadDir(filepath.Join(s.dir, "instances"))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list instances: %w", err)
	}
	states := make([]instanceState, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || !s.hasInstance(entry.Name()) {
			continue
		}
		state, err := s.readInstance(entry.Name())
		if err != nil {
			return nil, err
		}
		states = append(states, state)
	}
	return states, nil
}

func (s stateStore) writePID(name string, pid int) error {
	return writeFileAtomic(filepath.Join(s.getInstanceDir(name), "tart-run.pid"), []byte(strconv.Itoa(pid)+"\n"), 0o600)
}

func (s stateStore) writeSharedFile(name, file string, data []byte, perm os.FileMode) error {
	dir := s.getSharedDir(name)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	return writeFileAtomic(filepath.Join(dir, file), data, perm)
}

func (s stateStore) deleteInstance(name string) error {
	if err := os.RemoveAll(s.getInstanceDir(name)); err != nil {
		return fmt.Errorf("delete state of %s: %w", name, err)
	}
	return nil
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename %s: %w", tmp, err)
	}
	return nil
}
