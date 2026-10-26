# How a recording is transcribed

What a model writes down from a recording, where it is kept, and how a search lands on the second it was said. Transcription is the plugin `engines-transcription`: it keeps each transcript in its own folder of the vault's store, `.numen/md.numen.engines.transcription/`, and gives it to the core as a timed text version of the recording, which is what the window shows and the index cuts.

The recording itself is never moved, copied or renamed. It plays where it lies, from the address the core's `RecordingService` gives the player.

## Asking for a transcript

A recording is transcribed when a person asks: **Recognise the text with Audio/Video Transcription** on the recording's menu and in the palette, the button where the transcript would stand, or `numen-cli transcribe <vault> <file>`. The menu entry is the core's: the plugin declares that it reads `.wav`, `.mp3` and `.flac` files, and the menu of every such file offers it. The command is the plugin's: it declares `transcribe` over those files, and the command waits until the transcript is written down. The plugin does not run on its own.

The run waits for the core's shared queue as a person's request, behind any other work holding the models and ahead of work nobody asked for, and is shown in the status bar as a task with the seconds heard of the seconds the recording runs. A run that cannot open the recording stays there with the reason.

## The files of one transcription

Every file is named by the hash of the recording's bytes, so a file renamed or moved keeps its transcript, and two copies of one recording share it.

| File | What it holds |
| --- | --- |
| `<hash>.vtt` | the transcript, complete |
| `<hash>.partial.vtt` | a run still going, or one that stopped part way |
| `edit-<id>.vtt` | a person's edit of a version |

A recording asked for again whose transcript is complete is not heard again.

## What a transcript holds

WebVTT, which is the format a player and a browser already read:

```
WEBVTT

00:00:01.500 --> 00:00:04.200
The first thing to say about heat is that it moves.

00:00:04.200 --> 00:00:09.100
It moves one way, and that is the whole of the second law.
```

A cue is one stretch of speech between two times. The version's text is the cues' words, one to a line, and each part of it stands at the moments of its cue, so a search result about a recording names the time on the player and opening it sets the player to the second the words were said.

## A run

The speech is found first, and the model is given one stretch at a time. Cues are appended to the partial file as they are written down, and after them a note saying how many milliseconds have been reached:

```
NOTE heard 9100
```

**The note is what makes the cues in front of it count**: a batch that did not land whole is one no note claims, and the next run cuts back to the last note and does that batch again.

The version is put after every batch, so the core cuts the recording again and a search finds what has been transcribed while the rest of it is still being heard. Work waiting for the queue runs between two batches.

## An edit

The transcript under the player is edited in place. An edit is a new version of the plugin, made from the version shown, and it is the one shown from then on; editing it again rewrites it. Deleting the edit from the recording's menu gives back the version it was made from.

## Deleting a transcript

**Delete the text Audio/Video Transcription made** on the recording's menu asks the plugin to remove the version and its file, and the recording is as it was before anything heard it.

## Running without models

`numen -debug-mock-ml`, and `NUMEN_MOCK_ML=true` in the environment of a plugin, hear every recording with a mock model, for automated runs.
