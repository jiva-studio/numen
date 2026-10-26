# Audio/Video Transcription Plugin

Transcribes the recordings a person asks for with native ONNX models, and gives each transcript to the vault as a timed text version.

## Features

- Declares a text input of `.wav`, `.mp3` and `.flac` recordings, so the core offers "Transcribe" on their menu. It runs only when a person asks.
- Serves `numen.v1.TextProviderService`: `CreateText` waits for the core's shared queue as a person's request, reports the run as a task, and puts the version after every batch of stretches heard, so the words are searchable while the rest is still being heard. A run stopped part way takes up where it stopped.
- Keeps the transcripts in its own folder of the vault's store, `.numen/md.numen.engines.transcription/`: `<hash of the recording>.vtt` whole and `<hash>.partial.vtt` while a run is under way.
- Serves `numen.v1.TranscriptService` to the player: `ReadTranscript` is the cues of the live version, whichever plugin made it; `WriteTranscript` keeps a person's edit as a version of this plugin made from the live one.
- Declares the `transcribe` command over its recordings, hidden from the palette: `numen-cli transcribe <vault> <file>` transcribes the file and waits for the transcript.
- `NUMEN_MOCK_ML=true` hears every recording with a mock model, for automated runs.
- A model setting: Parakeet TDT 0.6B v3 (`nvidia/parakeet-tdt-0.6b-v3`, 25 European languages, detected by the model) or v2 (`nvidia/parakeet-tdt-0.6b-v2`, English), each the int8 sherpa-onnx export of its model and declared as the dependencies the option provisions. Neither model can be given a language, so the language row shows the language of the model chosen: "Detected by the model" for v3, English for v2.
- Declares `onnxruntime` as a `runtime` dependency; the host provisions it and launches the plugin with `NUMEN_ONNX_RUNTIME_LIB`.
