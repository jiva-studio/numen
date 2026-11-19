package note

import _ "embed"

// The statements the package reads through a pool of the index database,
// embedded when the package is built.

//go:embed sql/addressing.sql
var addressingSQL string

//go:embed sql/addressing_at.sql
var addressingAtSQL string

//go:embed sql/backlink_candidates.sql
var backlinkCandidatesSQL string

//go:embed sql/backlink_candidates_at.sql
var backlinkCandidatesAtSQL string

//go:embed sql/candidates.sql
var candidatesSQL string

//go:embed sql/candidates_at.sql
var candidatesAtSQL string

//go:embed sql/dangling.sql
var danglingSQL string

//go:embed sql/frontmatter_after.sql
var frontmatterAfterSQL string

//go:embed sql/frontmatter_at.sql
var frontmatterAtSQL string

//go:embed sql/frontmatter_errors.sql
var frontmatterErrorsSQL string

//go:embed sql/headings_of.sql
var headingsOfSQL string

//go:embed sql/identify.sql
var identifySQL string

//go:embed sql/links_at.sql
var linksAtSQL string

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

//go:embed sql/note_in_vault_by_identifier.sql
var noteInVaultByIdentifierSQL string

//go:embed sql/file_paths.sql
var filePathsSQL string

//go:embed sql/links_by_name.sql
var linksByNameSQL string

//go:embed sql/property_values.sql
var propertyValuesSQL string

//go:embed sql/headings_like.sql
var headingsLikeSQL string

//go:embed sql/all_headings.sql
var allHeadingsSQL string

//go:embed sql/count_frontmatter_errors.sql
var countFrontmatterErrorsSQL string

//go:embed sql/heads_at.sql
var headsAtSQL string
