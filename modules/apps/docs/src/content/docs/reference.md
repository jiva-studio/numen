---
title: Settings reference
description: Every key numen.json can hold, section by section, taken from the application itself.
---

Every key `numen.json` can hold. The pages under Settings say which of these are worth touching and why; this one is here so that nothing is missing.

Each heading below is a section of the file, and the keys under it are written the way they are written inside that section — `model.name` under `indexing.embedding` is `indexing.embedding.model.name` if you write it out in full.

A key you leave out keeps its default, so a file naming one setting is a complete file. A key numen does not know is carried through untouched. A row with no kind beside it is a group holding the keys under it.

Two things are not settings and are not in here: which files count as notes, and what a vault's own folder is called. Both are asked for on the command line, by [`numen-cli`](/cli/#where-it-puts-things).

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
| `recognition.runtime` | text | the ONNX shared library. Empty means the one beside the application, and then the one the platform holds. |
| `recognition.dir` | text | a folder holding the models. Empty means the folder beside the application, and then the download cache. |
| `recognition.download` | yes or no | allows fetching what is not on this machine. |
| `recognition.layout` |  | divides a page into its parts and puts them in reading order. |
| `recognition.layout.name` | text | where the model is fetched from, and `path` is a file on this machine. |
| `recognition.layout.path` | text | this model as a file on this machine. |
| `recognition.layout.labels` | a list of words | the parts the model knows, in the order of its class ids. |
| `recognition.layout.minimum` | a number | the score a part carries to be a part at all. |
| `recognition.layout.overlap` | a number | how much of two parts may be common before the second is taken to be the first found again. |
| `recognition.layout.margin` | a number | what a part is widened by before it is read, because the first letter of a line sits on the boundary the model drew. |
| `recognition.detect` |  | finds the lines one part holds. |
| `recognition.detect.name` | text | where this model is fetched from. |
| `recognition.detect.path` | text | this model as a file on this machine. |
| `recognition.detect.max_side` | a number | the longest side a part is read at. |
| `recognition.detect.expand` | a number | how many pixels a found line is widened by, in the image the detector reads: the part scaled so that its longest side is `max_side`. |
| `recognition.detect.minimum` | a number | the heat a pixel carries to be part of a line. |
| `recognition.recognise` |  | reads what a line says: which model, at what size a page and a line reach it, and how much of the machine it takes. |
| `recognition.recognise.name` | text | where this model is fetched from. |
| `recognition.recognise.path` | text | this model as a file on this machine. |
| `recognition.recognise.dict` | text | the model's characters, one to a line. Empty is the ordinary case: the model carries them, and asking it is the only way to be sure. |
| `recognition.recognise.classes` | a number | how many characters the model knows and two more. |
| `recognition.recognise.dpi` | a number | what a page is rendered at. |
| `recognition.recognise.height` | a number | what a line is scaled to before it is read. |
| `recognition.recognise.sessions` | a number | how many lines are read at once. |
| `recognition.recognise.threads` | a number | how many threads one model may use. |
| `recognition.regions` |  | says what the parts of a page are for. A part the model names that `body` does not carry is not read. |
| `recognition.regions.body` | a list of words | carry what the document says. |
| `recognition.regions.head` | a list of words | open a part of the document, outermost first: where a name stands in the list is how deep the part it opens sits. |
| `recognition.proofread` |  | names the profile a reading is put right at. Automatically there says whether a reading just made is put right without anybody asking. |
| `recognition.proofread.with` | text | the profile, by the name the profiles carry it under. |
| `recognition.proofread.automatically` | yes or no | whether a reading already written down is put right without anybody asking for it. |
| `proofreading` |  | what puts a reading right. Naming no profile here is naming no proofreader, and a reading is used as it was read. |
| `proofreading.max_edit_distance` | a number | how far a correction may stand from the line as read and still be a correction: the Levenshtein distance between their letters, as a share of the longer of the two. |
| `proofreading.profiles` |  | the places a reading is put right at, by the name a consumer asks for one under. |
| `transcription` |  | how a recording is listened to: which models hear it, where they came from, and how the speech in it is found. |
| `transcription.runtime` | text | the ONNX shared library. Empty means the one beside the application, and then the one the platform holds. |
| `transcription.dir` | text | a folder holding the models. Empty means the folder beside the application, and then the download cache. |
| `transcription.download` | yes or no | allows fetching what is not on this machine. |
| `transcription.threads` | a number | how many threads one model may use. |
| `transcription.model` |  | turns speech into words. It is exported as three graphs and the pieces they write, and all four are one model: three graphs from two exports answer with nothing anybody can read. |
| `transcription.model.name` | text | what this model is called in the record kept beside a text. |
| `transcription.model.from` | text | the folder the four files are fetched from. |
| `transcription.model.encoder` | text | `encoder`, `decoder`, `joiner` and `tokens` are the files on this machine. |
| `transcription.model.decoder` | text |  |
| `transcription.model.joiner` | text |  |
| `transcription.model.tokens` | text |  |
| `transcription.speech` |  | finds where in a recording somebody is speaking. |
| `transcription.speech.name` | text | what this segmenter is called in the record kept beside a text. |
| `transcription.speech.from` | text | where the model is fetched from, and `path` is a file on this machine. |
| `transcription.speech.path` | text |  |
| `transcription.speech.threshold` | a number | how sure the model has to be that a window carries speech. |
| `transcription.speech.silence` | a number | how much quiet, in milliseconds, closes a stretch of speech. |
| `transcription.speech.pad` | a number | how many milliseconds are kept on each side of a stretch, so that the first and last sound of a word are inside it. |
| `transcription.speech.longest` | a number | how many milliseconds one stretch may run to. |
| `transcription.speech.shortest` | a number | how many milliseconds a stretch carries to be a stretch at all. |
| `transcription.speech.least` | a number | how many milliseconds a stretch runs to before it stands as a line of its own. |
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
| `claude.model` | text | which of its models answers — `opus`, `sonnet`, `haiku`, or a full name. |
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
