# Audio/Video Transcription

Transcribes `.wav`, `.mp3`, and `.flac` audio recordings locally using NVIDIA Parakeet TDT ONNX models and indexes time-aligned transcript cues.

<!-- more -->

## Models & Languages

- **Parakeet TDT 0.6B v3** *(default)*: Multilingual `sherpa-onnx` int8 model supporting 25 European languages with automatic language detection (~670 MB, downloaded automatically from Hugging Face Hub on first run).
- **Parakeet TDT 0.6B v2**: English-specialized `sherpa-onnx` int8 model (~660 MB, downloaded automatically from Hugging Face Hub on first run).

## How It Works

- **Incremental streaming**: Saves cues after each processed audio batch into `.numen/md.numen.engines.transcription/<hash>.jsonl`, making transcripts readable and searchable while transcription is still in progress and resuming cleanly if interrupted.
- **Timestamped cues**: Each transcript line carries its start and end time in seconds (`{"t":[start,end]}`), allowing the Media Player to jump to and highlight spoken passages.
