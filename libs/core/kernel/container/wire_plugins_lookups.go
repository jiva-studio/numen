package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostvault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
)

func getPluginNotes(db *Index) hostvault.Notes {
	queries := db.NoteQueries()
	return hostvault.Notes{
		Notes:      query.NewGetNotes(queries),
		Types:      query.NewGetTypes(queries),
		NamedPaths: query.NewGetNamedPaths(queries),
		TypedPaths: query.NewGetPathsOfType(queries),
	}
}

func getPluginLinks(db *Index) hostvault.Links {
	queries := db.NoteQueries()
	return hostvault.Links{
		Links:     query.NewGetLinks(queries),
		Backlinks: query.NewGetBacklinks(queries),
		Targets:   query.NewResolveLinks(queries),
	}
}

func getIndexState(db *Index) *query.GetIndexState {
	return query.NewGetIndexState(db.NoteQueries())
}
