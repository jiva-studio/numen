# Glossary

The ubiquitous language of the project: domain terms, storage entities, protocol contracts, and specialized vocabulary. Only terms that carry specific project semantics or cross boundaries are defined here. Standard industry concepts (tree, window, file, vector, cache, database, agent, theme, title) are omitted.

## The vault and notes

| Term | What it is | Never called |
| --- | --- | --- |
| vault | A directory with an assigned ULID identity, containing notes and assets watched for changes. | |
| note | A markdown file inside a vault with structured frontmatter, typed links, and optional card blocks. | |
| asset | A non-markdown file inside a vault (images, PDFs, media), addressed via the `asset:` URI scheme. | |
| naming | The title resolution policy: whether `frontmatter` or `filename` takes precedence. | |
| link | A typed relationship to another note or asset, formatted as a URI (e.g. `note:<id>`, `asset:<path>`). | edge, address |
| role | The structural classification of a link: `parent`, `child`, `jump`, `ref`, `attachment`. | seat |
| identifier | The ULID assigned to a vault or note. | |
| stale | A write conflict state (`port.ErrStale`) occurring when a file on disk changed after the client read it. | overtaken |
| error code | The closed enum (`ErrorCode`) a successful call answers with when its answer is no: a name taken, a file that moved past the caller, a path the vault does not hold. | refusal, trouble |

## The index and search

| Term | What it is | Never called |
| --- | --- | --- |
| source | An indexable resource: `note`, `book` (EPUB), `document` (PDF), `recording`, or `url`. | |
| artifact | Extracted or generated data that cannot be reproduced deterministically for free (e.g. `ocr`, `transcript`, `article`). | cache |
| producer | The model or engine that generated source text or an artifact (e.g. `asr`, OCR engine name). | reader, extractor |
| recognition | The execution run of an OCR model over a document's pages to produce recognized text and coordinates. | reading |
| text layer | Deterministic text extracted directly from a PDF without OCR. | |
| spread | The layout unit of a reflowing EPUB book (one or two columns turned together). | page |
| span | A character or byte range (`{ from, to }`) indicating text offsets within a file or source. | stretch |
| passage | A search result match containing surrounding text excerpt and source coordinates. | chunk |
| highlight | Normalized page bounding boxes covering a text span on a document page. | |
| search mode | The retrieval ranking method: `Lexical`, `Dense`, `ByName`, or `Hybrid`. | |
| recipe | Deterministic indexing parameters (chunk sizes, embedding model, dimensionality). | |
| presence | Local availability status of model files: `present`, `not fetched`, `nothing to fetch`. | |

## Cards and review

| Term | What it is | Never called |
| --- | --- | --- |
| stencil | A note defining the field schema and face templates for flashcards. | note type, model |
| face | A layout template of a stencil specifying front and back sides with `{{Field}}` placeholders. | side |
| card | A flashcard entry generated from a stencil, carrying populated field values and a card mark. | note |
| card face | A single directional challenge of a card (e.g. front to back), scheduled independently via FSRS. | |
| mark (of a card) | A 10-character persistent ID (`^...`) at the end of a card heading that tracks the card across moves. | |
| deck | A note whose body consists of card sections and flashcards. | |
| preset | A note defining FSRS scheduling configuration (target retention, daily limits, review curve). | |
| budget unit | The accounting unit for daily review limits: `cards` (charged once per day) or `shows` (charged on every repetition). | |
| curve | Retention and workload projection computed over a preset's parameters. | graph, chart |

## Recordings and transcription

| Term | What it is | Never called |
| --- | --- | --- |
| recording | An audio or video file in a vault whose text is extracted via speech transcription. | |
| transcript | A WebVTT artifact containing timestamped speech cue segments. | spoken |
| cue | A single timestamped segment of speech in a transcript (`transcript.Cue`). | caption, segment |
| transcription | The execution run of a speech recognition model over a recording to produce a transcript. | |
| proofreading | LLM-assisted post-processing run that corrects OCR text or transcripts. | |
| profile | Named model or service configuration used for proofreading runs. | preset |

## Plex and navigation

| Term | What it is | Never called |
| --- | --- | --- |
| plex | The focused interactive graph view visualizing a note's neighbourhood. | |
| neighbourhood | A central note and all directly connected neighbour notes. | |
| focus | The central note around which a neighbourhood graph is rendered. | |
| node id | An ephemeral identifier assigned to a node for canvas layout tracking. | ticket |
| seat | Relative topological position of a node to the focus: `parent`, `child`, `jump`, `sibling`. | role |
| tab kind | An extension contract (`TabKind`) declaring lifecycle, title resolution, and view rendering for a tab type. | plugin, view |
| invocation | Execution context of a command: active note, vault, selection, and typed argument. | |
