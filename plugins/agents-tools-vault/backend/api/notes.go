package api

type NoteReadInput struct {
	Path string `json:"path" jsonschema:"relative path to the note file (e.g. Notes/Topic.md)"`
}

type NoteReadOutput struct {
	Path        string         `json:"path" jsonschema:"path of the note"`
	Title       string         `json:"title" jsonschema:"title of the note"`
	Text        string         `json:"text" jsonschema:"full raw markdown content of the file"`
	Body        string         `json:"body" jsonschema:"prose content of the note excluding frontmatter"`
	Fingerprint string         `json:"fingerprint" jsonschema:"concurrency fingerprint for atomic writes"`
	Frontmatter map[string]any `json:"frontmatter,omitempty" jsonschema:"parsed YAML frontmatter, if present"`
}

type NoteWriteInput struct {
	Path        string  `json:"path" jsonschema:"relative path to the note file"`
	Text        *string `json:"text,omitempty" jsonschema:"markdown content to write"`
	Fingerprint string  `json:"fingerprint,omitempty" jsonschema:"optional baseline fingerprint from note_read for optimistic concurrency"`
}

type NoteWriteOutput struct {
	Path        string `json:"path" jsonschema:"path written"`
	Fingerprint string `json:"fingerprint" jsonschema:"new concurrency fingerprint after write"`
	Bytes       int    `json:"bytes" jsonschema:"number of bytes written"`
}

// HeadingItem is a heading of a note.
type HeadingItem struct {
	Level int    `json:"level" jsonschema:"heading level (1 to 6)"`
	Text  string `json:"text" jsonschema:"heading text"`
	Line  int    `json:"line" jsonschema:"line of the heading in the body, counted from 1 at the first line of the body"`
	Start int    `json:"start" jsonschema:"byte offset of the heading in the body"`
}

type NoteHeadingListInput struct {
	Path string `json:"path" jsonschema:"relative path to the note file"`
}

type NoteHeadingListOutput struct {
	Path     string        `json:"path"`
	Headings []HeadingItem `json:"headings"`
}
