package service

import (
	"context"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/searchfunction"

	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/domain/tags"
	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/usecase/tagcache"
)

// The query functions the manifest declares.
const (
	functionHasAnyTag = "hasAnyTag"
	functionHasTag    = "hasTag"
	functionExactly   = "hasTagExactly"
	argumentTag       = "tag"
)

type listPaths func(ctx context.Context, id domain.VaultID, tag string) (paths []string, isStale bool, err error)

func newFunctions(cache *tagcache.Caches) (*searchfunction.Registry, error) {
	registry := searchfunction.New()
	if err := registry.Add(functionHasTag, newFunction(readTagArgument(functionHasTag), cache.ListTaggedPaths)); err != nil {
		return nil, err
	}
	if err := registry.Add(functionExactly, newFunction(readTagArgument(functionExactly), cache.ListExactlyTaggedPaths)); err != nil {
		return nil, err
	}
	if err := registry.Add(functionHasAnyTag, newFunction(func(searchfunction.Call) (string, error) { return "", nil }, cache.ListTaggedPaths)); err != nil {
		return nil, err
	}
	return registry, nil
}

func readTagArgument(function string) func(searchfunction.Call) (string, error) {
	return func(call searchfunction.Call) (string, error) {
		tag, _ := call.Arguments.GetString(argumentTag)
		if tag = tags.Clean(tag); tag == "" {
			return "", fmt.Errorf("%w: %s needs a tag", searchfunction.ErrInvalidArguments, function)
		}
		return tag, nil
	}
}

func newFunction(readTag func(searchfunction.Call) (string, error), listTagged listPaths) searchfunction.Function {
	list := func(ctx context.Context, call searchfunction.Call) ([]string, string, error) {
		parent, err := readTag(call)
		if err != nil {
			return nil, "", err
		}
		paths, isStale, err := listTagged(ctx, domain.VaultID(call.VaultID), parent)
		if err != nil {
			return nil, "", err
		}
		if isStale {
			return paths, fmt.Sprintf("the tags may be out of date (index generation %s)", call.IndexGeneration), nil
		}
		return paths, "", nil
	}
	return searchfunction.Function{
		Match: func(ctx context.Context, call searchfunction.Call, asked []string) (searchfunction.MatchResult, error) {
			paths, warning, err := list(ctx, call)
			if err != nil {
				return searchfunction.MatchResult{}, err
			}
			matched := slices.DeleteFunc(slices.Clone(asked), func(path string) bool {
				_, isTagged := slices.BinarySearch(paths, path)
				return !isTagged
			})
			return searchfunction.MatchResult{Paths: matched, Warning: warning}, nil
		},
		Find: func(ctx context.Context, call searchfunction.Call, limit int) (searchfunction.Found, error) {
			paths, warning, err := list(ctx, call)
			if err != nil {
				return searchfunction.Found{}, err
			}
			return searchfunction.Found{Paths: paths[:min(limit, len(paths))], EstimatedCount: int64(len(paths)), Warning: warning}, nil
		},
	}
}
