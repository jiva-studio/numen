# How a recording is heard

What a model hears in a recording, where it is kept, and how a search lands on the second it was said. A transcript is written into the vault's own folder, under `.numen/asr/`, and the source is cut from it (see [ADR-0043](adr/0043-a-transcript-is-webvtt.md)).

The recording itself is never moved, copied or renamed. It plays where it lies.

## The files of one transcription

Every file is named by the hash of the recording's bytes, so a file renamed or moved keeps what was heard in it, and two copies of one recording share the one transcript.

| File | What it holds |
| --- | --- |
| `<hash>.vtt` | the transcript, complete |
| `<hash>.partial.vtt` | a run still going, or one that stopped part way |
| `<hash>.corrected.vtt` | the transcript as it now reads, once something has been put right |
| `<hash>.answer` | why there will never be a transcript |
| `<hash>.json` | what listened: the model, the segmenter and where they came from |

Only one of `.vtt`, `.partial.vtt` and `.answer` exists at a time. A recording with none of them has not been heard yet.

`.corrected.vtt` is a transcript that has been put right — by a proofreader, by a person editing it in the recording tab, or by both. It is WebVTT, under that format's own extension, so whatever opens the artifact opens it too, and the artifact is not rewritten: it stays what the model heard. A recording the vault holds a corrected transcript for is cut from that file, and deleting it gives back what was heard. What a proofreader is shown and what it may change is [Proofreading](proofreading.md).

## What the artifact holds

WebVTT, which is the format a player and a browser already read:

```
WEBVTT

NOTE heard 9100

00:00:01.500 --> 00:00:04.200
Вещий стон гласит печаль.

00:00:04.200 --> 00:00:09.100
Кони торопливы, чутко смотрят в темную даль.
```

A cue is one stretch of speech between two times. The words of the transcript are the cues' text, one to a line; the timings are not part of them, so a chunk cut from the transcript holds what was said and none of the bookkeeping around it.

## How a moment is named

Where a chunk is, is the time on the player: `01:23:45`. A search result about a recording carries that, and opening it turns the tab to the recording and sets the player to the second the words were said.

## A run

The speech is found first, and the model is given one stretch at a time. Cues are appended as they are heard, and after them a note saying how many milliseconds have been written down. **The note is what makes the cues in front of it count**: a batch that did not land whole is one no note claims, and the next run cuts back to the last note and does that batch again.

The source is cut after every batch, so a recording answers a search about what has been heard while the rest of it is still playing to the model. A tab open on the recording shows the words appearing.

A note is a comment in this format, so a partial file is still a file every other reader understands.

## Every ending is an answer

A recording is transcribed without anybody asking ([ADR-0044](adr/0044-a-recording-is-transcribed-without-being-asked.md)), so a run that comes to nothing has to say so. Otherwise the scan offers the same file again forever.

| Ending | What is written | Offered again |
| --- | --- | --- |
| heard | `<hash>.vtt` | no |
| nothing to hear | `<hash>.answer` — silence, or music | no |
| will not open | `<hash>.answer`, with the reason | no |
| somebody else holds it | nothing | yes |
| stopped part way | `<hash>.partial.vtt` | yes, from the note |

Deleting an answer is how a person asks for a recording to be tried again, and `numen-cli transcribe <vault> <file> --again` is how they ask without going into the folder. It throws away the transcript, the run that was going and the answer, and listens from the start — which is what a person who changed the model wants.

In the window, **Delete transcript** stands where the words of the recording do. It takes away the transcript, what a person put right, the record of what listened, the answer and the chunks cut from any of them, leaving a recording nothing has listened to. A run holding the recording keeps it: the words go once that run ends.

The queue takes what it can carry: a recording larger than `indexing.transcribe_under_mb` is left alone until somebody asks for it by name. A folder of albums is days of a machine, and nobody put them in a vault to be read.

## Which recordings

`.mp3`, `.wav` and `.flac`. The container is decoded here, in Go, and brought to 16 kHz mono, which is what the model takes.

## Settings

Under `indexing.transcription`. The models are fetched when a recording is first heard and kept in the platform's cache folder; nothing is downloaded until then. See [settings](settings.md).
