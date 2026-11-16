package settags

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"

	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/domain/tags"
)

// readRecent returns the tags of the file recentFile, the latest first.
func readRecent(ctx context.Context, store Store) ([]string, error) {
	content, err := store.Read(ctx, recentFile)
	if errors.Is(err, host.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", recentFile, err)
	}
	var list []string
	if err := json.Unmarshal(content, &list); err != nil {
		return nil, fmt.Errorf("parse %s: %w", recentFile, err)
	}
	return list, nil
}

// remember puts the tags in front of the recent tags, the last of them first, and keeps the first recentLimit.
func remember(ctx context.Context, store Store, set []toggle) error {
	if len(set) == 0 {
		return nil
	}
	list, err := readRecent(ctx, store)
	if err != nil {
		return err
	}
	var front []string
	for _, one := range slices.Backward(set) {
		if !slices.ContainsFunc(front, func(tag string) bool { return tags.GetKey(tag) == one.key }) {
			front = append(front, one.tag)
		}
	}
	merged := slices.Clone(front)
	for _, tag := range list {
		isNew := !slices.ContainsFunc(front, func(one string) bool { return tags.GetKey(one) == tags.GetKey(tag) })
		if isNew {
			merged = append(merged, tag)
		}
	}
	content, err := json.Marshal(merged[:min(len(merged), recentLimit)])
	if err != nil {
		return fmt.Errorf("encode %s: %w", recentFile, err)
	}
	if err := store.Write(ctx, recentFile, content); err != nil {
		return fmt.Errorf("write %s: %w", recentFile, err)
	}
	return nil
}
