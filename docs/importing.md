# Importing an address

A person pastes an address into the palette and gets a link: a file that holds the address. The Media Importer plugin (`tools-media-importer`) fetches what is published there into a text version of the link, copies a video to play from this disk, and opens the link in its tab. The core knows nothing of links beyond what any plugin's text versions and copies give it.

The file holds the address and nothing else. What a person writes about what is there is a note of their own, pointing at the file.

## The gesture

**Import an address** in the palette asks for one on a step of its own, and the plugin makes a `.url` file of it, named by the address:

```ini
[InternetShortcut]
URL=https://www.youtube.com/watch?v=A4OZ4L9TCpM
```

That is what every system calls such a file, so a file manager and a browser open it too. An address dragged out of a browser onto the files tree is the same gesture.

At the command line the same run is `numen-cli import <vault> --address <url>`, a command the plugin declares. It waits for the text and, for a video, the copy. Over a link, `numen-cli download-text <vault> <file>` fetches the text again, `download-copy` copies the video and `delete-copy` takes the copy away; the palette offers the same three over a link.

## What is fetched

| At the address                          | The link's text version                          |
| --------------------------------------- | ------------------------------------------------ |
| a video somebody published captions for | the captions, timed text                         |
| anything else                           | the prose the page is written around, plain text |

An address that publishes nothing is kept as such, and the plugin is not asked for it again until a person asks. The plugin runs on its own over a link with no text, so a link made any other way is fetched too. Deleting the text version through the file's menu keeps the link from being fetched on its own again.

Everything is kept in the plugin's folder of the vault's store, `.numen/md.numen.tools.media.importer/`, under the content hash of the link file, so two links of one address share it and renaming the link keeps it:

| File          | What it holds                                                 |
| ------------- | ------------------------------------------------------------- |
| `<hash>.txt`  | the prose of a page                                           |
| `<hash>.vtt`  | the captions of a video, as WebVTT                            |
| `<hash>.json` | what fetched it: the address, the title, the tool             |
| `<hash>.none` | an address that publishes nothing, or a text a person deleted |
| `<hash>.mp4`  | the copy of a video; `<hash>.part` while it arrives           |

## What a search answers

The text version is the link's text, cut as any source's is. A search about a lecture lands on the link, and what a person wrote about it is a second hit in their own note.

## A copy

Importing the address of a video copies it, up to the size `copy_max_size_mb` names, through the core's queue and shown as a task. The plugin declares `playback` for `.url` files, so `RecordingService.GetRecording` of a link answers the address the copy is played from on the window's loopback socket, and the link's tab plays it: offline, and with nothing of the site it came from loaded.

## The link's commands

Over a `.url` file the plugin declares three commands, offered in the palette under **This file** and by `numen-cli`: **Download the text again** (`download-text`) fetches the link's text again, **Download a copy** (`download-copy`) copies the video the link points at, and **Delete the copy** (`delete-copy`) removes that copy from the plugin's folder.

## The tools

A page is fetched by the plugin itself — an ordinary request, and the prose found by a library — so every machine reaches one. `yt-dlp` reaches what a video site publishes, and `ffmpeg` joins what it fetched. Neither is shipped and neither is required: a machine without `yt-dlp` imports pages, and copies no video. Under `-debug-mock-ml` the plugin reaches video sites through a mock that publishes captions and copies a clip of quiet sound, offline.

## Settings

The plugin's own, see [Settings](settings.md).
