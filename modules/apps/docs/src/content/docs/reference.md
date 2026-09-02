---
title: Settings reference
description: Every key numen.json can hold, section by section, taken from the application itself.
---

Every key `numen.json` can hold. The pages under Settings say which of these are worth touching
and why; this one is here so that nothing is missing.

Each heading below is a section of the file, and the keys under it are written the way they are
written inside that section — `model.name` under `indexing.embedding` is
`indexing.embedding.model.name` if you write it out in full.

A key you leave out keeps its default, so a file naming one setting is a complete file. A key
numen does not know is carried through untouched. A row with no kind beside it is a group
holding the keys under it.

Two things are not settings and are not in here: which files count as notes, and what a vault's
own folder is called. Both are asked for on the command line, by
[`numen-cli`](/cli/#where-it-puts-things).

<!-- BEGIN AUTOGEN -->
| | | |
| --- | --- | --- |
| `v` | a number | the shape of the file. Nothing reads it yet, and it is written so that the day a section changes shape there is something to tell the two apart. |

### `appearance`

How the window is drawn.

| | | |
| --- | --- | --- |
| `interface_scale` | a number | how large the window is drawn: its chrome, its controls, the spacing between them and the type in them. |
| `text_scale` | a number | how large the text a person reads is set: a note, a book, an answer, the editor. |
| `mode` | text | which half of a colour pair the window takes: `system`, `light` or `dark`. |
| `theme` | text | the stylesheet the window wears, named by the shelf it came off and its filename: `preset:dracula` ships here, `mine:dracula` is the person's file. |
| `hang_parts_under_a_node` | yes or no | whether a node in the plex hangs the headings of its note under the box. |
| `parts_under_a_node` | a number | how many of those headings stand under a node at once, the rest being wound to. |

### `indexing`

How a vault is made searchable.

| | | |
| --- | --- | --- |
| `embedding` |  | which model turns text into vectors, and how it is reached. |
| `embedding.model` |  | what a vector is, and it is said once. A vector made while a vault is indexed is claimed again by a question, so the two are one model or the comparison between them means nothing. |
| `embedding.model.name` | text | what the model is called here. |
| `embedding.model.dimensions` | a number | how wide its vectors are. The coarse index is built for one width, and changing it builds that index again from the vectors held. |
| `embedding.model.max_tokens` | a number | where the model truncates what it is given. A window cut somewhere else is a window whose vector describes text it does not hold. |
| `embedding.model.pooling` | text | `mean` or `head`. Empty is `mean`. |
| `embedding.indexing` |  | makes the vectors a vault is searched by. `query` makes the vector a question is asked with, and taking nothing here is asking the way the vault was indexed. |
| `embedding.indexing.use` | text | `local` or `service`, and names which of the two sections below is the one in force. |
| `embedding.indexing.local` |  | how this machine reaches a model it runs. |
| `embedding.indexing.local.name` | text | a HuggingFace repository. |
| `embedding.indexing.local.dir` | text | holds the model and tokenizer.json. Empty means the download cache. |
| `embedding.indexing.local.file` | text | the model inside the repository or the directory. |
| `embedding.indexing.local.batch_texts` | a number | how many texts one forward pass carries. |
| `embedding.indexing.local.download` | yes or no | allows fetching the model when it is not on this machine. |
| `embedding.indexing.service` |  | how a hosted model is reached over HTTP. |
| `embedding.indexing.service.base_url` | text | points at anything speaking the /v1/embeddings request shape. |
| `embedding.indexing.service.name` | text | what that service calls the model. |
| `embedding.indexing.service.batch_characters` | a number | bounds one request by the characters of everything in it. |
| `embedding.indexing.service.key_env` | text | names the environment variable holding the key, for an installation that keeps it out of the file. |
| `embedding.query` |  | where a vector is made: on this machine, or by a service. |
| `embedding.query.use` | text | `local` or `service`, and names which of the two sections below is the one in force. |
| `embedding.query.local` |  | how this machine reaches a model it runs. |
| `embedding.query.local.name` | text | a HuggingFace repository. |
| `embedding.query.local.dir` | text | holds the model and tokenizer.json. Empty means the download cache. |
| `embedding.query.local.file` | text | the model inside the repository or the directory. |
| `embedding.query.local.batch_texts` | a number | how many texts one forward pass carries. |
| `embedding.query.local.download` | yes or no | allows fetching the model when it is not on this machine. |
| `embedding.query.service` |  | how a hosted model is reached over HTTP. |
| `embedding.query.service.base_url` | text | points at anything speaking the /v1/embeddings request shape. |
| `embedding.query.service.name` | text | what that service calls the model. |
| `embedding.query.service.batch_characters` | a number | bounds one request by the characters of everything in it. |
| `embedding.query.service.key_env` | text | names the environment variable holding the key, for an installation that keeps it out of the file. |
| `embedding.floor` | a number | the cosine similarity a passage reaches to be an answer, in the units the model in use measures in. |
| `recognition` |  | how a scanned document is read when a person asks for it. |
| `recognition.proofread` |  | names the profile a reading is put right at. Automatically there says whether a reading just made is put right without anybody asking. |
| `recognition.proofread.with` | text | the profile, by the name the profiles carry it under. |
| `recognition.proofread.automatically` | yes or no | whether a reading already written down is put right without anybody asking for it. |
| `proofreading` |  | what puts a reading right. `naming` no profile here is naming no proofreader, and a reading is used as it was read. |
| `proofreading.max_edit_distance` | a number | how far a correction may stand from the line as read and still be a correction: the Levenshtein distance between their letters, as a share of the longer of the two. |
| `proofreading.profiles` |  | the stations a reading is put right at, by the name a consumer asks for one under. |
| `transcription` |  | how a recording is listened to: which models hear it, where they came from, and how the speech in it is found. |
| `transcription.proofread` |  | names the profile a transcript is put right at. |
| `transcription.proofread.with` | text | the profile, by the name the profiles carry it under. |
| `transcription.proofread.automatically` | yes or no | whether a reading already written down is put right without anybody asking for it. |
| `transcribe_recordings` | yes or no | whether a recording the vault holds no transcript for is listened to without anybody asking. |
| `transcribe_under_mb` | a number | how large a recording may be and still be listened to without anybody asking, in megabytes. |

### `agent`

Which agent answers in the panel, and what it may reach.

| | | |
| --- | --- | --- |
| `use` | text | names the agent. Empty answers with none, and the panel says so. |
| `serve_tools` | yes or no | puts the tools on a port, which is how an agent a person runs themselves reaches this vault. |
| `claude` |  | code, reached by starting it and reading what it prints. |
| `claude.command` | a list of words | starts it: the command line's path, and anything it is started through. |
| `claude.model` | text | which of its models answers — `opus`, `sonnet`, or a full name. |
| `claude.max_steps` | a number | how many times it may go to the model before it is stopped. |
| `claude.reads_hooks_and_skills` | yes or no | lets it read what is configured for it on this machine: hooks, skills, standing instructions in CLAUDE. |

### `naming`

How a note's title and the name of its file are held together.

| | | |
| --- | --- | --- |
| `sync_title_and_filename` | yes or no | whether renaming either of the two brings the other into line. |

### `review`

What a day of review is, on this person's clock.

| | | |
| --- | --- | --- |
| `day_starts` | text | the hour a day of review begins at, on the clock on the wall, written as hours and minutes. |
<!-- END AUTOGEN -->
