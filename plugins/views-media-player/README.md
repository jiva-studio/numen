# Media Player Plugin

Plays the recordings of a vault with their transcripts, and records from the microphone.

## Features

- **Recording tab**: `Player` at the top plays the recording through the window's one audio element, from the address the core's `RecordingService` gives; under it the transcript stands in an editor, one cue to a line with the moment each was said in the gutter. A time in the gutter, and a search hit, play from that moment.
- **Transcript**: read from `TranscriptService` (the cues of the recording's live text version), and an edit written back to it once the typing has been still: the plugin that serves it keeps the edit as a new version.
- **Menu**: Transcribe, Proofread transcript and Delete transcript come from the core's `TextMenuService`, which derives them from the enabled plugins and the recording's versions. While the window's tasks name the recording, the transcript is read again as it grows.
- **Microphone Recording**: a voice recorder with a live level meter, a timer, a preview and export to the vault.
