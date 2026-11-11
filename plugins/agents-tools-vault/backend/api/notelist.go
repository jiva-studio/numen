package api

// NoteEntry is the path and title of a note.
type NoteEntry struct {
	Path  string `json:"path" jsonschema:"path of the note relative to the vault folder"`
	Title string `json:"title" jsonschema:"title of the note"`
}

// NoteTitle is the path, title and identifier of a note.
type NoteTitle struct {
	Path  string `json:"path" jsonschema:"path of the note relative to the vault folder"`
	Title string `json:"title" jsonschema:"title of the note"`
	ID    string `json:"id,omitempty" jsonschema:"identifier written in the note, which [[note://<identifier>]] names; absent for a note without one"`
}

type NoteListInput struct {
	Folder string `json:"folder,omitempty" jsonschema:"limit the result to this folder and its subfolders, relative to the vault folder; the whole vault when left out"`
	Limit  int    `json:"limit,omitempty" jsonschema:"how many notes to return (default: 100, max: 500)"`
	Offset int    `json:"offset,omitempty" jsonschema:"how many notes to skip; the nextOffset of the previous result continues the list"`
}

type NoteListOutput struct {
	Notes      []NoteEntry `json:"notes" jsonschema:"the notes of this page, ordered by path"`
	Total      int         `json:"total" jsonschema:"number of notes that match, before limit and offset"`
	NextOffset int         `json:"nextOffset,omitempty" jsonschema:"the offset of the next page; absent on the last page"`
}

type NoteTitleListInput struct {
	Paths []string `json:"paths" jsonschema:"paths of the notes, at most 50"`
}

type NoteTitleListOutput struct {
	Notes        []NoteTitle `json:"notes" jsonschema:"the notes found, with their titles"`
	MissingPaths []string    `json:"missingPaths" jsonschema:"the paths that name no note, in the order given"`
}

type NoteResolveInput struct {
	Name string `json:"name" jsonschema:"file name of a note without its extension, as a wikilink writes it"`
}

type NoteResolveOutput struct {
	Paths []string `json:"paths" jsonschema:"paths of every note with that name; empty when none has it"`
}
