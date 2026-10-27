# How a recording is transcribed

What a model writes down from a recording, where it is kept, and how a search lands on the second it was said. Transcription is the plugin `engines-transcription`: it keeps each transcript in its own folder of the vault's store, `.numen/md.numen.engines.transcription/`, and records it with the core as a text version of kind `transcript`, which is what the window shows and the index cuts. The core keeps the record in its index database and reads the file itself.

The recording itself is never moved, copied or renamed. It plays where it lies, from the address the core's `RecordingService` gives the player.

## Asking for a transcript

A recording is transcribed when a person asks: **Recognise the text with Audio/Video Transcription** on the recording's menu and in the palette, the button where the transcript would stand, or `numen-cli transcribe <vault> <file>`. The menu entry is the core's: the plugin declares that it reads `.wav`, `.mp3` and `.flac` files, and the menu of every such file offers it. The command is the plugin's: it declares `transcribe` over those files, and the command waits until the transcript is written down. The plugin does not run on its own.

The run waits for the core's shared queue as a person's request, behind any other work holding the models and ahead of work nobody asked for, and is shown in the status bar as a task with the seconds heard of the seconds the recording runs. A run that cannot open the recording stays there with the reason.

## The files of one transcription

Every file is named by the hash of the recording's bytes, so a file renamed or moved keeps its transcript, and two copies of one recording share it.

| File                   | What it holds                                   |
| ---------------------- | ----------------------------------------------- |
| `<hash>.jsonl`         | the transcript, complete                        |
| `<hash>.partial.jsonl` | a run still going, or one that stopped part way |
| `<hash>.reached`       | how far a partial run got, and its size then    |

A recording asked for again whose transcript is complete is not heard again.

## What a transcript holds

The common format of every text version: JSON Lines, one segment to a line, `{"text", "heading"?, "selector"?}`. A cue is one segment, its selector the span of the recording in seconds (`t`, as in W3C Media Fragments):

```
{"text":"The first thing to say about heat is that it moves.\n","selector":{"t":[1.5,4.2]}}
{"text":"It moves one way, and that is the whole of the second law.\n","selector":{"t":[4.2,9.1]}}
```

The version's text is the cues' words, one to a line. A search hit in a recording carries the selector of its segment, and the player goes to the second the words were said.

## A run

The speech is found first, and the model is given one stretch at a time. Cues are appended to the partial file as they are written down, and after each batch `<hash>.reached` records how many milliseconds have been reached and how many bytes the partial file held then. A batch that did not land whole is past that size: the next run cuts the partial file back to it and does that batch again.

The version is put after every batch, so the core cuts the recording again and a search finds what has been transcribed while the rest of it is still being heard. Work waiting for the queue runs between two batches.

## An edit

The transcript under the player is edited in place. The player saves the edit through the plugin SDK, and the core writes it as a new version made from the version shown, in the player's own folder `.numen/md.numen.views.media.player/`; it is the one shown from then on. Each save is a new version. Deleting the edit from the recording's menu gives back the version it was made from.

## Deleting a transcript

**Delete the text Audio/Video Transcription made** on the recording's menu removes the version's file and its record, and the recording is as it was before anything heard it. A transcript stays shown and searchable while the plugin is disabled.

## Running without models

`numen -debug-mock-ml`, and `NUMEN_MOCK_ML=true` in the environment of a plugin, hear every recording with a mock model, for automated runs.
