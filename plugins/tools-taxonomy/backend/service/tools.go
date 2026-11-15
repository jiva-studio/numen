package service

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"

	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/domain/tags"
	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/usecase/tagcache"
)

var errNoVault = errors.New("the call names no vault")

type TagListInput struct {
	Under string `json:"under,omitempty" jsonschema:"a tag: list only it and the tags nested under it"`
}

type TagListOutput struct {
	Tags []TagEntry `json:"tags" jsonschema:"the tags of the vault as a tree read depth first: each tag is followed by the tags nested under it, and siblings come by count, the larger first"`
}

type TagEntry struct {
	Tag   string `json:"tag" jsonschema:"the whole tag, its levels joined by slashes"`
	Name  string `json:"name" jsonschema:"the last level of the tag"`
	Depth int    `json:"depth" jsonschema:"the level of the tag in the tree, 0 for a tag nested under none"`
	Count int    `json:"count" jsonschema:"the number of notes that have the tag or a tag nested under it"`
}

func newTools(cache *tagcache.Caches) (*tools.Registry, error) {
	registry := tools.New()
	err := tools.Add(registry, tools.Spec{
		Name:  "tag_list",
		Title: "List tags",
		Description: "List the tags of the notes of the vault as a tree read depth first, each with the number of notes that have it " +
			"or a tag nested under it. A note's tags are the frontmatter key tags, a list or a single text; a tag " +
			"nests under another with a slash, as Projects/Web under Projects. Tags are compared without case, " +
			"and a leading # is not part of the tag. Pass under to list one tag and the tags nested under it.",
		Effect: tools.ReadOnly,
	}, func(ctx context.Context, in TagListInput) (TagListOutput, error) {
		vault := domain.VaultID(tools.GetVault(ctx))
		if vault == "" {
			return TagListOutput{}, errNoVault
		}
		tree, err := cache.GetTree(ctx, vault, in.Under)
		if err != nil {
			return TagListOutput{}, err
		}
		return TagListOutput{Tags: writeEntries(tree, 0)}, nil
	})
	if err != nil {
		return nil, err
	}
	return registry, nil
}

func writeEntries(nodes []tags.Node, depth int) []TagEntry {
	out := []TagEntry{}
	for _, node := range nodes {
		out = append(out, TagEntry{Tag: node.Tag, Name: node.Name, Depth: depth, Count: node.Count})
		out = append(out, writeEntries(node.Children, depth+1)...)
	}
	return out
}
