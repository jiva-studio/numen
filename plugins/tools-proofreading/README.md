# LLM Proofreading Plugin

Puts the text of a file right with a language model and gives the result to the vault as a text version made from the text it read.

## Features

- Declares a text input of timed and positioned text and serves `numen.v1.TextProviderService`: "Proofread" in a file's menu appears once the file has a live version in one of those formats.
- Reads the live version through the host, asks the model about its numbered lines (a transcript as speech, a recognition a page at a time), and keeps the corrected text in its folder of the vault's store, `.numen/md.numen.tools.proofreading/proofread-<hash>.json`. The places of the lines it read (times, pages and rectangles) stay with the corrected lines.
- Reports the corrected text as a version whose input is the version it read. A version made from a recognition is dead while the OCR plugin is off.
- Waits for its turn in the core's shared queue and shows its progress as a task.
- Declares the `proofread` command, which proofreads the live text of a file from `numen-cli`.
- Under `NUMEN_MOCK_ML=true` it asks no model: the mock proofreader adds `[proofread]` to the end of every line it is asked about.
