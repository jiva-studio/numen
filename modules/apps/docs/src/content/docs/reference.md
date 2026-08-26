---
title: Every setting
description: The complete list of what numen.json holds, taken from the application itself.
---

Every key `numen.json` can hold, in the order the application declares them. The pages under
Settings say which of these are worth touching and why; this one is here so that nothing is
missing.

A key you leave out keeps its default, so a file naming one setting is a complete file. A key
numen does not know is carried through untouched.

A row with no kind beside it is a section holding the keys under it.

<!-- BEGIN AUTOGEN -->
| | | |
| --- | --- | --- |
| `appearance.interface_scale` | a number | how large the window is drawn: its chrome, its controls, the spacing between them and the type in them. |
| `appearance.text_scale` | a number | how large the text a person reads is set: a note, a book, an answer, the editor. |
| `appearance.mode` | text | which half of a colour pair the window takes: `system`, `light` or `dark`. |
| `appearance.theme` | text | the stylesheet the window wears, named by the shelf it came off and its filename: `preset:dracula` ships here, `mine:dracula` is the person's file. |
| `indexing.embedding.model` |  | what a vector is, and it is said once. A vector made while a vault is indexed is claimed again by a question, so the two are one model or the comparison between them means nothing. |
| `indexing.embedding.model.name` | text | what the model is called here. |
| `indexing.embedding.model.dimensions` | a number | how wide its vectors are. The coarse index is built for one width, and changing it builds that index again from the vectors held. |
| `indexing.embedding.model.max_tokens` | a number | where the model truncates what it is given. A window cut somewhere else is a window whose vector describes text it does not hold. |
| `indexing.embedding.model.pooling` | text | `mean` or `head`. Empty is `mean`. |
| `indexing.embedding.indexing` |  | makes the vectors a vault is searched by. `query` makes the vector a question is asked with, and taking nothing here is asking the way the vault was indexed. |
| `indexing.embedding.indexing.use` | text | `local` or `service`, and names which of the two sections below is the one in force. |
| `indexing.embedding.indexing.local` |  | how this machine reaches a model it runs. |
| `indexing.embedding.indexing.local.name` | text | a HuggingFace repository. |
| `indexing.embedding.indexing.local.dir` | text | holds the model and tokenizer.json. Empty means the download cache. |
| `indexing.embedding.indexing.local.file` | text | the model inside the repository or the directory. |
| `indexing.embedding.indexing.local.batch_texts` | a number | how many texts one forward pass carries. |
| `indexing.embedding.indexing.local.download` | yes or no | allows fetching the model when it is not on this machine. |
| `indexing.embedding.indexing.service` |  | how a hosted model is reached over HTTP. |
| `indexing.embedding.indexing.service.base_url` | text | points at anything speaking the /v1/embeddings request shape. |
| `indexing.embedding.indexing.service.name` | text | what that service calls the model. |
| `indexing.embedding.indexing.service.batch_characters` | a number | bounds one request by the characters of everything in it. |
| `indexing.embedding.indexing.service.key_env` | text | names the environment variable holding the key, for an installation that keeps it out of the file. |
| `indexing.embedding.query` |  | where a vector is made: on this machine, or by a service. |
| `indexing.embedding.query.use` | text | `local` or `service`, and names which of the two sections below is the one in force. |
| `indexing.embedding.query.local` |  | how this machine reaches a model it runs. |
| `indexing.embedding.query.local.name` | text | a HuggingFace repository. |
| `indexing.embedding.query.local.dir` | text | holds the model and tokenizer.json. Empty means the download cache. |
| `indexing.embedding.query.local.file` | text | the model inside the repository or the directory. |
| `indexing.embedding.query.local.batch_texts` | a number | how many texts one forward pass carries. |
| `indexing.embedding.query.local.download` | yes or no | allows fetching the model when it is not on this machine. |
| `indexing.embedding.query.service` |  | how a hosted model is reached over HTTP. |
| `indexing.embedding.query.service.base_url` | text | points at anything speaking the /v1/embeddings request shape. |
| `indexing.embedding.query.service.name` | text | what that service calls the model. |
| `indexing.embedding.query.service.batch_characters` | a number | bounds one request by the characters of everything in it. |
| `indexing.embedding.query.service.key_env` | text | names the environment variable holding the key, for an installation that keeps it out of the file. |
| `indexing.embedding.floor` | a number | the cosine similarity a passage reaches to be an answer, in the units the model in use measures in. |
| `indexing.recognition.runtime` | text | the ONNX shared library. Empty means the one beside the application, and then the one the platform holds. |
| `indexing.recognition.dir` | text | a folder holding the models. Empty means the folder beside the application, and then the download cache. |
| `indexing.recognition.download` | yes or no | allows fetching what is not on this machine. |
| `indexing.recognition.layout` |  | divides a page into its parts and puts them in reading order. |
| `indexing.recognition.layout.name` | text | where the model is fetched from, and `path` is a file on this machine. |
| `indexing.recognition.layout.path` | text | this model as a file on this machine. |
| `indexing.recognition.layout.labels` | a list of words | the parts the model knows, in the order of its class ids. |
| `indexing.recognition.layout.minimum` | a number | the score a part carries to be a part at all. |
| `indexing.recognition.layout.overlap` | a number | how much of two parts may be common before the second is taken to be the first found again. |
| `indexing.recognition.layout.margin` | a number | what a part is widened by before it is read, because the first letter of a line sits on the boundary the model drew. |
| `indexing.recognition.detect` |  | finds the lines one part holds. |
| `indexing.recognition.detect.name` | text | where this model is fetched from. |
| `indexing.recognition.detect.path` | text | this model as a file on this machine. |
| `indexing.recognition.detect.max_side` | a number | the longest side a part is read at. |
| `indexing.recognition.detect.expand` | a number | how many pixels a found line is widened by, in the image the detector reads: the part scaled so that its longest side is `max_side`. |
| `indexing.recognition.detect.minimum` | a number | the heat a pixel carries to be part of a line. |
| `indexing.recognition.recognise` |  | reads what a line says. |
| `indexing.recognition.recognise.name` | text | where this model is fetched from. |
| `indexing.recognition.recognise.path` | text | this model as a file on this machine. |
| `indexing.recognition.recognise.dict` | text | the model's characters, one to a line. Empty is the ordinary case: the model carries them, and asking it is the only way to be sure. |
| `indexing.recognition.recognise.classes` | a number | how many characters the model knows and two more. |
| `indexing.recognition.recognise.height` | a number | what a line is scaled to before it is read. |
| `indexing.recognition.recognise.sessions` | a number | how many lines are read at once. |
| `indexing.recognition.page` |  | how a page becomes an image, and how much of the machine one page may use. |
| `indexing.recognition.page.dpi` | a number | what a page is rendered at. |
| `indexing.recognition.page.threads` | a number | how many threads one model may use. |
| `indexing.recognition.regions` |  | says what the parts of a page are for. A part the model names that `body` does not carry is not read. |
| `indexing.recognition.regions.body` | a list of words | carry what the document says. |
| `indexing.recognition.regions.head` | a list of words | open a part of the document, outermost first: where a name stands in the list is how deep the part it opens sits. |
| `indexing.proofreading.use` | text | `service`, or empty for an installation that proofreads nothing. |
| `indexing.proofreading.service` |  | a hosted model reached over HTTP. |
| `indexing.proofreading.service.base_url` | text | points at anything speaking the /v1/chat/completions request shape. |
| `indexing.proofreading.service.batch_url` | text | a queue the pages are left in and collected from later, at half the price. |
| `indexing.proofreading.service.name` | text | which model corrects a reading. It stands beside every line it corrected. |
| `indexing.proofreading.service.key_env` | text | names the environment variable holding the key, for an installation that keeps it out of the file. |
| `indexing.proofreading.service.pages_at_once` | a number | how many pages one request carries. |
| `indexing.proofreading.service.letters_apart` | a number | how far a correction may move a line's letters and still be a correction, as a share of the longer of the two. |
| `agent.use` | text | names the agent. Empty answers with none, and the panel says so. |
| `agent.claude` |  | code, reached by starting it and reading what it prints. |
| `agent.claude.command` | a list of words | starts it: the command line's path, and anything it is started through. |
| `agent.claude.model` | text | which of its models answers — `opus`, `sonnet`, or a full name. |
| `agent.claude.max_steps` | a number | how many times it may go to the model before it is stopped. |
| `agent.claude.reads_hooks_and_skills` | yes or no | lets it read what is configured for it on this machine: hooks, skills, standing instructions in CLAUDE. |
<!-- END AUTOGEN -->
