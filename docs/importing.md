# Importing an address

A person pastes an address into the palette and gets a link: a note of type `link` that holds the address. The Media Importer plugin (`tools-media-importer`) fetches what is published there into a text version of the note, copies a video to play from this disk, and opens the note in its tab. The core knows nothing of links beyond what any plugin's note types, text versions and copies give it.

## The gesture

**Import from URL** in the palette is the plugin's command `import`. It asks for the address, and the plugin makes a note of it, named by the address:

```markdown
---
id: 01JB2Z6Q8M4T7V1X3K5N9P0RSW
type: link
url: https://www.youtube.com/watch?v=A4OZ4L9TCpM
---
```

What a person writes below the frontmatter is theirs; the tab shows the address, the copy and the text.

At the command line the same run is `numen-cli import <vault> --address <url>`. It waits for the text; `--wait` also waits for the copy of a video, which is otherwise made in the background and needs the window. Over a link note, `numen-cli download-text <vault> <note>` fetches the text again, `download-copy` copies the video and `delete-copy` removes the copy; these three run over notes of type `link` and are offered in the tab, not the palette.

An address dragged out of a browser onto the files tree is the same gesture: the plugin registers a drop handler for `text/uri-list`, and the tree hands it the drop and the folder of its selection without knowing the plugin.

## What is fetched

| At the address                          | The link's text version (`kind`)                                        |
| --------------------------------------- | ----------------------------------------------------------------------- |
| a video somebody published captions for | the captions, one segment each, with `t` in the selector (`transcript`) |
| anything else                           | the prose the page is written around (`article`)                        |

The plugin fetches the text when it makes the link, and when the link's tab opens on a link that has none. An address that publishes nothing is marked, and is not asked again until a person asks. Deleting the text version through the file's menu keeps the text deleted: the tab says so, and only **Download text again** in the tab fetches it again.

Everything is kept in the plugin's folder of the vault's store, `.numen/md.numen.tools.media.importer/`. The text is kept under the hash of the address, `<key>`, and the copy under the content hash of the note, `<hash>`:

| File          | What it holds                                                         |
| ------------- | --------------------------------------------------------------------- |
| `<key>.jsonl` | the text, in the common format of text versions                       |
| `<key>.json`  | what fetched it: the address, the title, the tool, the kind, the copy |
| `<key>.none`  | an address that publishes nothing                                     |
| `<hash>.mp4`  | the copy of a video; `<hash>.part` while it arrives                   |

A record of a kind whose `.jsonl` is gone is a text the person deleted. An edit of the note changes its hash; the plugin puts the stored text again as a version of the edited note and renames the copy, without fetching.

## What a search answers

The text version is the link's text, cut as any source's is. A search about a lecture lands on the link.

## A copy

Importing the address of a video copies it, up to the size `copy_max_size_mb` names, through the core's queue and shown as a task. The plugin declares `playback` for notes of type `link`, so `MediaService.GetMediaSource` of a link note returns the address the copy is played from on the window's loopback socket, and the link's tab plays it offline. The tab offers **Download video** and **Delete downloaded video**.

While no copy is kept, the tab frames the site's player. The plugin declares `frames: ["https://www.youtube-nocookie.com"]` and serves `embed?address=<url>` on its route, a page that frames the player. The tab loads that page from the window's loopback socket (`MediaService.GetPageSource`), not from the window's own scheme: a page of a scheme that is not http sends no Referer, and the site refuses to play for a page that sends none. The loopback page is served with `Referrer-Policy: strict-origin-when-cross-origin`.

## The tools

A page is fetched by the plugin itself, an ordinary request with the prose found by a library, so every machine reaches one. `yt-dlp` reaches what a video site publishes, and `ffmpeg` joins what it fetched. Neither is shipped and neither is required: a machine without `yt-dlp` imports pages and copies no video. Under `-debug-mock-ml` the plugin reaches video sites through a stand-in that publishes captions and copies a clip of quiet sound, offline.

## Settings

The plugin's own, see [Settings](settings.md).
