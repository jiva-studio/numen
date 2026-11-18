package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"

	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/domain/tags"
	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/usecase/settags"
	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/usecase/tagcache"
)

func mountAPI(s *server.Server, cache *tagcache.Caches, setter *settags.UseCase) {
	a := s.NewAPI(api.Service)
	server.Handle(a, api.MethodGetTags, func(ctx context.Context, _ api.GetTagsRequest) (api.TagList, error) {
		return getTags(ctx, cache, setter)
	})
	server.Handle(a, api.MethodSetTags, func(ctx context.Context, request api.SetTagsRequest) (api.SetTagsResult, error) {
		return setTags(ctx, setter, request)
	})
	server.Handle(a, api.MethodGetTagMap, func(ctx context.Context, _ api.GetTagMapRequest) (api.TagMap, error) {
		return getTagMap(ctx, cache)
	})
}

func getRequestVault(ctx context.Context) (domain.VaultID, error) {
	id := host.GetRequestVault(server.GetHeader(ctx))
	if id == "" {
		return "", server.NewAPIError(server.CodeFailedPrecondition, "the request names no vault")
	}
	return id, nil
}

func convertError(err error) error {
	switch {
	case errors.Is(err, host.ErrNoVault):
		return server.NewAPIError(server.CodeFailedPrecondition, err.Error())
	case errors.Is(err, host.ErrPermissionDenied):
		return server.NewAPIError(server.CodePermissionDenied, err.Error())
	}
	return err
}

func getTagMap(ctx context.Context, cache *tagcache.Caches) (api.TagMap, error) {
	id, err := getRequestVault(ctx)
	if err != nil {
		return api.TagMap{}, err
	}
	found, err := cache.GetMap(ctx, id)
	if err != nil {
		return api.TagMap{}, convertError(fmt.Errorf("get the tag map: %w", err))
	}
	return api.TagMap{Tags: found}, nil
}

func getTags(ctx context.Context, cache *tagcache.Caches, setter *settags.UseCase) (api.TagList, error) {
	id, err := getRequestVault(ctx)
	if err != nil {
		return api.TagList{}, err
	}
	found, err := cache.GetMap(ctx, id)
	if err != nil {
		return api.TagList{}, convertError(fmt.Errorf("get the tags: %w", err))
	}
	recent, err := setter.ListRecent(ctx, id)
	if err != nil {
		return api.TagList{}, convertError(fmt.Errorf("get the recent tags: %w", err))
	}
	counts := tags.GetCounts(found)
	list := api.TagList{Tags: make([]api.TagCount, 0, len(counts)), Recent: []string{}}
	spelled := make(map[string]string, len(counts))
	for _, one := range counts {
		list.Tags = append(list.Tags, api.TagCount{Tag: one.Tag, Count: one.Count})
		spelled[tags.GetKey(one.Tag)] = one.Tag
	}
	for _, tag := range recent {
		if kept, ok := spelled[tags.GetKey(tag)]; ok {
			list.Recent = append(list.Recent, kept)
		}
	}
	return list, nil
}

func setTags(ctx context.Context, setter *settags.UseCase, request api.SetTagsRequest) (api.SetTagsResult, error) {
	id, err := getRequestVault(ctx)
	if err != nil {
		return api.SetTagsResult{}, err
	}
	results, err := setter.Execute(ctx, id, request.Paths, request.Toggles)
	if err != nil {
		return api.SetTagsResult{}, convertError(fmt.Errorf("set the tags: %w", err))
	}
	notes := make([]api.NoteTags, len(results))
	for i, one := range results {
		notes[i] = api.NoteTags{Path: one.Path, Tags: append([]string{}, one.Tags...), Error: ""}
		if one.Err != nil {
			notes[i].Error = one.Err.Error()
		}
	}
	return api.SetTagsResult{Notes: notes}, nil
}
