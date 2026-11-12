package note

import _ "embed"

// The statements the package reads through a pool of the index database,
// embedded when the package is built.

//go:embed sql/addressing.sql
var addressingSQL string

//go:embed sql/backlink_candidates.sql
var backlinkCandidatesSQL string

//go:embed sql/candidates.sql
var candidatesSQL string

//go:embed sql/dangling.sql
var danglingSQL string

//go:embed sql/frontmatter_errors.sql
var frontmatterErrorsSQL string

//go:embed sql/headings_of.sql
var headingsOfSQL string

//go:embed sql/identify.sql
var identifySQL string

//go:embed sql/links_of.sql
var linksOfSQL string

//go:embed sql/named.sql
var namedSQL string

//go:embed sql/note_by_identifier.sql
var noteByIdentifierSQL string

//go:embed sql/notes_at.sql
var notesAtSQL string

//go:embed sql/notes_of_type.sql
var notesOfTypeSQL string

//go:embed sql/opening.sql
var openingSQL string

//go:embed sql/problems.sql
var problemsSQL string

//go:embed sql/shared_name_links.sql
var sharedNameLinksSQL string

//go:embed sql/summary.sql
var summarySQL string

//go:embed sql/titles.sql
var titlesSQL string

//go:embed sql/types_at.sql
var typesAtSQL string

//go:embed sql/vault_row.sql
var vaultRowSQL string
