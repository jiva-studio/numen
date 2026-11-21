# Importing an address

A person pastes an address into the palette and gets a link: a note of type `link` that holds the address. The Media Importer plugin (`tools-media-importer`) fetches what is published there into a text version of the note, and copies a video to play from this disk. The note opens in the Media Viewer (`views-media-viewer`), whose tab kind takes notes of type `link` and notes with `media`. The core has no special handling for links; it knows only the note types, text versions and copies that plugins register.

## The gesture

**Import link** in the palette is the command `link.import` of the plugin. Its manifest declares the argument `address`; the palette asks for it, and the plugin creates a note for it, named after the address:

```markdown
---
id: 01JB2Z6Q8M4T7V1X3K5N9P0RSW
type: link
url: https://www.youtube.com/watch?v=A4OZ4L9TCpM
---
```

The plugin does not manage what a person writes below the frontmatter; the viewer shows the copy, or the site's player, and the text.

At the command line the same run is `numen-cli link.import <vault> --address <url>`. Over a link note, `numen-cli app:file.createText <vault> <note>` fetches the text, which the plugin handles as a shared command over notes of type `link`; its own commands `link.downloadMedia` and `link.deleteMedia` download the video and remove it.

An address dragged out of a browser onto the files tree is the same gesture: the plugin registers a drop handler for `text/uri-list`, and the tree passes it the drop and the selected folder, without knowing the plugin. A dropped address names its note after the title of the page, fetched with the timeout and the size limit of any page fetch (15 seconds, 8 MiB); a title that is no file name is reduced to one and kept as the `title` of the note, and a name taken by another file gets the suffix 1, 2, 3 and so on. When the title cannot be fetched the note is named after the address.

## What is fetched

| At the address                          | The link's text version (`kind`)                                        |
| --------------------------------------- | ----------------------------------------------------------------------- |
| a video somebody published captions for | the captions, one segment each, with `t` in the selector (`transcript`) |
| anything else                           | the prose the page is written around (`article`)                        |

The plugin fetches the text when it makes the link. A run of `app:file.createText` (**Download captions**) fetches the text of one link note. With the setting `downloadCaptionsAutomatically` on, the plugin reads the file change feed of the vault from the cursor it keeps in its store folder (`links.cursor.json`), on its start, when a setting changes and on each `onFilesChanged` event of the vault, and fetches, one note at a time, the text of each link note added or given another address since that has no text version of the plugin from the address it holds. A moved note keeps its text. When the feed resets, and when the setting is turned on, it does so over every link note of the vault. Deleting the text version through the file's menu keeps the text deleted until **Download captions** fetches it again, or the address of the note changes.

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

Importing an address (`link.import`) makes the link note and nothing else; an address that already has a link note anywhere in the vault opens that note, and `url_import` returns it. **Download media** (`link.downloadMedia`) offers Video and Audio (the sound alone) for a link whose address publishes a video, and downloads the one picked, up to the size `copyMaxSizeMb` sets, into the plugin's folder of the vault's store, through the core's queue and shown as a task; nothing downloaded goes into the vault. The plugin writes the address of the download into the note as `media: plugin://md.numen.tools.media.importer/<key>.video.mp4` (`<key>.audio.m4a` for the sound alone), and of the page that frames the player as `page:`, which `resolveAddress` turns into the address the window plays or frames it from. The link notes of one address share its download. **Delete media** (`link.deleteMedia`), or a switch between video and audio, takes `media` out of that note alone; the download is deleted once no link note refers to it, and a delete during a running download waits for it to end. **Download captions** (`app:file.createText`) fetches the text of a link note; with the setting `downloadCaptionsAutomatically` on, the plugin fetches the text of a link note that is added without one.

While no media are downloaded, the Media Viewer frames the site's player. The plugin declares `contributes.frameOrigins: ["https://www.youtube-nocookie.com"]` and writes a page that frames the player into its folder of the vault's store when it makes the link note, at the address the note holds under `page`. The viewer loads that page from the window's loopback socket (`resolveAddress`), not from the window's own scheme: a page of a scheme that is not http sends no Referer, and the site refuses to play for a page that sends none. The loopback page is served with `Referrer-Policy: strict-origin-when-cross-origin`.

## The tools

A page is fetched by the plugin itself, an ordinary request with the prose found by a library, so it works on every machine. `yt-dlp` fetches what a video site publishes, and `ffmpeg` merges what it fetched. Neither is shipped. A machine that holds neither gets each downloaded on first use into the application's dependency folder from the address and checksum the plugin's manifest pins, so a file that does not match is refused and a platform with no pinned file is reported by name. A newer version is a new pinned version in the manifest. A program found on the machine or named in the settings is used instead. Under `-debug-mock-ml` the plugin uses a stand-in for video sites that publishes captions and copies a clip of quiet sound, offline.

## Settings

The plugin's own, see [Settings](settings.md).
