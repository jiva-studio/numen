# Importing an address

A person pastes an address into the palette and gets a file: the video plays in its tab, the words said in it stand under the player, and a search over the vault answers about them.

The file holds the address and nothing else. What a person writes about what is there is a note of their own, pointing at the file — see [A url is a source of its own](adr/0038-a-url-is-a-source-of-its-own.md).

## The gesture

**Import an address** in the palette asks for one on a step of its own. The step offers a row for somewhere a browser would go and for nothing else; the vault reads the address again and is what refuses one nothing can be fetched from.

What it makes is a `.url` file:

```ini
[InternetShortcut]
URL=https://www.youtube.com/watch?v=A4OZ4L9TCpM
```

That is what every system calls such a file, so a file manager and a browser open it too. What is at the address is fetched as the file is made, and the file takes the name what is there calls itself — one the person has since renamed keeps the name they gave it.

An address dragged out of a browser onto the files tree is the same gesture: it arrives as text and not as a file, and what is made of it is the file that address is kept in.

At the command line the same run is `numen-cli import <vault> <url>`, and `--again` asks a site for its words afresh.

An agent asks for one with `url_import`, which makes the file and fetches what is at the address in one call, and takes `copy` for the video itself. A build that reaches no address serves the tool nowhere — see [Agents](agents.md).

## What is fetched

| At the address | What is kept | Which it is |
| --- | --- | --- |
| a video somebody published words for | those words, as WebVTT | a transcript |
| a video nobody published words for | that answer, so the address is not asked again | |
| anything else | the prose the page is written around | an article |

A transcript is a transcript whoever wrote it down: the words a site published with a video and the words a model here heard in a recording are one kind, read and put right by one editor, and what separates them is only the producer. An article is neither — it carries no times, and no places on pages either, which is what separates it from a reading off a scan.

Every file is named by the hash of the address, in the one form every spelling of it reaches, so two urls pointing at one video share what was fetched and renaming either keeps it.

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

What was fetched is the url's text, cut as any source's is. A search about a lecture lands on the url, and what a person wrote about it is a second hit in their own note.

## The tab

It is the tab a recording opens in. The video plays in a frame, from the hosts the window may frame and no others; the words stand under it, one line a cue with its time in the gutter, and choosing one plays the video from the moment it was said.

An address nothing plays is the address itself, and what was fetched from it is searched like any other.

## A copy

**Download a copy** fetches the video itself onto this disk, and the tab plays that instead of the frame: it plays offline, and nothing of the site it came from is loaded to play it. At the command line it is `numen-cli import <vault> <url> --copy`.

It is asked for by hand. An hour of video on somebody's disk is not what pasting an address asks for, and `importing.copy_max_size_mb` is what a copy may run to at all.

The copy is kept in the vault's own folder, named by the address like everything else fetched for it, and it is played from there — the vault's own reader is refused that folder, so what serves it is the same socket, reading the store. `importing.copies_to_vault` keeps it beside the url instead, under that file's own name, where the person sees it in their folder and every other program on the machine can play it. Taking the copy away leaves the url pointing where it pointed, and the tab frames the address again.

## What goes when the file goes

Deleting a url deletes what was fetched for it: the words, the prose, the copy. It is named by the address and shared by every url carrying it, so it goes when the last of them does, and two urls on one video keep it while either stands. It is the same sweep that takes away the reading of a document that left the vault.

## Without being asked

`importing.fetch_unasked` reaches the address of every url nothing has been fetched for, as a walk of the vault finds it. It is off: reaching off the machine is a gesture, and a file dropped in from a browser is not one. Turned on, a url that arrived elsewhere has what is at its address by the time it is opened, and an address that will not answer is that file's trouble and leaves the walk standing.

## The tools

Each source of what is published at an address is a provider of its own, and which one answers is which one supports that address.

A page is fetched in this process — an ordinary request, and the prose found by a library — so every machine reaches one. `yt-dlp` reaches what a site publishes as a video, on the sites that tool knows, and `ffmpeg` brings its sound to what a transcriber opens. Neither is shipped: each is a program the machine already has, named by a setting holding a command and what it is started through. A machine without `yt-dlp` imports pages and says what is missing when a video is asked for.

A site with an API of its own is another provider and nothing more: it says which addresses it supports, and it is written as the whole of that site's answer.

A site that refuses an unattended request says so in its own words, and those are what the person is shown. What answers such a refusal — cookies from a browser, a token — is handed to every run through `importing.yt_dlp.arguments`.

## Settings

Under `importing`. See [Settings](settings.md).
