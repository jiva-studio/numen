# Media Importer Plugin

Keeps web addresses as link notes (`type: link`, the address under `url`), fetches what is published there into text versions of them, and copies videos to play from this disk. See [Importing an address](../../docs/importing.md).

## Features

- Declares the note type `link` and a text input of notes of that type. An article is an `article` version of its link, captions a `transcript` one, both in the common JSON Lines format.
- Declares `playback` of `text/markdown`: a copy of a video is kept in the plugin's store folder as `<hash>.<ext>`, `<hash>` the content hash of the link note, and played through `numen.v1.MediaService`.
- Serves its own `numen.links.v1.LinkService` (`proto/numen/links/v1/link.proto`, generated with `make plugin_importer_generate`) to the link's tab, and the command `import` (**Import from URL**).
- Reaches a video site with `yt-dlp` and `ffmpeg`, from the paths the settings name or `PATH`; neither is required. Under `NUMEN_MOCK_ML` a video site is reached through a stand-in that copies a clip of quiet sound, offline.
