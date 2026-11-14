package chunk

import _ "embed"

// The statements the package reads through a pool of the index database,
// embedded when the package is built.

//go:embed sql/coarse.sql
var coarseSQL string

//go:embed sql/enclosing.sql
var enclosingSQL string

//go:embed sql/extensions.sql
var extensionsSQL string

//go:embed sql/file_stats.sql
var fileStatsSQL string

//go:embed sql/file_stats_under.sql
var fileStatsUnderSQL string

//go:embed sql/identify.sql
var identifySQL string

//go:embed sql/kept_vectors.sql
var keptVectorsSQL string

//go:embed sql/lexical.sql
var lexicalSQL string

//go:embed sql/lexical_kinds.sql
var lexicalKindsSQL string

//go:embed sql/record.sql
var recordSQL string

//go:embed sql/progress.sql
var progressSQL string

//go:embed sql/recognized.sql
var recognizedSQL string

//go:embed sql/rerank.sql
var rerankSQL string

//go:embed sql/sections.sql
var sectionsSQL string

//go:embed sql/sections_kinds.sql
var sectionsKindsSQL string

//go:embed sql/sizes.sql
var sizesSQL string

//go:embed sql/source_text.sql
var sourceTextSQL string

//go:embed sql/stale_recipe.sql
var staleRecipeSQL string

//go:embed sql/unchunked.sql
var unchunkedSQL string

//go:embed sql/coarse_prefix.sql
var coarsePrefixSQL string

//go:embed sql/unembedded.sql
var unembeddedSQL string

//go:embed sql/vault_row.sql
var vaultRowSQL string
