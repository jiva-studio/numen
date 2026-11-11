package api

// FileMove is the move of a note's file and the links the host rewrote for it.
type FileMove struct {
	From          string   `json:"from" jsonschema:"previous path of the note"`
	Path          string   `json:"path" jsonschema:"new path of the note"`
	RepairedNotes []string `json:"repairedNotes" jsonschema:"paths of the notes whose links to the old name were rewritten"`
	BrokenLinks   []string `json:"brokenLinks" jsonschema:"paths of the notes whose links to the old name could not be rewritten and are broken"`
}

type NoteRenameInput struct {
	Path  string `json:"path" jsonschema:"path of the note"`
	Title string `json:"title" jsonschema:"the new title"`
}

type NoteRenameOutput struct {
	Path       string    `json:"path" jsonschema:"path of the note after the rename"`
	Title      string    `json:"title" jsonschema:"title of the note after the rename"`
	NameSource string    `json:"nameSource" jsonschema:"where the title is written: frontmatter for the title key, filename for the file name"`
	Move       *FileMove `json:"move,omitempty" jsonschema:"the move of the file; absent when the file kept its path"`
}

type NoteMoveInput struct {
	Paths  []string `json:"paths" jsonschema:"paths of the notes, at most 50"`
	Folder string   `json:"folder" jsonschema:"folder to move the notes to, relative to the vault folder; \"/\" for the vault root"`
}

// NoteMoveItem is the outcome of moving one note.
type NoteMoveItem struct {
	From          string   `json:"from" jsonschema:"path of the note as given"`
	Path          string   `json:"path,omitempty" jsonschema:"path of the note now; absent when it did not move"`
	RepairedNotes []string `json:"repairedNotes,omitempty" jsonschema:"paths of the notes whose links to the note were rewritten"`
	BrokenLinks   []string `json:"brokenLinks,omitempty" jsonschema:"paths of the notes whose links to the note are now broken"`
	Error         string   `json:"error,omitempty" jsonschema:"error message when the note was not moved"`
}

type NoteMoveOutput struct {
	Results []NoteMoveItem `json:"results" jsonschema:"one entry per path, in the order given"`
}

type NoteRemoveInput struct {
	Paths []string `json:"paths" jsonschema:"paths of the notes, at most 50"`
}

// NoteRemoveItem is the outcome of removing one note.
type NoteRemoveItem struct {
	Path        string   `json:"path" jsonschema:"path of the note as given"`
	TrashPath   string   `json:"trashPath,omitempty" jsonschema:"path of the file in the vault's trash, from which it can be restored"`
	BrokenLinks []string `json:"brokenLinks,omitempty" jsonschema:"paths of the notes whose links to the note are now broken"`
	Error       string   `json:"error,omitempty" jsonschema:"error message when the note was not removed"`
}

type NoteRemoveOutput struct {
	Results []NoteRemoveItem `json:"results" jsonschema:"one entry per path, in the order given"`
}
