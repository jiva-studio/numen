# Media Importer Plugin

Keeps web addresses as links (`.url` files), fetches what is published there into text versions of them, and copies videos to play from this disk. See [Importing an address](../../docs/importing.md).

## Features

- Declares a text input of `.url` files, run on its own: an article is a plain text version of its link, captions a timed one.
- Declares `playback` of `.url` files: a copy of a video is kept in the plugin's store folder as `<hash>.<ext>` and played through `RecordingService`.
- Serves `numen.v1.ArticleService` (`CreateLink`, `GetLink`, `ReadArticle`) to the link's tab, and the commands `import`, `download-text`, `download-copy` and `delete-copy`.
- Reaches a video site with `yt-dlp` and `ffmpeg`, from the paths the settings name or `PATH`; neither is required. Under `NUMEN_MOCK_ML` a video site is reached through a mock that copies a clip of quiet sound, offline.
