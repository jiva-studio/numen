# Importing an address

A person pastes an address into the palette and gets a link: a note of type `link` that holds the address. The Media Importer plugin (`tools-media-importer`) fetches what is published there into a text version of the note, and copies a video to play from this disk. The note opens in the Media Viewer (`views-media-viewer`), whose tab kind takes notes of type `link` and notes with `media`. The core has no special handling for links; it knows only the note types, text versions and copies that plugins register.

## The gesture

**Import link** in the palette is the shared command `app:link.import`, which the plugin handles. The command declares its argument `address` in `libs/core/protocol/commands.json`; the palette asks for it, and the plugin creates a note for it, named after the address:

```markdown
---
id: 01JB2Z6Q8M4T7V1X3K5N9P0RSW
type: link
url: https://www.youtube.com/watch?v=A4OZ4L9TCpM
---
```

The plugin does not manage what a person writes below the frontmatter; the viewer shows the copy, or the site's player, and the text.

At the command line the same run is `numen-cli app:link.import <vault> --address <url>`. Over a link note, `numen-cli app:file.createText <vault> <note>` fetches the text, `app:link.downloadMedia` downloads the video and `app:link.deleteMedia` removes it; the plugin handles these shared commands over notes of type `link`.

An address dragged out of a browser onto the files tree is the same gesture: the plugin registers a drop handler for `text/uri-list`, and the tree passes it the drop and the selected folder, without knowing the plugin.

## What is fetched

| At the address                          | The link's text version (`kind`)                                        |
| --------------------------------------- | ----------------------------------------------------------------------- |
| a video somebody published captions for | the captions, one segment each, with `t` in the selector (`transcript`) |
| anything else                           | the prose the page is written around (`article`)                        |

The plugin fetches the text when it makes the link. A run of `app:file.createText` (**Download captions**) fetches the text whoever started it, a person or the plugin's `file.added` trigger (setting `downloadCaptionsAutomatically`). Deleting the text version through the file's menu keeps the text deleted until **Download captions** fetches it again; with `downloadCaptionsAutomatically` on, the next pass of the trigger fetches it again.

Everything is kept in the plugin's folder of the vault's store, `.numen/md.numen.tools.media.importer/`. The text is kept under the hash of the address, `<key>`, and so are the copy and the page:

| File          | What it holds                                                            |
| ------------- | ------------------------------------------------------------------------ |
| `<key>.jsonl` | the text, in the common format of text versions                          |
| `<key>.json`  | how it was fetched: the address, the title, the tool, the kind, the copy |
| `<key>.mp4`   | the copy of a video; `<key>.part` while the download is in progress      |
| `<key>.html`  | the page that frames the site's player                                   |

A record of a kind whose `.jsonl` is gone is a text the person deleted. The text version is rooted in the note's `url`: an edit elsewhere in the note keeps it live, and a new `url` makes it not live.

## What a search returns

The text version is the link's text, split into passages as any source's text is. A search that matches a lecture returns the link.

## A copy

Importing an address (`app:link.import`) makes the link note and nothing else; an address that already has a link note anywhere in the vault opens that note, and `url_import` returns it. **Download media** (`app:link.downloadMedia`) offers Video and Audio (the sound alone) for a link whose address publishes a video, and downloads the one picked, up to the size `copyMaxSizeMb` sets, into the plugin's folder of the vault's store, through the core's queue and shown as a task; nothing downloaded goes into the vault. The plugin writes the address of the download into the note as `media: plugin://md.numen.tools.media.importer/<key>.video.mp4` (`<key>.audio.m4a` for the sound alone), and of the page that frames the player as `page:`, which `resolveAddress` turns into the address the window plays or frames it from. The link notes of one address share its download. **Delete media** (`app:link.deleteMedia`), or a switch between video and audio, takes `media` out of that note alone; the download is deleted once no link note refers to it, and a delete during a running download waits for it to end. **Download captions** (`app:file.createText`) fetches the text of a link note; with the setting `downloadCaptionsAutomatically` on, it runs when a link note is added.

While no media are downloaded, the Media Viewer frames the site's player. The plugin declares `contributes.frameOrigins: ["https://www.youtube-nocookie.com"]` and writes a page that frames the player into its folder of the vault's store when it makes the link note, at the address the note holds under `page`. The viewer loads that page from the window's loopback socket (`resolveAddress`), not from the window's own scheme: a page of a scheme that is not http sends no Referer, and the site refuses to play for a page that sends none. The loopback page is served with `Referrer-Policy: strict-origin-when-cross-origin`.

## The tools

A page is fetched by the plugin itself, an ordinary request with the prose found by a library, so it works on every machine. `yt-dlp` fetches what a video site publishes, and `ffmpeg` merges what it fetched. Neither is shipped and neither is required: a machine without `yt-dlp` imports pages and copies no video. Under `-debug-mock-ml` the plugin uses a stand-in for video sites that publishes captions and copies a clip of quiet sound, offline.

## Settings

The plugin's own, see [Settings](settings.md).
