# Importing an address

A person pastes an address into the palette and gets a note: the video plays in its tab, the words said in it stand under the player, and a search over the vault answers about them.

The note is a note. Its body is prose the person writes, and what was fetched stands beside it in the vault's own folder — see [A link is a note that carries an address](adr/0038-a-link-is-a-note-that-carries-an-address.md).

## The gesture

**Import an address** in the palette asks for one on a step of its own. The step offers a row for somewhere a browser would go and for nothing else; the vault reads the address again and is what refuses one nothing can be fetched from.

What it makes is a note carrying `type: link` and `url`. What is at the address is fetched as the note is made, and the note takes the name what is there calls itself — a note the person has since named keeps the name they gave it.

At the command line the same run is `numen-cli import <vault> <note>`, and `--again` asks a site for its words afresh.

An agent asks for one with `note_import`, which makes the note and fetches what is at the address in one call, and takes `copy` for the video itself. A build that reaches no address serves the tool nowhere — see [Agents](agents.md).

## What is fetched

| At the address | What is kept | Which it is |
| --- | --- | --- |
| a video somebody published words for | those words, as WebVTT | a transcript |
| a video nobody published words for | that answer, so the address is not asked again | |
| anything else | the prose the page is written around | an article |

A transcript is a transcript whoever wrote it down: the words a site published with a video and the words a model here heard in a recording are one kind, read and put right by one editor, and what separates them is only the producer. An article is neither — it carries no times, and no places on pages either, which is what separates it from a reading off a scan.

Every file is named by the hash of the address, in the one form every spelling of it reaches, so two notes pointing at one video share what was fetched and typing in either of them keeps it.

A folder is a kind, and the producer stands in the file's name where a kind has more than one:

| File | What it holds |
| --- | --- |
| `transcript/<hash>.captions.vtt` | the words a site published with a video |
| `transcript/<hash>.asr.vtt` | the words a model here heard in it |
| `transcript/<hash>.captions.answer` | why there are no words |
| `transcript/<hash>.captions.json` | what fetched it: the address, the title, the tool |
| `article/<hash>.txt` | the prose of a page |
| `copy/<hash>.mp4` | the video itself, where a copy was asked for |

One language is asked for and not a list: a site that publishes a machine translation into every language it knows answers a request for the lot by refusing it. What a person published is preferred over what a machine wrote; among those, the languages `importing.captions` names, and then the language the video was spoken in.

## What a search answers

A link note is cut over its prose and what was fetched, as one text. A search about either is a search about that note, and the passage shows which of the two it fell in — see [A link note's text is its prose and what was fetched](adr/0040-a-link-notes-text-is-its-prose-and-what-was-fetched.md).

## The tab

The video plays in a frame, from the hosts the window may frame and no others. The words stand under it, and choosing one plays the video from the moment it was said.

An address nothing plays is the address itself over the prose, and what was fetched from it is searched like any other.

## A copy

**Download a copy of this video** fetches the video itself onto this disk, and the tab plays that instead of the frame: it plays offline, and nothing of the site it came from is loaded to play it. At the command line it is `numen-cli import <vault> <note> --copy`.

It is asked for by hand. An hour of video on somebody's disk is not what pasting an address asks for, and `importing.copy_under_mb` is what a copy may run to at all.

The copy is kept in the vault's own folder, named by the address like everything else fetched for it, and it is played from there — the vault's own reader is refused that folder, so what serves it is the same socket, reading the store. `importing.copies_to_vault` keeps it beside the note instead, under the note's own name, where the person sees it in their folder and every other program on the machine can play it. Taking the copy away leaves the note pointing where it pointed, and the tab frames the address again.

## What goes when the note goes

Deleting a link note deletes what was fetched for it: the words, the prose, the copy. It is named by the address and shared by every note carrying it, so it goes when the last of them does, and two notes on one video keep it while either stands. The walk that notices the file is gone is what notices this.

## Without being asked

`importing.fetch_unasked` reaches the address of every link note nothing has been fetched for, as a walk of the vault finds it. It is off: reaching off the machine is a gesture, and a note somebody wrote in another editor is not one. Turned on, a note written elsewhere has what is at its address by the time it is opened, and an address that will not answer is that note's trouble and leaves the walk standing.

## The tools

Each source of what is published at an address is a provider of its own, and which one answers is which one supports that address.

A page is fetched in this process — an ordinary request, and the prose found by a library — so every machine reaches one. `yt-dlp` reaches what a site publishes as a video, on the sites that tool knows, and `ffmpeg` brings its sound to what a transcriber opens. Neither is shipped: each is a program the machine already has, named by a setting holding a command and what it is started through. A machine without `yt-dlp` imports pages and says what is missing when a video is asked for.

A site with an API of its own is another provider and nothing more: it says which addresses it supports, and it is written as the whole of that site's answer.

A site that refuses an unattended request says so in its own words, and those are what the person is shown. What answers such a refusal — cookies from a browser, a token — is handed to every run through `importing.yt_dlp.arguments`.

## Settings

Under `importing`. See [Settings](settings.md).
