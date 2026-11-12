package api

type LinkListInput struct {
	Path string `json:"path" jsonschema:"path of the note to inspect for links and backlinks"`
}

// ResolvedLinkItem is a link of a note with the path it resolves to.
type ResolvedLinkItem struct {
	From        string `json:"from" jsonschema:"source note path"`
	To          string `json:"to,omitempty" jsonschema:"resolved target note path"`
	Address     string `json:"address" jsonschema:"target link address as written"`
	Role        string `json:"role,omitempty" jsonschema:"link role (e.g. parent, child, jump, ref)"`
	Label       string `json:"label,omitempty" jsonschema:"link label"`
	IsAmbiguous bool   `json:"isAmbiguous,omitempty" jsonschema:"true if link target matches multiple notes"`
}

type LinkListOutput struct {
	Path      string             `json:"path"`
	Links     []ResolvedLinkItem `json:"links" jsonschema:"outgoing links from this note"`
	Backlinks []ResolvedLinkItem `json:"backlinks" jsonschema:"incoming backlinks pointing to this note"`
}

// LinkAddition is one link to write into the frontmatter of a note.
type LinkAddition struct {
	From        string `json:"from" jsonschema:"path of the source note"`
	Fingerprint string `json:"fingerprint" jsonschema:"the fingerprint note_read returned for the note; the same for every link written into one note"`
	Address     string `json:"address" jsonschema:"the other note's name, or note://<identifier> when the name is ambiguous"`
	Role        string `json:"role" jsonschema:"the kind of relationship: parent, child, jump, ref or attachment"`
	Type        string `json:"type,omitempty" jsonschema:"an open type of the link, such as source"`
	Label       string `json:"label,omitempty" jsonschema:"a few words naming the relationship"`
	Note        string `json:"note,omitempty" jsonschema:"why the link exists"`
}

type LinkAddInput struct {
	Links []LinkAddition `json:"links" jsonschema:"the links to write, at most 50"`
}

// LinkAddItem is the outcome of writing one link.
type LinkAddItem struct {
	From        string `json:"from" jsonschema:"path of the source note"`
	Address     string `json:"address" jsonschema:"the target as given"`
	Fingerprint string `json:"fingerprint,omitempty" jsonschema:"fingerprint of the note after the write, for its next write; absent when the link was not written"`
	Error       string `json:"error,omitempty" jsonschema:"error message when the link was not written"`
}

type LinkAddOutput struct {
	Results []LinkAddItem `json:"results" jsonschema:"one entry per link, in the order given"`
}

type LinkUpdateInput struct {
	From        string `json:"from" jsonschema:"path of the source note"`
	Address     string `json:"address" jsonschema:"the target as it is written now"`
	Fingerprint string `json:"fingerprint" jsonschema:"the fingerprint note_read returned for the note"`
	Role        string `json:"role,omitempty" jsonschema:"the link role: parent, child, jump, ref or attachment; kept when left out"`
	Type        string `json:"type,omitempty" jsonschema:"the type of the link; kept when left out"`
	Label       string `json:"label,omitempty" jsonschema:"a few words naming the relationship; kept when left out"`
	Note        string `json:"note,omitempty" jsonschema:"why the link exists; kept when left out"`
}

type LinkRemoveInput struct {
	From        string `json:"from" jsonschema:"path of the source note"`
	Address     string `json:"address" jsonschema:"the target as it is written"`
	Fingerprint string `json:"fingerprint" jsonschema:"the fingerprint note_read returned for the note"`
	Role        string `json:"role,omitempty" jsonschema:"remove only the link of this role; every link to the target when left out"`
}

// LinkEditOutput is the output of the link_update and link_remove tools.
type LinkEditOutput struct {
	Path        string `json:"path" jsonschema:"path of the note written"`
	Fingerprint string `json:"fingerprint" jsonschema:"fingerprint of the note after the write, for its next write"`
}
