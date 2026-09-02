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

### `indexing.embedding`

Which model turns text into vectors, and how it is reached.

| | | |
| --- | --- | --- |
| `model` |  | what a vector is, and it is said once. A vector made while a vault is indexed is claimed again by a question, so the two are one model or the comparison between them means nothing. |
| `model.name` | text | what the model is called here. |
| `model.dimensions` | a number | how wide its vectors are. The coarse index is built for one width, and changing it builds that index again from the vectors held. |
| `model.max_tokens` | a number | where the model truncates what it is given. A window cut somewhere else is a window whose vector describes text it does not hold. |
| `model.pooling` | text | `mean` or `head`. Empty is `mean`. |
| `indexing` |  | makes the vectors a vault is searched by. `query` makes the vector a question is asked with, and taking nothing here is asking the way the vault was indexed. |
| `indexing.use` | text | `local` or `service`, and names which of the two sections below is the one in force. |
| `indexing.local` |  | how this machine reaches a model it runs. |
| `indexing.local.name` | text | a HuggingFace repository. |
| `indexing.local.dir` | text | holds the model and tokenizer.json. Empty means the download cache. |
| `indexing.local.file` | text | the model inside the repository or the directory. |
| `indexing.local.batch_texts` | a number | how many texts one forward pass carries. |
| `indexing.local.download` | yes or no | allows fetching the model when it is not on this machine. |
| `indexing.service` |  | how a hosted model is reached over HTTP. |
| `indexing.service.base_url` | text | points at anything speaking the /v1/embeddings request shape. |
| `indexing.service.name` | text | what that service calls the model. |
| `indexing.service.batch_characters` | a number | bounds one request by the characters of everything in it. |
| `indexing.service.key_env` | text | names the environment variable holding the key, for an installation that keeps it out of the file. |
| `query` |  | where a vector is made: on this machine, or by a service. |
| `query.use` | text | `local` or `service`, and names which of the two sections below is the one in force. |
| `query.local` |  | how this machine reaches a model it runs. |
| `query.local.name` | text | a HuggingFace repository. |
| `query.local.dir` | text | holds the model and tokenizer.json. Empty means the download cache. |
| `query.local.file` | text | the model inside the repository or the directory. |
| `query.local.batch_texts` | a number | how many texts one forward pass carries. |
| `query.local.download` | yes or no | allows fetching the model when it is not on this machine. |
| `query.service` |  | how a hosted model is reached over HTTP. |
| `query.service.base_url` | text | points at anything speaking the /v1/embeddings request shape. |
| `query.service.name` | text | what that service calls the model. |
| `query.service.batch_characters` | a number | bounds one request by the characters of everything in it. |
| `query.service.key_env` | text | names the environment variable holding the key, for an installation that keeps it out of the file. |
| `floor` | a number | the cosine similarity a passage reaches to be an answer, in the units the model in use measures in. |

### `indexing.recognition`

How a scanned document is read when a person asks for it.

| | | |
| --- | --- | --- |
| `runtime` | text | the ONNX shared library. Empty means the one beside the application, and then the one the platform holds. |
| `dir` | text | a folder holding the models. Empty means the folder beside the application, and then the download cache. |
| `download` | yes or no | allows fetching what is not on this machine. |
| `layout` |  | divides a page into its parts and puts them in reading order. |
| `layout.name` | text | where the model is fetched from, and `path` is a file on this machine. |
| `layout.path` | text | this model as a file on this machine. |
| `layout.labels` | a list of words | the parts the model knows, in the order of its class ids. |
| `layout.minimum` | a number | the score a part carries to be a part at all. |
| `layout.overlap` | a number | how much of two parts may be common before the second is taken to be the first found again. |
| `layout.margin` | a number | what a part is widened by before it is read, because the first letter of a line sits on the boundary the model drew. |
| `detect` |  | finds the lines one part holds. |
| `detect.name` | text | where this model is fetched from. |
| `detect.path` | text | this model as a file on this machine. |
| `detect.max_side` | a number | the longest side a part is read at. |
| `detect.expand` | a number | how many pixels a found line is widened by, in the image the detector reads: the part scaled so that its longest side is `max_side`. |
| `detect.minimum` | a number | the heat a pixel carries to be part of a line. |
| `recognise` |  | reads what a line says. |
| `recognise.name` | text | where this model is fetched from. |
| `recognise.path` | text | this model as a file on this machine. |
| `recognise.dict` | text | the model's characters, one to a line. Empty is the ordinary case: the model carries them, and asking it is the only way to be sure. |
| `recognise.classes` | a number | how many characters the model knows and two more. |
| `recognise.height` | a number | what a line is scaled to before it is read. |
| `recognise.sessions` | a number | how many lines are read at once. |
| `page` |  | how a page becomes an image, and how much of the machine one page may use. |
| `page.dpi` | a number | what a page is rendered at. |
| `page.threads` | a number | how many threads one model may use. |
| `regions` |  | says what the parts of a page are for. A part the model names that `body` does not carry is not read. |
| `regions.body` | a list of words | carry what the document says. |
| `regions.head` | a list of words | open a part of the document, outermost first: where a name stands in the list is how deep the part it opens sits. |
| `proofread.with` | text | which profile under `indexing.proofreading.profiles` puts a reading right. Empty proofreads nothing. A name no profile carries is an error at startup. |
| `proofread.automatically` | yes or no | whether a reading is put right as soon as it is read. Off leaves it to the hand. |

### `indexing.proofreading`

What puts a reading right. It holds one threshold and the profiles, and naming no profile is naming no proofreader: a reading is used as it was read.

| | | |
| --- | --- | --- |
| `max_edit_distance` | a number | how far a correction may move a line's letters and still be a correction: the Levenshtein distance between them, as a share of the longer of the two. 0.30. A correction standing further apart is dropped and that line is left as it was. It stands above the profiles because it is one threshold for the installation: how far a correction may move says nothing about what it was asked for through. |
| `profiles` |  | a map of name to profile. The name is what a consumer says under `proofread.with`, and it is your own word. |
| `profiles.<name>.use` | text | `service` or `agent`. Every key below stands at the profile's own level, and one `use` does not apply to is ignored. |
| `profiles.<name>.base_url` | text | `service`: points at anything speaking the /v1/chat/completions request shape. |
| `profiles.<name>.batch_url` | text | `service`: a queue the batches are left in and collected from later, at half the price. Empty asks a batch at a time and waits. |
| `profiles.<name>.name` | text | `service`: which model corrects a reading. It stands beside every line it corrected. |
| `profiles.<name>.key_env` | text | `service`: names the environment variable holding the key, for an installation that keeps it out of the file. |
| `profiles.<name>.key` | text | `service`: the key, where you write it into the file. It is never written back. |
| `profiles.<name>.command` | a list of words | `agent`: what starts the command line, and anything it is started through. Empty runs `claude` from the path. |
| `profiles.<name>.model` | text | `agent`: which of its models answers — `opus`, `sonnet`, `haiku`, or a full name. |
| `profiles.<name>.batch_size` | a number | how many lines one request carries. 40 at a service, 60 at the command line. |
| `profiles.<name>.overlap` | a number | how many lines neighbouring batches share, so a phrase torn at a batch boundary is still seen whole by one of them. |
| `profiles.<name>.in_flight` | a number | how many batches are being asked about at any moment. 4 at a service, 2 at the command line. |

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
