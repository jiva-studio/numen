# Media Importer

Save web articles and online videos (such as YouTube lectures) into your vault for offline reading, watching, and searching.

<!-- more -->

## What You Can Do

- **Import a link**: Run **Import link** with a web address to make a link note for it. An address that already has a link note in the vault opens that note instead of making a second one.
- **Save the text of a link**: Run **Download captions** on a link note to save a clean copy of an article, or the captions of a video, as the note's text. It can be read and searched from then on.
- **Download the media of a link**: Run **Download media** on a link note and pick **Video** or **Audio** (the sound alone) to watch or listen offline in the Media Viewer. Link notes of the same address share one download. **Delete media** takes the download off that note, and the file is deleted once no link note refers to it.
- **Ask the agent to import a link**: The agent in the chat panel can save an address with the tool `url_import`, which makes the link note, or finds the one the address has, fetches its text and, when requested, downloads the video.

## Settings

- **yt-dlp**: Location of the `yt-dlp` program used to download online videos (detected automatically when installed on your system).
- **ffmpeg**: Location of the `ffmpeg` program used to process downloaded audio and video (detected automatically when installed on your system).
- **Caption languages**: Caption languages in order of preference, comma-separated.
- **Automatic captions**: Use auto-generated captions when a site has no others.
- **Download captions automatically**: Download the captions or the article of every link note added to the vault that has no text yet.
- **Maximum download size**: The largest download in megabytes; 0 for no limit.
