# A transcript is WebVTT

- **Status:** Accepted
- **Date:** 2026-09-01
- **Applies to:** `modules/libs/core`, `modules/apps/desktop`
- **Related:** [A book's text is a cache or an artifact](0015-a-books-text-is-a-cache-or-an-artifact.md), [A passage is a range of bytes](0016-a-passage-is-a-range-of-bytes.md), [A recording is a source of its own](0030-a-recording-is-a-source-of-its-own.md), [A recording is transcribed without being asked](0032-a-recording-is-transcribed-without-being-asked.md)

## Context

What a model heard in a recording is an artifact: no machine here makes it again, so it is written into the service folder and the source is cut from it afterwards.

It is words with times attached. A chunk is a range of bytes in the words, and a search hit has to be played from the moment those words were said.

## Decision

### The artifact is a WebVTT file

```
WEBVTT

NOTE heard 9100

00:00:01.500 --> 00:00:04.200
what was said

00:00:04.200 --> 00:00:09.100
what was said next
```

The unit is a cue, which is what the format calls one stretch of text between two times.

The format is somebody else's, and that is the whole of its appeal. It opens in a player, it renders against the recording in a browser without anything being written to draw it, and a person who takes the file elsewhere can use it there.

### The times are not the words

Reading a transcript gives the words and the cues separately. An offset in the words is an offset in the speech, so a chunk holds what was said and none of the bookkeeping around it, and a cue says which run of bytes was spoken when.

Where a chunk is, is the time on the player. That is the one position a person can act on.

### A run says how far it got in a `NOTE`

A note is a comment in this format, so a file carrying one is still a file every other reader understands. It is appended after the cues it claims, and a batch no note claims is one that did not land whole.

## Consequences

- A transcript in a vault is worth something without this application.
- The words of a recording cost a parse of the format wherever they are read.
- A cue is as long as the stretch of speech a segmenter cut, so the smallest
moment a search can be played from is that stretch.
- A transcript is not a text layer: a file that is already WebVTT in a vault is
a file with no text of its own, like any other.

## Alternatives considered

**A format of this repository's own**, with a mark before each stretch as a scan's artifact marks its pages. Rejected: it would be a private format for something the world has a public one for, and nothing outside this application could open it.

**JSON, as a speech service answers.** Rejected: a person opening the file finds a data structure, and the words have to be assembled out of fields before anything can be read.

**The times beside the words, in a second file.** Rejected: two files written by one run can be torn apart by a run that dies between them, and the words alone would be a transcript that cannot be played.
