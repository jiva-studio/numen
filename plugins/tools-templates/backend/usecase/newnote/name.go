package newnote

import (
	"context"
	"errors"
	"fmt"
	pathpkg "path"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

func suggestName(ctx context.Context, vault Vault, folder, template string) (string, error) {
	stem := validateName(getName(template))
	if stem == "" {
		return "", nil
	}
	for number := 1; number <= maxNames; number++ {
		name := stem
		if number > 1 {
			name = fmt.Sprintf("%s %d", stem, number)
		}
		isOpen, err := isFree(ctx, vault, getNotePath(folder, name))
		if err != nil {
			return "", err
		}
		if isOpen {
			return name, nil
		}
	}
	return "", nil
}

<<<<<<< HEAD
func checkedName(name string) string {
	reduced, _, err := domain.CreateFilename(name)
||||||| a8fdfeeb1
func checkedName(name string) string {
	reduced, _, err := domain.Filename(name)
=======
func validateName(name string) string {
	reduced, _, err := domain.Filename(name)
>>>>>>> origin/main
	if err != nil {
		return ""
	}
	return reduced
}

func checkName(name string) error {
	reduced, isExact, err := domain.CreateFilename(name)
	if err != nil {
		return err
	}
	if !isExact {
		return fmt.Errorf("%w: %q cannot be a file name, %q can", domain.ErrInvalidName, name, reduced)
	}
	return nil
}

func getNotePath(folder, name string) string {
	return pathpkg.Join(folder, name+noteExtension)
}

func isFree(ctx context.Context, vault Vault, path string) (bool, error) {
	_, err := vault.StatFile(ctx, path)
	if errors.Is(err, host.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("check the name %s: %w", path, err)
	}
	return false, nil
}

func getName(template string) string {
	stem := strings.TrimSuffix(pathpkg.Base(template), noteExtension)
	if name := strings.TrimSuffix(stem, templateExtension); name != "" {
		return name
	}
	return stem
}
