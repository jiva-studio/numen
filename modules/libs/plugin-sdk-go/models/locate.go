package models

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// Source is where a plugin looks for a model file and how it fetches one.
type Source struct {
	// GetPaths returns every place the file of that name may be.
	GetPaths func(file string) []string
	// Fetch downloads the address, checked against the sum when one is given.
	Fetch func(ctx context.Context, address, sum string) (string, error)
}

// FindLocal returns the file of a model that is in one of the places the source
// names. A model with a pinned sum is found only when the file holds that sum.
func FindLocal(src Source, pins Pins, name string) (string, bool) {
	sum := pins.GetSum(name)
	for _, p := range src.GetPaths(filepath.Base(name)) {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		if sum == "" || VerifyFile(p, sum) == nil {
			return p, true
		}
	}
	return "", false
}

// Locate returns where one model's file is. A path written down is used as
// given, and its absence is an error. A name is looked for in the places the
// source names and then fetched. The kind names the model in an error.
func Locate(ctx context.Context, src Source, pins Pins, path, name, kind string) (string, error) {
	if path != "" {
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("the %s: %w", kind, err)
		}
		return path, nil
	}
	if name == "" {
		return "", fmt.Errorf("no %s: name one, or say where it is", kind)
	}
	if foundPath, ok := FindLocal(src, pins, name); ok {
		return foundPath, nil
	}
	if !IsAddress(name) {
		return "", fmt.Errorf("the %s %q is not beside the application, and is not somewhere to fetch it from", kind, name)
	}
	found, err := src.Fetch(ctx, name, pins.GetSum(name))
	if err != nil {
		return "", fmt.Errorf("the %s: %w", kind, err)
	}
	return found, nil
}

// Wanted is a model file to locate.
type Wanted struct {
	Dst  *string
	Path string
	Name string
	Kind string
}

// LocateAll locates every wanted model file into its destination.
func LocateAll(ctx context.Context, src Source, pins Pins, wanted []Wanted) error {
	for _, one := range wanted {
		at, err := Locate(ctx, src, pins, one.Path, one.Name, one.Kind)
		if err != nil {
			return err
		}
		if one.Dst != nil {
			*one.Dst = at
		}
	}
	return nil
}
