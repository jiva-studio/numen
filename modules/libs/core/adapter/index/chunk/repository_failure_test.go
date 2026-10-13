package chunk_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/index/chunk"
)

// holdsBook indexes a book of a large chunk and two small ones, names for the
// section the large one opens, and a vector for every chunk.
func holdsBook(s *store) {
	s.t.Helper()
	s.cut(repoVaultA, "book", "a.epub", []chunk.Chunk{{
		Start: 0, Length: 20, Text: "whole", Opens: []string{"Beginnings"}, Small: []chunk.Chunk{
			{Start: 0, Length: 10, Text: "left"}, {Start: 10, Length: 10, Text: "right"},
		},
	}})
	var vectors []chunk.Vector
	for _, row := range s.rowsOf(repoVaultA, "a.epub") {
		vectors = append(vectors, s.vectorFor(row, "m", 1))
	}
	if err := s.db.Chunks().SaveVectors(s.t.Context(), vectors); err != nil {
		s.t.Fatal(err)
	}
}

func holdsNotes(s *store) {
	s.t.Helper()
	s.saveNote(repoVaultA, "Old.md", "Old", false)
	s.saveNote(repoVaultA, "Target.md", "Target", false)
	s.cut(repoVaultA, "book", "Target.md.d", []chunk.Chunk{{Start: 0, Length: 4, Text: "standing"}})
}

func replaceWith(chunks ...chunk.Chunk) func(s *store) error {
	return func(s *store) error {
		return s.db.Chunks().ReplaceChunks(s.t.Context(), repoVaultA, "book", "a.epub", chunks)
	}
}

func TestAWriteThatTheDatabaseRefusesSaysWhichStatementAndKeepsTheDatabasesReason(t *testing.T) {
	unchanged := []chunk.Chunk{{
		Start: 0, Length: 20, Text: "whole", Opens: []string{"Beginnings"}, Small: []chunk.Chunk{
			{Start: 0, Length: 10, Text: "left"}, {Start: 10, Length: 10, Text: "right"},
		},
	}}
	opening := []chunk.Chunk{{Start: 0, Length: 20, Text: "whole", Opens: []string{"Beginnings"}}}
	fresh := []chunk.Chunk{{Start: 0, Length: 5, Text: "brand new words", Opens: []string{"Fresh"}}}
	keptLeaf := []chunk.Chunk{{Start: 0, Length: 5, Text: "whole", Opens: []string{"Beginnings"}}}
	move := func(from, to string) func(s *store) error {
		return func(s *store) error { return s.db.Chunks().MoveSources(s.t.Context(), repoVaultA, from, to) }
	}
	source := chunk.Source{Path: "b.epub", Kind: "book", Size: 1, MTime: 1}

	for _, c := range []struct {
		name   string
		before func(s *store)
		poison func(s *store)
		run    func(s *store) error
		want   string
	}{
		{"save source", func(*store) {}, func(s *store) { s.failOn("sources", "INSERT") },
			func(s *store) error { return s.db.Chunks().SaveSource(s.t.Context(), repoVaultA, source) }, "record the source b.epub: "},
		{"extraction source", func(*store) {}, func(s *store) { s.failOn("sources", "INSERT") },
			func(s *store) error { return s.db.Chunks().SaveExtraction(s.t.Context(), repoVaultA, source, nil) }, "record the source b.epub: "},
		{"extraction chunk", func(*store) {}, func(s *store) { s.failOn("chunks", "INSERT") },
			func(s *store) error { return s.db.Chunks().SaveExtraction(s.t.Context(), repoVaultA, source, fresh) }, "store a chunk of this source: "},
		{"new chunk", holdsBook, func(s *store) { s.failOn("chunks", "INSERT") }, replaceWith(fresh...), "store a chunk of this source: "},
		{"new words", holdsBook, func(s *store) { s.failOn("chunks_fts", "INSERT") }, replaceWith(fresh...), "index a chunk for the words in it: "},
		{"new section names", holdsBook, func(s *store) { s.failOn("sections_fts", "INSERT") }, replaceWith(fresh...),
			"index the names of the sections a chunk opens: "},
		{"kept section names", holdsBook, func(s *store) { s.failOn("sections_fts", "INSERT") }, replaceWith(unchanged...),
			"index the names of the sections a chunk opens: "},
		{"old section names", holdsBook, func(s *store) { s.failOn("sections_fts", "DELETE") }, replaceWith(unchanged...),
			"take the names of this source's sections out of the index: "},
		{"moved chunk", holdsBook, func(s *store) { s.failOn("chunks", "UPDATE") }, replaceWith(unchanged...),
			"move a chunk to where its text now is: "},
		{"chunks held", holdsBook, func(s *store) { s.exec(`ALTER TABLE chunks RENAME COLUMN parent_id TO enclosing`) }, replaceWith(unchanged...),
			"the chunks this source is already cut into: "},
		{"gone words", holdsBook, func(s *store) { s.failOn("chunks_fts", "DELETE") }, replaceWith(opening...), "delete_fts: "},
		{"gone coarse", holdsBook, func(s *store) { s.failOn("chunks_vec", "DELETE") }, replaceWith(opening...), "delete_vec: "},
		{"gone chunk", holdsBook, func(s *store) { s.failOn("chunks", "DELETE") }, replaceWith(opening...), "delete_chunk: "},
		{"gone vector", holdsBook, func(s *store) { s.failOn("vectors", "DELETE") }, replaceWith(keptLeaf...), "forget_vector: "},
		{"coarse row", holdsBook, func(s *store) { s.failOn("chunks_vec", "INSERT") },
			func(s *store) error {
				return s.db.Chunks().SaveVectors(s.t.Context(), []chunk.Vector{s.vectorFor(s.rowsOf(repoVaultA, "a.epub")[0], "m", 3)})
			}, "insert_vec: "},
		{"old coarse row", holdsBook, func(s *store) { s.failOn("chunks_vec", "DELETE") },
			func(s *store) error {
				return s.db.Chunks().SaveVectors(s.t.Context(), []chunk.Vector{s.vectorFor(s.rowsOf(repoVaultA, "a.epub")[0], "m", 3)})
			}, "delete_vec: "},
		{"vector", holdsBook, func(s *store) { s.failOn("vectors", "UPDATE") },
			func(s *store) error {
				return s.db.Chunks().SaveVectors(s.t.Context(), []chunk.Vector{s.vectorFor(s.rowsOf(repoVaultA, "a.epub")[0], "m", 3)})
			}, "keep_vector: "},
		{"remove words", holdsBook, func(s *store) { s.failOn("chunks_fts", "DELETE") },
			func(s *store) error { return s.db.Chunks().RemoveSources(s.t.Context(), repoVaultA, "book", []string{"a.epub"}) }, "clear_fts: "},
		{"remove coarse", holdsBook, func(s *store) { s.failOn("chunks_vec", "DELETE") },
			func(s *store) error { return s.db.Chunks().RemoveSources(s.t.Context(), repoVaultA, "book", []string{"a.epub"}) }, "clear_vec: "},
		{"remove sections", holdsBook, func(s *store) { s.failOn("sections_fts", "DELETE") },
			func(s *store) error { return s.db.Chunks().RemoveSources(s.t.Context(), repoVaultA, "book", []string{"a.epub"}) }, "clear_sections: "},
		{"remove chunks", holdsBook, func(s *store) { s.failOn("chunks", "DELETE") },
			func(s *store) error { return s.db.Chunks().RemoveSources(s.t.Context(), repoVaultA, "book", []string{"a.epub"}) }, "clear_chunks: "},
		{"remove source", holdsBook, func(s *store) { s.failOn("sources", "DELETE") },
			func(s *store) error { return s.db.Chunks().RemoveSources(s.t.Context(), repoVaultA, "book", []string{"a.epub"}) }, "delete_source: "},
		{"move", holdsNotes, func(s *store) { s.failOn("sources", "UPDATE") }, move("Old.md", "New.md"), "move_sources: "},
		{"rename note", holdsNotes, func(s *store) { s.failOn("notes", "UPDATE") }, move("Old.md", "New.md"), "rename_note: "},
		{"rename title", holdsNotes, func(s *store) { s.failOn("titles_fts", "INSERT") }, move("Old.md", "New.md"), "rename_title: "},
		{"note naming", holdsNotes, func(s *store) { s.exec(`ALTER TABLE notes RENAME COLUMN frontmatter TO front`) }, move("Old.md", "New.md"),
			"what the note Old.md is called: "},
		{"displaced sources", holdsNotes, func(s *store) { s.exec(`ALTER TABLE sources RENAME COLUMN path TO filed`) }, move("Old.md", "Target.md"),
			"what the vault holds at Target.md and under it: "},
		{"displaced words", holdsNotes, func(s *store) { s.failOn("chunks_fts", "DELETE") }, move("Old.md", "Target.md"), "clear_fts: "},
		{"displaced source", holdsNotes, func(s *store) { s.failOn("sources", "DELETE") }, move("Old.md", "Target.md"), "delete_source: "},
	} {
		t.Run(c.name, func(t *testing.T) {
			s := newStore(t)
			c.before(s)
			c.poison(s)

			wantSQLiteError(t, c.run(s), c.want)
		})
	}
}

