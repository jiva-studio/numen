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
	FunctionTags     = "tags"
	FunctionIsTagged = "isTagged"
	argumentTag      = "tag"
)

func newFunctions(cache *tagcache.Caches) (*searchfunction.Registry, error) {
	registry := searchfunction.New()
	if err := registry.Add(FunctionTags, newFunction(cache, readTagArgument)); err != nil {
		return nil, err
	}
	if err := registry.Add(FunctionIsTagged, newFunction(cache, func(searchfunction.Call) (string, error) { return "", nil })); err != nil {
		return nil, err
	}
	return registry, nil
}

func readTagArgument(call searchfunction.Call) (string, error) {
	tag, _ := call.Arguments.GetString(argumentTag)
	if tag = tags.Clean(tag); tag == "" {
		return "", fmt.Errorf("%w: tags needs a tag", searchfunction.ErrInvalidArguments)
	}
	return tag, nil
}

func newFunction(cache *tagcache.Caches, readParent func(searchfunction.Call) (string, error)) searchfunction.Function {
	list := func(ctx context.Context, call searchfunction.Call) ([]string, string, error) {
		parent, err := readParent(call)
		if err != nil {
			return nil, "", err
		}
		paths, isStale, err := cache.ListTaggedPaths(ctx, domain.VaultID(call.VaultID), parent)
		if err != nil {
			return nil, "", err
		}
		if isStale {
			return paths, fmt.Sprintf("the tags may be out of date (index generation %s)", call.IndexGeneration), nil
		}
		return paths, "", nil
	}
	return searchfunction.Function{
		Match: func(ctx context.Context, call searchfunction.Call, asked []string) (searchfunction.Matched, error) {
			paths, warning, err := list(ctx, call)
			if err != nil {
				return searchfunction.Matched{}, err
			}
			matched := slices.DeleteFunc(slices.Clone(asked), func(path string) bool {
				_, isTagged := slices.BinarySearch(paths, path)
				return !isTagged
			})
			return searchfunction.Matched{Paths: matched, Warning: warning}, nil
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
