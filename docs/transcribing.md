# How a recording is transcribed

What a model writes down from a recording, where it is kept, and how a search finds the second it was said. Transcription is the plugin `engines-transcription`: it keeps each transcript in its own folder of the vault's store, `.numen/md.numen.engines.transcription/`, and records it with the core as a text version of kind `transcript`, which is what the window shows and the index splits into chunks. The core keeps the record in its index database and reads the file itself.

The recording itself is never moved, copied or renamed. It is played from where it is stored, from the address the core's `MediaService` returns to the player.

## Asking for a transcript

A recording is transcribed when a person asks: **Transcribe** on the recording's menu and under **Create text** in the palette, the button where the transcript would be shown, or `numen-cli app:file.createText <vault> <file>`. All of them run the plugin's handler of the shared command `app:file.createText`, which the manifest declares over audio and video files and which waits until the transcript is written. The plugin runs on its own only with its setting `transcribeAutomatically`, through its `file.added` trigger, which transcribes every recording with no valid transcript, a transcript removed with Delete text included. **Delete text** (`app:version.delete`) is a command of the application: it lists every text version of the file, and the core deletes the one picked, its record and its file.

The run waits in the core's shared queue as a person's request, after any other work using the models and before work nobody requested, and is shown in the list of background tasks of the title bar as a task with the seconds transcribed out of the recording's total seconds. A run that cannot open the recording stays in that list with the reason.

## The files of one transcription

Every file is named by the hash of the recording's bytes, so a file renamed or moved keeps its transcript, and two copies of one recording share it.

| File                   | What it holds                                   |
| ---------------------- | ----------------------------------------------- |
| `<hash>.jsonl`         | the transcript, complete                        |
| `<hash>.partial.jsonl` | a run still going, or one that stopped part way |
| `<hash>.reached`       | how far a partial run got, and its size then    |

A recording asked for again whose transcript is complete is not transcribed again.

## What a transcript holds

The common format of every text version: JSON Lines, one segment to a line, `{"text", "heading"?, "selector"?}`. A cue is one segment, its selector the span of the recording in seconds (`t`, as in W3C Media Fragments):

```
{"text":"The first thing to say about heat is that it moves.\n","selector":{"t":[1.5,4.2]}}
{"text":"It moves one way, and that is the whole of the second law.\n","selector":{"t":[4.2,9.1]}}
```

The version's text is the cues' words, one to a line. A search hit in a recording includes the selector of its segment, and the player seeks to the second the words were said.

## A run

Speech segments are detected first, and the model receives one segment at a time. Cues are appended to the partial file as they are produced, and after each batch `<hash>.reached` records how many milliseconds have been reached and how many bytes the partial file held then. A batch that was not written completely extends the file beyond that size: the next run truncates the partial file to that size and repeats that batch.

The version is stored after every batch, so the core splits the transcript into chunks again and a search finds what has been transcribed while the rest of the recording is still being processed. Work waiting for the queue runs between two batches.

## An edit

The transcript under the player is edited in place. The player saves the edit through the plugin SDK, and the core writes it as a new version made from the version shown, in the player's own folder `.numen/md.numen.views.media.viewer/`; it is the one shown from then on. Each save is a new version. Deleting the edit from the recording's menu restores the version it was made from.

## Deleting a transcript

**Delete the text Audio/Video Transcription made** on the recording's menu removes the version's file and its record, and the recording is in the state it had before it was transcribed. A transcript stays shown and searchable while the plugin is disabled.

## Running without models

`numen -debug-mock-ml`, and `NUMEN_TEST_MOCK_ML=true` in the environment of a plugin, transcribe every recording with a mock model, for automated runs.
