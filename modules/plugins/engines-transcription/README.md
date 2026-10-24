# Audio/Video Transcription Plugin

Speech-to-text audio and video transcription using native ONNX models.

## Features

- Transcribes speech audio into timestamped subtitle tracks.
- Serves `numen.v1.TranscriptionService`.
- Configurable transcription language.
- Activates on media files (`.mp3`, `.wav`, `.m4a`, `.mp4`) and `onCommand:transcribe`.
- Declares `onnxruntime` as a `runtime` dependency; the host provisions it and launches the plugin with `NUMEN_ONNX_RUNTIME_LIB`.
