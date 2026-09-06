# Importing an address

A person pastes an address into the palette and gets a note: the video plays in its tab, the words said in it stand under the player, and a search over the vault answers about them.

The note is a note. Its body is prose the person writes, and what was fetched stands beside it in the vault's own folder — see [A link is a note that carries an address](adr/0038-a-link-is-a-note-that-carries-an-address.md).

## The gesture

**Import an address** in the palette asks for one on a step of its own. The step offers a row for somewhere a browser would go and for nothing else; the vault reads the address again and is what refuses one nothing can be fetched from.

What it makes is a note carrying `type: link` and `url`. What is at the address is fetched as the note is made, and the note takes the name what is there calls itself — a note the person has since named keeps the name they gave it.

At the command line the same run is `numen-cli import <vault> <note>`, and `--again` asks a site for its words afresh.

## What is fetched

| At the address | What is kept | Under |
| --- | --- | --- |
| a video somebody published words for | those words, as WebVTT | `captions` |
| a video nobody published words for | that answer, so the address is not asked again | `captions` |
| anything else | the article the page is written around | `article` |

Every file is named by the hash of the address, in the one form every spelling of it reaches, so two notes pointing at one video share what was fetched and typing in either of them keeps it.

| File | What it holds |
| --- | --- |
| `<hash>.vtt` | the words published with a video |
| `<hash>.txt` | the prose of a page |
| `<hash>.answer` | why there are no words |
| `<hash>.json` | what fetched it: the address, the title, the tool |

One language is asked for and not a list: a site that publishes a machine translation into every language it knows answers a request for the lot by refusing it. What a person published is preferred over what a machine wrote; among those, the languages `importing.captions` names, and then the language the video was spoken in.

## What a search answers

A link note is cut over its prose and what was fetched, as one text. A search about either is a search about that note, and the passage shows which of the two it fell in — see [A link note's text is its prose and what was fetched](adr/0040-a-link-notes-text-is-its-prose-and-what-was-fetched.md).

## The tab

The video plays in a frame, from the hosts the window may frame and no others. The words stand under it, and choosing one plays the video from the moment it was said.

An address nothing plays is the address itself over the prose, and what was fetched from it is searched like any other.

## The tools

`yt-dlp` reaches a video, and `ffmpeg` brings sound to what a transcriber opens. Neither is shipped: each is a program the machine already has, named by a setting holding a command and what it is started through. A machine with neither cannot import, says so, and offers it nowhere afterwards.

A site that refuses an unattended request says so in its own words, and those are what the person is shown. What answers such a refusal — cookies from a browser, a token — is handed to every run through `importing.yt_dlp.arguments`.

## Settings

Under `importing`. See [Settings](settings.md).
