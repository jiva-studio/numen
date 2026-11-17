package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostvault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
)

func getPluginLookups(db *Index) hostvault.Lookups {
	queries, frontmatter, feed := db.NoteQueries(), db.NoteQueryReads(), db.FileChanges()
	return hostvault.Lookups{
		Notes: hostvault.Notes{
			Notes:      query.NewGetNotes(queries),
			Types:      query.NewGetTypes(queries),
			NamedPaths: query.NewGetNamedPaths(queries),
			TypedPaths: query.NewGetPathsOfType(queries),
		},
		Changes: hostvault.Changes{
			Frontmatter: query.NewReadFrontmatter(frontmatter, feed, query.GetFrontmatterLimits()),
			FileChanges: query.NewListFileChanges(feed),
		},
		Links: hostvault.Links{
			Links:     query.NewGetLinks(queries),
			Backlinks: query.NewGetBacklinks(queries),
			Targets:   query.NewResolveLinks(queries),
		},
	}
}

func getIndexState(db *Index) *query.GetIndexState {
	return query.NewGetIndexState(db.NoteQueries())
}
