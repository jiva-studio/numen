# Media Player Plugin

Plays the recordings of a vault with their transcripts, and records from the microphone.

## Features

- **Recording tab**: `Player` at the top plays the recording through the window's one audio element, from the address the core's `RecordingService` gives; under it the transcript stands in an editor, one cue to a line with the moment each was said in the gutter. A time in the gutter, and a search hit, play from that moment.
- **Transcript**: the cues of the recording's live text version, read through `ctx.texts` of the plugin SDK; an edit is saved through it once the typing has been still, and the core keeps it as a new version in this plugin's folder.
- **Menu**: Transcribe, Proofread transcript and Delete transcript come from the core's `TextMenuService`, which derives them from the enabled plugins and the recording's versions. While the window's tasks name the recording, the transcript is read again as it grows.
- **Microphone Recording**: a voice recorder with a live level meter, a timer, a preview and export to the vault.
